// SPDX-License-Identifier: GPL-2.0-only
//! Native ribbon appearance scenarios from the live application benchmark.
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};

fn open_tab_menu(world: &mut ApplicationWorld, tab: &str) {
    let position = diagnostics::pointer_pixels(world.run(tab).center());
    world.right_click(position);
    world.pump(Vec::new());
    world.expect_text("Show Panel Titles");
}
fn select_tab(world: &mut ApplicationWorld, tab: &str) {
    world.click_text(tab);
    world.pump(Vec::new());
}
fn select_title_setting(world: &mut ApplicationWorld) {
    world.click_text("Show Panel Titles");
    world.pump(Vec::new());
}
#[given("the full 3D Model ribbon shows panel titles")]
fn model_titles_shown(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
    world.expect_text("Modify");
}
#[when("the user clears Show Panel Titles from the 3D Model tab context menu")]
fn hide_titles(world: &mut ApplicationWorld) {
    open_tab_menu(world, "3D Model");
    assert!(world.checked("Show Panel Titles"));
    select_title_setting(world);
}
#[when("the user selects the Tools tab")]
fn tools_tab(world: &mut ApplicationWorld) {
    select_tab(world, "Tools");
}
#[then("the Tools panel titles are hidden")]
fn tools_titles_hidden(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
    assert!(
        world.frame.runs.iter().all(|run| run.text != "Options"),
        "expected the Options panel title to be hidden"
    );
}
#[when("the user opens the Tools tab context menu")]
fn tools_tab_menu(world: &mut ApplicationWorld) {
    open_tab_menu(world, "Tools");
}
#[then("Show Panel Titles is unchecked")]
fn titles_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Show Panel Titles"));
}
#[when("the user selects Show Panel Titles from that menu")]
fn show_titles(world: &mut ApplicationWorld) {
    select_title_setting(world);
}
#[then("the Tools panel titles appear")]
fn tools_titles_visible(world: &mut ApplicationWorld) {
    assert!(
        world.frame.runs.iter().any(|run| run.text == "Options"),
        "expected the Options panel title to be visible"
    );
    world.expect_text("Application Options");
}
#[when("the user returns to 3D Model")]
fn return_to_model(world: &mut ApplicationWorld) {
    select_tab(world, "3D Model");
}
#[then("the 3D Model panel titles are visible")]
fn model_titles_visible(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
    world.expect_text("Start 2D Sketch");
}

#[test]
fn appearance_menu_marks_normal_and_disables_unavailable_layouts() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tab_menu(&mut world, "3D Model");
    world.click_text("Ribbon Appearance");
    world.pump(Vec::new());
    assert!(world.checked("Normal"), "expected the current appearance");
    for label in ["Text Off", "Small", "Compact", "Large", "Reset Ribbon"] {
        world.expect_text(label);
        assert!(
            !world.run_is_enabled(label),
            "{label}: expected an unavailable control"
        );
    }
    world.click_text("Normal");
    world.pump(Vec::new());
    world.expect_no_text("Show Panel Titles");
    world.expect_text("Start 2D Sketch");
    world.close();
}

#[test]
fn part_feature_icons_keep_unavailable_commands_disabled() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.pump(Vec::new());
    for label in ["Revolve", "Hole", "Fillet", "Chamfer"] {
        world.expect_text(label);
        assert!(
            !world.run_is_enabled(label),
            "{label}: expected an unavailable command"
        );
    }
    world.save_capture("part-feature-icons-disabled");
    world.close();
}
