#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Track changes in the external MCAD scenario source, without claiming test results."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import tempfile

DECLARATION = re.compile(r"^\s*(Scenario(?: Outline| Template)?):", re.MULTILINE)


def feature_record(path: Path) -> dict:
    """Describe one source file. Example: feature_record(Path('sample.feature'))."""
    content = path.read_bytes()
    declarations = DECLARATION.findall(content.decode('utf-8-sig'))
    return {
        'sha256': hashlib.sha256(content).hexdigest(),
        'scenario_declarations': len(declarations),
        'outlines': sum(item != 'Scenario' for item in declarations),
    }


def collect_features(root: Path) -> dict:
    """Index all feature files. Example: collect_features(Path('/work/MCAD-Scenarios'))."""
    features = root / 'features'
    if not features.is_dir():
        raise ValueError(f'scenario root {root}: expected a features directory')
    return {p.relative_to(root).as_posix(): feature_record(p)
            for p in sorted(features.rglob('*.feature')) if p.is_file()}


def changes(previous: dict, current: dict) -> dict:
    """Compare indexed sources. Example: changes({}, {'a': {'sha256': 'x'}})."""
    old, new = set(previous), set(current)
    return {
        'added': sorted(new - old),
        'changed': sorted(p for p in old & new if previous[p] != current[p]),
        'removed': sorted(old - new),
    }


def load_baseline(path: Path, root: Path) -> dict:
    """Read a compatible baseline. Example: load_baseline(Path('index.json'), Path('/work'))."""
    if not path.exists():
        return {}
    record = json.loads(path.read_text(encoding='utf-8'))
    if record.get('version') != 1 or record.get('source') != str(root):
        raise ValueError(f'inventory {path}: expected version 1 for source {root}')
    return record['features']


def save_baseline(path: Path, root: Path, features: dict) -> None:
    """Atomically save an inventory. Example: save_baseline(Path('index.json'), Path('/work'), {})."""
    path.parent.mkdir(parents=True, exist_ok=True)
    record = {'version': 1, 'source': str(root), 'features': features}
    with tempfile.NamedTemporaryFile('w', encoding='utf-8', dir=path.parent, delete=False) as stream:
        temporary = Path(stream.name)
        json.dump(record, stream, indent=2, sort_keys=True)
        stream.write('\n')
    try:
        os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source', type=Path, help='path to MCAD-Scenarios')
    parser.add_argument('--baseline', type=Path, default=Path('target/bdd-tracking/inventory.json'))
    parser.add_argument('--update', action='store_true', help='save the current source as baseline')
    args = parser.parse_args()
    root = args.source.resolve(strict=True)
    current = collect_features(root)
    delta = changes(load_baseline(args.baseline, root), current)
    areas = {}
    for path, record in current.items():
        area = path.split('/')[1]
        counts = areas.setdefault(area, {'features': 0, 'scenario_declarations': 0, 'outlines': 0})
        counts['features'] += 1
        counts['scenario_declarations'] += record['scenario_declarations']
        counts['outlines'] += record['outlines']
    print(json.dumps({'source': str(root), 'areas': areas, 'changes': delta}, indent=2))
    if args.update:
        save_baseline(args.baseline, root, current)


if __name__ == '__main__':
    main()
