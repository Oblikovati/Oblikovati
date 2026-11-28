// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Surface Stitch scenarios (stitch-01..05).
use super::super::super::*;
use oblikovati_gui::GuiStitchPreview;
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey};

pub(super) const STITCH_WINDOW: &str = "Stitch";
const THICKEN_WINDOW: &str = "Thicken/Offset";

pub(super) fn open_stitch_ribbon(world: &mut ApplicationWorld) {
    // Top ribbon Surface panel exposes Stitch directly (or via collapsed Surface popup).
    if world.shows("Stitch") {
        let center = world.run("Stitch").center();
        world.click(super::super::super::diagnostics::pointer_pixels(center));
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
    if super::super::super::diagnostics::window_geometry(STITCH_WINDOW).is_none() {
        let surface = world.run("Surface").center();
        world.click(super::super::super::diagnostics::pointer_pixels(surface));
        world.pump(Vec::new());
        world.pump(Vec::new());
        let stitch = world.run("Stitch").center();
        world.click(super::super::super::diagnostics::pointer_pixels(stitch));
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
    assert!(
        super::super::super::diagnostics::window_geometry(STITCH_WINDOW).is_some(),
        "expected Stitch dialog to open, saw {}",
        world.text()
    );
}

pub(super) fn click_stitch_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(STITCH_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn replace_stitch_field(world: &mut ApplicationWorld, label: &str, value: &str) {
    let row = world.run_in_window(STITCH_WINDOW, label).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(STITCH_WINDOW).unwrap();
    let field = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.72,
        row.center()[1],
    ]);
    world.click(field);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::A, true);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::Backspace, false);
    world.pump(Vec::new());
    world.type_text(value);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::Tab, false);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_model_point(world: &mut ApplicationWorld, point: [f64; 3]) -> bool {
    let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
        return false;
    };
    world.click(pixel);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
    true
}

pub(super) fn pan_clear_of_panel(world: &mut ApplicationWorld) {
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.35 },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Pan([360, 0]),
    )
    .unwrap();
    world.pump(Vec::new());
}

pub(super) fn create_zero_offset_surface_on_point(world: &mut ApplicationWorld, point: [f64; 3]) {
    super::super::super::ribbon_panel_expansion::click_expansion_arrow(world, "Modify");
    world.pump(Vec::new());
    world.pump(Vec::new());
    let run = world
        .run_in_window("##ribbon-additional-PartModify", "Thicken/Offset")
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(run));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(super::super::super::diagnostics::window_geometry(THICKEN_WINDOW).is_some());

    let surface_btn = world.run_in_window(THICKEN_WINDOW, "Surface").center();
    world.click(super::super::super::diagnostics::pointer_pixels(
        surface_btn,
    ));
    world.pump(Vec::new());

    // Set Distance = 0 mm
    let row = world.run_in_window(THICKEN_WINDOW, "Distance").clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(THICKEN_WINDOW).unwrap();
    let field = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.68,
        row.center()[1],
    ]);
    world.click(field);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::A, true);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::Backspace, false);
    world.pump(Vec::new());
    world.type_text("0 mm");
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::Tab, false);
    world.pump(Vec::new());

    pan_clear_of_panel(world);
    assert!(click_model_point(world, point));
    let ok = world.run_in_window(THICKEN_WINDOW, "OK").center();
    world.click(super::super::super::diagnostics::pointer_pixels(ok));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(THICKEN_WINDOW).is_none(),
        "expected Thicken/Offset to close after creating zero-offset surface"
    );
}

pub(super) fn hide_solid_body_in_browser(world: &mut ApplicationWorld, name: &str) {
    if !world.shows(name) {
        let folder = world.run("Solid Bodies (1)").center();
        world.click(super::super::super::diagnostics::pointer_pixels(folder));
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
    let row = world.run(name).center();
    world.right_click(super::super::super::diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    world.pump(Vec::new());
    let vis = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| world.frame.is_popup_row(run) && run.text == "Visibility")
        .unwrap_or_else(|| {
            panic!(
                "expected Visibility popup item for {name}, saw {}",
                world.text()
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(vis));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_browser_surface(world: &mut ApplicationWorld, name: &str) {
    let row = world.run(name).center();
    world.click(super::super::super::diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn double_click_stitch_feature(world: &mut ApplicationWorld, name: &str) {
    let center = world.run(name).center();
    world.double_click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
    if super::super::super::diagnostics::window_geometry(STITCH_WINDOW).is_none() {
        let row = world.run(name).center();
        world.right_click(super::super::super::diagnostics::pointer_pixels(row));
        world.pump(Vec::new());
        world.pump(Vec::new());
        let edit = world
            .frame
            .runs
            .iter()
            .rev()
            .find(|run| world.frame.is_popup_row(run) && run.text == "EditFeature")
            .unwrap_or_else(|| {
                panic!(
                    "expected EditFeature popup item for {name}, saw {}",
                    world.text()
                )
            })
            .center();
        world.click(super::super::super::diagnostics::pointer_pixels(edit));
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
    assert!(
        super::super::super::diagnostics::window_geometry(STITCH_WINDOW).is_some(),
        "expected Stitch dialog after reopening {name}, saw {}",
        world.text()
    );
}

pub(super) fn expect_ready_preview(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .stitch_preview
        .expect("expected stitch preview");
    assert!(
        matches!(preview.result, GuiStitchPreview::Ready { .. }),
        "expected Ready stitch preview, got {preview:?}"
    );
}
