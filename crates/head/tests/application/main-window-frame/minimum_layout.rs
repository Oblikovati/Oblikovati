// SPDX-License-Identifier: GPL-2.0-only
//! Keep the active tab, ribbon, and a floating tool usable at the native frame minimum.
use super::super::{desktop_input, diagnostics};
use super::*;

const RIBBON: &str = "##application-ribbon";
const SHELL: &str = "Shell";

pub(super) fn remember_docks(world: &mut ApplicationWorld) {
    world.observed_counts = dock_widths(world).into();
}

fn dock_widths(world: &mut ApplicationWorld) -> [usize; 2] {
    let [width, _] = world.extent();
    let [left, _, right, _] = world.viewport_bounds();
    [left as usize, (width - right) as usize]
}

#[then(
    "docked side panels scale proportionally without hiding the active document tab or overflowing the ribbon"
)]
fn minimum_docks_and_ribbon(world: &mut ApplicationWorld) {
    let after = dock_widths(world);
    let before = &world.observed_counts;
    assert!(
        after
            .iter()
            .zip(before)
            .all(|(narrow, wide)| (1..*wide).contains(narrow)),
        "side docks {before:?} -> {after:?}"
    );
    assert!(after[0].abs_diff(after[1]) <= 4, "side docks {after:?}");
    let width = world.extent()[0] as usize;
    assert!((after[0] + after[1]) * 100 <= 45 * width + 200);
    assert_active_tab_visible(world);
    assert_ribbon_fits();
    world.save_capture("minimum-window-docks-and-ribbon");
    let row = diagnostics::child_scroll(RIBBON, "##ribbon-panels").unwrap();
    let print = world
        .frame
        .runs
        .iter()
        .find(|run| run.list == row.list && run.text.starts_with("3DPri"))
        .expect("the compact 3D Print group must remain visible");
    desktop_input::click(world, diagnostics::pointer_pixels(print.center()));
    world.expect_text("Export STL");
    world.escape();
    world.pump(Vec::new());
}

fn assert_active_tab_visible(world: &mut ApplicationWorld) {
    assert_eq!(world.state().document, world.previous_document);
    let tab = world.expect_tab("main");
    let bounds = diagnostics::window_pixel_bounds("##open-document-tabs").unwrap();
    let x = diagnostics::pointer_pixels(tab.center())[0] as usize;
    let pixel = world.pixel_at([x, bounds[3] as usize - 2]);
    let accent =
        unsafe { imgui::sys::igGetColorU32_Col(imgui::sys::ImGuiCol_NavHighlight as i32, 1.) }
            .to_le_bytes();
    assert_eq!(
        pixel[..3],
        accent[..3],
        "active tab underline for {:?}",
        tab.text
    );
}

fn assert_ribbon_fits() {
    let tabs = diagnostics::child_scroll(RIBBON, "##ribbon-tab-region").expect("ribbon tabs");
    let panels = diagnostics::child_scroll(RIBBON, "##ribbon-panels").expect("ribbon panels");
    assert!(!tabs.horizontal_scrollbar && !panels.horizontal_scrollbar);
    assert!(
        panels.content_width <= panels.size[0] + 2.,
        "panel content {}/{}",
        panels.content_width,
        panels.size[0]
    );
}

#[then("any open floating feature panel remains clamped inside the visible viewport bounds")]
fn floating_panel_is_clamped(world: &mut ApplicationWorld) {
    let [_, _, width, height] = world.window_frame_before.unwrap();
    super::resize_from_bottom_right(world, [width - 2, height - 2]);
    click_native_text(world, SHELL);
    assert!(diagnostics::window_geometry(SHELL).is_some());
    super::resize_from_bottom_right(world, [100, 100]);
    super::minimum_window_bounds(world);
    let panel = diagnostics::window_pixel_bounds(SHELL).expect("visible floating Shell panel");
    let view = world.viewport_bounds().map(|edge| edge as f32);
    assert!(
        panel[0] >= view[0] && panel[1] >= view[1] && panel[2] <= view[2] && panel[3] <= view[3],
        "floating Shell {panel:?} outside viewport {view:?}"
    );
    world.save_capture("floating-shell-at-minimum");
    let cancel = world.run_in_window(SHELL, "Cancel").center();
    desktop_input::click(world, diagnostics::pointer_pixels(cancel));
    assert!(diagnostics::window_geometry(SHELL).is_none());
    assert_eq!(world.state().document, world.previous_document);
}

fn click_native_text(world: &mut ApplicationWorld, label: &str) {
    let point = diagnostics::pointer_pixels(world.run(label).center());
    desktop_input::click(world, point);
}
