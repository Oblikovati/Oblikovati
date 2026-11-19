// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::{DocumentSnapshot, DocumentSnapshotTarget},
    parameter::ParameterCollection,
};
use oblikovati_persistence::package::{ParameterPackageTarget, SnapshotPackageError};

fn snapshot() -> DocumentSnapshot {
    DocumentSnapshot {
        work_geometry: Default::default(),
        sketches: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
        assets: Default::default(),
        model_states: None,
        assembly: None,
        drawing: None,
        body_names: Default::default(),
        design_views: Default::default(),
        part_design_views: None,
        sketch_settings: None,
        parameters: ParameterCollection::new(Default::default()),
        properties: Default::default(),
    }
}
#[test]
fn assembly_occurrence_edits_are_saved_from_the_owned_snapshot() {
    use oblikovati_persistence::atomic::NativeAtomicFiles;
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("assembly.oad");
    let mut package =
        DocumentPackage::decode(include_bytes!("../fixtures/go-assembly-occurrences.yaml"))
            .unwrap();
    let mut records = package.assembly_occurrences().unwrap().unwrap();
    records[1].transform[3] = 8.;
    let mut document = snapshot();
    document.assembly = Some(records.clone());
    let mut guids = CountingGuidSource(0);
    let identity = FileIdentity::new("alpha", &mut guids).unwrap();
    let mut files = NativeAtomicFiles;
    let mut target = ParameterPackageTarget::new(&mut package, &path, &mut files);
    let digest = target
        .prepare(&document, 0, 0, Default::default(), 0)
        .unwrap();
    let identity = identity.prepare_save(digest, "alpha", &mut guids).unwrap();
    target.persist(&identity).unwrap();
    let saved = DocumentPackage::decode(&std::fs::read(path).unwrap()).unwrap();
    assert_eq!(saved.assembly_occurrences().unwrap(), Some(records));
}
#[test]
fn staged_title_changes_do_not_reach_the_retained_package_on_failed_storage() {
    let mut package = DocumentPackage::empty(oblikovati_api::types::DOCUMENT_PART, "Previous");
    let mut source = CountingGuidSource(0);
    let before = package.clone();
    let identity = FileIdentity::new("beta", &mut source).unwrap();
    let mut files = RefusingFiles { calls: 0 };
    let mut document = snapshot();
    document.part_design_views = Some(Default::default());
    let mut target = ParameterPackageTarget::new(&mut package, Path::new("part.opd"), &mut files)
        .with_display_name("Current title".into());
    let digest = target
        .prepare(&document, 0, 0, Default::default(), 0)
        .unwrap();
    let identity = identity.prepare_save(digest, "beta", &mut source).unwrap();
    let error = target.persist(&identity).unwrap_err();
    assert!(error.to_string().contains("fake storage unavailable"));
    drop(target);
    assert_eq!(package, before);
    assert_eq!(files.calls, 1);
}
#[test]
fn a_title_change_after_preparation_invalidates_the_old_candidate() {
    let mut package = DocumentPackage::empty(oblikovati_api::types::DOCUMENT_PART, "Previous");
    let mut source = CountingGuidSource(0);
    let before = package.clone();
    let identity = FileIdentity::new("beta", &mut source).unwrap();
    let mut files = RefusingFiles { calls: 0 };
    let mut document = snapshot();
    document.part_design_views = Some(Default::default());
    let mut target = ParameterPackageTarget::new(&mut package, Path::new("part.opd"), &mut files);
    target
        .prepare(&document, 0, 0, Default::default(), 0)
        .unwrap();
    let mut target = target.with_display_name("New title".into());
    assert!(matches!(
        target.persist(&identity),
        Err(SnapshotPackageError::Unprepared)
    ));
    drop(target);
    assert_eq!(package, before);
    assert_eq!(files.calls, 0);
}
