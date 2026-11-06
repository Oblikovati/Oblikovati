// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::SketchPlacement, history::DocumentSketchCreation, sketch::SketchPlane,
};

/// Queue one gesture without presentation gaps; example: `double_click_browser_row(window, owner)`.
pub(super) fn double_click_browser_row(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
) {
    // #3798: two separately presented single clicks depend on display speed, not input intent.
    drive_pointer_clicks(window, owner, [90, 389], 2);
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
