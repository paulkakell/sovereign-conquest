#!/usr/bin/env python3
"""Exercise password byte boundaries against a disposable real API/database."""

import json
import os
import secrets
import unittest
from urllib.error import HTTPError
from urllib.request import Request, urlopen


class PasswordIntegrationTests(unittest.TestCase):
    def request(self, route, payload, token=None):
        headers = {"Content-Type": "application/json"}
        if token:
            headers["Authorization"] = "Bearer " + token
        request = Request(os.environ["SC_SMOKE_BASE_URL"] + "/api/" + route,
                          data=json.dumps(payload).encode(), headers=headers, method="POST")
        try:
            with urlopen(request, timeout=15) as response:
                return response.status, json.load(response)
        except HTTPError as error:
            return error.code, json.load(error)

    def test_registration_rejects_invalid_passwords_with_client_errors(self):
        for password in ("short", "x" * 73, "x" * 100, "é" * 37):
            with self.subTest(bytes=len(password.encode())):
                status, body = self.request("register", {"username": "badlength", "password": password})
                self.assertEqual(status, 400)
                self.assertIn("8-72 bytes", json.dumps(body))

    def test_valid_boundaries_round_trip_and_invalid_changes_preserve_credentials(self):
        for password in ("x" * 8, "x" * 72, "é" * 36):
            with self.subTest(bytes=len(password.encode())):
                username = "length" + secrets.token_hex(4)
                status, body = self.request("register", {"username": username, "password": password})
                self.assertEqual(status, 200)
                token = body["token"]
                status, _ = self.request("login", {"username": username, "password": password})
                self.assertEqual(status, 200)
                for invalid in ("short", "x" * 73, "x" * 100, "é" * 37):
                    status, body = self.request("change_password", {"old_password": password, "new_password": invalid}, token)
                    self.assertEqual(status, 400)
                    self.assertIn("8-72 bytes", json.dumps(body))
                status, _ = self.request("login", {"username": username, "password": password})
                self.assertEqual(status, 200)
                replacement = "y" * 72
                status, _ = self.request("change_password", {"old_password": password, "new_password": replacement}, token)
                self.assertEqual(status, 200)
                status, _ = self.request("login", {"username": username, "password": replacement})
                self.assertEqual(status, 200)
                status, _ = self.request("login", {"username": username, "password": password})
                self.assertEqual(status, 401)


if __name__ == "__main__":
    unittest.main(verbosity=2)
