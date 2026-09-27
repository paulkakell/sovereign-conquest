#!/usr/bin/env python3
"""Exercise both Compose profiles with disposable, synthetic environment files.

No daemon is needed. Set SC_COMPOSE_COMMAND to a standalone Compose executable
when `docker compose` is unavailable. Rendering alone does not test containers.
"""

import json
import os
from pathlib import Path
import re
import shlex
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[1]
PROFILES = ("docker-compose.yml", "deploy/docker-compose.bind.yml")
EXAMPLE = dict(
    line.split("=", 1)
    for line in (ROOT / ".env.example").read_text().splitlines()
    if line and not line.startswith("#")
)
REQUIRED = {
    "POSTGRES_USER": "compose_fixture_user",
    "POSTGRES_PASSWORD": "Compose-Database-Fixture-Only-2026",
    "POSTGRES_DB": "compose_fixture_database",
    "JWT_SECRET": "Compose-Session-Fixture-Only-2026-Long",
    "INITIAL_ADMIN_PASSWORD": "Compose-Admin-Fixture-Only-2026",
}


class ComposeConfigurationTests(unittest.TestCase):
    def render(self, profile, settings, success=True):
        command = shlex.split(os.environ.get("SC_COMPOSE_COMMAND", "docker compose"))
        # Do not let the caller's deployment credentials or COMPOSE_FILE override
        # fixtures. Never write a real .env inside the source checkout.
        environment = {key: os.environ[key] for key in ("PATH", "HOME") if key in os.environ}
        with tempfile.TemporaryDirectory(prefix="sc-compose-test-") as directory:
            env_file = Path(directory) / "fixture.env"
            lines = []
            for key, value in settings.items():
                escaped = value.replace("'", "\\'")
                lines.append(f"{key}='{escaped}'")
            env_file.write_text("\n".join(lines) + "\n")
            result = subprocess.run(
                [*command, "--env-file", str(env_file), "-f", str(ROOT / profile),
                 "config", "--format", "json"],
                cwd=ROOT, env=environment, capture_output=True, text=True,
                timeout=30, check=False,
            )
        if success:
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stderr, "")
            return json.loads(result.stdout)
        self.assertNotEqual(result.returncode, 0)
        return result

    def test_every_example_setting_is_consumed_by_root_compose(self):
        variables = set(re.findall(r"\$\{([A-Z_]+)", (ROOT / PROFILES[0]).read_text()))
        self.assertEqual(set(EXAMPLE), variables)

    def test_example_requires_operator_supplied_secrets(self):
        for key in ("POSTGRES_PASSWORD", "JWT_SECRET", "INITIAL_ADMIN_PASSWORD", "ADMIN_SECRET"):
            self.assertEqual(EXAMPLE[key], "")
        for profile in PROFILES:
            with self.subTest(profile=profile):
                self.render(profile, EXAMPLE, success=False)

    def test_missing_or_empty_required_values_fail_before_startup(self):
        for profile in PROFILES:
            for key in REQUIRED:
                for missing in (True, False):
                    with self.subTest(profile=profile, key=key, missing=missing):
                        settings = dict(REQUIRED)
                        if missing:
                            settings.pop(key)
                        else:
                            settings[key] = ""
                        result = self.render(profile, settings, success=False)
                        self.assertIn(key, result.stderr)

    def test_release_defaults_match_version_and_completed_example(self):
        image = "ghcr.io/paulkakell/sovereign-conquest:" + (ROOT / "VERSION").read_text().strip()
        self.assertEqual(EXAMPLE["SC_IMAGE"], image)
        for profile in PROFILES:
            with self.subTest(profile=profile):
                default = self.render(profile, REQUIRED)["services"]["api"]
                completed = self.render(profile, {**EXAMPLE, **REQUIRED})["services"]["api"]
                self.assertEqual(default["image"], image)
                self.assertEqual(completed["image"], image)
                self.assertEqual(default["environment"], completed["environment"])
                self.assertEqual(default["pull_policy"], "always")
                self.assertEqual(default["environment"]["TRUST_PROXY_HEADERS"], "false")
                self.assertEqual(default["environment"]["ADMIN_SECRET"], "")
                self.assertEqual(default["environment"]["DATABASE_URL"], "postgres://db:5432/?sslmode=disable")
                self.assertEqual(default["environment"]["HTTP_ADDR"], ":8080")
                self.assertEqual(default["environment"]["WEB_ROOT"], "/app/web")

    def test_all_runtime_overrides_reach_both_api_profiles(self):
        overrides = {
            "APP_ENV": "production", "TRUST_PROXY_HEADERS": "true",
            "ADMIN_SECRET": "Compose-Reset-Fixture-Only-2026-Long",
            "INITIAL_ADMIN_USERNAME": "fixture_operator", "UNIVERSE_SEED": "-17",
            "UNIVERSE_SECTORS": "42", "TURN_REGEN_SECONDS": "30",
            "PORT_TICK_SECONDS": "0", "PLANET_TICK_SECONDS": "10",
            "EVENT_TICK_SECONDS": "20", "PROTECTORATE_TICK_SECONDS": "25",
        }
        for profile in PROFILES:
            with self.subTest(profile=profile):
                api = self.render(profile, {**EXAMPLE, **REQUIRED, **overrides})["services"]["api"]
                for key, value in overrides.items():
                    self.assertEqual(api["environment"][key], value)

    def test_image_pull_policy_and_port_overrides(self):
        overrides = {"SC_IMAGE": "example.invalid/conquest@sha256:" + "a" * 64,
                     "SC_PULL_POLICY": "never", "WEB_PORT": "5000", "API_PORT": "5080"}
        for profile in PROFILES:
            with self.subTest(profile=profile):
                api = self.render(profile, {**REQUIRED, **overrides})["services"]["api"]
                self.assertEqual(api["image"], overrides["SC_IMAGE"])
                self.assertEqual(api["pull_policy"], "never")
                expected = {"5000", "5080"} if profile == PROFILES[0] else {"5000"}
                self.assertEqual({port["published"] for port in api["ports"]}, expected)
                for port in api["ports"]:
                    self.assertEqual(port["target"], 8080)
                    self.assertEqual(port["host_ip"], "127.0.0.1")

    def test_default_and_shared_example_ports_are_explicit(self):
        for profile, expected in zip(PROFILES, ({"3000", "8080"}, {"5000"})):
            with self.subTest(profile=profile):
                api = self.render(profile, REQUIRED)["services"]["api"]
                self.assertEqual({port["published"] for port in api["ports"]}, expected)
        api = self.render(PROFILES[1], {**EXAMPLE, **REQUIRED})["services"]["api"]
        self.assertEqual([port["published"] for port in api["ports"]], ["3000"])

    def test_literal_credentials_are_shared_without_url_interpolation(self):
        values = {"POSTGRES_USER": 'fixture"$role', "POSTGRES_DB": 'fixture=db$literal',
                  "POSTGRES_PASSWORD": "fixture'\\:@/?#%-${DO_NOT_EXPAND}$$"}
        for profile in PROFILES:
            with self.subTest(profile=profile):
                services = self.render(profile, {**REQUIRED, **values})["services"]
                for key, api_key in (("POSTGRES_USER", "PGUSER"), ("POSTGRES_DB", "PGDATABASE"),
                                     ("POSTGRES_PASSWORD", "PGPASSWORD")):
                    # Compose escapes dollars for reusable serialized configuration.
                    for service, setting in (("db", key), ("api", api_key)):
                        self.assertEqual(services[service]["environment"][setting].replace("$$", "$"), values[key])
                self.assertNotIn("fixture", services["api"]["environment"]["DATABASE_URL"])

    def test_storage_health_and_hardening_are_preserved(self):
        for profile in PROFILES:
            with self.subTest(profile=profile):
                model = self.render(profile, REQUIRED)
                db, api = model["services"]["db"], model["services"]["api"]
                self.assertEqual(set(model["services"]), {"db", "api"})
                self.assertEqual(db["image"], "postgres:16-alpine")
                self.assertFalse(db.get("ports"))
                self.assertEqual(db["expose"], ["5432"])
                self.assertEqual(db["entrypoint"], ["bash", "/opt/sc-db/entrypoint.sh"])
                self.assertEqual(db["command"], ["postgres"])
                self.assertEqual(db["healthcheck"]["test"], ["CMD", "bash", "/opt/sc-db/check.sh"])
                self.assertEqual(db["healthcheck"]["interval"], "5s")
                self.assertEqual(db["healthcheck"]["timeout"], "3s")
                self.assertEqual(db["healthcheck"]["retries"], 20)
                self.assertEqual(db["healthcheck"]["start_period"], "1m0s")
                self.assertEqual(api["depends_on"]["db"]["condition"], "service_healthy")
                self.assertTrue(api["read_only"])
                self.assertTrue(api["init"])
                self.assertEqual(api["cap_drop"], ["ALL"])
                self.assertIn("no-new-privileges:true", api["security_opt"])
                self.assertIn("/tmp:size=32m,mode=1777", api["tmpfs"])
                for service in (db, api):
                    self.assertEqual(service["restart"], "unless-stopped")
                mount = next(v for v in db["volumes"] if v["target"] == "/opt/sc-db")
                self.assertTrue(mount["read_only"])
                # Some Compose versions omit false-valued options from JSON.
                # An explicit true must still fail this hardening check.
                self.assertFalse(mount.get("bind", {}).get("create_host_path", False))
                data = next(v for v in db["volumes"] if v["target"] == "/var/lib/postgresql/data")
                if profile == PROFILES[0]:
                    self.assertEqual(mount["source"], str(ROOT / "docker/db"))
                    self.assertEqual(data["type"], "volume")
                    self.assertEqual(data["source"], "db_data")
                    self.assertEqual(set(api["networks"]), {"backend"})
                    self.assertEqual(set(db["networks"]), {"backend"})
                else:
                    self.assertEqual(model["name"], "conquest")
                    self.assertEqual(mount["source"], "/dockershare/containers/conquest/config/db")
                    self.assertEqual(data["source"], "/dockershare/containers/conquest/db")
                    self.assertEqual(set(db["networks"]), {"net_internal"})
                    self.assertEqual(set(api["networks"]), {"net_internal", "net_external"})
                    self.assertTrue(model["networks"]["net_external"]["external"])
                    self.assertEqual(model["networks"]["net_external"]["name"], "containers-external")
                    self.assertEqual(model["networks"]["net_internal"]["name"], "conquest-internal")


if __name__ == "__main__":
    unittest.main()
