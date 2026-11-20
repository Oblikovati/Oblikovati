// SPDX-License-Identifier: GPL-2.0-only
//! Steps for switching between the appearance and panel visibility lists.
use super::super::ribbon_customization::{
    click_in_customize, open_tools_customization, stage_sketch_visibility,
};
use super::*;

fn open_tools_panel_list(world: &mut ApplicationWorld) {
    let title = diagnostics::pointer_pixels(world.run("Options").center());
    world.right_click(title);
    world.click_text("Show Panels");
    world.pump(Vec::new());
}

fn set_tools_panel_visibility(world: &mut ApplicationWorld, label: &str, visible: bool) {
    open_tools_panel_list(world);
    world.expect_text(label);
    if world.checked(label) != visible {
        world.click_text(label);
        world.pump(Vec::new());
        open_tools_panel_list(world);
    }
    assert_eq!(
        world.checked(label),
        visible,
        "{label}: unexpected visibility"
    );
    world.escape();
    world.escape();
}

fn enable_user_commands(world: &mut ApplicationWorld) {
    open_tools_panel_list(world);
    let present = world.text().lines().any(|line| line == "User Commands");
    world.escape();
    world.escape();
    if !present {
        open_tools_customization(world);
        stage_sketch_visibility(world);
        click_in_customize(world, "OK");
    }
    set_tools_panel_visibility(world, "User Commands", true);
}

#[given("Team Web, Tutorials, and User Commands are enabled in Show Panels")]
fn enable_captured_panels(world: &mut ApplicationWorld) {
    enable_user_commands(world);
    set_tools_panel_visibility(world, "Tutorials", true);
    set_tools_panel_visibility(world, "Team Web", true);
}

#[given("Team Web is disabled while Tutorials and User Commands are enabled in Show Panels")]
fn setup_reset_panels(world: &mut ApplicationWorld) {
    enable_user_commands(world);
    set_tools_panel_visibility(world, "Tutorials", true);
    set_tools_panel_visibility(world, "Team Web", false);
}

#[then("Team Web, Tutorials, and User Commands are checked in the captured panel state")]
fn captured_panels_checked(world: &mut ApplicationWorld) {
    for label in ["Team Web", "Tutorials", "User Commands"] {
        world.expect_text(label);
        assert!(world.checked(label), "expected {label} checked");
    }
}

#[test]
fn tools_panel_fixture_changes_and_restores_visibility() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    select_tab(&mut world, "Tools");
    set_tools_panel_visibility(&mut world, "Find", false);
    world.expect_no_text("Find");
    set_tools_panel_visibility(&mut world, "Find", true);
    world.expect_text("Find");
    world.close();
}

#[test]
fn optional_tools_panels_follow_show_panels_and_reset() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    select_tab(&mut world, "Tools");
    world.expect_no_text("Tutorials");
    enable_user_commands(&mut world);
    world.expect_text("User Commands");
    set_tools_panel_visibility(&mut world, "Tutorials", true);
    world.expect_text("Tutorials");
    open_panel_appearance(&mut world, "Options");
    world.click_text("Reset Ribbon");
    world.click_text("Yes");
    select_tab(&mut world, "Tools");
    world.expect_no_text("Tutorials");
    world.expect_no_text("User Commands");
    world.close();
}

#[test]
fn team_web_opens_a_closeable_resource_window() {
    let mut world = ApplicationWorld::new();
    let document = open_appearance_part(&mut world);
    select_tab(&mut world, "Tools");
    let button = diagnostics::pointer_pixels(world.run("Open Team Web").center());
    super::super::x11_input::click(button);
    world.pump(Vec::new());
    let content = world.window_text("Team Web");
    assert!(content.contains("OpeninBrowser"), "{content:?}");
    assert!(content.contains("https://docs.oblikovati.org/"), "{content:?}");
    world.save_capture("team-web-resource");
    let (position, size) = diagnostics::window_geometry("Team Web").unwrap();
    let close = [position[0] + size[0] - 10., position[1] + 10.];
    super::super::x11_input::click(diagnostics::pointer_pixels(close));
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Team Web").is_none());
    assert_eq!(world.active_id(), document);
    world.close();
}

#[then("Normal is marked as selected")]
#[then("Normal is still marked as selected")]
fn normal_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Normal"), "expected Normal checked");
}

#[when("the user opens Show Panels from the same context menu")]
fn open_panel_list(world: &mut ApplicationWorld) {
    world.click_text("Show Panels");
    world.pump(Vec::new());
}

#[then("the appearance submenu closes")]
fn appearance_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Text Off");
}

#[then(
    "the Show Panels submenu lists Measure, Material and Appearance, Options, Clipboard, Find, Team Web, Tutorials, and User Commands with visibility checks"
)]
fn tools_panel_choices(world: &mut ApplicationWorld) {
    let submenu = world.run("Measure").list;
    for label in [
        "Measure",
        "Material and Appearance",
        "Options",
        "Clipboard",
        "Find",
        "Team Web",
        "Tutorials",
        "User Commands",
    ] {
        world.expect_text(label);
        assert_eq!(
            world.run(label).list,
            submenu,
            "{label}: expected in panel list"
        );
    }
    for label in [
        "Measure",
        "Material and Appearance",
        "Options",
        "Clipboard",
        "Find",
        "Team Web",
    ] {
        assert!(world.checked(label), "expected {label} checked");
    }
}

#[then("the panel context menu remains open")]
fn parent_menu_open(world: &mut ApplicationWorld) {
    world.expect_text("Show Panels");
}

#[then("the Tools ribbon remains visible")]
fn tools_visible(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
}
