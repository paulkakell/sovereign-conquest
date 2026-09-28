#!/usr/bin/env python3
"""Verify startup exit codes and redacted container logs with synthetic secrets."""

import os
import subprocess
import unittest


class StartupValidationTests(unittest.TestCase):
    def run_container(self, overrides):
        settings = {
            "APP_ENV": "development",
            "DATABASE_URL": "postgres://fixture@127.0.0.1:1/fixture?sslmode=disable&connect_timeout=1",
            "PGPASSWORD": "Database-Fixture-Only",
            "JWT_SECRET": "j" * 32,
            "ADMIN_SECRET": "a" * 32,
            "INITIAL_ADMIN_PASSWORD": "p" * 16,
        }
        settings.update(overrides)
        command = ["docker", "run", "--rm", "--network", "none", "--read-only",
                   "--tmpfs", "/tmp:rw,size=32m,mode=1777", "--cap-drop", "ALL",
                   "--security-opt", "no-new-privileges"]
        for key, value in settings.items():
            command.extend(["--env", f"{key}={value}"])
        command.append(os.environ["SC_IMAGE"])
        result = subprocess.run(command, capture_output=True, text=True, timeout=20, check=False)
        output = result.stdout + result.stderr
        self.assertEqual(result.returncode, 1, "API must exit 1 for invalid configuration or the deliberately unavailable database")
        for key in ("PGPASSWORD", "JWT_SECRET", "ADMIN_SECRET", "INITIAL_ADMIN_PASSWORD"):
            value = settings[key]
            if value.strip():
                self.assertNotIn(value, output, "container log exposed a synthetic secret")
        return output

    def test_all_failures_are_reported_in_every_environment(self):
        for environment in ("development", "production"):
            with self.subTest(environment=environment):
                output = self.run_container({"APP_ENV": environment, "PGPASSWORD": "",
                    "JWT_SECRET": "jwt-fixture", "ADMIN_SECRET": "admin-fixture",
                    "INITIAL_ADMIN_PASSWORD": "p" * 73})
                for diagnostic in ("configuration validation failed", "POSTGRES_PASSWORD",
                                   "JWT_SECRET", "ADMIN_SECRET", "INITIAL_ADMIN_PASSWORD", "16-72", "32 bytes"):
                    self.assertIn(diagnostic, output)
                self.assertNotIn("database connection failed", output)

    def test_missing_secrets_do_not_use_defaults(self):
        output = self.run_container({"JWT_SECRET": "", "INITIAL_ADMIN_PASSWORD": ""})
        self.assertIn("JWT_SECRET", output)
        self.assertIn("INITIAL_ADMIN_PASSWORD", output)
        self.assertNotIn("database connection failed", output)

    def test_unicode_password_limit_is_measured_in_bytes(self):
        output = self.run_container({"INITIAL_ADMIN_PASSWORD": "é" * 37})
        self.assertIn("INITIAL_ADMIN_PASSWORD", output)
        self.assertNotIn("database connection failed", output)

    def test_valid_boundaries_and_optional_reset_key_reach_database_connection(self):
        for password in ("p" * 16, "p" * 72, "é" * 36):
            with self.subTest(bytes=len(password.encode())):
                output = self.run_container({"INITIAL_ADMIN_PASSWORD": password, "ADMIN_SECRET": ""})
                self.assertNotIn("configuration validation failed", output)
                self.assertIn("database connection failed", output)


if __name__ == "__main__":
    unittest.main(verbosity=2)
