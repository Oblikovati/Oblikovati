// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::identity::{
    DocumentGuid, DocumentIdentityIndex, DocumentSaveTarget, FileIdentity, GuidSource,
    IdentityTransitionError,
};
use oblikovati_persistence::{
    atomic::{AtomicFiles, NativeAtomicFiles, StagedFile},
    package::{DocumentPackage, PackageSaveError, PackageSaveTarget},
};
use std::{fs, io, path::Path};

struct CountingGuidSource(u64);
impl GuidSource for CountingGuidSource {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        self.0 += 1;
        Ok(format!("00000000-0000-4000-8000-{:012x}", self.0)
            .parse()
            .unwrap())
    }
}

struct RefusingFiles {
    calls: usize,
}
impl AtomicFiles for RefusingFiles {
    type Stage = UnreachableStage;
    fn stage(&mut self, _destination: &Path) -> io::Result<Self::Stage> {
        self.calls += 1;
        Err(io::Error::other("fake storage unavailable"))
    }
}
struct UnreachableStage;
impl StagedFile for UnreachableStage {
    fn write_contents(&mut self, _contents: &[u8]) -> io::Result<()> {
        unreachable!()
    }
    fn synchronize(&mut self) -> io::Result<()> {
        unreachable!()
    }
    fn commit(self, _destination: &Path) -> io::Result<()> {
        unreachable!()
    }
}

fn open_fixture() -> (DocumentPackage, DocumentIdentityIndex, CountingGuidSource) {
    let package = DocumentPackage::decode(include_bytes!("fixtures/go-package.yaml")).unwrap();
    let mut source = CountingGuidSource(0);
    let fresh = FileIdentity::new("alpha", &mut source).unwrap();
    let identity = package
        .identity_record()
        .unwrap()
        .unwrap()
        .restore(&fresh)
        .unwrap();
    let mut index = DocumentIdentityIndex::default();
    let _registration = index.register(1, identity, &mut source).unwrap();
    (package, index, source)
}

#[test]
fn native_save_reopen_matches_registered_identity_and_preserves_sections() {
    let (mut package, mut index, mut source) = open_fixture();
    let before = package.clone();
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("part.obk");
    let digest = package.recipe_digest().unwrap();
    let mut native = NativeAtomicFiles;
    let mut target = PackageSaveTarget::new(&mut package, &path, &mut native);
    index
        .save(1, digest, "beta", &mut source, &mut target)
        .unwrap();
    let loaded = DocumentPackage::decode(&fs::read(&path).unwrap()).unwrap();
    let current = index.identity(1).unwrap();
    assert_eq!(
        loaded
            .identity_record()
            .unwrap()
            .unwrap()
            .restore(current)
            .unwrap(),
        *current
    );
    assert_eq!(loaded, package);
    assert_eq!(current.save_counter, 3);
    assert_preserved_sections(&before, &loaded);
}

fn assert_preserved_sections(before: &DocumentPackage, after: &DocumentPackage) {
    for name in ["model", "data", "resources", "attributes", "bodyNames"] {
        assert_eq!(
            before.section_yaml(name).unwrap(),
            after.section_yaml(name).unwrap()
        );
    }
    assert_eq!(before.manifest(), after.manifest());
    assert_eq!(before.resources(), after.resources());
    assert_eq!(before.streams(), after.streams());
    assert_eq!(
        before.recipe_digest().unwrap(),
        after.recipe_digest().unwrap()
    );
}

#[test]
fn failed_storage_changes_neither_registered_identity_nor_package() {
    let (mut package, mut index, mut source) = open_fixture();
    let before = package.clone();
    let original = index.identity(1).unwrap().clone();
    let digest = package.recipe_digest().unwrap();
    let mut files = RefusingFiles { calls: 0 };
    let mut target = PackageSaveTarget::new(&mut package, Path::new("part.obk"), &mut files);
    let error = index
        .save(1, digest, "beta", &mut source, &mut target)
        .unwrap_err();
    assert!(error.to_string().contains("fake storage unavailable"));
    assert!(std::error::Error::source(&error).is_some());
    assert_eq!(package, before);
    assert_eq!(index.identity(1), Some(&original));
    assert_eq!(files.calls, 1);
}

#[test]
fn mismatched_recipe_digest_is_refused_before_io() {
    let (mut package, index, _) = open_fixture();
    let before = package.clone();
    let candidate = index.identity(1).unwrap().clone();
    let mut files = RefusingFiles { calls: 0 };
    let mut target = PackageSaveTarget::new(&mut package, Path::new("part.obk"), &mut files);
    let error = target.persist(&candidate).unwrap_err();
    assert!(matches!(error, PackageSaveError::DigestMismatch { .. }));
    assert!(error.to_string().contains("expected snapshot digest"));
    assert!(std::error::Error::source(&error).is_none());
    assert_eq!(files.calls, 0);
    assert_eq!(package, before);
}

#[test]
fn unknown_identity_fields_cannot_be_lost_by_replacement() {
    let input = b"schemaVersion: 2\ndocumentType: 1\nidentity: {futureMeaning: secret}\n";
    let mut package = DocumentPackage::decode(input).unwrap();
    let before = package.clone();
    let mut source = CountingGuidSource(0);
    let candidate = FileIdentity::new("alpha", &mut source).unwrap();
    let mut files = RefusingFiles { calls: 0 };
    let mut target = PackageSaveTarget::new(&mut package, Path::new("part.obk"), &mut files);
    let error = target.persist(&candidate).unwrap_err();
    assert!(matches!(error, PackageSaveError::Codec(_)));
    assert!(error.to_string().contains("futureMeaning"));
    assert!(std::error::Error::source(&error).is_some());
    assert_eq!(package, before);
    assert_eq!(files.calls, 0);
}

#[test]
fn absent_or_null_recipe_has_no_fingerprint_and_absent_identity_is_optional() {
    for suffix in ["", "model: null\nidentity: null\n"] {
        let input = format!("schemaVersion: 2\ndocumentType: 1\n{suffix}");
        let package = DocumentPackage::decode(input.as_bytes()).unwrap();
        assert_eq!(package.recipe_digest().unwrap(), None);
        assert_eq!(package.identity_record().unwrap(), None);
    }
}

#[test]
fn repeat_native_save_keeps_the_model_revision_and_advances_the_file_revision() {
    let (mut package, mut index, mut source) = open_fixture();
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("part.obk");
    let digest = package.recipe_digest().unwrap();
    let mut native = NativeAtomicFiles;
    let mut target = PackageSaveTarget::new(&mut package, &path, &mut native);
    index
        .save(1, digest, "beta", &mut source, &mut target)
        .unwrap();
    let first = index.identity(1).unwrap().clone();
    index
        .save(1, digest, "gamma", &mut source, &mut target)
        .unwrap();
    let second = index.identity(1).unwrap();
    assert_eq!(second.database_revision_id, first.database_revision_id);
    assert_ne!(second.revision_id, first.revision_id);
    assert_eq!(second.save_counter, first.save_counter + 1);
}

#[test]
fn malformed_existing_identity_cannot_be_silently_replaced() {
    let input = b"schemaVersion: 2\ndocumentType: 1\nidentity: {internalName: invalid-guid}\n";
    let mut package = DocumentPackage::decode(input).unwrap();
    let mut source = CountingGuidSource(0);
    let candidate = FileIdentity::new("alpha", &mut source).unwrap();
    let mut files = RefusingFiles { calls: 0 };
    let mut target = PackageSaveTarget::new(&mut package, Path::new("part.obk"), &mut files);
    let error = target.persist(&candidate).unwrap_err();
    assert!(error.to_string().contains("invalid-guid"));
    assert_eq!(files.calls, 0);
}

#[test]
fn edited_parameter_recipe_survives_native_save_and_reopen() {
    use oblikovati_api::types::USER_PARAM;
    use oblikovati_model::{parameter::ParameterCollection, units::UnitsOfMeasure};
    let (original, mut index, mut source) = open_fixture();
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "10 mm")
        .unwrap();
    parameters.set_expression(1, "25 mm").unwrap();
    let mut updated = original
        .with_parameters(&parameters, UnitsOfMeasure::default())
        .unwrap();
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("edited.obk");
    let digest = updated.recipe_digest().unwrap();
    let mut files = NativeAtomicFiles;
    let mut target = PackageSaveTarget::new(&mut updated, &path, &mut files);
    index
        .save(1, digest, "beta", &mut source, &mut target)
        .unwrap();
    let reopened = DocumentPackage::decode(&fs::read(path).unwrap()).unwrap();
    let restored = reopened
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap();
    assert_eq!(
        restored
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        2.5
    );
    assert_eq!(reopened.streams(), original.streams());
}

#[path = "package_save/names.rs"]
mod names;
