#!/usr/bin/env python3
"""Build installers and a universal managed bundle (Go + Python at build time only)."""
import argparse
import hashlib
import io
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tarfile
import zipfile

ROOT = Path(__file__).resolve().parents[1]
DIST = ROOT / 'dist'


def prepare_bundle(version):
    stage = DIST / 'bundle'
    if stage.exists():
        shutil.rmtree(stage)
    (stage / 'manager').mkdir(parents=True)
    go = os.environ.get('OMP_GO', 'go')
    for platform in ('linux', 'windows'):
        for arch in ('amd64', 'arm64'):
            name = f'manage-{platform}-{arch}' + ('.exe' if platform == 'windows' else '')
            env = dict(os.environ, GOOS=platform, GOARCH=arch, CGO_ENABLED='0')
            subprocess.run([go, 'build', '-trimpath', '-buildvcs=false', '-ldflags=-s -w -buildid=',
                            '-o', str(stage / 'manager' / name), './manager'], cwd=ROOT, env=env, check=True)
    for name in ('catalog.json', 'manage.sh', 'manage.cmd', 'README.md', 'install-omp.sh',
                 'install-omp.bat', 'install-omp-tui.sh', 'install-omp-tui.bat', 'install-omp-tui.ps1'):
        shutil.copyfile(ROOT / name, stage / name)
    (stage / 'docs').mkdir()
    for name in ('INSTALLATION.md', 'RELEASE.md'):
        shutil.copyfile(ROOT / 'docs' / name, stage / 'docs' / name)
    for directory in ('defaults', 'scripts', 'plugins'):
        if (ROOT / directory).is_dir():
            shutil.copytree(ROOT / directory, stage / directory)
    shutil.copyfile(ROOT / 'packaging/VERSION', stage / 'VERSION')
    shutil.copyfile(ROOT / 'packaging/linux-uninstall.sh', stage / 'uninstall.sh')
    shutil.copyfile(ROOT / 'packaging/windows-launch.bat', stage / 'launch-installer.bat')
    # An optional component map supports independent script/plugin ownership.
    owners = json.loads((ROOT / 'packaging/components.json').read_text())
    components = {name: {'version': version, 'files': {}} for name in owners}
    for path in sorted(stage.rglob('*')):
        if not path.is_file():
            continue
        rel = path.relative_to(stage).as_posix()
        matches = [name for name, prefixes in owners.items()
                   if any(rel == prefix or (prefix.endswith('/') and rel.startswith(prefix)) for prefix in prefixes)]
        if len(matches) != 1:
            raise SystemExit(f'Expected exactly one owner for {rel}, got {matches}')
        components[matches[0]]['files'][rel] = hashlib.sha256(path.read_bytes()).hexdigest()
        if rel.endswith('.sh') or rel.startswith('manager/manage-'):
            path.chmod(0o755)
    manifest = {'version': version, 'components': components}
    (stage / 'bundle.json').write_text(json.dumps(manifest, indent=2) + '\n', encoding='utf-8')
    output = DIST / f'omp-harness-bundle-{version}.zip'
    with zipfile.ZipFile(output, 'w', compression=zipfile.ZIP_DEFLATED) as archive:
        for path in sorted(stage.rglob('*')):
            if path.is_file():
                info = zipfile.ZipInfo(path.relative_to(stage).as_posix())
                info.compress_type = zipfile.ZIP_DEFLATED
                info.external_attr = (0o100755 if path.suffix == '.sh' or path.parent.name == 'manager' else 0o100644) << 16
                archive.writestr(info, path.read_bytes())
    return stage, output


def build_linux(version, stage):
    buffer = io.BytesIO()
    with tarfile.open(fileobj=buffer, mode='w:gz') as archive:
        for path in sorted(stage.rglob('*')):
            if path.is_file():
                archive.add(path, arcname=path.relative_to(stage).as_posix(), recursive=False)
    header = (ROOT / 'packaging/linux-install.sh').read_text(encoding='utf-8').rstrip('\n') + '\n'
    header = header.replace('@PAYLOAD_LINE@', str(header.count('\n') + 1))
    output = DIST / f'omp-harness-scripts-{version}-linux.run'
    output.write_bytes(header.encode() + buffer.getvalue())
    output.chmod(0o755)
    return output


def build_windows(version):
    compiler = shutil.which('ISCC.exe') or shutil.which('iscc')
    if not compiler:
        candidate = Path(os.environ.get('ProgramFiles(x86)', r'C:\Program Files (x86)')) / 'Inno Setup 6/ISCC.exe'
        if candidate.is_file():
            compiler = str(candidate)
    if not compiler:
        raise SystemExit('Install Inno Setup 6 and add ISCC.exe to PATH.')
    subprocess.run([compiler, f'/DPackageVersion={version}', str(ROOT / 'packaging/windows.iss')], check=True)
    return DIST / f'omp-harness-scripts-{version}-windows-setup.exe'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('platform', choices=['linux', 'windows', 'bundle'])
    args = parser.parse_args()
    version = (ROOT / 'packaging/VERSION').read_text().strip()
    if not re.fullmatch(r'(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)', version):
        raise SystemExit('packaging/VERSION must contain MAJOR.MINOR.PATCH')
    DIST.mkdir(exist_ok=True)
    stage, bundle = prepare_bundle(version)
    outputs = [bundle]
    if args.platform == 'linux':
        outputs.append(build_linux(version, stage))
    elif args.platform == 'windows':
        outputs.append(build_windows(version))
    for output in outputs:
        digest = hashlib.sha256(output.read_bytes()).hexdigest()
        output.with_name(output.name + '.sha256').write_text(f'{digest}  {output.name}\n', encoding='ascii')
        print(output)


if __name__ == '__main__':
    main()
