// SPDX-License-Identifier: GPL-2.0-only
//! Shared desktop BDD helpers for Part Create Unwrap scenarios (unwrap-01..unwrap-06).
use super::super::super::*;
use oblikovati_gui::GuiUnwrapPreview;
use oblikovati_viewport::ViewportNavigation;

pub(super) const UNWRAP_WINDOW: &str = "Unwrap";

pub(super) fn open_disposable_unwrap_fixture(world: &mut ApplicationWorld, relative: &str) {
    let source = super::super::super::scenario_root().join(relative);
    let copy = world
        .scratch_folder()
        .join(source.file_name().unwrap().to_str().unwrap());
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("unwrap fixture {source:?}: {error}"));
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

pub(super) fn click_unwrap_ribbon(world: &mut ApplicationWorld) {
    world.click_text("Unwrap");
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_unwrap_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(UNWRAP_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_unwrap_row_button(world: &mut ApplicationWorld, row_label: &str) {
    let row = world.run_in_window(UNWRAP_WINDOW, row_label).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(UNWRAP_WINDOW).unwrap();
    let button = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.72,
        row.center()[1],
    ]);
    world.click(button);
    world.pump(Vec::new());
    world.pump(Vec::new());
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

pub(super) fn select_lateral_face(world: &mut ApplicationWorld) {
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let cz = f64::from((min[2] + max[2]) * 0.5);
    let rx = f64::from((max[0] - min[0]) * 0.5);
    let ry = f64::from((max[1] - min[1]) * 0.5);
    let candidates = [
        [cx + rx * 0.85, cy, cz],
        [cx, cy - ry * 0.85, cz],
        [cx + rx * 0.65, cy - ry * 0.65, cz],
        [cx - rx * 0.85, cy, cz],
        [cx, cy + ry * 0.85, cz],
        [cx, cy, cz],
    ];
    for point in candidates {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .unwrap_preview
            .as_ref()
            .is_some_and(|preview| preview.result == GuiUnwrapPreview::Ready)
        {
            return;
        }
    }
    panic!(
        "expected lateral face click to produce a Ready Unwrap preview, saw {:?}",
        world.state().unwrap_preview
    );
}
