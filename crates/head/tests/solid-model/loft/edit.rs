// SPDX-License-Identifier: GPL-2.0-only
//! Native browser editing and cancellation of a committed Loft.
use super::*;

const OPTIONS: [&str; 2] = ["ClosedLoop", "MergeTangentFaces"];

#[given("the user creates a solid Loft1 by selecting both circular sketches as sections")]
#[given("a solid Loft1 made from Sketch1 and Sketch2")]
fn create_loft_to_edit(world: &mut ApplicationWorld) {
    parallel_sections_if_needed(world);
    pick_both_circles(world);
    assert_eq!(world.state().loft_preview, Some(Ok(())));
    accept_loft(world);
    loft_body_key(world);
}

#[when("Loft1 is double-clicked in the browser")]
#[when("Loft1 is reopened")]
pub(super) fn double_click_loft(world: &mut ApplicationWorld) {
    let pixel = diagnostics::pointer_pixels(world.run("Loft1").center());
    world.pump(crate::native::gui_loop_tests::pointer_click_events(
        pixel, 2,
    ));
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Loft").is_some());
}

#[then("its Loft dialog displays both sections and a preview")]
fn shows_saved_sections_and_preview(world: &mut ApplicationWorld) {
    assert_eq!(world.state().loft_preview, Some(Ok(())));
    assert!(loft_panel_has_text(world, "SectionsSketch1Sketch2"));
    let section_names: Vec<_> = world
        .state()
        .loft_sections
        .into_iter()
        .map(|section| section.name)
        .collect();
    assert_eq!(section_names, ["Sketch1", "Sketch2"]);
    let capture = capture_framebuffer(&mut world.app().window.native);
    assert!(solid_visibility::visible_preview_fill(
        &capture,
        &diagnostics::window_occlusions()
    ));
}

#[when("Closed Loop and Merge Tangent Faces are checked and Cancel is clicked")]
fn cancel_dirty_loft_options(world: &mut ApplicationWorld) {
    let generation = world.state().document_generation;
    let key = loft_body_key(world);
    for label in OPTIONS {
        click_loft_control(world, label);
        assert!(world.checked_run(world.run_in_window("Loft", label)));
    }
    click_loft_control(world, "Cancel");
    assert_eq!(world.state().document_generation, generation);
    assert_eq!(loft_body_key(world), key);
}

#[then("Loft1 remains visible")]
fn committed_loft_remains_visible(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Loft").is_none());
    assert!(world.state().browser.unwrap().bodies[0].visible);
    assert!(
        world
            .app()
            .window
            .scene_triangles
            .iter()
            .all(|triangle| { triangle.color != crate::part_scene::NEW_SOLID_PREVIEW })
    );
    assert!(!world.app().window.scene_triangles.is_empty());
}

#[then("both options are unchecked")]
fn saved_options_remain_unchecked(world: &mut ApplicationWorld) {
    for label in OPTIONS {
        assert!(!world.checked_run(world.run_in_window("Loft", label)));
    }
    shows_saved_sections_and_preview(world);
}

#[given("Loft1 is open for editing in its default Rails mode")]
fn starts_in_rails_mode(world: &mut ApplicationWorld) {
    if world.state().document.is_none() {
        create_loft_to_edit(world);
    }
    double_click_loft(world);
    rails_and_preview_return(world);
}

#[when("the second path mode is selected")]
fn view_center_line_mode(world: &mut ApplicationWorld) {
    click_loft_control(world, "CenterLine");
}

#[then("the dialog requests a Center Line sketch and OK is disabled")]
fn center_line_awaits_geometry(world: &mut ApplicationWorld) {
    assert!(loft_panel_has_text(world, "SelectaCenterLinesketch"));
    assert!(!loft_panel_has_text(world, "Norailsselected"));
    assert!(world.state().loft_preview.is_none());
    assert!(
        world
            .app()
            .window
            .scene_triangles
            .iter()
            .all(|triangle| { triangle.color != crate::part_scene::NEW_SOLID_PREVIEW })
    );
    assert!(!world.run_is_enabled("OK"));
}

#[when("the third path mode is selected")]
fn view_placed_sections_mode(world: &mut ApplicationWorld) {
    click_loft_control(world, "CenterLinewithPlacedSections");
}

#[then("the dialog requests a Center Line and offers Placed Sections")]
fn section_placement_needs_a_center_line(world: &mut ApplicationWorld) {
    center_line_awaits_geometry(world);
    world.run_in_window("Loft", "PlacedSections");
}

#[when("the first mode is selected again")]
fn restore_rails_mode(world: &mut ApplicationWorld) {
    click_loft_control(world, "Rails");
}

#[then("the Rails list and solid preview return")]
fn rails_and_preview_return(world: &mut ApplicationWorld) {
    assert!(loft_panel_has_text(world, "Norailsselected"));
    assert!(!loft_panel_has_text(world, "SelectaCenterLinesketch"));
    assert!(world.run_is_enabled("OK"));
    shows_saved_sections_and_preview(world);
}
