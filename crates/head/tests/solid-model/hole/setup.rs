// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Modify Hole scenarios (hole-01..hole-11 + contract).
use super::super::super::*;
use oblikovati_gui::GuiHolePreview;
use oblikovati_viewport::{ViewportNavigation, ViewportPhysicalKey};
use std::path::Path;

pub(super) const HOLE_WINDOW: &str = "Hole";

pub(super) fn open_hole_fixture(world: &mut ApplicationWorld, relative: &str) {
    let source = super::super::super::scenario_root().join(relative);
    let copy = world
        .scratch_folder()
        .join(source.file_name().unwrap().to_str().unwrap());
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("hole fixture {source:?}: {error}"));
    let document = world
        .app()
        .owner
        .document_host()
        .open_path(&copy, true)
        .unwrap();
    world.pump(Vec::new());
    world.activate_through_tab(document);
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.25 },
    )
    .unwrap();
    world.pump(Vec::new());
}

pub(super) fn open_visible_solid_part(world: &mut ApplicationWorld) {
    let source = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../test-utilities/fixtures/docking-visible-solid.opd");
    let copy = world.scratch_folder().join("hole-contract-part.opd");
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("hole contract fixture {source:?}: {error}"));
    let document = world
        .app()
        .owner
        .document_host()
        .open_path(&copy, true)
        .unwrap();
    world.pump(Vec::new());
    world.activate_through_tab(document);
    world.pump(Vec::new());
}

pub(super) fn click_hole_ribbon(world: &mut ApplicationWorld) {
    world.click_text("Hole");
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(HOLE_WINDOW).is_some(),
        "expected Hole panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_hole_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(HOLE_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn reopen_hole1(world: &mut ApplicationWorld) {
    let row = world.run("Hole1").center();
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
        super::super::super::diagnostics::window_geometry(HOLE_WINDOW).is_some(),
        "expected Hole panel to reopen for Hole1"
    );
}

pub(super) fn replace_hole_field(
    world: &mut ApplicationWorld,
    label: &str,
    value: &str,
    press_enter: bool,
) {
    let row = world.run_in_window(HOLE_WINDOW, label).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(HOLE_WINDOW).unwrap();
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

pub(super) fn select_hole_combo(world: &mut ApplicationWorld, row_label: &str, choice: &str) {
    let row = world.run_in_window(HOLE_WINDOW, row_label).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(HOLE_WINDOW).unwrap();
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

pub(super) fn open_hole_preset_settings(world: &mut ApplicationWorld) {
    let (origin, size) = super::super::super::diagnostics::window_geometry(HOLE_WINDOW).unwrap();
    let no_preset = world.run_in_window(HOLE_WINDOW, "NoPreset").clone();
    let gear = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] - 18.0,
        no_preset.center()[1],
    ]);
    world.click(gear);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_popup_item(world: &mut ApplicationWorld, label: &str) {
    let squeezed = super::super::super::world::squeezed(label);
    let run = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| world.frame.is_popup_row(run) && run.text == squeezed)
        .unwrap_or_else(|| panic!("expected popup item {label:?}, saw {}", world.text()))
        .clone();
    world.click(super::super::super::diagnostics::pointer_pixels(
        run.center(),
    ));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn popup_item<'a>(
    world: &'a ApplicationWorld,
    label: &str,
) -> &'a super::super::super::diagnostics::RenderedRun {
    let squeezed = super::super::super::world::squeezed(label);
    world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| world.frame.is_popup_row(run) && run.text == squeezed)
        .unwrap_or_else(|| panic!("expected popup item {label:?}, saw {}", world.text()))
}

pub(super) fn accept_hole_and_wait(world: &mut ApplicationWorld) {
    click_hole_control(world, "OK");
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(HOLE_WINDOW).is_none(),
        "expected Hole window to close after OK, preview={:?}",
        world.state().hole_preview
    );
}

pub(super) fn click_end_face_for_hole(world: &mut ApplicationWorld) {
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
    let hole_rect = super::super::super::diagnostics::window_geometry(HOLE_WINDOW);
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
        let preview = world.state().hole_preview;
        if preview
            .as_ref()
            .is_some_and(|p| p.result == GuiHolePreview::Ready)
        {
            return;
        }
        trace.push(format!("{point:?} -> {pixel:?} -> {preview:?}"));
    }
    panic!(
        "expected end face click to produce a Ready Hole preview; bounds=({min:?},{max:?}) hole={hole_rect:?} trace={trace:#?}"
    );
}

pub(super) fn scene_bounds(world: &mut ApplicationWorld) -> ([f32; 3], [f32; 3]) {
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

pub(super) fn red_preview_z_bounds(world: &mut ApplicationWorld) -> Option<(f32, f32)> {
    let mut lo = f32::INFINITY;
    let mut hi = f32::NEG_INFINITY;
    let mut found = false;
    for tri in &world.app().window.scene_triangles {
        if tri.color[0] > 0.7 && tri.color[1] < 0.35 && tri.color[2] < 0.35 {
            found = true;
            for p in tri.positions {
                lo = lo.min(p[2]);
                hi = hi.max(p[2]);
            }
        }
    }
    found.then_some((lo, hi))
}
