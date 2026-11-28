// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Work Features > Work Axis (#3918).
use super::super::super::*;
use oblikovati_viewport::ViewportNavigation;

pub(super) fn open_work_axis_fixture(world: &mut ApplicationWorld, fixture: &str) {
    let source = super::super::super::scenario_root().join(fixture);
    let copy = world
        .scratch_folder()
        .join(source.file_name().unwrap().to_str().unwrap());
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("work-axis fixture {source:?}: {error}"));
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

pub(super) fn open_axis_dropdown_and_choose(world: &mut ApplicationWorld, choice: &str) {
    let axis_run = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Axis" && run.center()[1] < 160.0)
        .unwrap_or_else(|| panic!("expected Axis ribbon button, saw {}", world.text()))
        .clone();
    let trigger = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.text.contains('▼')
                && run.center()[1] < 160.0
                && run.center()[0] > axis_run.center()[0]
                && (run.center()[1] - axis_run.center()[1]).abs() < 14.0
        })
        .min_by(|a, b| a.center()[0].total_cmp(&b.center()[0]))
        .unwrap_or_else(|| {
            panic!(
                "expected Axis dropdown trigger beside Axis button, saw {}",
                world.text()
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(trigger));
    world.pump(Vec::new());
    world.pump(Vec::new());
    let squeezed = super::super::super::world::squeezed(choice);
    let item = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| run.text == squeezed && run.center()[1] > axis_run.maximum[1])
        .unwrap_or_else(|| {
            panic!(
                "expected Axis dropdown option {choice:?} ({squeezed:?}), saw {}",
                world.text()
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(item));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn ensure_origin_expanded(world: &mut ApplicationWorld) {
    if world
        .frame
        .runs
        .iter()
        .any(|run| run.text == "XYPlane" || run.text == "ZAxis")
    {
        return;
    }
    world.click_text("Origin");
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_browser_entry(world: &mut ApplicationWorld, label: &str) {
    let squeezed = super::super::super::world::squeezed(label);
    let run = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == squeezed)
        .unwrap_or_else(|| {
            panic!(
                "expected browser row {label:?} ({squeezed:?}), saw {}",
                world.text()
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(run));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn scene_bounds(world: &mut ApplicationWorld) -> ([f64; 3], [f64; 3]) {
    let mut min = [f64::INFINITY; 3];
    let mut max = [f64::NEG_INFINITY; 3];
    for tri in &world.app().window.scene_triangles {
        for vertex in tri.positions {
            for axis in 0..3 {
                min[axis] = min[axis].min(f64::from(vertex[axis]));
                max[axis] = max[axis].max(f64::from(vertex[axis]));
            }
        }
    }
    assert!(
        min.iter().chain(&max).all(|v| v.is_finite()),
        "expected finite scene bounds"
    );
    (min, max)
}

pub(super) fn click_model_point(world: &mut ApplicationWorld, point: [f64; 3]) {
    let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, point).unwrap();
    world.click(pixel);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_inner_cylindrical_wall(world: &mut ApplicationWorld, radius_cm: f64) {
    let (min, max) = scene_bounds(world);
    let cx = (min[0] + max[0]) * 0.5;
    let cy = (min[1] + max[1]) * 0.5;
    let height = (max[2] - min[2]).max(0.05);
    let wall_point = [cx, cy + radius_cm, min[2] + height * 0.5];
    let delta_y = (radius_cm * 0.4).min(height * 0.6);
    let delta_z = height * 0.5;
    let dir_len = delta_y.hypot(delta_z);
    let span = (max[0] - min[0]).max(height).max(1.0);
    let dist = span * 3.0;
    let mut camera = crate::HeadWindow::view_camera(&world.app().window.native).unwrap();
    camera.target = [cx, cy, min[2] + height * 0.5];
    camera.eye = [
        wall_point[0],
        wall_point[1] - (delta_y / dir_len) * dist,
        wall_point[2] + (delta_z / dir_len) * dist,
    ];
    crate::HeadWindow::set_view_camera(&mut world.app().window.native, camera).unwrap();
    world.pump(Vec::new());
    click_model_point(world, wall_point);
}

pub(super) fn work_axis_count(world: &mut ApplicationWorld) -> usize {
    world
        .state()
        .browser
        .as_ref()
        .map(|browser| {
            browser
                .features
                .iter()
                .filter(|feature| feature.kind == "Work Axis")
                .count()
        })
        .unwrap_or(0)
}
