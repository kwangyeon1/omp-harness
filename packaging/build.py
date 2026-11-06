#!/usr/bin/env python3
"""Build script installers; Python standard library only."""
import argparse
import hashlib
import io
import os
from pathlib import Path
import re
import shutil
import subprocess
import tarfile

ROOT = Path(__file__).resolve().parents[1]
DIST = ROOT / "dist"


def build_linux(version):
    files = {
        "install-omp.sh": ROOT / "install-omp.sh",
        "install-omp-tui.sh": ROOT / "install-omp-tui.sh",
        "uninstall.sh": ROOT / "packaging/linux-uninstall.sh",
        "VERSION": ROOT / "packaging/VERSION",
        "README.md": ROOT / "README.md",
    }
    buffer = io.BytesIO()
    with tarfile.open(fileobj=buffer, mode="w:gz") as archive:
        for name, path in files.items():
            data = path.read_bytes()
            info = tarfile.TarInfo(name)
            info.size = len(data)
            info.mode = 0o755 if name.endswith(".sh") else 0o644
            archive.addfile(info, io.BytesIO(data))
    header = (ROOT / "packaging/linux-install.sh").read_text(encoding="utf-8")
    if not header.endswith("\n"):
        header += "\n"
    header = header.replace("@PAYLOAD_LINE@", str(header.count("\n") + 1))
    output = DIST / f"omp-harness-scripts-{version}-linux.run"
    output.write_bytes(header.encode("utf-8") + buffer.getvalue())
    output.chmod(0o755)
    return output


def build_windows(version):
    compiler = shutil.which("ISCC.exe") or shutil.which("iscc")
    if not compiler:
        candidate = Path(os.environ.get("ProgramFiles(x86)", r"C:\Program Files (x86)")) / "Inno Setup 6/ISCC.exe"
        if candidate.is_file():
            compiler = str(candidate)
    if not compiler:
        raise SystemExit("Install Inno Setup 6 and add ISCC.exe to PATH.")
    subprocess.run([compiler, f"/DPackageVersion={version}", str(ROOT / "packaging/windows.iss")], check=True)
    return DIST / f"omp-harness-scripts-{version}-windows-setup.exe"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("platform", choices=["linux", "windows"])
    args = parser.parse_args()
    version = (ROOT / "packaging/VERSION").read_text().strip()
    if not re.fullmatch(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", version):
        raise SystemExit("packaging/VERSION must contain MAJOR.MINOR.PATCH")
    DIST.mkdir(exist_ok=True)
    output = build_linux(version) if args.platform == "linux" else build_windows(version)
    digest = hashlib.sha256(output.read_bytes()).hexdigest()
    output.with_name(output.name + ".sha256").write_text(f"{digest}  {output.name}\n", encoding="ascii")
    print(output)


if __name__ == "__main__":
    main()
