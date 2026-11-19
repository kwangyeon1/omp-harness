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
            state = base / 'state.json'
            manager = ROOT / 'dist/bundle/manager/manage-linux-amd64'
            fake_bin = base / 'fake-bin'
            fake_bin.mkdir()
            installer = base / 'upstream.sh'
            # Replace only HTTPS transport; execute the real standalone leaf.
            curl = fake_bin / 'curl'
            curl.write_text('#!/bin/sh\nwhile [ "$#" -gt 0 ]; do\nif [ "$1" = -o ]; then cp "$FIXTURE_UPSTREAM" "$2"; exit; fi\nshift\ndone\nexit 2\n')
            curl.chmod(0o755)
            env = dict(os.environ, OMP_HARNESS_MANAGER=str(manager),
                       OMP_HARNESS_STATE_FILE=str(state), PI_INSTALL_DIR=str(base / 'bin'),
                       FIXTURE_UPSTREAM=str(installer), PATH=str(fake_bin)+os.pathsep+os.environ['PATH'])
            installer.write_text('exit 1\n')
            result = subprocess.run(['sh', str(ROOT / 'install-omp.sh')], env=env, capture_output=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse(state.exists())
            installer.write_text('mkdir -p "$PI_INSTALL_DIR"\nprintf \'#!/bin/sh\\nexit 0\\n\' > "$PI_INSTALL_DIR/omp"\nchmod +x "$PI_INSTALL_DIR/omp"\n')
            subprocess.run(['sh', str(ROOT / 'install-omp.sh'), 'v18.8.9'], env=env, check=True)
            self.assertEqual(json.loads(state.read_text()), {'omp': True})

    def test_restore_entries_and_help(self):
        bundle = ROOT / 'dist/bundle'
        for name in ('restore.sh', 'restore-tui.sh'):
            result = subprocess.run(['sh', str(bundle / name), '--help'], check=True, capture_output=True, text=True)
            self.assertIn('restore run', result.stdout)
        result = subprocess.run(['sh', str(bundle / 'manage.sh'), 'help', 'omp'], check=True, capture_output=True, text=True)
        self.assertIn('install-omp.sh', result.stdout)
        self.assertNotIn('install-omp-tui.sh', result.stdout)
        for name in ('install-omp-tui.sh', 'manage.sh'):
            subprocess.run(['sh', str(bundle / name)], input='\n', text=True, check=True, capture_output=True)

    def test_checksum(self):
        expected = PACKAGE.with_name(PACKAGE.name + ".sha256").read_text().split()[0]
        self.assertEqual(hashlib.sha256(PACKAGE.read_bytes()).hexdigest(), expected)


if __name__ == "__main__":
    unittest.main()
