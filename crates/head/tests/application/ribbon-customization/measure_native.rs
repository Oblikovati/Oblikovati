// SPDX-License-Identifier: GPL-2.0-only
//! Verify the Tools ribbon opens the existing sketch Measure command.
use super::*;

#[test]
fn tools_measure_respects_context_and_opens_sketch_inspection() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("Tools");
    assert_ne!(measure_command(&world).alpha, u8::MAX);
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
