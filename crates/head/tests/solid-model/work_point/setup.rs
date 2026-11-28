// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Work Point scenarios (work-point-01..work-point-08, #3918).
use super::super::super::*;
use oblikovati_viewport::ViewportNavigation;

pub(super) const GROUNDED_WINDOW: &str = "Grounded Point";

pub(super) fn open_work_point_fixture(world: &mut ApplicationWorld, relative: &str) {
    let source = super::super::super::scenario_root().join(relative);
    let copy = world
        .scratch_folder()
        .join(source.file_name().unwrap().to_str().unwrap());
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("work-point fixture {source:?}: {error}"));
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
        ViewportNavigation::Home { margin: 0.35 },
    )
    .unwrap();
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn open_point_dropdown(world: &mut ApplicationWorld) {
    let point_run = find_ribbon_point_run(world).unwrap_or_else(|| {
        world.click_text("Work Features");
        world.pump(Vec::new());
        find_ribbon_point_run(world).unwrap_or_else(|| {
            panic!(
                "expected Work Features Point button in ribbon, saw {}",
                world.text()
            )
        })
    });
    let trigger = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.text == "▼"
                && (run.center()[1] - point_run.center()[1]).abs() < 28.0
                && run.minimum[0] >= point_run.maximum[0] - 4.0
        })
        .min_by(|a, b| a.minimum[0].total_cmp(&b.minimum[0]))
        .map(|run| run.center())
        .unwrap_or([point_run.maximum[0] + 8.0, point_run.center()[1]]);
    world.click(super::super::super::diagnostics::pointer_pixels(trigger));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

fn find_ribbon_point_run(
    world: &ApplicationWorld,
) -> Option<super::super::super::diagnostics::RenderedRun> {
    world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Point" && run.center()[1] < 160.0)
        .cloned()
}

pub(super) fn choose_point_menu_item(world: &mut ApplicationWorld, label: &str) {
    open_point_dropdown(world);
    let squeezed = super::super::super::world::squeezed(label);
    let center = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| world.frame.is_popup_row(run) && run.text == squeezed)
        .unwrap_or_else(|| {
            panic!(
                "expected Point menu item {label:?} ({squeezed:?}), saw {}",
                world.text()
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn ensure_origin_expanded(world: &mut ApplicationWorld) {
    if !world.shows("XY Plane") {
        world.press_disclosure("Origin");
        world.pump(Vec::new());
    }
    world.expect_text("XY Plane");
}

pub(super) fn click_browser_row(world: &mut ApplicationWorld, label: &str) {
    let squeezed = super::super::super::world::squeezed(label);
    let center = world.run_in_window("Model", &squeezed).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_solid_vertex(world: &mut ApplicationWorld) -> [f64; 3] {
    let (_, max) = scene_bounds(world);
    let target = [f64::from(max[0]), f64::from(max[1]), f64::from(max[2])];
    let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, target).unwrap();
    world.click(pixel);
    world.pump(Vec::new());
    world.pump(Vec::new());
    target
}

pub(super) fn click_circular_end_edge(world: &mut ApplicationWorld) -> [f64; 3] {
    let (min, max) = scene_bounds(world);
    let cx = 0.5 * f64::from(min[0] + max[0]);
    let cy = 0.5 * f64::from(min[1] + max[1]);
    let radius = 0.5 * f64::from(max[0] - min[0]);
    let edge_point = [cx + radius, cy, f64::from(max[2])];
    let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, edge_point).unwrap();
    world.click(pixel);
    world.pump(Vec::new());
    world.pump(Vec::new());
    [cx, cy, f64::from(max[2])]
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
    assert!(
        min.iter().all(|v| v.is_finite()) && max.iter().all(|v| v.is_finite()),
        "expected visible solid scene bounds"
    );
    (min, max)
}
