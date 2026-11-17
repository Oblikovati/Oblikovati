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

#[test]
fn measure_initial_face_requires_one_current_unmixed_selection() {
    let (mut owner, document) = measure_face_fixture();
    start_measure(&mut owner);
    let rows = owner.gui_command_state().unwrap().measure.unwrap().rows;
    assert_eq!(rows[0], "Selection 1: Face");
    assert!(rows[1].starts_with("Area "));
    finish_measure(&mut owner);
    toggle_back_face(&mut owner, document);
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .selected_face_appearances
            .len(),
        2
    );
    assert_measure_requests_face(&mut owner);
    toggle_back_face(&mut owner, document);
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .selected_face_appearances
            .len(),
        1
    );
    set_parameter(&mut owner, "depth", "3 cm");
    assert_measure_requests_face(&mut owner);
}
fn measure_face_fixture() -> (TestOwner, u64) {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    owner.take_scene().unwrap();
    owner
        .click_model(document, [[1., 1., 10.], [0., 0., -1.]])
        .unwrap();
    (owner, document)
}
fn toggle_back_face(owner: &mut TestOwner, document: u64) {
    owner
        .click_model_with_modifiers(document, [[1., 1., -10.], [0., 0., 1.]], None, true)
        .unwrap();
}
fn start_measure(owner: &mut TestOwner) {
    owner
        .inspect_from_gui(oblikovati_gui::GuiInspectAction::StartMeasure)
        .unwrap();
}
fn finish_measure(owner: &mut TestOwner) {
    owner
        .inspect_from_gui(oblikovati_gui::GuiInspectAction::FinishMeasure)
        .unwrap();
}
fn assert_measure_requests_face(owner: &mut TestOwner) {
    start_measure(owner);
    assert_eq!(
        owner.gui_command_state().unwrap().measure.unwrap().rows,
        ["Select a solid face to measure its area"]
    );
    finish_measure(owner);
}
#[test]
fn measure_does_not_preselect_a_face_when_a_body_is_also_selected() {
    let (mut owner, document) = measure_face_fixture();
    let state = owner.gui_command_state().unwrap();
    let body = &state.browser.unwrap().bodies[0];
    owner
        .files
        .commands(document)
        .unwrap()
        .set_browser_body_selection(&body.key, body.feature);
    let state = owner.gui_command_state().unwrap();
    assert_eq!(state.selected_face_appearances.len(), 1);
    assert_eq!(state.selected_body_keys.len(), 1);
    assert_measure_requests_face(&mut owner);
}

#[test]
fn face_measure_command_rejects_changed_selection_and_stale_identity() {
    let (mut owner, document) = measure_face_fixture();
    let key = owner.gui_command_state().unwrap().selected_face_appearances[0]
        .0
        .clone();
    assert!(owner.measure_selected_face(document, "wrong").is_err());
    toggle_back_face(&mut owner, document);
    assert!(owner.measure_selected_face(document, &key).is_err());
    toggle_back_face(&mut owner, document);
    assert!(owner.measure_selected_face(document, &key).unwrap());
    finish_measure(&mut owner);
    set_parameter(&mut owner, "depth", "3 cm");
    assert!(owner.measure_selected_face(document, &key).is_err());
    assert!(owner.gui_command_state().unwrap().measure.is_none());
}

#[test]
fn face_properties_use_exact_area_and_expire_after_model_change() {
    let (mut owner, document) = measure_face_fixture();
    let key = owner.gui_command_state().unwrap().selected_face_appearances[0]
        .0
        .clone();
    assert!(owner.open_face_properties(document, "wrong").is_err());
    assert!(owner.open_face_properties(document, &key).unwrap());
    let face = owner.gui_command_state().unwrap().face_properties.unwrap();
    assert_eq!(face.key, key);
    assert!((face.area_mm2 - 1200.).abs() < 1e-6);
    assert!(owner.close_face_properties().unwrap());
    assert!(owner.gui_command_state().unwrap().face_properties.is_none());
    assert!(owner.open_face_properties(document, &key).unwrap());
    set_parameter(&mut owner, "depth", "3 cm");
    assert!(owner.gui_command_state().unwrap().face_properties.is_none());
    assert!(owner.retire_stale_face_properties());
    assert!(!owner.retire_stale_face_properties());
    assert!(owner.open_face_properties(document, &key).is_err());
}
