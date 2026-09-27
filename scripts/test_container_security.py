"""Exercise real tar inventory and vulnerability policy failure modes."""

import copy
import importlib.util
import io
import pathlib
import struct
import tarfile
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


if __name__ == "__main__":
    unittest.main()
