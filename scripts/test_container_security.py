"""Exercise real tar inventory and vulnerability policy failure modes."""

import copy
import importlib.util
import io
import json
import os
import pathlib
import shutil
import struct
import subprocess
import tarfile
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("container_security", pathlib.Path(__file__).with_name("container-security-check.py"))
security = importlib.util.module_from_spec(spec)
spec.loader.exec_module(security)


def elf(segment_type=1, bits=64, endian="<"):
    data = bytearray(128)
    data[:7] = b"\x7fELF" + bytes([2 if bits == 64 else 1, 1 if endian == "<" else 2, 1])
    struct.pack_into(endian + ("Q" if bits == 64 else "I"), data, 32 if bits == 64 else 28, 64)
    struct.pack_into(endian + "HH", data, 54 if bits == 64 else 42, 56 if bits == 64 else 32, 1)
    struct.pack_into(endian + "I", data, 64, segment_type)
    return bytes(data)


class InventoryTests(unittest.TestCase):
    def setUp(self):
        self.config = {
            "User": "10001:10001", "Cmd": ["/app/sovereign-api"],
            "Healthcheck": {"Test": ["CMD", "/app/sovereign-api", "healthcheck"]},
        }
        self.files = {
            "app/sovereign-api": elf(),
            "etc/ssl/certs/ca-certificates.crt": b"-----BEGIN CERTIFICATE-----\ntest",
            "usr/share/zoneinfo/Etc/UTC": b"TZif2",
        }

    def check(self, extra=None):
        buffer = io.BytesIO()
        with tarfile.open(fileobj=buffer, mode="w") as archive:
            for name, data in self.files.items():
                info = tarfile.TarInfo(name)
                info.size = len(data)
                info.mode = 0o755 if name == "app/sovereign-api" else 0o644
                archive.addfile(info, io.BytesIO(data))
            tmp = tarfile.TarInfo("tmp")
            tmp.type = tarfile.DIRTYPE
            tmp.mode = 0o1777
            archive.addfile(tmp)
            if extra is not None:
                for member in extra if isinstance(extra, list) else [extra]:
                    archive.addfile(member)
        buffer.seek(0)
        with tarfile.open(fileobj=buffer) as archive:
            return security.check_inventory(self.config, archive)

    def test_minimal_static_runtime_passes(self):
        self.assertEqual(self.check()["status"], "passed")

    def test_root_and_shell_healthchecks_fail(self):
        for config in ({"User": "root"}, {"Healthcheck": {"Test": ["CMD-SHELL", "wget localhost"]}}):
            with self.subTest(config=config):
                saved = copy.deepcopy(self.config)
                self.config.update(config)
                with self.assertRaises(ValueError):
                    self.check()
                self.config = saved

    def test_reported_components_and_package_database_fail(self):
        for name in ("usr/bin/wget", "bin/busybox", "bin/busybox-binsh", "usr/bin/ssl_client",
                     "lib/libz.so.1", "bin/sh", "lib/apk/db/installed"):
            with self.subTest(name=name):
                self.files[name] = b"present"
                with self.assertRaises(ValueError):
                    self.check()
                del self.files[name]

    def test_dynamic_binary_and_dynamic_library_fail_for_all_elf_formats(self):
        for bits in (32, 64):
            for endian in ("<", ">"):
                for segment in (2, 3):
                    with self.subTest(bits=bits, endian=endian, segment=segment):
                        self.files["app/sovereign-api"] = elf(segment, bits, endian)
                        with self.assertRaisesRegex(ValueError, "Dynamic ELF"):
                            self.check()

    def test_extra_static_executable_fails(self):
        self.files["usr/bin/extra"] = elf()
        with self.assertRaisesRegex(ValueError, "Expected only"):
            self.check()

    def test_missing_runtime_data_fails(self):
        for name in ("etc/ssl/certs/ca-certificates.crt", "usr/share/zoneinfo/Etc/UTC"):
            with self.subTest(name=name):
                data = self.files.pop(name)
                with self.assertRaisesRegex(ValueError, "Required runtime data"):
                    self.check()
                self.files[name] = data

    def test_timezone_symlinks_and_hardlinks_resolve_inside_archive(self):
        name = "usr/share/zoneinfo/Etc/UTC"
        data = self.files.pop(name)
        self.files["usr/share/zoneinfo/UTC"] = data
        for kind, target in ((tarfile.LNKTYPE, "usr/share/zoneinfo/UTC"),
                             (tarfile.SYMTYPE, "../UTC"),
                             (tarfile.SYMTYPE, "/usr/share/zoneinfo/UTC")):
            with self.subTest(kind=kind, target=target):
                link = tarfile.TarInfo(name)
                link.type = kind
                link.linkname = target
                self.assertEqual(self.check(link)["status"], "passed")

    def test_chained_timezone_links_validate_target_contents(self):
        name = "usr/share/zoneinfo/Etc/UTC"
        data = self.files.pop(name)
        self.files["usr/share/zoneinfo/Zulu"] = data
        links = []
        for source, target, kind in ((name, "../UTC", tarfile.SYMTYPE),
                                     ("usr/share/zoneinfo/UTC", "usr/share/zoneinfo/Zulu", tarfile.LNKTYPE)):
            link = tarfile.TarInfo(source)
            link.type = kind
            link.linkname = target
            links.append(link)
        self.assertEqual(self.check(links)["status"], "passed")
        self.files["usr/share/zoneinfo/Zulu"] = b"invalid timezone data"
        with self.assertRaisesRegex(ValueError, "Required runtime data is invalid"):
            self.check(links)

    def test_timezone_link_escape_cycle_and_missing_target_fail(self):
        name = "usr/share/zoneinfo/Etc/UTC"
        self.files.pop(name)
        for target, message in (("../../../../../etc/passwd", "escapes archive root"),
                                ("UTC", "link cycle"),
                                ("../Missing", "Required runtime data is missing")):
            with self.subTest(target=target):
                link = tarfile.TarInfo(name)
                link.type = tarfile.SYMTYPE
                link.linkname = target
                with self.assertRaisesRegex(ValueError, message):
                    self.check(link)

    def test_tmp_permissions_remain_required(self):
        tmp = tarfile.TarInfo("tmp")
        tmp.type = tarfile.DIRTYPE
        tmp.mode = 0o755
        # Test the permission check with a single temporary-directory member.
        buffer = io.BytesIO()
        with tarfile.open(fileobj=buffer, mode="w") as archive:
            for name, data in self.files.items():
                info = tarfile.TarInfo(name)
                info.size = len(data)
                info.mode = 0o755
                archive.addfile(info, io.BytesIO(data))
            archive.addfile(tmp)
        buffer.seek(0)
        with tarfile.open(fileobj=buffer) as archive:
            with self.assertRaisesRegex(ValueError, "sticky world-writable"):
                security.check_inventory(self.config, archive)

    def test_renamed_symlink_to_busybox_fails(self):
        link = tarfile.TarInfo("bin/probe")
        link.type = tarfile.SYMTYPE
        link.linkname = "/bin/busybox"
        with self.assertRaisesRegex(ValueError, "symlink target"):
            self.check(link)

    def test_malformed_elf_fails(self):
        for value in (b"\x7fELF", elf()[:64]):
            with self.subTest(size=len(value)):
                self.files["app/sovereign-api"] = value
                with self.assertRaisesRegex(ValueError, "Truncated ELF"):
                    self.check()


class VulnerabilityTests(unittest.TestCase):
    def report(self, vulnerabilities):
        return {"SchemaVersion": 2, "ArtifactType": "container_image", "ArtifactName": "test:latest",
                "Results": [{"Type": "gobinary", "Target": "app/sovereign-api", "Vulnerabilities": vulnerabilities}]}

    def test_clean_go_scan_passes(self):
        self.assertEqual(security.check_vulnerabilities(self.report([]))["status"], "passed")

    def test_every_reported_cve_blocks_even_without_fix_or_severity(self):
        for cve in security.REPORTED_CVES:
            with self.subTest(cve=cve):
                with self.assertRaisesRegex(ValueError, cve):
                    security.check_vulnerabilities(self.report([{"VulnerabilityID": cve, "Severity": "MEDIUM"}]))

    def test_general_high_and_critical_findings_block_even_without_fix(self):
        for severity in ("HIGH", "CRITICAL"):
            with self.subTest(severity=severity):
                with self.assertRaisesRegex(ValueError, "CVE-2099-0001"):
                    security.check_vulnerabilities(self.report([{"VulnerabilityID": "CVE-2099-0001", "Severity": severity}]))

    def test_unrelated_medium_does_not_expand_release_policy(self):
        self.assertEqual(security.check_vulnerabilities(self.report([
            {"VulnerabilityID": "CVE-2099-0001", "Severity": "MEDIUM"},
        ]))["status"], "passed")

    def test_missing_analysis_and_wrong_schema_fail_closed(self):
        for report in ({}, {"SchemaVersion": 2, "ArtifactType": "container_image", "Results": []},
                       {"SchemaVersion": 2, "ArtifactType": "filesystem", "Results": [{"Type": "gobinary"}]}):
            with self.subTest(report=report):
                with self.assertRaises(ValueError):
                    security.check_vulnerabilities(report)


@unittest.skipUnless(shutil.which("jq"), "The publication workflow requires jq")
class PublicationWorkflowTests(unittest.TestCase):
    def run_publication(self, local_id=None, remote_id=None, registry_digest=None):
        repo = pathlib.Path(__file__).resolve().parents[1]
        workflow = (repo / ".github/workflows/publish-ghcr.yml").read_text()
        step = workflow.split("      - name: Push validated release image\n", 1)[1]
        step = step.split("\n      - name:", 1)[0]
        block = step.split("        run: |\n", 1)[1]
        shell = "\n".join(line[10:] for line in block.splitlines())
        validated_id = "sha256:" + "a" * 64
        expected_digest = "sha256:" + "b" * 64
        with tempfile.TemporaryDirectory(prefix="sc-publication-test-") as directory:
            root = pathlib.Path(directory)
            evidence = root / "container-security"
            evidence.mkdir()
            (evidence / "combined-inventory.json").write_text(json.dumps({"image_id": validated_id}))
            docker = root / "docker"
            docker.write_text("""#!/usr/bin/env python3
import json
import os
import sys
args = sys.argv[1:]
with open(os.environ["MOCK_DOCKER_LOG"], "a") as log:
    log.write(json.dumps(args) + "\\n")
if args[:2] == ["image", "inspect"]:
    print(os.environ["MOCK_LOCAL_ID"])
elif args[:3] == ["buildx", "imagetools", "inspect"]:
    if args[-1] == "--raw":
        print(json.dumps({"schemaVersion": 2, "config": {"digest": os.environ["MOCK_REMOTE_ID"]}}))
    else:
        print(json.dumps({"digest": os.environ["MOCK_REGISTRY_DIGEST"]}))
elif args[0] == "push":
    # Docker quiet mode prints the image reference, not the manifest digest.
    print(args[-1])
elif args[0] != "tag":
    sys.exit("Unexpected Docker command: " + repr(args))
""")
            docker.chmod(0o755)
            env = os.environ.copy()
            env.update({
                "PATH": str(root) + os.pathsep + env["PATH"],
                "IMAGE": "ghcr.io/example/sovereign-conquest",
                "VERSION": "01.06.06", "SHORT_SHA": "abc123",
                "GITHUB_OUTPUT": str(root / "github-output"),
                "MOCK_DOCKER_LOG": str(root / "docker-calls"),
                "MOCK_LOCAL_ID": local_id or validated_id,
                "MOCK_REMOTE_ID": remote_id or validated_id,
                "MOCK_REGISTRY_DIGEST": registry_digest or expected_digest,
            })
            process = subprocess.run(["bash", "-c", shell], cwd=root, env=env, text=True, capture_output=True)
            calls = [json.loads(line) for line in (root / "docker-calls").read_text().splitlines()]
            outputs = (root / "github-output").read_text() if (root / "github-output").exists() else ""
            published = (evidence / "published-digest.txt").read_text() if (evidence / "published-digest.txt").exists() else ""
            return process, calls, outputs, published

    def test_push_reference_output_is_ignored_and_registry_digest_is_published(self):
        process, calls, outputs, published = self.run_publication()
        self.assertEqual(process.returncode, 0, process.stderr)
        digest = "sha256:" + "b" * 64
        self.assertEqual(outputs, f"digest={digest}\n")
        self.assertEqual(published, f"ghcr.io/example/sovereign-conquest@{digest}\n")
        pushes = [call[-1] for call in calls if call[0] == "push"]
        self.assertEqual(pushes, ["ghcr.io/example/sovereign-conquest:sha-abc123",
                                 "ghcr.io/example/sovereign-conquest:01.06.06",
                                 "ghcr.io/example/sovereign-conquest:main"])
        raw_index = next(index for index, call in enumerate(calls) if call[-1] == "--raw")
        tag_index = next(index for index, call in enumerate(calls) if call[0] == "tag")
        self.assertLess(raw_index, tag_index)

    def test_registry_configuration_mismatch_blocks_release_aliases(self):
        process, calls, outputs, published = self.run_publication(remote_id="sha256:" + "c" * 64)
        self.assertNotEqual(process.returncode, 0)
        self.assertIn("does not match", process.stderr)
        self.assertEqual([call[-1] for call in calls if call[0] == "push"],
                         ["ghcr.io/example/sovereign-conquest:sha-abc123"])
        self.assertFalse(any(call[0] == "tag" for call in calls))
        self.assertEqual(outputs + published, "")

    def test_invalid_registry_digest_blocks_release_aliases(self):
        for digest in ("not-a-digest", "sha256:1234", "ghcr.io/example/image:tag"):
            with self.subTest(digest=digest):
                process, calls, outputs, published = self.run_publication(registry_digest=digest)
                self.assertNotEqual(process.returncode, 0)
                self.assertEqual([call[-1] for call in calls if call[0] == "push"],
                                 ["ghcr.io/example/sovereign-conquest:sha-abc123"])
                self.assertFalse(any(call[0] == "tag" for call in calls))
                self.assertEqual(outputs + published, "")

    def test_changed_local_image_blocks_every_push(self):
        process, calls, outputs, published = self.run_publication(local_id="sha256:" + "d" * 64)
        self.assertNotEqual(process.returncode, 0)
        self.assertFalse(any(call[0] in {"tag", "push"} for call in calls))
        self.assertEqual(outputs + published, "")


if __name__ == "__main__":
    unittest.main()
