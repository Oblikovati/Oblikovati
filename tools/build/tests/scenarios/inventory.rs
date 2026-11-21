// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::collections::BTreeMap;
use std::path::PathBuf;
#[derive(Default)]
struct FakeScenarioSource {
    files: BTreeMap<PathBuf, Vec<u8>>,
}
impl ScenarioSource for FakeScenarioSource {
    fn feature_paths(&mut self, _: &Path) -> BuildResult<Vec<PathBuf>> {
        Ok(self.files.keys().rev().cloned().collect())
    }
    fn read(&mut self, path: &Path) -> BuildResult<Vec<u8>> {
        self.files
            .get(path)
            .cloned()
            .ok_or_else(|| format!("missing fake source {path:?}"))
    }
}
#[test]
fn inventory_preserves_feature_folders_and_fingerprints_exact_crlf_bytes() {
    let root = Path::new("benchmark");
    let path = root.join("features/application/document-creation/scenario.feature");
    let mut source = FakeScenarioSource {
        files: BTreeMap::from([(path, b"Feature: New\r\n".to_vec())]),
    };
    let first = scan(root, &mut source).unwrap();
    assert_eq!(
        first.features[0].path,
        "features/application/document-creation/scenario.feature"
    );
    assert_eq!(first.features[0].area, "application");
    source
        .files
        .values_mut()
        .next()
        .unwrap()
        .retain(|byte| *byte != b'\r');
    assert_ne!(first, scan(root, &mut source).unwrap());
}
#[test]
fn rescan_detects_new_features_and_rejects_empty_or_malformed_trees() {
    let root = Path::new("benchmark");
    let mut source = FakeScenarioSource::default();
    assert!(
        scan(root, &mut source)
            .unwrap_err()
            .contains("expected features/")
    );
    source
        .files
        .insert(root.join("features/application/a/a.feature"), Vec::new());
    assert_eq!(scan(root, &mut source).unwrap().features.len(), 1);
    source
        .files
        .insert(root.join("features/application/b/b.feature"), Vec::new());
    assert_eq!(scan(root, &mut source).unwrap().features.len(), 2);
    source
        .files
        .insert(root.join("features/bad.feature"), Vec::new());
    assert!(
        scan(root, &mut source)
            .unwrap_err()
            .contains("expected features/")
    );
}
