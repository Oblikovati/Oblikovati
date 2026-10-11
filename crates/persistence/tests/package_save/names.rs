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
        parameters: ParameterCollection::new(Default::default()),
        properties: Default::default(),
    }
}
#[test]
fn staged_title_changes_do_not_reach_the_retained_package_on_failed_storage() {
    let mut package = DocumentPackage::empty(oblikovati_api::types::DOCUMENT_PART, "Previous");
    let mut source = CountingGuidSource(0);
    let before = package.clone();
    let identity = FileIdentity::new("beta", &mut source).unwrap();
    let mut files = RefusingFiles { calls: 0 };
    let mut target = ParameterPackageTarget::new(&mut package, Path::new("part.opd"), &mut files)
        .with_display_name("Current title".into());
    let digest = target
        .prepare(&snapshot(), 0, 0, Default::default(), 0)
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
    let mut target = ParameterPackageTarget::new(&mut package, Path::new("part.opd"), &mut files);
    target
        .prepare(&snapshot(), 0, 0, Default::default(), 0)
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
