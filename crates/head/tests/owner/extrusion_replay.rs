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

#[test]
fn repeated_extrusion_previews_reuse_the_current_part_and_cancel_cleanly() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    assert!(
        owner
            .extrude_from_gui(document, crate::scene_fixture::SKETCH, 0, "width")
            .unwrap()
    );
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let initial = owner.part_scenes.extrusion_prefix_previews();
    let mut preview = oblikovati_gui::GuiExtrudePreview {
        generation: Some(generation),
        document,
        sketch: crate::scene_fixture::SKETCH,
        profile: 0,
        distance: "1 cm".into(),
    };
    assert!(owner.extrusion_preview_from_gui(Some(&preview)).unwrap());
    preview.distance = "2 cm".into();
    assert!(owner.extrusion_preview_from_gui(Some(&preview)).unwrap());
    assert_eq!(owner.part_scenes.extrusion_prefix_previews(), initial + 2);
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    assert!(owner.extrusion_preview_from_gui(None).unwrap());
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
}

#[test]
fn extrusion_preview_does_not_reuse_a_different_policy() {
    use oblikovati_automation::PartUpdateAccess;
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    assert!(
        owner
            .extrude_from_gui(document, crate::scene_fixture::SKETCH, 0, "width")
            .unwrap()
    );
    let mut alternate = resolution();
    alternate.weld *= 2.;
    let mut replay = oblikovati_automation::PartReplayService::with_policy(
        oblikovati_martensite::PrismResolutionPolicy::Fixed(alternate),
    )
    .unwrap();
    replay.ensure_part(&mut owner.files, document).unwrap();
    let before = owner.part_scenes.extrusion_prefix_previews();
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let preview = oblikovati_gui::GuiExtrudePreview {
        generation: Some(generation),
        document,
        sketch: crate::scene_fixture::SKETCH,
        profile: 0,
        distance: "1 cm".into(),
    };
    assert!(owner.extrusion_preview_from_gui(Some(&preview)).unwrap());
    assert_eq!(owner.part_scenes.extrusion_prefix_previews(), before);
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    assert!(replay.current_part(&owner.files, document).is_ok());
    assert!(owner.extrusion_preview_from_gui(None).unwrap());
}
