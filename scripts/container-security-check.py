#!/usr/bin/env python3
"""Fail closed on unexpected API runtime contents or container vulnerabilities."""

import argparse
import json
import pathlib
import struct
import subprocess
import sys
import tarfile
import tempfile


REPORTED_CVES = frozenset({
    "CVE-2026-85091", "CVE-2026-58469", "CVE-2026-58470",
    "CVE-2026-58471", "CVE-2026-58472", "CVE-2025-60876",
})
FORBIDDEN_NAMES = frozenset({
    "wget", "busybox", "busybox-binsh", "ssl_client", "zlib",
    "sh", "ash", "bash", "apk", "libz.a", "libz.so",
})


def require(condition, message):
    if not condition:
        raise ValueError(message)


def check_elf(stream, name):
    """Reject ELF interpreters and dynamic linking without invoking image tools."""
    header = stream.read(64)
    if header[:4] != b"\x7fELF":
        return False
    require(len(header) == 64, f"Truncated ELF header: {name}")
    require(header[4] in (1, 2) and header[5] in (1, 2), f"Unsupported ELF: {name}")
    endian = "<" if header[5] == 1 else ">"
    is_64 = header[4] == 2
    offset = struct.unpack_from(endian + ("Q" if is_64 else "I"), header, 32 if is_64 else 28)[0]
    size, count = struct.unpack_from(endian + "HH", header, 54 if is_64 else 42)
    require(size >= (56 if is_64 else 32) and 0 < count < 65535, f"Invalid ELF program headers: {name}")
    for index in range(count):
        stream.seek(offset + index * size)
        entry = stream.read(size)
        require(len(entry) == size, f"Truncated ELF program header: {name}")
        segment_type = struct.unpack_from(endian + "I", entry)[0]
        require(segment_type not in (2, 3), f"Dynamic ELF segment in runtime file: {name}")
    return True


def resolve_data_member(members, name):
    """Follow container links inside the archive, without extracting host paths."""
    pending = list(pathlib.PurePosixPath(name).parts)
    resolved = []
    visited = set()
    while pending:
        part = pending.pop(0)
        if part in ("/", "."):
            continue
        if part == "..":
            require(bool(resolved), f"Runtime data link escapes archive root: {name}")
            resolved.pop()
            continue
        resolved.append(part)
        current = "/".join(resolved)
        member = members.get(current)
        if member is not None and (member.issym() or member.islnk()):
            require(current not in visited, f"Runtime data link cycle: {name}")
            visited.add(current)
            target = pathlib.PurePosixPath(member.linkname)
            require(bool(member.linkname), f"Empty runtime data link: {current}")
            pending = list(target.parts) + pending
            # Tar hardlink targets are archive-relative; symlinks are relative
            # to their containing directory, or to the container root.
            if member.islnk() or target.is_absolute():
                resolved = []
            else:
                resolved.pop()
    member = members.get("/".join(resolved))
    require(member is not None and member.isfile(), f"Required runtime data is missing: {name}")
    return member


def check_inventory(config, archive):
    require(config.get("User") == "10001:10001", "Runtime must use USER 10001:10001")
    require(config.get("Healthcheck", {}).get("Test") == ["CMD", "/app/sovereign-api", "healthcheck"],
            "Healthcheck must directly execute the native API healthcheck")
    require(config.get("Cmd") == ["/app/sovereign-api"] and not config.get("Entrypoint"),
            "Runtime command must directly execute the API")
    members = {}
    elf_files = []
    for member in archive.getmembers():
        path = pathlib.PurePosixPath(member.name)
        require(not path.is_absolute() and ".." not in path.parts, f"Unsafe archive path: {member.name}")
        name = str(path)
        require(name not in members, f"Duplicate archive path: {name}")
        members[name] = member
        require(path.name not in FORBIDDEN_NAMES and not path.name.startswith(("libz.so.", "zlib-")),
                f"Forbidden runtime component: {name}")
        require(not name.startswith(("etc/apk/", "lib/apk/", "usr/lib/apk/")),
                f"Package-manager database in scratch runtime: {name}")
        require(not member.mode & 0o6000, f"Setuid/setgid runtime file: {name}")
        if member.isfile():
            with archive.extractfile(member) as stream:
                if check_elf(stream, name):
                    elf_files.append(name)
        if member.issym() or member.islnk():
            target = pathlib.PurePosixPath(member.linkname)
            require(target.name not in FORBIDDEN_NAMES and not target.name.startswith("libz.so."),
                    f"Forbidden runtime symlink target: {name} -> {target}")
    require(elf_files == ["app/sovereign-api"], f"Expected only the statically linked API ELF; found {elf_files}")
    binary = members.get("app/sovereign-api")
    require(binary is not None and binary.mode & 0o111, "API binary is missing or not executable")
    for name, signature in {
        "etc/ssl/certs/ca-certificates.crt": b"-----BEGIN CERTIFICATE-----",
        "usr/share/zoneinfo/Etc/UTC": b"TZif",
    }.items():
        member = resolve_data_member(members, name)
        with archive.extractfile(member) as stream:
            require(signature in stream.read(), f"Required runtime data is invalid: {name}")
    tmp = members.get("tmp")
    require(tmp is not None and tmp.isdir() and tmp.mode & 0o1777 == 0o1777,
            "Runtime /tmp must exist with sticky world-writable permissions")
    return {"files": len(members), "elf_files": elf_files, "user": config["User"], "status": "passed"}


def inspect_image(image):
    inspection = json.loads(subprocess.check_output(["docker", "image", "inspect", image], text=True))
    require(len(inspection) == 1, "Expected exactly one image")
    container = subprocess.check_output(["docker", "create", image], text=True).strip()
    try:
        with tempfile.TemporaryDirectory(prefix="sc-inventory-") as directory:
            output = pathlib.Path(directory) / "runtime.tar"
            subprocess.run(["docker", "export", "--output", str(output), container], check=True)
            with tarfile.open(output, mode="r:") as archive:
                result = check_inventory(inspection[0]["Config"], archive)
            result.update({"image": image, "image_id": inspection[0]["Id"]})
            print(json.dumps(result, indent=2))
    finally:
        subprocess.run(["docker", "rm", "--force", container], check=True, stdout=subprocess.DEVNULL)


def check_vulnerabilities(report):
    require(report.get("SchemaVersion") == 2, "Unrecognized Trivy report schema")
    require(report.get("ArtifactType") == "container_image", "Trivy report is not a container image scan")
    results = report.get("Results")
    require(isinstance(results, list) and any(result.get("Type") == "gobinary" for result in results),
            "Trivy report does not contain a Go binary analysis")
    blocked = []
    for result in results:
        for vulnerability in result.get("Vulnerabilities") or []:
            cve = vulnerability.get("VulnerabilityID", "")
            severity = vulnerability.get("Severity", "UNKNOWN").upper()
            if cve in REPORTED_CVES or severity in {"HIGH", "CRITICAL"}:
                blocked.append(f"{cve} {severity} {vulnerability.get('PkgName', '?')} "
                               f"{vulnerability.get('InstalledVersion', '?')} ({result.get('Target', '?')})")
    require(not blocked, "Blocked container vulnerabilities, including unfixed findings:\n" + "\n".join(blocked))
    return {"artifact": report.get("ArtifactName"), "status": "passed", "results": len(results)}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("inventory", "vulnerabilities"))
    parser.add_argument("target", help="Docker image reference or Trivy JSON report path")
    args = parser.parse_args()
    try:
        if args.mode == "inventory":
            inspect_image(args.target)
        else:
            with open(args.target, encoding="utf-8") as source:
                print(json.dumps(check_vulnerabilities(json.load(source)), indent=2))
    except (ValueError, OSError, tarfile.TarError, subprocess.CalledProcessError) as error:
        print(f"Container security gate failed: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
