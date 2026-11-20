import json
import os
from pathlib import Path
import re
import shutil
import subprocess
base = Path('/tmp/obk-feature-all-XOS0QQ')
root = base / 'runtime'
root.mkdir()
entries = json.loads(Path('/tmp/obk-sealed-execution-final.json').read_text())['runtime_inputs']
for name, info in sorted(entries.items(), key=lambda x: (x[0].count('/'), x[0])):
    if name == '.':
        continue
    target = root / name
    target.parent.mkdir(parents=True, exist_ok=True)
    if info['kind'] == 'directory':
        target.mkdir(exist_ok=True)
    elif name == 'runner':
        target.touch()
    elif info['kind'] == 'symlink':
        target.symlink_to(os.readlink('/' + name))
    elif info['kind'] == 'file':
        shutil.copy2('/' + name, target)
    else:
        raise ValueError(f"{name}: unsupported {info['kind']}")
(root/'desktop').touch()
(root/'proc').mkdir()
def copy_library(name):
    source = Path(name)
    real = source.resolve(strict=True)
    target = root / real.relative_to('/')
    target.parent.mkdir(parents=True, exist_ok=True)
    if not target.exists():
        shutil.copy2(real, target)
    if source.is_symlink():
        link = root / 'usr/lib/x86_64-linux-gnu' / source.name
        if not link.exists() and not link.is_symlink():
            link.symlink_to(os.path.relpath(target, link.parent))
for name in ('python3','openbox','import','compare','xvfb-run','Xvfb','xauth'):
    source = Path('/usr/bin') / name
    target = root / 'usr/bin' / name
    if target.is_symlink() or target.exists():
        target.unlink()
    shutil.copy2(source.resolve(strict=True), target)
    if name != 'xvfb-run':
        result = subprocess.run(['ldd',str(source)],capture_output=True,text=True,check=True)
        for dependency in re.findall(r'(/[^\s()]+)',result.stdout):
            if Path(dependency).is_file():
                copy_library(dependency)
for name in ('libXtst.so.6','libX11.so.6'):
    copy_library(Path('/usr/lib/x86_64-linux-gnu') / name)
awk = root / 'usr/bin/awk'
awk.unlink()
awk.symlink_to('mawk')
for name in ('usr/lib/python3.12','etc/ImageMagick-6','usr/share/ImageMagick-6','usr/lib/x86_64-linux-gnu/ImageMagick-6.9.12','etc/xdg/openbox','etc/fonts','usr/share/openbox','usr/share/themes','usr/share/icons'):
    source = Path('/') / name
    if source.exists():
        shutil.copytree(source,root/name,dirs_exist_ok=True,symlinks=True)
print('Diagnostic runtime staged; provenance unbounded')
