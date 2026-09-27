#!/usr/bin/env python3
"""Exercise the actual PostgreSQL 16 image and startup scripts on disposable volumes."""
import os
from pathlib import Path
import subprocess
import time
import unittest
import uuid

ROOT = Path(__file__).resolve().parents[1]
IMAGE = os.environ.get("SC_TEST_POSTGRES_IMAGE", "postgres:16-alpine")


def docker(*args, check=True, stdin=None):
    result = subprocess.run(["docker", *args], input=stdin, text=True,
                            capture_output=True, timeout=120, check=False)
    if check and result.returncode:
        raise AssertionError(f"Docker operation {args[0]} failed: {result.stderr}")
    return result


class DatabaseStartupTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        docker("pull", IMAGE)

    def setUp(self):
        self.volume = "sc-db-test-" + uuid.uuid4().hex[:12]
        self.container = self.volume + "-server"
        self.password = "Test-'quoted-\\-:@/?#%-$-password"
        self.user = "sovereign"
        self.database = "sovereign_conquest"
        docker("volume", "create", self.volume)

    def tearDown(self):
        docker("rm", "-f", self.container, check=False)
        docker("volume", "rm", self.volume, check=False)

    def start(self, *, raw=False, user=None, password=None, database=None, extra=(), ready=True):
        docker("rm", "-f", self.container, check=False)
        self.user = self.user if user is None else user
        self.password = self.password if password is None else password
        self.database = self.database if database is None else database
        args = ["run", "-d", "--name", self.container, "--network", "none",
                "-v", f"{self.volume}:/var/lib/postgresql/data",
                "-v", f"{ROOT / 'docker/db'}:/opt/sc-db:ro",
                "-e", f"POSTGRES_USER={self.user}",
                "-e", f"POSTGRES_PASSWORD={self.password}",
                "-e", f"POSTGRES_DB={self.database}"]
        if not raw:
            args += ["--entrypoint", "bash", "--health-cmd", "bash /opt/sc-db/check.sh",
                     "--health-interval", "1s", "--health-timeout", "3s",
                     "--health-retries", "90"]
        args += list(extra) + [IMAGE]
        if not raw:
            args += ["/opt/sc-db/entrypoint.sh", "postgres"]
        docker(*args)
        if ready:
            self.wait_ready(raw=raw)

    def wait_ready(self, *, raw=False):
        started = time.monotonic()
        deadline = started + 90
        while time.monotonic() < deadline:
            state = docker("inspect", "-f", "{{.State.Status}}", self.container).stdout.strip()
            if state != "running":
                logs = docker("logs", self.container)
                self.fail("Database exited before readiness: " + logs.stdout + logs.stderr)
            if raw:
                # Force TCP: the official init server accepts only Unix sockets.
                if self.query("SELECT 1", check=False).returncode == 0:
                    print(f"Legacy database ready in {time.monotonic() - started:.2f}s", flush=True)
                    return
            else:
                result = docker("exec", self.container, "bash", "/opt/sc-db/check.sh", check=False)
                if result.returncode == 0:
                    print(f"Reconciled database ready in {time.monotonic() - started:.2f}s", flush=True)
                    return
            time.sleep(0.25)
        self.fail("Database readiness timeout")

    def query(self, sql, *, password=None, user=None, database=None, check=True):
        return docker("exec", "-i", "-e", "PGHOST=127.0.0.1", "-e", "PGPORT=5432",
                      "-e", f"PGPASSWORD={self.password if password is None else password}",
                      "-e", f"PGUSER={self.user if user is None else user}",
                      "-e", f"PGDATABASE={self.database if database is None else database}",
                      self.container, "psql", "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1",
                      check=check, stdin=sql)

    def stop(self):
        docker("stop", "-t", "30", self.container)

    def seed(self):
        self.query("CREATE TABLE retained_game_data (value text); INSERT INTO retained_game_data VALUES ('keep');")

    def assert_retained(self, **kwargs):
        self.assertEqual(self.query("SELECT value FROM retained_game_data", **kwargs).stdout.strip(), "keep")

    def assert_failed_start(self):
        result = docker("wait", self.container)
        self.assertNotEqual(result.stdout.strip(), "0")
        state = docker("inspect", "-f", "{{.State.Health.Status}}", self.container).stdout.strip()
        self.assertNotEqual(state, "healthy")

    def test_fresh_then_unchanged_restart_preserves_password_hash_and_data(self):
        self.start()
        self.seed()
        before = self.query("SELECT rolpassword FROM pg_authid WHERE rolname = current_user").stdout
        self.stop()
        self.start()
        self.assert_retained()
        self.assertEqual(before, self.query("SELECT rolpassword FROM pg_authid WHERE rolname = current_user").stdout)
        self.assertNotEqual(self.query("SELECT 1", password="incorrect", check=False).returncode, 0)
        logs = docker("logs", self.container)
        self.assertNotIn(self.password, logs.stdout + logs.stderr)
        self.assertIn('"event":"credentials_verified"', logs.stdout)
        self.assertNotIn('"event":"reconciling_database_and_role"', logs.stdout)

    def test_password_rotation_on_legacy_volume_preserves_data(self):
        self.start(raw=True)
        self.seed()
        old_password = self.password
        self.stop()
        self.start(password="New-'quoted-\\-:@/?#%-$-password")
        self.assert_retained()
        self.assertNotEqual(self.query("SELECT 1", password=old_password, check=False).returncode, 0)
        logs = docker("logs", self.container)
        self.assertNotIn(self.password, logs.stdout + logs.stderr)
        self.assertIn('"event":"reconciling_database_and_role"', logs.stdout)
        # Reverting the environment and recreating the container restores the old credential.
        self.stop()
        self.start(password=old_password)
        self.assert_retained()

    def test_live_password_repair_preserves_rows_and_is_idempotent(self):
        self.start(raw=True)
        self.seed()
        old_password = self.password
        before = self.query("SELECT rolpassword FROM pg_authid WHERE rolname=current_user").stdout
        self.password = "Live-'quoted-\\-:@/?#%-$-password"
        def repair():
            return docker("exec", "-e", f"POSTGRES_PASSWORD={self.password}",
                          self.container, "bash", "/opt/sc-db/repair-password.sh")
        result = repair()
        self.assertIn('"event":"password_updated"', result.stdout)
        self.assertNotIn(self.password, result.stdout + result.stderr)
        self.assert_retained()
        self.assertNotEqual(before, self.query("SELECT rolpassword FROM pg_authid WHERE rolname=current_user").stdout)
        # Raw official images can trust localhost. Start the SCRAM-enforcing
        # wrapper before proving that the old password is rejected over TCP.
        self.stop()
        self.start()
        self.assertNotEqual(self.query("SELECT 1", password=old_password, check=False).returncode, 0)
        self.assertNotIn('"event":"reconciling_database_and_role"', docker("logs", self.container).stdout)
        self.assertIn('"event":"password_updated"', repair().stdout)
        self.assert_retained()

    def test_live_repair_requires_existing_local_administrator(self):
        self.start(raw=True)
        self.seed()
        result = docker("exec", "-e", "POSTGRES_USER=missing_role", self.container,
                        "bash", "/opt/sc-db/repair-password.sh", check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('"event":"local_administrator_password_repair_failed"', result.stderr)
        self.assertNotIn(self.password, result.stdout + result.stderr)
        self.assert_retained()

    def test_missing_database_is_created_without_removing_old_database(self):
        self.start(raw=True)
        self.seed()
        old_database = self.database
        self.stop()
        self.start(database="new_database")
        self.assertEqual(self.query("SELECT current_database()").stdout.strip(), "new_database")
        self.assert_retained(database=old_database)

    def test_missing_role_and_database_with_renamed_bootstrap_admin(self):
        self.start(raw=True, user="legacy admin")
        # A role cannot rename the current session user. Use a second administrator.
        self.query("CREATE ROLE maintenance LOGIN SUPERUSER PASSWORD 'maintenance-test';")
        self.query('ALTER ROLE "legacy admin" RENAME TO "renamed admin";', user="maintenance", password="maintenance-test")
        self.stop()
        self.start(user="new application", database="new app database")
        self.assertEqual(self.query("SELECT rolsuper, rolcreatedb, rolcreaterole FROM pg_roles WHERE rolname = current_user").stdout.strip(), "f|f|f")
        self.query("CREATE TABLE owned_by_app (id integer)")

    def test_missing_role_on_existing_database_keeps_ownership(self):
        self.start(raw=True, user="legacy_owner")
        self.seed()
        self.stop()
        self.start(user="new_login")
        self.assertEqual(self.query("SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname=current_database()").stdout.strip(), "legacy_owner")
        self.assert_retained(user="legacy_owner")

    def test_disabled_and_expired_login_is_repaired(self):
        self.start()
        self.query("ALTER ROLE sovereign NOLOGIN VALID UNTIL '2000-01-01';")
        self.stop()
        self.start()
        self.assertEqual(self.query("SELECT rolcanlogin, rolvaliduntil FROM pg_roles WHERE rolname=current_user").stdout.strip(), "t|infinity")

    def test_quoted_names_and_connection_string_looking_names(self):
        self.start(raw=True)
        self.stop()
        user = 'new"; SELECT 1; -- role'
        database = "host=unreachable dbname='app'"
        self.start(user=user, database=database)
        self.assertEqual(self.query("SELECT current_user").stdout.strip(), user)
        self.assertEqual(self.query("SELECT current_database()").stdout.strip(), database)

    def test_password_is_checked_even_with_legacy_trust_hba(self):
        self.start(raw=True, extra=("-e", "POSTGRES_HOST_AUTH_METHOD=trust"))
        self.stop()
        self.start(password="changed-password-on-trust-volume")
        stored = self.query("SELECT rolpassword FROM pg_authid WHERE rolname=current_user").stdout
        self.assertTrue(stored.startswith("SCRAM-SHA-256$"))
        logs = docker("logs", self.container).stdout
        self.assertIn('"event":"reconciling_database_and_role"', logs)
        self.assertNotEqual(self.query("SELECT 1", password="wrong", check=False).returncode, 0)
        self.assertEqual(self.query("SELECT count(*) FROM pg_hba_file_rules WHERE error IS NOT NULL").stdout.strip(), "0")

    def test_unrelated_disabled_administrator_is_not_enabled(self):
        self.start(raw=True)
        self.query("ALTER ROLE sovereign NOLOGIN;")
        self.stop()
        self.start(user="another_user", database="another_database", ready=False)
        self.assert_failed_start()
        logs = docker("logs", self.container)
        self.assertIn('"event":"administrator_login_disabled"', logs.stdout + logs.stderr)

    def test_unrepairable_database_fails_closed_and_preserves_data(self):
        self.start()
        self.seed()
        self.query("ALTER DATABASE sovereign_conquest ALLOW_CONNECTIONS false;", database="postgres")
        self.stop()
        self.start(ready=False)
        self.assert_failed_start()
        self.start(raw=True, database="postgres")
        self.query("ALTER DATABASE sovereign_conquest ALLOW_CONNECTIONS true;")
        self.assert_retained(database="sovereign_conquest")

    def test_invalid_configuration_fails_closed(self):
        for kwargs in ({"password": ""}, {"user": "x" * 64}, {"database": "template1"},
                       {"database": "bad\nname"}, {"extra": ("-e", "POSTGRES_HOST_AUTH_METHOD=trust")}):
            with self.subTest(case=list(kwargs)):
                self.user, self.password, self.database = "sovereign", "valid-test-password", "sovereign_conquest"
                self.start(ready=False, **kwargs)
                self.assert_failed_start()

    def test_healthcheck_rejects_wrong_password_and_missing_marker(self):
        self.start()
        self.assertNotEqual(docker("exec", "-e", "POSTGRES_PASSWORD=wrong", self.container,
                                   "bash", "/opt/sc-db/check.sh", check=False).returncode, 0)
        docker("exec", self.container, "rm", "/var/run/postgresql/sc-db-ready")
        self.assertNotEqual(docker("exec", self.container, "bash", "/opt/sc-db/check.sh", check=False).returncode, 0)

    def test_abrupt_restart_recovers_without_data_loss(self):
        self.start()
        self.seed()
        docker("kill", "--signal", "KILL", self.container)
        docker("start", self.container)
        self.wait_ready()
        self.assert_retained()


if __name__ == "__main__":
    unittest.main(verbosity=2)
