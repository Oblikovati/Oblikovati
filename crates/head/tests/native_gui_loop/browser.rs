// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::SketchPlacement, history::DocumentSketchCreation, sketch::SketchPlane,
};

#[test]
fn viewport_click_uses_priority_for_a_body_and_its_face() {
    let directory = tempfile::tempdir().unwrap();
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner_at(native.wake.clone(), directory.path()).unwrap();
    let document = open_native_part(&mut owner, &directory.path().join("priority.opd"));
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    let pixel = window.native.view.model_pixel([2., 1.5, 2.]).unwrap();
    window.scripted = pointer_click_events(pixel, 1);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    let state = owner.gui_command_state().unwrap();
    assert!(state.browser.unwrap().selected_body.is_some());
    let filter = owner.gui_command_state().unwrap().selection_filter.unwrap();
    let face = filter
        .entries
        .iter()
        .position(|entry| entry.kind == oblikovati_gui::GuiSelectionKind::Face)
        .unwrap();
    owner.move_selection_kind_from_gui(face, 0).unwrap();
    window.scripted = pointer_click_events(pixel, 1);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    assert!(owner.gui_command_state().unwrap().model_face_chosen);
    assert_eq!(
        owner.documents().workspace().active_document(),
        Some(document)
    );
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn native_face_sketch_and_measure_choose_a_face_when_body_has_priority() {
    let directory = tempfile::tempdir().unwrap();
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner_at(native.wake.clone(), directory.path()).unwrap();
    let document = open_native_part(&mut owner, &directory.path().join("face-tools.opd"));
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Start 2D Sketch"),
    );
    let pixel = window.native.view.model_pixel([2., 1.5, 2.]).unwrap();
    window.scripted = pointer_click_events(pixel, 1);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    let edit = owner
        .active_sketch_edit()
        .expect("a face sketch starts from the viewport click");
    let session = owner.documents().workspace().session(document).unwrap();
    assert!(matches!(
        session
            .snapshot()
            .sketches
            .get(edit.sketch)
            .unwrap()
            .placement(),
        SketchPlacement::Face(_)
    ));
    owner.finish_sketch_edit().unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    owner
        .inspect_from_gui(oblikovati_gui::GuiInspectAction::StartMeasure)
        .unwrap();
    let pixel = window.native.view.model_pixel([2., 1.5, 2.]).unwrap();
    window.scripted = pointer_click_events(pixel, 1);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    assert!(owner.gui_command_state().unwrap().model_face_chosen);
    assert!(
        owner
            .gui_command_state()
            .unwrap()
            .measure
            .unwrap()
            .rows
            .iter()
            .any(|row| row.contains("Area"))
    );
    owner.finish().unwrap();
    window.finish().unwrap();
}

/// Queue one gesture without presentation gaps; example: `double_click_browser_row(window, owner)`.
pub(super) fn double_click_browser_row(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
) {
    // #3798: two separately presented single clicks depend on display speed, not input intent.
    drive_pointer_clicks(
        window,
        owner,
        application_bdd::rendered_sketch_row_center(),
        2,
    );
}
fn seed_browser_sketch(owner: &mut NativeDesktopOwner) -> crate::HeadSketchEdit {
    owner.create_part_document().unwrap();
    let document = owner.documents().workspace().active_document().unwrap();
    let request = DocumentSketchCreation {
        name: "Sketch".into(),
        placement: SketchPlacement::Fixed(SketchPlane::xy()),
        orientation: None,
        project_origin: true,
    };
    let sketch = owner.create_sketch_for_edit(document, &request).unwrap();
    owner.finish_sketch_edit().unwrap();
    crate::HeadSketchEdit {
        document,
        sketch,
        plane: SketchPlane::xy(),
        look_at_plane: true,
    }
}
#[test]
fn browser_double_click_is_independent_of_slow_presentation() {
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    let edit = seed_browser_sketch(&mut owner);
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    window.presentation_delay = Duration::from_millis(400);
    double_click_browser_row(&mut window, &mut owner);
    assert_eq!(owner.active_sketch_edit(), Some(edit));
    owner.finish().unwrap();
    window.finish().unwrap();
}
