// SPDX-License-Identifier: GPL-2.0-only
//! Verify the actual committed sketch edges in the unobstructed document view.
use super::*;
use oblikovati_viewport::ViewportCapture;

/// Check the committed edges against their projected location before sketch re-entry.
/// Example: `assert_committed_rectangle_visible(window, owner, edit, &capture)`.
pub(super) fn assert_committed_rectangle_visible(
    window: &mut ScriptedGuiWindow,
    owner: &NativeDesktopOwner,
    edit: crate::HeadSketchEdit,
    capture: &ViewportCapture,
) {
    save_optional_capture(capture);
    assert!(
        visible_in_view(window, owner, edit, capture),
        "sketch {:?} in viewport {:?}: expected four unobstructed committed rectangle edges",
        edit.sketch,
        capture.extent
    );
}
/// Hide the sketch in an aborted transaction to test the oracle without history edits.
/// Example: `assert_hidden_rectangle_fails(window, owner, edit)`.
pub(super) fn assert_hidden_rectangle_fails(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    edit: crate::HeadSketchEdit,
) {
    let history = history_entries(owner, edit);
    let before = begin_visibility_probe(owner, edit);
    assert_visible_frame(window, owner, edit, false);
    let after = oblikovati_api::client::Client::new(owner.document_host())
        .transactions()
        .abort()
        .unwrap();
    assert_eq!(before, after, "visibility probe must not change history");
    assert_eq!(
        history,
        history_entries(owner, edit),
        "visibility probe must not add a history entry"
    );
    assert_visible_frame(window, owner, edit, true);
}
fn history_entries(
    owner: &NativeDesktopOwner,
    edit: crate::HeadSketchEdit,
) -> Vec<oblikovati_api::wire::TransactionHistoryEntry> {
    owner
        .documents()
        .workspace()
        .session(edit.document)
        .unwrap()
        .history()
        .entries()
}
fn assert_visible_frame(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    edit: crate::HeadSketchEdit,
    expected: bool,
) {
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    let capture = capture_framebuffer(&mut window.native);
    assert_eq!(
        visible_in_view(window, owner, edit, &capture),
        expected,
        "sketch {:?}: expected rectangle visibility {expected}",
        edit.sketch
    );
}
fn begin_visibility_probe(
    owner: &mut NativeDesktopOwner,
    edit: crate::HeadSketchEdit,
) -> oblikovati_api::wire::UndoState {
    use oblikovati_api::{client::Client, wire::TransactionBeginArgs};
    let before = Client::new(owner.document_host())
        .transactions()
        .state()
        .unwrap();
    Client::new(owner.document_host())
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Visibility probe".into(),
        })
        .unwrap();
    set_sketch_hidden(owner, edit);
    before
}
fn set_sketch_hidden(owner: &mut NativeDesktopOwner, edit: crate::HeadSketchEdit) {
    use oblikovati_api::{client::Client, wire::SetSketchPropertyArgs};
    let index = sketch_index(owner, edit);
    Client::new(owner.document_host())
        .sketch()
        .set_property(&SetSketchPropertyArgs {
            sketch_index: index,
            property: "visible".into(),
            value: "false".into(),
        })
        .unwrap();
}
fn sketch_index(owner: &NativeDesktopOwner, edit: crate::HeadSketchEdit) -> i64 {
    let session = owner
        .documents()
        .workspace()
        .session(edit.document)
        .unwrap();
    crate::sketch_geometric::wire_sketch_index(&session.snapshot().sketches, edit.sketch).unwrap()
}
fn visible_in_view(
    window: &mut ScriptedGuiWindow,
    owner: &NativeDesktopOwner,
    edit: crate::HeadSketchEdit,
    capture: &ViewportCapture,
) -> bool {
    let corners = projected_rectangle(window, owner, edit);
    let area = document_area(window, capture.extent);
    application_bdd::rectangle_visibility::rectangle_visible_in_view(
        capture,
        corners,
        area,
        &application_bdd::window_occlusions(),
    )
}
fn document_area(window: &ScriptedGuiWindow, extent: [u32; 2]) -> [u32; 4] {
    window
        .native
        .gui
        .as_ref()
        .unwrap()
        .document_area(extent)
        .unwrap()
}
fn save_optional_capture(capture: &ViewportCapture) {
    let Some(path) = std::env::var_os("OBK_FIRST_PART_SCREENSHOT") else {
        return;
    };
    use crate::capture_png::WindowCaptureFiles;
    crate::capture_png::NativeWindowCaptureFiles
        .write_capture(std::path::Path::new(&path), capture)
        .unwrap();
}
fn projected_rectangle(
    window: &mut ScriptedGuiWindow,
    owner: &NativeDesktopOwner,
    edit: crate::HeadSketchEdit,
) -> [[i32; 2]; 2] {
    let sketch = sketch_geometry(owner, edit);
    assert_eq!(sketch.lines().len(), 4);
    let mut bounds = [[i32::MAX; 2], [i32::MIN; 2]];
    for id in sketch.lines().iter().flat_map(|line| line.endpoints) {
        let point = sketch.point(id).unwrap().position;
        let pixel = window.native.view.sketch_pixel(edit.plane, point).unwrap();
        include_pixel(&mut bounds, pixel);
    }
    bounds
}
fn sketch_geometry(
    owner: &NativeDesktopOwner,
    edit: crate::HeadSketchEdit,
) -> &oblikovati_model::sketch::Sketch2D {
    owner
        .documents()
        .workspace()
        .session(edit.document)
        .unwrap()
        .snapshot()
        .sketches
        .get(edit.sketch)
        .unwrap()
        .geometry()
}
fn include_pixel(bounds: &mut [[i32; 2]; 2], pixel: [i32; 2]) {
    for (axis, value) in pixel.into_iter().enumerate() {
        bounds[0][axis] = bounds[0][axis].min(value);
        bounds[1][axis] = bounds[1][axis].max(value);
    }
}
