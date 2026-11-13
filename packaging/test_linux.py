"""Exercise the generated installer without downloading or installing OMP."""
import hashlib
import json
import os
import shutil
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
            state = Path(temp) / 'user' / 'state.json'
            state.parent.mkdir()
            state.write_text('{"omp": true}')
            env = dict(os.environ, OMP_HARNESS_STATE_FILE=str(state))
            subprocess.run(cmd, check=True, env=env)
            result = subprocess.run(['sh', str(prefix / 'manage.sh'), 'list'], env=env, check=True, capture_output=True, text=True)
            self.assertIn('omp | installed', result.stdout)
            for name in ["install-omp.sh", "install-omp-tui.sh", "README.md"]:
                self.assertEqual((prefix / name).read_bytes(), (ROOT / name).read_bytes())
            self.assertEqual((prefix / "VERSION").read_text().strip(), VERSION)
            sentinel = prefix / "user-file.txt"
            sentinel.write_text("preserve me")
            subprocess.run(cmd, check=True)
            subprocess.run(["sh", str(prefix / "uninstall.sh")], check=True)
            self.assertEqual(sentinel.read_text(), "preserve me")
            self.assertEqual(list(prefix.iterdir()), [sentinel])
            self.assertEqual(json.loads(state.read_text()), {'omp': True})

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

    def test_leaf_records_only_after_success(self):
        with tempfile.TemporaryDirectory() as temp:
            base = Path(temp)
            leaf = base / 'install-omp-tui.sh'
            shutil.copyfile(ROOT / leaf.name, leaf)
            state = base / 'state.json'
            manager = ROOT / 'dist/bundle/manager/manage-linux-amd64'
            env = dict(os.environ, OMP_HARNESS_MANAGER=str(manager), OMP_HARNESS_STATE_FILE=str(state))
            installer = base / 'install-omp.sh'
            answers = ['', '', str(base / 'bin'), '', '', 'y', 'n', 'n', '', '', '', 'y']
            # A real leaf is exercised with a local fixture replacing the network installer.
            installer.write_text('#!/bin/sh\nexit 1\n')
            result = subprocess.run(['sh', str(leaf)], input='\n'.join(answers)+'\n', env=env, text=True, capture_output=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse(state.exists())
            installer.write_text('#!/bin/sh\nexit 0\n')
            result = subprocess.run(['sh', str(leaf)], input='\n'.join(answers[:-1]+['n'])+'\n', env=env, text=True, capture_output=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertFalse(state.exists())
            result = subprocess.run(['sh', str(leaf)], input='\n'.join(answers)+'\n', env=env, text=True, capture_output=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(json.loads(state.read_text()), {'omp': True})

    def test_checksum(self):
        expected = PACKAGE.with_name(PACKAGE.name + ".sha256").read_text().split()[0]
        self.assertEqual(hashlib.sha256(PACKAGE.read_bytes()).hexdigest(), expected)


if __name__ == "__main__":
    unittest.main()
