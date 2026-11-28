// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Surface Ruled Surface scenarios (ruled-surface-01..10).
use super::super::super::*;
use oblikovati_gui::GuiRuledSurfacePreview;
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey};
use std::cell::RefCell;

pub(super) const RULED_WINDOW: &str = "Ruled Surface";

thread_local! {
    static PREV_SURFACE_COUNT: RefCell<usize> = const { RefCell::new(0) };
    static PREV_ARROW_DIR: RefCell<Option<[f64; 3]>> = const { RefCell::new(None) };
}

pub(super) fn record_surface_count(world: &mut ApplicationWorld) {
    let count = world.state().browser.map_or(0, |b| b.surface_bodies.len());
    PREV_SURFACE_COUNT.with(|cell| *cell.borrow_mut() = count);
}

pub(super) fn previous_surface_count() -> usize {
    PREV_SURFACE_COUNT.with(|cell| *cell.borrow())
}

pub(super) fn record_arrow_direction(dir: [f64; 3]) {
    PREV_ARROW_DIR.with(|cell| *cell.borrow_mut() = Some(dir));
}

pub(super) fn previous_arrow_direction() -> Option<[f64; 3]> {
    PREV_ARROW_DIR.with(|cell| *cell.borrow())
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

pub(super) fn open_ruled_surface_ribbon(world: &mut ApplicationWorld) {
    record_surface_count(world);
    if !world.shows("Ruled Surface") {
        world.click_text("Surface");
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
    world.click_text("Ruled Surface");
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(RULED_WINDOW).is_some(),
        "expected Ruled Surface panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_ruled_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(RULED_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn replace_field_in(
    world: &mut ApplicationWorld,
    window: &str,
    label: &str,
    value: &str,
) {
    let row = world.run_in_window(window, label).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(window).unwrap();
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
    world.type_text(value);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::Tab, false);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn set_automatic_edge_chain(world: &mut ApplicationWorld, wanted: bool) {
    let current = world
        .state()
        .ruled_surface_preview
        .as_ref()
        .is_none_or(|p| p.request.automatic_edge_chain);
    if current != wanted {
        click_ruled_control(world, "AutomaticEdgeChain");
    }
}

pub(super) fn pick_offset_surface_edge(world: &mut ApplicationWorld, expected_count: usize) {
    pan_clear_of_panel(world);
    let edge_midpoints = [
        [-2.0, 0.0, 2.1],
        [0.0, -1.25, 2.1],
        [-2.0, -2.5, 2.1],
        [-4.0, -1.25, 2.1],
    ];
    for point in edge_midpoints {
        if !click_model_point(world, point) {
            continue;
        }
        if world
            .state()
            .ruled_surface_preview
            .as_ref()
            .is_some_and(|p| p.request.edges.len() == expected_count)
        {
            return;
        }
    }
    panic!(
        "expected clicking offset-surface edge to select {expected_count} edge(s), saw {:?}",
        world.state().ruled_surface_preview
    );
}

pub(super) fn choose_origin_z_axis(world: &mut ApplicationWorld) {
    if !world
        .frame
        .runs
        .iter()
        .any(|run| run.text == "XYPlane" || run.text == "ZAxis")
    {
        world.click_text("Origin");
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
    let z_row = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "ZAxis")
        .unwrap_or_else(|| panic!("expected Z Axis browser row, saw {}", world.text()))
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(z_row));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn double_click_ruled_feature(world: &mut ApplicationWorld, name: &str) {
    let center = world.run(name).center();
    world.double_click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
    if super::super::super::diagnostics::window_geometry(RULED_WINDOW).is_none() {
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
        super::super::super::diagnostics::window_geometry(RULED_WINDOW).is_some(),
        "expected Ruled Surface panel after reopening {name}, saw {}",
        world.text()
    );
}

pub(super) fn expect_ready_preview(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .ruled_surface_preview
        .expect("expected ruled surface preview");
    assert_eq!(
        preview.result,
        GuiRuledSurfacePreview::Ready,
        "expected Ready preview, got {preview:?}"
    );
}
