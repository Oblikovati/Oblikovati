// SPDX-License-Identifier: GPL-2.0-only
//! Shared Wayland desktop BDD helpers for Part Modify Split scenarios (split-01..split-07, #3918).
use super::super::super::*;
use oblikovati_gui::GuiSplitPreview;
use oblikovati_model::feature::split::{SplitKeepSide, SplitMode};
use oblikovati_viewport::ViewportNavigation;

pub(super) const SPLIT_WINDOW: &str = "Split";

pub(super) fn open_disposable_split_fixture(world: &mut ApplicationWorld, relative: &str) {
    let source = super::super::super::scenario_root().join(relative);
    let copy = world
        .scratch_folder()
        .join(source.file_name().unwrap().to_str().unwrap());
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("split fixture {source:?}: {error}"));
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

pub(super) fn open_split_ribbon(world: &mut ApplicationWorld) {
    super::super::super::ribbon_panel_expansion::click_expansion_arrow(world, "Modify");
    world.pump(Vec::new());
    world.pump(Vec::new());
    let run = world
        .run_in_window("##ribbon-additional-PartModify", "Split")
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(run));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(SPLIT_WINDOW).is_some(),
        "expected Split panel to open, saw {}",
        world.text()
    );
}

pub(super) fn click_split_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(SPLIT_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_clear_on_split_row(world: &mut ApplicationWorld, row_label: &str) {
    let row_y = world.run_in_window(SPLIT_WINDOW, row_label).center()[1];
    let (origin, size) = super::super::super::diagnostics::window_geometry(SPLIT_WINDOW).unwrap();
    let clear = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.text == "Clear"
                && run.center()[0] >= origin[0]
                && run.center()[0] <= origin[0] + size[0]
                && (run.center()[1] - row_y).abs() <= 14.0
        })
        .min_by(|a, b| {
            (a.center()[1] - row_y)
                .abs()
                .total_cmp(&(b.center()[1] - row_y).abs())
        })
        .unwrap_or_else(|| {
            panic!(
                "expected Clear button on {row_label} row in Split panel, saw {}",
                world.window_text(SPLIT_WINDOW)
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(clear));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn set_all_faces(world: &mut ApplicationWorld, wanted: bool) {
    let current = world
        .state()
        .split_preview
        .as_ref()
        .is_some_and(|preview| preview.request.all_faces);
    if current != wanted {
        click_split_control(world, "AllFaces");
    }
    let after = world
        .state()
        .split_preview
        .as_ref()
        .is_some_and(|preview| preview.request.all_faces);
    assert_eq!(after, wanted, "expected All Faces == {wanted}");
}

pub(super) fn switch_split_to_solid_mode(world: &mut ApplicationWorld) {
    let is_solid = world
        .state()
        .split_preview
        .as_ref()
        .is_some_and(|preview| preview.request.mode == SplitMode::Solid);
    if !is_solid {
        click_split_control(world, "Solid");
    }
    let mode = world
        .state()
        .split_preview
        .as_ref()
        .map(|preview| preview.request.mode);
    assert_eq!(mode, Some(SplitMode::Solid), "expected Split in Solid mode");
}

pub(super) fn pan_clear_of_panel(world: &mut ApplicationWorld) {
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.35 },
    )
    .unwrap();
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Pan([380, 0]),
    )
    .unwrap();
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

pub(super) fn click_visible_sketch_line_tool(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let z_top = f64::from(max[2]);
    let candidates = [
        [cx, cy, z_top],
        [cx, f64::from(min[1]) - 0.15, z_top],
        [cx, f64::from(max[1]) + 0.15, z_top],
        [f64::from(min[0]) + 0.2, cy, z_top],
        [cx, f64::from(min[1]) + 0.2, z_top],
    ];
    for point in candidates {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .split_preview
            .as_ref()
            .is_some_and(|preview| preview.request.tool.is_some())
        {
            return;
        }
    }
    panic!(
        "expected viewport click on visible sketch line to populate Split tool; preview={:?}",
        world.state().split_preview
    );
}

pub(super) fn click_top_planar_face(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let z_top = f64::from(max[2]);
    let candidates = [
        [f64::from(min[0]) + 0.18, cy, z_top],
        [cx, f64::from(min[1]) + 0.18, z_top],
        [f64::from(max[0]) - 0.18, cy, z_top],
        [cx, f64::from(max[1]) - 0.18, z_top],
        [cx, cy, z_top],
    ];
    for point in candidates {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .split_preview
            .as_ref()
            .is_some_and(|preview| !preview.request.faces.is_empty())
        {
            return;
        }
    }
    panic!(
        "expected viewport click on top planar face to populate Split faces; preview={:?}",
        world.state().split_preview
    );
}

pub(super) fn click_rectangular_solid_in_viewport(world: &mut ApplicationWorld) {
    pan_clear_of_panel(world);
    let (min, max) = scene_bounds(world);
    let cx = f64::from((min[0] + max[0]) * 0.5);
    let cy = f64::from((min[1] + max[1]) * 0.5);
    let z_top = f64::from(max[2]);
    let candidates = [[cx, cy, z_top], [cx + 0.2, cy + 0.2, z_top]];
    for point in candidates {
        let Ok(pixel) = crate::HeadWindow::model_pixel(&world.app().window.native, point) else {
            continue;
        };
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .split_preview
            .as_ref()
            .is_some_and(|preview| preview.request.body.is_some())
        {
            return;
        }
    }
    panic!(
        "expected viewport click on rectangular solid to populate Split solid; preview={:?}",
        world.state().split_preview
    );
}

pub(super) fn pick_browser_xy_origin_plane(world: &mut ApplicationWorld) {
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

pub(super) fn pick_browser_solid_body(world: &mut ApplicationWorld, name: &str) {
    if !world.frame.runs.iter().any(|run| run.text == name) {
        let folder = world
            .frame
            .runs
            .iter()
            .find(|run| run.text.starts_with("SolidBodies"))
            .unwrap_or_else(|| {
                panic!(
                    "expected Solid Bodies folder in browser, saw {}",
                    world.text()
                )
            })
            .clone();
        let arrow = super::super::super::diagnostics::pointer_pixels([
            folder.minimum[0] - 10.,
            folder.center()[1],
        ]);
        world.click(arrow);
        world.pump(Vec::new());
        if !world.frame.runs.iter().any(|run| run.text == name) {
            let center = super::super::super::diagnostics::pointer_pixels(folder.center());
            world.click(center);
            world.pump(Vec::new());
        }
    }
    let row = super::super::super::diagnostics::pointer_pixels(world.run(name).center());
    world.click(row);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn pick_browser_work_plane1(world: &mut ApplicationWorld) {
    let row = super::super::super::diagnostics::pointer_pixels(world.run("WorkPlane1").center());
    world.click(row);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn accept_split_and_wait(world: &mut ApplicationWorld) {
    click_split_control(world, "OK");
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(SPLIT_WINDOW).is_none(),
        "expected Split panel to close after OK, preview={:?}",
        world.state().split_preview
    );
}

pub(super) fn double_click_split1(world: &mut ApplicationWorld) {
    let center = world.run("Split1").center();
    world.double_click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry(SPLIT_WINDOW).is_some(),
        "expected Split panel after double-clicking Split1, saw {}",
        world.text()
    );
}

pub(super) fn create_split1_all_faces(world: &mut ApplicationWorld) {
    open_split_ribbon(world);
    set_all_faces(world, true);
    click_visible_sketch_line_tool(world);
    assert_eq!(
        world.state().split_preview.unwrap().result,
        GuiSplitPreview::Ready
    );
    accept_split_and_wait(world);
}

pub(super) fn create_split1_midplane_solid(world: &mut ApplicationWorld) {
    open_split_ribbon(world);
    switch_split_to_solid_mode(world);
    pick_browser_work_plane1(world);
    click_rectangular_solid_in_viewport(world);
    click_split_control(world, "KeepBoth");
    let preview = world.state().split_preview.unwrap();
    assert_eq!(preview.request.keep_side, SplitKeepSide::Both);
    assert_eq!(preview.result, GuiSplitPreview::Ready);
    accept_split_and_wait(world);
}

pub(super) fn total_evaluated_edge_count(world: &mut ApplicationWorld) -> usize {
    let document = world.active_id();
    let app = world.app();
    let workspace = app.owner.documents().workspace();
    let session = workspace.session(document).expect("expected session");
    let guid = workspace
        .identity(document)
        .expect("expected identity")
        .internal_name;
    let eval = app
        .owner
        .part_scenes_current(document)
        .expect("expected current part evaluation");
    eval.ordered_bodies()
        .map(|body| {
            body.edge_reference_keys(session.history(), guid)
                .map_or(0, |keys| keys.len())
        })
        .sum()
}
