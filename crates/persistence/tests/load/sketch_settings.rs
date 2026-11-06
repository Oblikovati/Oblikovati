// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_persistence::load::{
    LoadedParameterPackage, PackageLoadError, load_parameter_package,
};
fn load_settings(
    contents: &[u8],
    index: &mut DocumentIdentityIndex,
    source: &mut FakeGuidSequence,
) -> Result<LoadedParameterPackage, PackageLoadError> {
    load_parameter_package(
        Path::new("settings.obk"),
        1,
        "test",
        index,
        source,
        &mut reader(contents),
    )
}
#[test]
fn malformed_settings_claim_no_identity_and_consume_no_guid() {
    let mut index = DocumentIdentityIndex::default();
    let mut generator = source();
    let encoded = b"schemaVersion: 2\ndocumentType: 1\nsketchSettings: {inferConstraints: bad}\n";
    let error = load_settings(encoded, &mut index, &mut generator).unwrap_err();
    assert!(error.to_string().contains("sketchSettings"));
    assert!(index.identity(1).is_none());
    assert_eq!(generator.next, 0);
}
#[test]
fn loaded_owner_exposes_explicit_settings_without_filling_absent_fields() {
    let encoded = b"schemaVersion: 2\ndocumentType: 1\nsketchSettings: {inferConstraints: true, constraintPriority: 50434}\n";
    let loaded = load_settings(
        encoded,
        &mut DocumentIdentityIndex::default(),
        &mut source(),
    )
    .unwrap();
    let settings = loaded.sketch_settings.unwrap();
    assert!(settings.infer_constraints);
    assert_eq!(settings.constraint_priority.0, 50434);
    assert!(!settings.auto_apply_constraints);
    assert_eq!(settings.x_snap_spacing, 0.0);
}
