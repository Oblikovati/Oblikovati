// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Modify Slot scenarios (slot-01..slot-08).
use super::super::super::*;
use oblikovati_gui::GuiSlotPreview;
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey};

pub(super) const SLOT_WINDOW: &str = "Slot";

pub(super) fn click_slot_ribbon(world: &mut ApplicationWorld) {
    super::super::super::ribbon_panel_expansion::click_expansion_arrow(world, "Modify");
    world.pump(Vec::new());
    world.pump(Vec::new());
    let slot_run = world
        .run_in_window("##ribbon-additional-PartModify", "Slot")
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(slot_run));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(SLOT_WINDOW).is_some(),
        "expected Slot panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_slot_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(SLOT_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn reopen_slot1(world: &mut ApplicationWorld) {
    let row = world.run("Slot1").center();
    world.right_click(super::super::super::diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    world.pump(Vec::new());
    let edit = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| world.frame.is_popup_row(run) && run.text == "EditFeature")
        .unwrap_or_else(|| panic!("expected EditFeature popup item, saw {}", world.text()))
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(edit));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(SLOT_WINDOW).is_some(),
        "expected Slot panel to reopen for Slot1"
    );
}

pub(super) fn replace_slot_field(
    world: &mut ApplicationWorld,
    label: &str,
    value: &str,
    press_enter: bool,
) {
    let row = world.run_in_window(SLOT_WINDOW, label).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(SLOT_WINDOW).unwrap();
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
    if press_enter {
        world.key(ViewportPhysicalKey::Enter, false);
        world.pump(Vec::new());
        world.pump(Vec::new());
    } else {
        world.key(ViewportPhysicalKey::Tab, false);
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
}

pub(super) fn select_slot_combo(world: &mut ApplicationWorld, row_label: &str, choice: &str) {
    let row = world.run_in_window(SLOT_WINDOW, row_label).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(SLOT_WINDOW).unwrap();
    let combo = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.72,
        row.center()[1],
    ]);
    world.click(combo);
    world.pump(Vec::new());
    world.pump(Vec::new());
    let squeezed_choice = super::super::super::world::squeezed(choice);
    let item = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| world.frame.is_popup_row(run) && run.text == squeezed_choice)
        .unwrap_or_else(|| {
            panic!(
                "expected combo option {choice:?} ({squeezed_choice:?}), saw {}",
                world.text()
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(item));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn accept_slot_and_wait(world: &mut ApplicationWorld) {
    click_slot_control(world, "OK");
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(SLOT_WINDOW).is_none(),
        "expected Slot window to close after OK, preview={:?}",
        world.state().slot_preview
    );
}

pub(super) fn pick_xy_origin_plane(world: &mut ApplicationWorld) {
    if !world.frame.runs.iter().any(|run| run.text == "XYPlane") {
        let origin = world.run("Origin").clone();
        let arrow = super::super::super::diagnostics::pointer_pixels([
            origin.minimum[0] - 10.,
            origin.center()[1],
        ]);
        world.click(arrow);
        world.pump(Vec::new());
        if !world.frame.runs.iter().any(|run| run.text == "XYPlane") {
            world.click_text("Origin");
            world.pump(Vec::new());
        }
    }
    let xy = super::super::super::diagnostics::pointer_pixels(world.run("XYPlane").center());
    world.click(xy);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

fn scene_bounds(world: &mut ApplicationWorld) -> ([f32; 3], [f32; 3]) {
    let mut min = [f32::INFINITY; 3];
    let mut max = [f32::NEG_INFINITY; 3];
    for tri in &world.app().window.scene_triangles {
        for vertex in tri.positions {
            for axis in 0..3 {
                min[axis] = min[axis].min(vertex[axis]);
                max[axis] = max[axis].max(vertex[axis]);
            }
        }
    }
    (min, max)
}

pub(super) fn click_top_face_for_slot(world: &mut ApplicationWorld) {
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.35 },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Pan([400, 0]),
    )
    .unwrap();
    world.pump(Vec::new());
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let candidates = [
        [cx, cy, f64::from(max[2])],
        [cx + 0.15, cy, f64::from(max[2])],
        [cx - 0.15, cy, f64::from(max[2])],
        [cx, cy + 0.15, f64::from(max[2])],
        [cx, cy - 0.15, f64::from(max[2])],
        [cx, cy, f64::from(min[2])],
    ];
    let slot_rect = super::super::super::diagnostics::window_geometry(SLOT_WINDOW);
    let mut trace = Vec::new();
    for point in candidates {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
            trace.push(format!("{point:?}: no pixel"));
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        world.pump(Vec::new());
        let preview = world.state().slot_preview;
        if preview
            .as_ref()
            .is_some_and(|p| p.result == GuiSlotPreview::Ready)
        {
            return;
        }
        trace.push(format!("{point:?} -> {pixel:?} -> {preview:?}"));
    }
    panic!(
        "expected face click to produce a Ready Slot preview; bounds=({min:?},{max:?}) slot={slot_rect:?} trace={trace:#?}"
    );
}

pub(super) fn create_baseline_slot1(world: &mut ApplicationWorld) {
    click_slot_ribbon(world);
    click_slot_control(world, "Rounded");
    replace_slot_field(world, "Length", "4 mm", false);
    replace_slot_field(world, "Width", "2 mm", false);
    select_slot_combo(world, "Termination", "Distance");
    replace_slot_field(world, "Depth", "8 mm", false);
    click_top_face_for_slot(world);
    accept_slot_and_wait(world);
}
