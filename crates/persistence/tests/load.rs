// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::identity::{
    DocumentGuid, DocumentIdentityIndex, GuidSource, IdentityRegistration, IdentityTransitionError,
};
use oblikovati_persistence::{
    atomic::NativeAtomicFiles,
    load::{NativePackageReader, PackageFileReader, load_registered},
    package::PackageSaveTarget,
};
use std::{io, path::Path};

const FIXTURE: &[u8] = include_bytes!("fixtures/go-package.yaml");

struct FakePackageReader {
    contents: Vec<u8>,
    fail: bool,
}
impl PackageFileReader for FakePackageReader {
    fn read(&mut self, _path: &Path) -> io::Result<Vec<u8>> {
        if self.fail {
            return Err(io::Error::other("fake read refused"));
        }
        Ok(self.contents.clone())
    }
}

struct FakeGuidSequence {
    next: u64,
    fail: bool,
}
impl GuidSource for FakeGuidSequence {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        if self.fail {
            return Err(IdentityTransitionError::Randomness(
                "fake entropy refused".into(),
            ));
        }
        self.next += 1;
        Ok(format!("00000000-0000-4000-8000-{:012x}", self.next)
            .parse()
            .unwrap())
    }
}

fn source() -> FakeGuidSequence {
    FakeGuidSequence {
        next: 0,
        fail: false,
    }
}
fn reader(contents: &[u8]) -> FakePackageReader {
    FakePackageReader {
        contents: contents.to_vec(),
        fail: false,
    }
}

#[test]
fn loaded_identity_is_preserved_and_copied_file_collision_is_explicit() {
    let mut index = DocumentIdentityIndex::default();
    let mut source = source();
    let mut reader = reader(FIXTURE);
    let first = load_registered(
        Path::new("first.obk"),
        1,
        "alpha",
        &mut index,
        &mut source,
        &mut reader,
    )
    .unwrap();
    assert_eq!(first.registration, IdentityRegistration::Preserved);
    let original = index.identity(1).unwrap().internal_name;
    let second = load_registered(
        Path::new("copy.obk"),
        2,
        "alpha",
        &mut index,
        &mut source,
        &mut reader,
    )
    .unwrap();
    assert_eq!(first.package, second.package);
    assert_eq!(
        second.registration,
        IdentityRegistration::Reassigned {
            previous: original,
            current: index.identity(2).unwrap().internal_name
        }
    );
    assert_eq!(index.owner(original), Some(1));
}

#[test]
fn every_failed_load_leaves_the_session_unregistered() {
    for contents in [
        b"not a mapping".as_slice(),
        b"schemaVersion: 2\ndocumentType: 1\nidentity: {internalName: bad}",
        b"schemaVersion: 2\ndocumentType: 1\nidentity: {future: unknown}",
    ] {
        let mut index = DocumentIdentityIndex::default();
        let error = load_registered(
            Path::new("bad.obk"),
            1,
            "alpha",
            &mut index,
            &mut source(),
            &mut reader(contents),
        )
        .unwrap_err();
        assert_eq!(error.path, Path::new("bad.obk"));
        assert!(error.to_string().contains("bad.obk"));
        let cause = std::error::Error::source(&error).unwrap();
        assert!(cause.source().is_some());
        assert!(index.identity(1).is_none());
    }
}

#[test]
fn read_and_entropy_failures_are_reported_without_registration() {
    let mut index = DocumentIdentityIndex::default();
    let mut input = reader(FIXTURE);
    input.fail = true;
    let mut generator = source();
    let error = load_registered(
        Path::new("part.obk"),
        1,
        "alpha",
        &mut index,
        &mut generator,
        &mut input,
    )
    .unwrap_err();
    assert!(error.to_string().contains("fake read refused"));
    assert_eq!(generator.next, 0);
    input.fail = false;
    generator.fail = true;
    let error = load_registered(
        Path::new("part.obk"),
        1,
        "alpha",
        &mut index,
        &mut generator,
        &mut input,
    )
    .unwrap_err();
    assert!(error.to_string().contains("fake entropy refused"));
    assert!(index.identity(1).is_none());
}

#[test]
fn absent_identity_gains_a_stable_unsaved_identity_and_duplicate_session_is_refused() {
    let mut index = DocumentIdentityIndex::default();
    let mut generator = source();
    let mut input = reader(b"schemaVersion: 2\ndocumentType: 1\n");
    let _loaded = load_registered(
        Path::new("old.obk"),
        1,
        "alpha",
        &mut index,
        &mut generator,
        &mut input,
    )
    .unwrap();
    let before = index.identity(1).unwrap().clone();
    assert_eq!(before.save_counter, 0);
    assert_eq!(before.version_created, "alpha");
    let error = load_registered(
        Path::new("other.obk"),
        1,
        "alpha",
        &mut index,
        &mut generator,
        &mut input,
    )
    .unwrap_err();
    assert!(error.to_string().contains("unregistered ID"));
    assert_eq!(index.identity(1), Some(&before));
}

#[test]
fn native_load_save_close_reopen_preserves_the_written_identity() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("part.obk");
    std::fs::write(&path, FIXTURE).unwrap();
    let mut index = DocumentIdentityIndex::default();
    let mut generator = source();
    let mut loaded = load_registered(
        &path,
        1,
        "alpha",
        &mut index,
        &mut generator,
        &mut NativePackageReader,
    )
    .unwrap();
    let digest = loaded.package.recipe_digest().unwrap();
    let mut files = NativeAtomicFiles;
    let mut target = PackageSaveTarget::new(&mut loaded.package, &path, &mut files);
    index
        .save(1, digest, "beta", &mut generator, &mut target)
        .unwrap();
    let saved = index.close(1).unwrap();
    let reopened = load_registered(
        &path,
        2,
        "beta",
        &mut index,
        &mut generator,
        &mut NativePackageReader,
    )
    .unwrap();
    assert_eq!(reopened.registration, IdentityRegistration::Preserved);
    assert_eq!(index.identity(2), Some(&saved));
    assert_eq!(reopened.package, loaded.package);
}

#[test]
fn missing_native_file_returns_its_requested_path() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("missing.obk");
    let mut index = DocumentIdentityIndex::default();
    let error = load_registered(
        &path,
        1,
        "alpha",
        &mut index,
        &mut source(),
        &mut NativePackageReader,
    )
    .unwrap_err();
    assert_eq!(error.path, path);
    assert!(index.identity(1).is_none());
}

#[test]
fn parameter_reconstruction_errors_never_claim_identity_or_consume_entropy() {
    use oblikovati_persistence::load::load_parameter_package;
    let mut index = DocumentIdentityIndex::default();
    let mut generator = source();
    for model in [
        "{units: {workingScale: '0'}}",
        "{parameters: [{name: p, kind: user, expression: missing}]}",
        "{parameters: [{name: p, kind: user, expression: '1', tolerance: {type: bad}}]}",
        "{parameterGroups: [{internalName: group, members: [missing]}]}",
        "{derivedParameterTables: [{id: 0, sourceDocument: part.opd}]}",
        "{parameterSettings: {dimensionDisplayType: bad}}",
    ] {
        let encoded = format!("schemaVersion: 2\ndocumentType: 1\nmodel: {model}\n");
        let error = load_parameter_package(
            Path::new("bad.opd"),
            1,
            "alpha",
            &mut index,
            &mut generator,
            &mut reader(encoded.as_bytes()),
        )
        .unwrap_err();
        assert_eq!(error.path, Path::new("bad.opd"));
        assert!(error.to_string().contains("bad.opd"));
        assert!(std::error::Error::source(&error).is_some());
        assert!(index.identity(1).is_none());
        assert_eq!(generator.next, 0, "{model}");
    }
}

#[test]
fn validated_parameter_packages_keep_collision_outcomes_and_existing_claims() {
    use oblikovati_persistence::load::load_parameter_package;
    let encoded = b"schemaVersion: 2\ndocumentType: 1\nidentity: {internalName: '11111111-2222-3333-4444-555555555555'}\nmodel:\n  units: {workingScale: '0.0001'}\n  parameters: [{name: width, kind: user, expression: '1 mm'}]\n";
    let mut index = DocumentIdentityIndex::default();
    let mut generator = source();
    let first = load_parameter_package(
        Path::new("part.opd"),
        1,
        "alpha",
        &mut index,
        &mut generator,
        &mut reader(encoded),
    )
    .unwrap();
    let identity = index.identity(1).unwrap().clone();
    assert_eq!(
        first
            .restored
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        1000.0
    );
    assert_eq!(first.loaded.registration, IdentityRegistration::Preserved);
    let second = load_parameter_package(
        Path::new("copy.opd"),
        2,
        "alpha",
        &mut index,
        &mut generator,
        &mut reader(encoded),
    )
    .unwrap();
    assert!(matches!(
        second.loaded.registration,
        IdentityRegistration::Reassigned { .. }
    ));
    assert_eq!(index.identity(1).unwrap(), &identity);
    assert!(
        load_parameter_package(
            Path::new("part.opd"),
            1,
            "alpha",
            &mut index,
            &mut generator,
            &mut reader(encoded)
        )
        .is_err()
    );
    assert_eq!(index.identity(1).unwrap(), &identity);
}

#[test]
fn native_parameter_load_edit_save_and_reopen_preserves_stored_scale() {
    use oblikovati_persistence::load::load_parameter_package;
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("part.opd");
    std::fs::write(&path, b"schemaVersion: 2\ndocumentType: 1\ndata: {addin: YWJj}\nmodel:\n  units: {workingScale: '0.0001'}\n  parameters: [{name: width, kind: user, expression: '1 mm'}]\n").unwrap();
    let mut index = DocumentIdentityIndex::default();
    let mut generator = source();
    let mut loaded = load_parameter_package(
        &path,
        1,
        "alpha",
        &mut index,
        &mut generator,
        &mut NativePackageReader,
    )
    .unwrap();
    loaded
        .restored
        .parameters
        .set_expression(1, "2 mm")
        .unwrap();
    let units = loaded.loaded.package.document_units().unwrap();
    let mut updated = loaded
        .loaded
        .package
        .with_parameters(&loaded.restored.parameters, units)
        .unwrap();
    let digest = updated.recipe_digest().unwrap();
    let mut files = NativeAtomicFiles;
    let mut target = PackageSaveTarget::new(&mut updated, &path, &mut files);
    index
        .save(1, digest, "alpha", &mut generator, &mut target)
        .unwrap();
    let saved_identity = index.close(1).unwrap();
    let reopened = load_parameter_package(
        &path,
        2,
        "alpha",
        &mut index,
        &mut generator,
        &mut NativePackageReader,
    )
    .unwrap();
    assert_eq!(index.identity(2).unwrap(), &saved_identity);
    assert_eq!(
        reopened
            .restored
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        2000.0
    );
    assert_eq!(reopened.loaded.package.streams()["addin"], b"abc");
}

#[test]
fn parameter_package_identity_errors_do_not_register_a_reconstructed_candidate() {
    use oblikovati_persistence::load::load_parameter_package;
    let encoded = b"schemaVersion: 2\ndocumentType: 1\nidentity: {internalName: malformed}\nmodel: {parameters: [{name: p, kind: user, expression: '1'}]}\n";
    let mut index = DocumentIdentityIndex::default();
    let error = load_parameter_package(
        Path::new("bad.opd"),
        1,
        "alpha",
        &mut index,
        &mut source(),
        &mut reader(encoded),
    )
    .unwrap_err();
    assert!(error.to_string().contains("malformed"));
    assert!(index.identity(1).is_none());
}

#[test]
fn document_properties_are_reconstructed_before_registration() {
    use oblikovati_api::types::VariantValue;
    use oblikovati_persistence::load::load_parameter_package;
    let mut index = DocumentIdentityIndex::default();
    let mut generator = source();
    for (property_type, expected_ok) in [("unknown", false), ("integer", true)] {
        let encoded = format!(
            "schemaVersion: 2\ndocumentType: 1\nmodel: {{properties: [{{set: Custom, name: Count, type: {property_type}, value: '4'}}]}}\n"
        );
        let result = load_parameter_package(
            Path::new("properties.opd"),
            1,
            "alpha",
            &mut index,
            &mut generator,
            &mut reader(encoded.as_bytes()),
        );
        if !expected_ok {
            assert!(result.is_err());
            assert!(index.identity(1).is_none());
            assert_eq!(generator.next, 0);
            continue;
        }
        assert_eq!(
            result
                .unwrap()
                .properties
                .get("Custom", "Count")
                .unwrap()
                .value,
            VariantValue::Integer(4)
        );
        assert!(index.identity(1).is_some());
    }
}

#[test]
fn loaded_parameter_export_flags_rebuild_property_provenance() {
    use oblikovati_api::types::VariantValue;
    use oblikovati_model::property::USER_DEFINED;
    use oblikovati_persistence::load::load_parameter_package;
    let encoded = b"schemaVersion: 2\ndocumentType: 1\nmodel: {parameters: [{name: label, kind: user, valueType: text, text: steel, export: true}]}\n";
    let loaded = load_parameter_package(
        Path::new("exposed.opd"),
        1,
        "alpha",
        &mut DocumentIdentityIndex::default(),
        &mut source(),
        &mut reader(encoded),
    )
    .unwrap();
    let property = loaded.properties.get(USER_DEFINED, "label").unwrap();
    assert_eq!(property.value, VariantValue::String("steel".into()));
    assert_eq!(property.from_parameter, "label");
}

#[test]
fn malformed_attributes_claim_no_document_identity() {
    use oblikovati_persistence::load::load_parameter_package;
    let mut index = DocumentIdentityIndex::default();
    let mut generator = source();
    let encoded = b"schemaVersion: 2\ndocumentType: 1\nattributes: [1, 0, 0, 0]\n";
    let result = load_parameter_package(
        Path::new("attributes.opd"),
        1,
        "alpha",
        &mut index,
        &mut generator,
        &mut reader(encoded),
    );
    assert!(result.unwrap_err().to_string().contains("attributes"));
    assert!(index.identity(1).is_none());
    assert_eq!(generator.next, 0);
}

#[path = "load/interests.rs"]
mod interests;
