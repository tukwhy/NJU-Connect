"""Build clean source/runtime ZIPs using only the Python standard library."""
import argparse
import hashlib
import json
import pathlib
import re
import subprocess
import zipfile

ROOT = pathlib.Path(__file__).resolve().parents[1]
ROOT_FILES = {
    '.gitignore', '.gitattributes', 'go.mod', 'go.sum', 'LICENSE', 'README.md', 'README_NJU.md',
    'README.upstream.md', 'README_en.md', 'NOTICE.md', 'CHANGELOG.md',
    'Build-NJU.ps1', 'Start-NJU.ps1', 'CFW-Mixin-NJU.js', 'CFW-Mixin.template.js',
    'clash-nju.yaml', 'nju.toml', 'config.toml.example', 'Dockerfile',
    'docker-compose.yml', 'com.zju.connect.plist',
}
SOURCE_DIRS = {
    'client', 'configs', 'dial', 'internal', 'log', 'resolve', 'service',
    'stack', 'underlay', 'mobile', 'docs', '.github', 'tests', 'scripts',
}
FORBIDDEN_SUFFIXES = {'.har', '.pcap', '.log', '.keylog', '.zip', '.exe', '.pyc', '.p12', '.key'}

def digest(data):
    return hashlib.sha256(data).hexdigest()

def source_files():
    if (ROOT / '.git').exists():
        result = subprocess.run(
            ['git', 'ls-files', '--cached', '--others', '--exclude-standard', '-z'],
            cwd=ROOT, check=True, capture_output=True,
        )
        candidates = [ROOT / name for name in result.stdout.decode('utf8').split('\0') if name]
    else:
        candidates = [file for file in ROOT.rglob('*') if file.is_file()]
    selected = {}
    for file in candidates:
        if not file.is_file() or file.is_symlink():
            continue
        relative = file.relative_to(ROOT)
        if relative.parts[0] not in SOURCE_DIRS and not (
            relative.as_posix() in ROOT_FILES or len(relative.parts) == 1 and file.suffix == '.go'
        ):
            continue
        if any(part in {'__pycache__', 'node_modules', '.git'} for part in relative.parts):
            continue
        if file.suffix.lower() in FORBIDDEN_SUFFIXES or '.generated.' in file.name or '.local.' in file.name:
            continue
        selected[relative.as_posix()] = file
    return {name: selected[name].read_bytes() for name in sorted(selected)}

def write_zip(path, contents, version, kind):
    manifest = {
        'version': version,
        'kind': kind,
        'upstream_commit': '923672d',
        'files_sha256': {name: digest(data) for name, data in sorted(contents.items())},
    }
    entries = {**contents, 'PACKAGE-MANIFEST.json': (json.dumps(manifest, indent=2, ensure_ascii=False)+'\n').encode('utf8')}
    with zipfile.ZipFile(path, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        for name, data in sorted(entries.items()):
            archive.writestr(name, data)
    # Verify every archived file, not just archive creation success.
    with zipfile.ZipFile(path) as archive:
        if archive.testzip() is not None:
            raise RuntimeError('ZIP integrity check failed')
        if set(archive.namelist()) != set(entries):
            raise RuntimeError('ZIP file inventory mismatch')
        for name, data in entries.items():
            if archive.read(name) != data:
                raise RuntimeError('ZIP content mismatch: '+name)

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--version', default='v0.1.0-nju.1')
    parser.add_argument('--binary', default='dist/nju-connect.exe')
    parser.add_argument('--output', default='releases')
    args = parser.parse_args()
    if not re.fullmatch(r'[A-Za-z0-9._-]+', args.version):
        parser.error('version must contain only letters, numbers, dots, underscores or hyphens')
    binary = pathlib.Path(args.binary)
    if not binary.is_absolute():
        binary = ROOT / binary
    if not binary.is_file():
        parser.error('build the Windows binary first: '+str(binary))
    binary_data = binary.read_bytes()
    if not binary_data.startswith(b'MZ'):
        parser.error('binary is not a Windows PE executable')
    output = pathlib.Path(args.output)
    if not output.is_absolute():
        output = ROOT / output
    output.mkdir(parents=True, exist_ok=True)
    source = source_files()
    for required in {'go.mod', 'main.go', 'init.go', 'nju.go', 'nju_rules.go', 'CFW-Mixin.template.js', 'LICENSE', 'README.md'}:
        if required not in source:
            raise RuntimeError('missing required source: '+required)
    runtime = {
        name: (ROOT / name).read_bytes()
        for name in ['Start-NJU.ps1', 'nju.toml', 'CFW-Mixin-NJU.js', 'README.md', 'LICENSE', 'NOTICE.md', 'CHANGELOG.md']
    }
    runtime['dist/nju-connect.exe'] = binary_data
    source_zip = output / f'NJU-Connect-{args.version}-source.zip'
    windows_zip = output / f'NJU-Connect-{args.version}-windows-amd64.zip'
    write_zip(source_zip, source, args.version, 'source')
    write_zip(windows_zip, runtime, args.version, 'windows-amd64')
    checksums = ''.join(f'{digest(file.read_bytes())}  {file.name}\n' for file in [source_zip, windows_zip])
    (output / 'SHA256SUMS.txt').write_text(checksums, encoding='ascii')
    print(json.dumps({
        'source_archive': str(source_zip), 'source_files': len(source),
        'windows_archive': str(windows_zip), 'runtime_files': sorted(runtime),
        'checksums': str(output / 'SHA256SUMS.txt'),
    }, indent=2))

if __name__ == '__main__':
    main()
