// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn closing_the_document_clears_its_preview_from_the_scene() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    let preview = oblikovati_gui::GuiExtrudePreview {
        generation: None,
        document,
        sketch: crate::scene_fixture::SKETCH,
        profile: 0,
        distance: "width".into(),
        second_distance: None,
        direction: Default::default(),
    };
    owner.extrusion_preview_from_gui(Some(&preview)).unwrap();
    assert!(
        !owner
            .take_scene()
            .unwrap()
            .unwrap()
            .scene
            .triangles
            .is_empty()
    );
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    assert!(
        owner
            .take_scene()
            .unwrap()
            .unwrap()
            .scene
            .triangles
            .is_empty()
    );
}
