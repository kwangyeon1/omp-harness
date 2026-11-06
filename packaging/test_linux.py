"""Exercise the generated installer without downloading or installing OMP."""
import hashlib
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
VERSION = (ROOT / "packaging/VERSION").read_text().strip()
PACKAGE = ROOT / f"dist/omp-harness-scripts-{VERSION}-linux.run"


class InstallerTest(unittest.TestCase):
    def test_install_reinstall_uninstall(self):
        with tempfile.TemporaryDirectory() as temp:
            prefix = Path(temp) / "scripts with spaces"
            cmd = ["sh", str(PACKAGE), "--prefix", str(prefix)]
            subprocess.run(cmd, check=True)
            for name in ["install-omp.sh", "install-omp-tui.sh", "README.md"]:
                self.assertEqual((prefix / name).read_bytes(), (ROOT / name).read_bytes())
            self.assertEqual((prefix / "VERSION").read_text().strip(), VERSION)
            sentinel = prefix / "user-file.txt"
            sentinel.write_text("preserve me")
            subprocess.run(cmd, check=True)
            subprocess.run(["sh", str(prefix / "uninstall.sh")], check=True)
            self.assertEqual(sentinel.read_text(), "preserve me")
            self.assertEqual(list(prefix.iterdir()), [sentinel])

    def test_reject_unowned_destination(self):
        with tempfile.TemporaryDirectory() as temp:
            sentinel = Path(temp) / "install-omp.sh"
            sentinel.write_text("user owned")
            result = subprocess.run(["sh", str(PACKAGE), "--prefix", temp])
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(sentinel.read_text(), "user owned")

    def test_help_and_invalid_arguments(self):
        subprocess.run(["sh", str(PACKAGE), "--help"], check=True)
        for args in [["--prefix"], ["--prefix", "relative"], ["--unknown"]]:
            self.assertNotEqual(subprocess.run(["sh", str(PACKAGE), *args]).returncode, 0)

    def test_checksum(self):
        expected = PACKAGE.with_name(PACKAGE.name + ".sha256").read_text().split()[0]
        self.assertEqual(hashlib.sha256(PACKAGE.read_bytes()).hexdigest(), expected)


if __name__ == "__main__":
    unittest.main()
