"""Host-independent failure and secret-handling checks for live DB recovery.

Real PostgreSQL quoting and data preservation are covered in test-db-startup.py.
"""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "docker/db/repair-password.sh"
PSQL = '''#!/usr/bin/env python3
import json, os, pathlib, sys
args = sys.argv[1:]
path = pathlib.Path(os.environ['SC_TEST_WORK'])
host = os.environ.get('PGHOST')
sql = '' if '-c' in args else sys.stdin.read()
with (path / 'calls.jsonl').open('a') as output:
    output.write(json.dumps({'args': args, 'host': host, 'sql': sql,
        'hostaddr': os.environ.get('PGHOSTADDR'),
        'options': os.environ.get('PGOPTIONS')}) + '\\n')
mode = os.environ['SC_TEST_MODE']
if host == '127.0.0.1':
    if mode == 'valid' or (mode == 'repair' and (path / 'repaired').exists()):
        print('1')
        sys.exit(0)
else:
    if mode != 'denied':
        (path / 'repaired').touch()
        sys.exit(0)
print(os.environ['POSTGRES_PASSWORD'] + sql, file=sys.stderr)
sys.exit(2)
'''


class RecoveryTests(unittest.TestCase):
    def run_repair(self, mode, updates=None, args=()):
        with tempfile.TemporaryDirectory(prefix="sc-recovery-test-") as directory:
            path = Path(directory)
            stub = path / "psql"
            stub.write_text(PSQL)
            stub.chmod(0o700)
            env = dict(os.environ, PATH=directory + os.pathsep + os.environ['PATH'],
                       POSTGRES_USER='existing_admin', POSTGRES_DB='conquest',
                       POSTGRES_PASSWORD="Test-'quoted-\\-:$-secret",
                       SC_TEST_WORK=directory, SC_TEST_MODE=mode,
                       PGHOSTADDR='unexpected-host', PGOPTIONS='unexpected-options')
            env.update(updates or {})
            result = subprocess.run(['bash', str(SCRIPT), *args], env=env,
                                    capture_output=True, text=True, timeout=10)
            self.assertNotIn(env['POSTGRES_PASSWORD'], result.stdout + result.stderr)
            calls_path = path / 'calls.jsonl'
            calls = [json.loads(line) for line in calls_path.read_text().splitlines()] if calls_path.exists() else []
            for call in calls:
                self.assertNotIn(env['POSTGRES_PASSWORD'], ' '.join(call['args']))
                self.assertIsNone(call['hostaddr'])
                self.assertIsNone(call['options'])
            return result, calls

    def test_localhost_trust_does_not_skip_password_update(self):
        result, calls = self.run_repair('valid')
        self.assertEqual(result.returncode, 0)
        self.assertIn('password_updated', result.stdout)
        self.assertTrue(any('ALTER ROLE' in call['sql'] for call in calls))

    def test_password_repair_requires_successful_tcp_recheck(self):
        result, calls = self.run_repair('repair')
        self.assertEqual(result.returncode, 0)
        self.assertIn('password_updated', result.stdout)
        self.assertEqual(calls[-1]['host'], '127.0.0.1')

    def test_failed_local_authentication_is_redacted(self):
        result, _ = self.run_repair('denied')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('local_administrator_password_repair_failed', result.stderr)

    def test_unsuccessful_recheck_does_not_report_success(self):
        result, _ = self.run_repair('unrepairable')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('database_login_still_fails', result.stderr)
        self.assertNotIn('password_updated', result.stdout)

    def test_missing_settings_and_arguments_fail_before_sql(self):
        for updates, args in [({'POSTGRES_USER': ''}, ()), ({'POSTGRES_DB': ''}, ()), ({}, ('extra',))]:
            with self.subTest(updates=updates, args=args):
                result, calls = self.run_repair('repair', updates, args)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(calls, [])


if __name__ == '__main__':
    unittest.main(verbosity=2)
