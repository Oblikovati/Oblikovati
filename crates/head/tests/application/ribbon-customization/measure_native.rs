// SPDX-License-Identifier: GPL-2.0-only
//! Verify Measure in both part and sketch contexts.
use super::*;

#[test]
fn tools_measure_respects_context_and_opens_sketch_inspection() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("Tools");
    assert_eq!(measure_command(&world).alpha, u8::MAX);
    click_measure_command(&mut world);
    world.expect_text("Select a solid face or edge to measure");
    world.click_text("Done");
    world.click_text("3D Model");
    world.start_sketch();
    world.click_text("Tools");
    assert_eq!(measure_command(&world).alpha, u8::MAX);
    click_measure_command(&mut world);
    world.expect_text("Select geometry to measure");
    assert!(diagnostics::window_geometry("Measure").is_some());
    world.click_text("Done");
    assert!(diagnostics::window_geometry("Measure").is_none());
    world.close();
}

fn click_measure_command(world: &mut ApplicationWorld) {
    let center = measure_command(world).center();
    world.click(diagnostics::pointer_pixels(center));
}

fn measure_command(world: &ApplicationWorld) -> &diagnostics::RenderedRun {
    world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Measure")
        .min_by(|left, right| left.center()[1].total_cmp(&right.center()[1]))
        .expect("Measure: expected the ribbon command")
}

#[test]
fn moved_measure_runs_in_the_expanded_panel_and_returns_to_main() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.start_sketch();
    world.click_text("Tools");
    let position = diagnostics::pointer_pixels(measure_command(&world).center());
    world.right_click(position);
    world.click_text("Move to Expanded Panel");
    world.click_text("Measure");
    let title = "##ribbon-additional-ToolMeasure";
    assert!(diagnostics::window_geometry(title).is_some());
    let position = diagnostics::pointer_pixels(world.run_in_window(title, "Measure").center());
    world.click(position);
    world.expect_text("Select geometry to measure");
    world.click_text("Done");
    world.click_text("Measure");
    let position = diagnostics::pointer_pixels(world.run_in_window(title, "Measure").center());
    world.right_click(position);
    world.click_text("Move to Main Panel");
    assert!(diagnostics::window_geometry(title).is_none());
    click_measure_command(&mut world);
    world.expect_text("Select geometry to measure");
    world.click_text("Done");
    world.close();
}

#[test]
fn part_measure_reports_analytic_face_area_and_survives_ribbon_tab_changes() {
    let mut world = ApplicationWorld::new();
    open_measurable_part_without_browser(&mut world);
    world.click_text("Tools");
    click_measure_command(&mut world);
    select_visible_solid_face(&mut world);
    let text = world.window_text("Measure");
    assert!(
        text.contains("Area600.000")
            || text.contains("Area800.000")
            || text.contains("Area1200.000"),
        "{text:?}: expected an analytic face area in mm²"
    );
    world.save_capture("part-measure-analytic-face");
    world.click_text("View");
    world.expect_text("Selection 1: Face");
    world.click([1000, 800]);
    world.expect_text("Select a solid face or edge to measure");
    world.click_text("Done");
    assert!(diagnostics::window_geometry("Measure").is_none());
    world.close();
}

fn open_measurable_part_without_browser(world: &mut ApplicationWorld) {
    super::super::super::workspace_layout::browser_feature::open_visible_extruded_part(world);
    world.click_text("View");
    world.click_text("User Interface");
    world.click_text("Model Browser");
    world.escape();
    world.expect_no_text("Origin");
}
fn select_visible_solid_face(world: &mut ApplicationWorld) {
    // The shaded silhouette can place the first visible pixel inside an edge pick radius.
    let camera = oblikovati_viewport::ViewportCamera {
        eye: [2., 1.5, 22.],
        target: [2., 1.5, 2.],
        scale: 7.,
        ..Default::default()
    };
    crate::HeadWindow::set_view_camera(&mut world.app().window, camera).unwrap();
    world.pump(Vec::new());
    let at = world
        .app()
        .window
        .native
        .view
        .model_pixel([2., 1.5, 2.])
        .unwrap();
    world.click(at);
    world.expect_text("Selection 1: Face");
}
