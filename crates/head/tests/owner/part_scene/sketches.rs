// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn authoring_display_is_cached_and_independent_of_solid_failure() {
    let (mut owner, _) = owner();
    assert!(owner.take_sketch_scene().unwrap().is_none());
    let document = open_prism(&mut owner, false);
    let first = owner.take_sketch_scene().unwrap().unwrap();
    assert_eq!(first.source.unwrap().0, document);
    assert_eq!(first.sketches.len(), 1);
    assert!(first.plane_declines.is_empty());
    assert!(owner.take_sketch_scene().unwrap().is_none());
    set_parameter(&mut owner, "depth", "0 cm");
    let failed = owner.take_scene().unwrap().unwrap();
    assert_eq!(failed.status.visible_bodies, 0);
    let update = owner.take_sketch_scene().unwrap().unwrap();
    assert!(update.source.unwrap().1 > first.source.unwrap().1);
    assert_eq!(update.sketches, first.sketches);
    verify_closed_owner_scene(&mut owner, document);
    let closed = owner.take_sketch_scene().unwrap().unwrap();
    assert!(closed.source.is_none() && closed.sketches.is_empty());
}
#[test]
fn edit_finish_preserves_display_and_history_restores_geometry() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let first = owner.take_sketch_scene().unwrap().unwrap();
    owner.begin_sketch_edit(document, fixture::SKETCH).unwrap();
    assert!(owner.take_sketch_scene().unwrap().is_none());
    owner.finish_sketch_edit().unwrap();
    assert!(owner.take_sketch_scene().unwrap().is_none());
    set_parameter(&mut owner, "width", "6 cm");
    let edited = owner.take_sketch_scene().unwrap().unwrap();
    assert_ne!(edited.sketches, first.sketches);
    verify_display_history(&mut owner, &first, &edited);
}

#[test]
fn document_focus_replaces_the_owned_sketch_packet() {
    let (mut owner, awake) = owner();
    let first = open_prism(&mut owner, false);
    let before = owner.take_sketch_scene().unwrap().unwrap();
    let second = create(&mut owner, &awake);
    let empty = owner.take_sketch_scene().unwrap().unwrap();
    assert_eq!(empty.source.unwrap().0, second);
    assert!(empty.sketches.is_empty());
    Client::new(owner.document_host())
        .documents()
        .activate(&ActivateDocumentArgs { id: first })
        .unwrap();
    let restored = owner.take_sketch_scene().unwrap().unwrap();
    assert_eq!(restored.source, before.source);
    assert_eq!(restored.sketches, before.sketches);
}

fn verify_display_history(
    owner: &mut TestOwner,
    first: &crate::HeadSketchSceneUpdate,
    edited: &crate::HeadSketchSceneUpdate,
) {
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(
        owner.take_sketch_scene().unwrap().unwrap().sketches,
        first.sketches
    );
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(
        owner.take_sketch_scene().unwrap().unwrap().sketches,
        edited.sketches
    );
}
