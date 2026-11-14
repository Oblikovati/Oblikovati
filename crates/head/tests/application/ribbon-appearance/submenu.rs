// SPDX-License-Identifier: GPL-2.0-only
//! Steps for switching between the appearance and panel visibility lists.
use super::*;

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
