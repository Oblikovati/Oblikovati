// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::SketchPlacement;

#[test]
fn picked_face_creates_a_direct_associative_sketch_without_a_work_plane() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    let feature_count = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .work_geometry
        .features()
        .len();
    assert!(
        !owner
            .start_sketch_on_face_line(document, [[40., 40., 10.], [0., 0., -1.]])
            .unwrap()
    );
    assert!(
        owner
            .start_sketch_on_face_line(document, [[1., 1., 10.], [0., 0., -1.]])
            .unwrap()
    );
    let sketch = owner.active_sketch_edit().unwrap().sketch;
    let snapshot = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    assert_eq!(snapshot.work_geometry.features().len(), feature_count);
    assert!(matches!(
        snapshot.sketches.get(sketch).unwrap().placement(),
        SketchPlacement::Face(_)
    ));
    let state = owner.gui_command_state().unwrap();
    assert_eq!(state.dimensions_needed, Some(0));
    assert!(
        state
            .browser
            .unwrap()
            .sketches
            .iter()
            .find(|row| row.id == sketch)
            .unwrap()
            .inspection
            .as_ref()
            .unwrap()
            .healthy
    );
    owner.finish_sketch_edit().unwrap();
    set_parameter(&mut owner, "depth", "3 cm");
    assert_direct_sketch_height(&mut owner, sketch, 3.);
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_direct_sketch_height(&mut owner, sketch, 2.);
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_direct_sketch_height(&mut owner, sketch, 3.);
    let path = "/memory/direct-face-sketch.opd";
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
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap();
    assert_direct_sketch_height(&mut owner, sketch, 3.);
}
fn assert_direct_sketch_height(
    owner: &mut TestOwner,
    sketch: oblikovati_model::document_sketch::DocumentSketchId,
    expected: f64,
) {
    let scene = owner.take_sketch_scene().unwrap().unwrap();
    assert!(scene.plane_declines.is_empty());
    let plane = scene
        .sketches
        .iter()
        .find(|item| item.sketch == sketch)
        .unwrap()
        .plane;
    assert!((plane.recipe().origin[2] - expected).abs() < 1e-9);
}

#[test]
fn selected_face_starts_associative_sketch_and_rejects_stale_identity() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    owner.take_scene().unwrap();
    owner
        .click_model(document, [[1., 1., 10.], [0., 0., -1.]])
        .unwrap();
    let key = owner.gui_command_state().unwrap().selected_face_appearances[0]
        .0
        .clone();
    owner
        .click_model_with_modifiers(document, [[1., 1., -10.], [0., 0., 1.]], None, true)
        .unwrap();
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .selected_face_appearances
            .len(),
        2
    );
    assert!(owner.start_sketch_on_selected_face(document, &key).is_err());
    owner
        .click_model_with_modifiers(document, [[1., 1., -10.], [0., 0., 1.]], None, true)
        .unwrap();
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .selected_face_appearances
            .len(),
        1
    );
    let before = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .sketches()
        .len();
    assert!(
        owner
            .start_sketch_on_selected_face(document, "wrong")
            .is_err()
    );
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .sketches
            .sketches()
            .len(),
        before
    );
    set_parameter(&mut owner, "depth", "3 cm");
    assert!(owner.start_sketch_on_selected_face(document, &key).is_err());
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .sketches
            .sketches()
            .len(),
        before
    );
    owner.take_scene().unwrap();
    owner
        .click_model(document, [[1., 1., 10.], [0., 0., -1.]])
        .unwrap();
    assert!(owner.start_sketch_on_selected_face(document, &key).unwrap());
    let sketch = owner.active_sketch_edit().unwrap().sketch;
    let snapshot = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    let SketchPlacement::Face(face) = snapshot.sketches.get(sketch).unwrap().placement() else {
        panic!("expected a face-based sketch")
    };
    assert_eq!(
        face.key()
            .iter()
            .map(|byte| format!("{byte:02x}"))
            .collect::<String>(),
        key
    );
    owner.finish_sketch_edit().unwrap();
    set_parameter(&mut owner, "depth", "4 cm");
    assert_direct_sketch_height(&mut owner, sketch, 4.);
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_direct_sketch_height(&mut owner, sketch, 3.);
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_direct_sketch_height(&mut owner, sketch, 4.);
}
