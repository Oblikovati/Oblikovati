// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn extrusion_commit_publishes_the_certified_document_result_before_display() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    assert!(
        owner
            .extrude_from_gui(document, crate::scene_fixture::SKETCH, 0, "width")
            .unwrap()
    );
    let session = owner.files.workspace().session(document).unwrap();
    let policy = oblikovati_martensite::PrismResolutionPolicy::Fixed(resolution());
    let result = session
        .owned_part_slot(Some(policy))
        .unwrap()
        .current(session)
        .unwrap();
    assert_eq!(result.ordered_bodies().count(), 1);
}

#[test]
fn second_extrusion_reuses_the_current_document_prefix() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    assert!(
        owner
            .extrude_from_gui(document, crate::scene_fixture::SKETCH, 0, "width")
            .unwrap()
    );
    let policy = oblikovati_martensite::PrismResolutionPolicy::Fixed(resolution());
    let session = owner.files.workspace().session(document).unwrap();
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let first = session
        .owned_part_slot(Some(policy))
        .unwrap()
        .current(session)
        .unwrap();
    let keys = first
        .body(oblikovati_model::feature::FeatureId(1))
        .unwrap()
        .face_reference_keys(session.history(), guid)
        .unwrap();
    assert!(
        owner
            .extrude_from_gui(document, crate::scene_fixture::SKETCH, 0, "width")
            .unwrap()
    );
    let session = owner.files.workspace().session(document).unwrap();
    let result = session
        .owned_part_slot(Some(policy))
        .unwrap()
        .current(session)
        .unwrap();
    assert_eq!(result.ordered_bodies().count(), 2);
    assert_eq!(
        result
            .body(oblikovati_model::feature::FeatureId(1))
            .unwrap()
            .face_reference_keys(session.history(), guid)
            .unwrap(),
        keys
    );
}
