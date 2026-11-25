// SPDX-License-Identifier: GPL-2.0-only
//! Ribbon tab wheel navigation and Tools command tests.
use super::*;

#[given("a part is active in the MCAD Software with the full ribbon visible")]
fn part_with_full_ribbon(world: &mut ApplicationWorld) {
    super::super::workspace_layout::part_open(world);
    world.expect_text("Extrude");
}
#[given("the 3D Model tab is the first visible command tab and is active")]
fn model_is_first(world: &mut ApplicationWorld) {
    world.expect_text("3D Model");
    world.expect_text("Extrude");
}
fn scroll_tab(world: &mut ApplicationWorld, label: &str, notches: i32) {
    use oblikovati_viewport::{ViewportEvent, ViewportInput};
    let position = super::super::diagnostics::pointer_pixels(world.run(label).center());
    world.hover(position);
    world.pump(vec![ViewportEvent::Input(ViewportInput::Wheel {
        notches,
        position,
    })]);
}
#[when("the user scrolls up over the 3D Model tab")]
fn wheel_up_on_model(world: &mut ApplicationWorld) {
    scroll_tab(world, "3D Model", 1);
}
#[then("the 3D Model tab remains active")]
fn model_remains(world: &mut ApplicationWorld) {
    world.expect_text("Extrude");
}
#[when("the user selects the last visible command tab")]
fn select_last_tab(world: &mut ApplicationWorld) {
    world.click_text("Annotate");
}
#[when("the user scrolls down over that tab")]
#[when("the user scrolls down over the last visible command tab")]
fn wheel_down_on_last(world: &mut ApplicationWorld) {
    scroll_tab(world, "Annotate", -1);
}
#[then("the last visible command tab remains active")]
fn last_remains(world: &mut ApplicationWorld) {
    world.expect_text("General Note");
}
#[then("its command groups remain visible")]
fn current_groups_remain(world: &mut ApplicationWorld) {
    assert!(
        world.shows("Extrude") || world.shows("General Note"),
        "expected current tab commands"
    );
}

#[test]
fn mouse_wheel_selects_adjacent_ribbon_tabs() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    let camera = crate::HeadWindow::view_camera(&world.app().window).unwrap();
    scroll_tab(&mut world, "3D Model", -1);
    world.expect_text("Finish Sketch");
    world.expect_no_text("Extrude");
    scroll_tab(&mut world, "Sketch", -1);
    world.expect_text("Application Options");
    scroll_tab(&mut world, "Tools", -1);
    world.expect_text("Parameters");
    scroll_tab(&mut world, "Manage", 1);
    world.expect_text("Application Options");
    scroll_tab(&mut world, "Tools", 1);
    world.expect_text("Finish Sketch");
    scroll_tab(&mut world, "Sketch", 1);
    world.expect_text("Extrude");
    assert_eq!(
        crate::HeadWindow::view_camera(&world.app().window).unwrap(),
        camera
    );
    world.close();
}

#[test]
fn mouse_wheel_keeps_tabs_only_closed_until_a_tab_is_clicked() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    choose_tabs_mode(&mut world);
    scroll_tab(&mut world, "3D Model", -1);
    world.expect_no_text("Extrude");
    world.expect_no_text("Finish Sketch");
    world.click_text("Tools");
    world.expect_text("Application Options");
    world.close();
}

#[test]
fn mouse_wheel_switches_an_open_temporary_ribbon() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    choose_tabs_mode(&mut world);
    world.click_text("3D Model");
    world.expect_text("Extrude");
    scroll_tab(&mut world, "3D Model", -1);
    world.expect_no_text("Extrude");
    world.expect_text("Finish Sketch");
    world.close();
}

#[given("the full Tools ribbon is visible")]
#[given("the full Tools ribbon is displayed")]
pub(super) fn full_tools(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.expect_text("Application Options");
}
#[when("the user scrolls down over the Tools tab")]
fn wheel_down_on_tools(world: &mut ApplicationWorld) {
    scroll_tab(world, "Tools", -1);
}
#[then("the full Manage command groups are visible")]
fn manage_active(world: &mut ApplicationWorld) {
    assert!(
        world
            .frame
            .runs
            .iter()
            .filter(|run| run.text == "Parameters")
            .count()
            > 1
    );
}
#[then("Manage becomes the active tab")]
#[then("Manage remains active in tabs-only display")]
fn manage_tab_selected(world: &mut ApplicationWorld) {
    world.hover([1000, 600]);
    let inactive = world.pixel_left_of("View");
    assert_eq!(world.pixel_left_of("Tools"), inactive);
    assert_ne!(world.pixel_left_of("Manage"), inactive);
}
#[when("the user scrolls up over the same position on the tab row")]
fn wheel_up_over_same_tab_position(world: &mut ApplicationWorld) {
    scroll_tab(world, "Tools", 1);
}
#[then("the full Tools command groups are visible")]
fn tools_active(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
    world.expect_text("Document Settings");
}
#[then("Tools becomes the active tab")]
fn tools_tab_selected(world: &mut ApplicationWorld) {
    world.hover([1000, 600]);
    let inactive = world.pixel_left_of("View");
    assert_eq!(world.pixel_left_of("Manage"), inactive);
    assert_ne!(world.pixel_left_of("Tools"), inactive);
}

#[given("no temporary ribbon commands are open")]
#[then("the ribbon remains in tabs-only display without opening temporary commands")]
fn no_temporary_commands(world: &mut ApplicationWorld) {
    assert!(window_geometry("##ribbon-temporary-panel").is_none());
    world.expect_no_text("Application Options");
}

#[given("the temporary Tools command groups are open")]
fn temporary_tools_open(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.expect_text("Application Options");
    assert!(window_geometry("##ribbon-temporary-panel").is_some());
}

#[then("temporary Manage command groups replace the Tools command groups")]
fn temporary_manage_replaces_tools(world: &mut ApplicationWorld) {
    assert!(window_geometry("##ribbon-temporary-panel").is_some());
    world.expect_no_text("Application Options");
    manage_active(world);
}

#[cfg(target_os = "linux")]
#[test]
fn native_wheel_switches_open_tabs_only_commands() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    full_tools(&mut world);
    choose_tabs_mode(&mut world);
    temporary_tools_open(&mut world);
    let at = super::super::diagnostics::pointer_pixels(world.run("Tools").center());
    super::super::x11_input::focus_window();
    super::super::x11_input::scroll(at, -1);
    world.pump(Vec::new());
    world.pump(Vec::new());
    manage_tab_selected(&mut world);
    temporary_manage_replaces_tools(&mut world);
    world.close();
}

#[test]
fn tools_ribbon_opens_application_options() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    full_tools(&mut world);
    world.click_text("Application Options");
    world.expect_text("Text size");
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    world.expect_no_text("Text size");
    world.close();
}

#[test]
fn tools_document_settings_edit_the_active_part_units() {
    use crate::HeadWork;
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    full_tools(&mut world);
    let units = world
        .app()
        .owner
        .gui_command_state()
        .unwrap()
        .document_units
        .unwrap();
    let selected = if units.length_unit == "in" {
        "mm"
    } else {
        "in"
    };
    world.click_text("Document Settings");
    select_document_units(&mut world);
    world.click_text(&units.length_unit);
    world.click_text(selected);
    assert_eq!(
        world
            .app()
            .owner
            .gui_command_state()
            .unwrap()
            .document_units
            .unwrap()
            .length_unit,
        units.length_unit,
        "a staged edit must not change the document before Apply"
    );
    let angle = if units.angle_unit == "rad" {
        "deg"
    } else {
        "rad"
    };
    world.click_text(&units.angle_unit);
    world.click_text(angle);
    let mass = if units.mass_unit == "lb" { "kg" } else { "lb" };
    world.click_text(&units.mass_unit);
    world.click_text(mass);
    let time = if units.time_unit == "min" { "s" } else { "min" };
    world.click_text(&units.time_unit);
    world.click_text(time);
    let format = if units.length_display_format == "fractional" {
        "decimal"
    } else {
        "fractional"
    };
    world.click_text(&units.length_display_format);
    world.click_text(format);
    world.click_text("Apply");
    let updated = world
        .app()
        .owner
        .gui_command_state()
        .unwrap()
        .document_units
        .unwrap();
    assert_eq!(
        (
            updated.angle_unit.as_str(),
            updated.mass_unit.as_str(),
            updated.time_unit.as_str(),
            updated.length_display_format.as_str()
        ),
        (angle, mass, time, format)
    );
    assert_eq!(updated.length_unit, selected);
    world.click_text("Close");
    world.expect_no_text("Length precision");
    world.close();
}

#[test]
fn document_settings_cancel_discards_staged_unit_edits() {
    use crate::HeadWork;
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    full_tools(&mut world);
    let before = world
        .app()
        .owner
        .gui_command_state()
        .unwrap()
        .document_units
        .unwrap();
    let selected = if before.length_unit == "in" {
        "mm"
    } else {
        "in"
    };
    world.click_text("Document Settings");
    select_document_units(&mut world);
    world.click_text(&before.length_unit);
    world.click_text(selected);
    world.click_text("Cancel");
    let after = world
        .app()
        .owner
        .gui_command_state()
        .unwrap()
        .document_units
        .unwrap();
    assert_eq!(after, before, "Cancel must discard the full staged edit");
    world.click_text("Document Settings");
    assert!(
        world
            .window_text("Document Settings")
            .contains("PartMaterial:")
    );
    select_document_units(&mut world);
    world.expect_text(&before.length_unit);
    world.click_text("Close");
    world.close();
}

fn select_document_units(world: &mut ApplicationWorld) {
    let tab = world.run_in_window("Document Settings", "Units").center();
    world.click(super::super::diagnostics::pointer_pixels(tab));
    world.expect_text("Length precision");
}

#[test]
fn compact_tools_closes_its_panel_when_document_settings_opens() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    full_tools(&mut world);
    click_display_menu(&mut world);
    world.click_text("Minimize to Panel Buttons");
    world.click_text("Options");
    world.expect_text("Document Settings");
    let position =
        super::super::diagnostics::pointer_pixels(world.run("Document Settings").center());
    world.hover(position);
    let button = |pressed| {
        oblikovati_viewport::ViewportEvent::Input(oblikovati_viewport::ViewportInput::Button {
            button: oblikovati_viewport::ViewportButton::Left,
            pressed,
            position,
            modifiers: Default::default(),
        })
    };
    world.pump(vec![button(true)]);
    world.expect_text("Document Settings");
    world.pump(vec![button(false)]);
    world.run_in_window("Document Settings", "Standard");
    world.click_text("Close");
    world.expect_no_text("Length precision");
    world.expect_no_text("Document Settings");
    world.close();
}

#[test]
fn tabs_only_tools_closes_temporary_commands_after_opening_settings() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    full_tools(&mut world);
    choose_tabs_mode(&mut world);
    world.click_text("Tools");
    world.expect_text("Document Settings");
    world.click_text("Document Settings");
    world.run_in_window("Document Settings", "Standard");
    world.expect_no_text("Application Options");
    world.click_text("Close");
    world.expect_no_text("Length precision");
    world.expect_no_text("Document Settings");
    world.close();
}

#[test]
fn widening_an_open_width_collapsed_panel_closes_its_popup() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    world.app().window.native.view.resize([700, 600]).unwrap();
    world.pump(Vec::new());
    let modify = world.run("Modify").center();
    world.click(super::super::diagnostics::pointer_pixels([
        modify[0],
        modify[1] - 35.,
    ]));
    world.expect_text("Hole");
    assert!(world.run("Hole").minimum[1] > band_height());
    world.app().window.native.view.resize([1600, 900]).unwrap();
    world.pump(Vec::new());
    world.expect_text("Hole");
    assert!(world.run("Hole").minimum[1] < band_height());
    world.app().window.native.view.resize([700, 600]).unwrap();
    world.pump(Vec::new());
    world.expect_no_text("Hole");
    world.close();
}

#[test]
fn compact_options_stays_highlighted_when_clicked_again() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    full_tools(&mut world);
    click_display_menu(&mut world);
    world.click_text("Minimize to Panel Buttons");
    let label = world.run("Options").clone();
    let sample = [label.minimum[0] as usize - 4, label.center()[1] as usize];
    world.hover([1000, 600]);
    let idle = world.pixel_at(sample);
    world.click_text("Options");
    world.hover([1000, 600]);
    world.expect_text("Document Settings");
    let selected = world.pixel_at(sample);
    assert_ne!(idle, selected, "open Options button: expected a highlight");
    world.click(super::super::diagnostics::pointer_pixels(
        compact_options::options_button(&world).center(),
    ));
    world.hover([1000, 600]);
    world.expect_text("Document Settings");
    assert_eq!(world.pixel_at(sample), selected);
    world.close();
}

#[test]
fn document_settings_is_unavailable_while_home_is_in_front() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    full_tools(&mut world);
    world.click_text("Home");
    world.click_text("Document Settings");
    world.expect_no_text("Length precision");
    world.click_tab(world.previous_document.unwrap());
    world.click_text("Document Settings");
    world.run_in_window("Document Settings", "Standard");
    world.close();
}
