// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn desktop_resolution_supports_preview_commit_and_parameter_replay() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_sketch(&mut owner);
    let preview = oblikovati_gui::GuiExtrudePreview {
        generation: None,
        document,
        sketch: crate::scene_fixture::SKETCH,
        profile: 0,
        distance: "width".into(),
        flipped: false,
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
    owner
        .extrude_from_gui(document, crate::scene_fixture::SKETCH, 0, "width", false)
        .unwrap();
    owner.take_scene().unwrap().unwrap();
    assert_cube(&owner, document, 1.);
    set_size(&mut owner, "20 mm");
    owner.take_scene().unwrap().unwrap();
    assert_cube(&owner, document, 2.);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    owner.take_scene().unwrap().unwrap();
    assert_cube(&owner, document, 1.);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    owner.take_scene().unwrap().unwrap();
    assert_cube(&owner, document, 2.);
    let keys = cube_keys(&owner, document);
    let path = "/memory/default-policy-cube.opd";
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.into(),
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    let reopened = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    owner.take_scene().unwrap().unwrap();
    assert_cube(&owner, reopened, 2.);
    assert_eq!(cube_keys(&owner, reopened), keys);
}

#[test]
fn stale_extrusion_request_does_not_replace_the_current_preview() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    let generation = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let mut preview = oblikovati_gui::GuiExtrudePreview {
        generation: Some(generation),
        document,
        sketch: crate::scene_fixture::SKETCH,
        profile: 0,
        distance: "width".into(),
        flipped: false,
    };
    set_size(&mut owner, "20 mm");
    preview.generation = Some(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
    );
    assert!(owner.extrusion_preview_from_gui(Some(&preview)).unwrap());
    preview.generation = Some(generation);
    assert!(!owner.extrusion_preview_from_gui(Some(&preview)).unwrap());
    assert!(owner.gui_command_state().unwrap().extrude_preview_active);
}
