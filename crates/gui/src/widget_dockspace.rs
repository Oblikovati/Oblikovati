// SPDX-License-Identifier: GPL-2.0-only
//! Scalar ABI for workspace dockspace geometry and hit testing.
use imgui::Ui;

unsafe extern "C" {
    fn oblikovati_gui_workspace_dockspace(width: f32, height: f32) -> u32;
    fn oblikovati_gui_outer_dockspace(
        width: f32,
        height: f32,
        seed_asset_dock: bool,
        anchor: *mut u32,
    ) -> u32;
    fn oblikovati_gui_prepare_dock_hit_test(x: f32, y: f32);
    fn oblikovati_gui_outer_bottom_span(id: u32, left: *mut f32, right: *mut f32) -> bool;
    fn oblikovati_gui_dockspace_center(
        id: u32,
        left: *mut f32,
        top: *mut f32,
        right: *mut f32,
        bottom: *mut f32,
    ) -> bool;
}

/// Route the next drag frame to the outer target when the pointer is over it.
/// Example: `prepare_dock_hit_test(position)` before starting the next ImGui frame.
pub(crate) fn prepare_dock_hit_test(position: [f32; 2]) {
    unsafe { oblikovati_gui_prepare_dock_hit_test(position[0], position[1]) };
}
/// Draw a dockspace in the current host and return its central bounds when available.
/// Example: `workspace_dockspace(ui, [800., 600.])` in a transparent host window.
pub(crate) fn workspace_dockspace(_: &Ui, size: [f32; 2]) -> Option<[f32; 4]> {
    let id = unsafe { oblikovati_gui_workspace_dockspace(size[0], size[1]) };
    dockspace_center(id)
}
/// Draw the outer workspace, including the default Model dock.
/// Example: `outer_dockspace(ui, [800., 600.], false, &mut anchor)` in its host.
pub(crate) fn outer_dockspace(
    _: &Ui,
    size: [f32; 2],
    seed_asset_dock: bool,
    anchor: &mut u32,
) -> Option<([f32; 4], Option<[f32; 2]>)> {
    // The native helper updates the anchor while the same GUI frame owns this reference.
    let id = unsafe { oblikovati_gui_outer_dockspace(size[0], size[1], seed_asset_dock, anchor) };
    let mut span = [0.; 2];
    let [left, right] = &mut span;
    let has_bottom = unsafe { oblikovati_gui_outer_bottom_span(id, left, right) };
    dockspace_center(id).map(|center| (center, has_bottom.then_some(span)))
}

fn dockspace_center(id: u32) -> Option<[f32; 4]> {
    let mut bounds = [0.; 4];
    let [left, top, right, bottom] = &mut bounds;
    let found = unsafe { oblikovati_gui_dockspace_center(id, left, top, right, bottom) };
    found.then_some(bounds)
}
