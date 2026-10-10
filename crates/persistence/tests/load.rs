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
