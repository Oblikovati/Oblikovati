# SPDX-License-Identifier: GPL-2.0-only
import json
from pathlib import Path
import tempfile
import unittest

import inventory


class InventoryTests(unittest.TestCase):
    def test_discovers_and_compares_source_without_expanding_outlines(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            folder = root / 'features' / 'sketch'
            folder.mkdir(parents=True)
            feature = folder / 'first.feature'
            feature.write_text('Feature: one\n  Scenario: first\n  Scenario Outline: values\n', encoding='utf-8')
            before = inventory.collect_features(root)
            self.assertEqual(before['features/sketch/first.feature']['scenario_declarations'], 2)
            self.assertEqual(before['features/sketch/first.feature']['outlines'], 1)
            baseline = root / 'output' / 'baseline.json'
            inventory.save_baseline(baseline, root, before)
            self.assertEqual(inventory.load_baseline(baseline, root), before)
            feature.write_text('Feature: changed\n', encoding='utf-8')
            (folder / 'second.feature').write_text('Feature: new\n', encoding='utf-8')
            delta = inventory.changes(before, inventory.collect_features(root))
            self.assertEqual(delta['changed'], ['features/sketch/first.feature'])
            self.assertEqual(delta['added'], ['features/sketch/second.feature'])
            feature.unlink()
            self.assertEqual(inventory.changes(before, inventory.collect_features(root))['removed'],
                             ['features/sketch/first.feature'])

    def test_rejects_missing_source_and_wrong_baseline_root(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with self.assertRaises(ValueError):
                inventory.collect_features(root)
            baseline = root / 'baseline.json'
            baseline.write_text(json.dumps({'version': 1, 'source': '/other', 'features': {}}))
            with self.assertRaises(ValueError):
                inventory.load_baseline(baseline, root)


if __name__ == '__main__':
    unittest.main()
