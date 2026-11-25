// SPDX-License-Identifier: GPL-2.0-only
//! Finish a sketch from a non-sketch ribbon tab through the native window.
use super::*;

#[test]
fn tools_keeps_finish_until_the_sketch_is_finished() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let document = world.state().document;
    world.click_text("Tools");
    world.pump(Vec::new());
    world.expect_text("Application Options");
    world.expect_text("Finish Sketch");
    assert!(world.run_is_enabled("Finish Sketch"));
    world.click_text("Finish Sketch");
    world.pump(Vec::new());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    assert_eq!(world.state().document, document);
    world.expect_text("Application Options");
    world.expect_no_text("Finish Sketch");
    world.close();
}

#[test]
fn view_keeps_finish_until_the_sketch_is_finished() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.click_text("View");
    world.pump(Vec::new());
    world.expect_text("User Interface");
    world.click_text("Finish Sketch");
    world.pump(Vec::new());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    world.expect_text("User Interface");
    world.expect_no_text("Finish Sketch");
    world.close();
}

use super::super::workspace_layout::browser_feature::{
    FEATURE, SKETCH, click_branch_arrow, expect_solid_visible, open_visible_extruded_part,
    right_click_row,
};
use cucumber::{given, then, when};

#[given("a part with an editable sketch is active in the MCAD Software")]
fn editable_sketch_active(world: &mut ApplicationWorld) {
    if world.state().authoring != oblikovati_gui::GuiAuthoringContext::Sketch {
        world.start_empty_sketch();
    }
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
}
#[given("the full Sketch command groups are visible")]
fn prepare_full_sketch_ribbon(world: &mut ApplicationWorld) {
    if !world.shows("Finish Sketch") {
        activate_display_control(world);
    }
    full_sketch_ribbon(world);
}
#[then("the full Sketch command groups are visible")]
fn full_sketch_ribbon(world: &mut ApplicationWorld) {
    world.expect_text("Rectangle");
    world.expect_text("Finish Sketch");
    assert!(window_geometry("##ribbon-temporary-panel").is_none());
}
#[then("the command groups disappear and the ribbon tabs remain visible")]
fn sketch_tabs_without_commands(world: &mut ApplicationWorld) {
    world.expect_text("Sketch");
    world.expect_no_text("Rectangle");
    world.expect_no_text("Finish Sketch");
    assert!(window_geometry("##ribbon-temporary-panel").is_none());
}

#[cfg(target_os = "linux")]
#[test]
fn native_display_toggle_keeps_sketch_editing() {
    let mut world = ApplicationWorld::new();
    editable_sketch_active(&mut world);
    choose_tabs_mode(&mut world);
    prepare_full_sketch_ribbon(&mut world);
    let at = super::super::diagnostics::pointer_pixels(world.run("Ribbon Display").center());
    super::super::x11_input::focus_window();
    for show_commands in [false, true, false, true] {
        super::super::x11_input::click(at);
        world.pump(Vec::new());
        if show_commands {
            full_sketch_ribbon(&mut world);
        } else {
            sketch_tabs_without_commands(&mut world);
        }
        assert_eq!(
            world.state().authoring,
            oblikovati_gui::GuiAuthoringContext::Sketch
        );
    }
    world.close();
}

#[given("a part containing an existing extrusion and sketch is active in the MCAD Software")]
fn part_with_existing_sketch(world: &mut ApplicationWorld) {
    open_visible_extruded_part(world);
    if world.shows(SKETCH) {
        click_branch_arrow(world, FEATURE);
    }
    world.expect_no_text(SKETCH);
}
#[given("the ribbon is minimized to tabs with Tools selected")]
fn minimized_tools_ribbon(world: &mut ApplicationWorld) {
    full_tools(world);
    choose_tabs_mode(world);
    world.expect_no_text("Application Options");
}
#[when("the user expands the extrusion in the model browser")]
fn expand_existing_extrusion(world: &mut ApplicationWorld) {
    click_branch_arrow(world, FEATURE);
    world.expect_text(SKETCH);
}
#[when("the user double-clicks its sketch")]
fn edit_existing_sketch_by_double_click(world: &mut ApplicationWorld) {
    let at = super::super::diagnostics::pointer_pixels(world.run(SKETCH).center());
    world.pump(crate::native::gui_loop_tests::pointer_click_events(at, 2));
    world.pump(Vec::new());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
}
#[then("the Sketch tab is selected while the ribbon remains minimized to tabs")]
fn sketch_selected_without_commands(world: &mut ApplicationWorld) {
    sketch_tabs_without_commands(world);
    world.hover([1000, 600]);
    let inactive = world.pixel_left_of("View");
    assert_eq!(world.pixel_left_of("Tools"), inactive);
    assert_ne!(world.pixel_left_of("Sketch"), inactive);
}
#[then("temporary Tools command groups appear beneath the tabs")]
fn temporary_tools_with_sketch(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
    assert!(window_geometry("##ribbon-temporary-panel").is_some());
}
#[then("Finish is available in an Exit group")]
fn finish_in_exit_group(world: &mut ApplicationWorld) {
    world.expect_text("Exit");
    assert!(world.run_is_enabled("Finish Sketch"));
}
#[then("the ribbon remains minimized to tabs with Tools selected")]
fn minimized_tools_selected(world: &mut ApplicationWorld) {
    world.expect_no_text("Application Options");
    world.hover([1000, 600]);
    let inactive = world.pixel_left_of("View");
    assert_eq!(world.pixel_left_of("Sketch"), inactive);
    assert_ne!(world.pixel_left_of("Tools"), inactive);
}

#[given("an existing sketch is being edited in the MCAD Software")]
fn existing_sketch_under_edit(world: &mut ApplicationWorld) {
    open_visible_extruded_part(world);
    right_click_row(world, SKETCH);
    world.click_text("Edit Sketch");
    world.pump(Vec::new());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
}
#[given("the Sketch ribbon is minimized to tabs")]
fn minimized_sketch_ribbon(world: &mut ApplicationWorld) {
    choose_tabs_mode(world);
    world.expect_no_text("Finish Sketch");
}
#[when("the user clicks the Sketch tab")]
fn open_sketch_tab(world: &mut ApplicationWorld) {
    world.click_text("Sketch");
    world.pump(Vec::new());
}
#[then("the Sketch command groups appear temporarily beneath the tabs")]
fn temporary_sketch_groups(world: &mut ApplicationWorld) {
    assert!(window_geometry("##ribbon-temporary-panel").is_some());
    world.expect_text("Rectangle");
}
#[then("the temporary ribbon includes the Finish action")]
fn temporary_finish(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Finish Sketch"));
}
#[when("the user selects Finish")]
fn finish_from_ribbon(world: &mut ApplicationWorld) {
    world.click_text("Finish Sketch");
    world.pump(Vec::new());
}
#[then("sketch editing ends and the part model is displayed")]
fn finished_model_visible(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    expect_solid_visible(world);
}
