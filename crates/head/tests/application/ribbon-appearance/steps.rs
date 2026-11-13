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
#[given("the full 3D Model ribbon uses the Normal appearance")]
fn model_normal_appearance(world: &mut ApplicationWorld) {
    expect_normal_appearance(world, "3D Model", "Start 2D Sketch");
}
fn expect_normal_appearance(world: &mut ApplicationWorld, tab: &str, command: &str) {
    open_tab_menu(world, tab);
    world.click_text("Ribbon Appearance");
    assert!(world.checked("Normal"));
    world.escape();
    world.escape();
    world.expect_text(command);
}
#[given("the Sketch ribbon uses Normal appearance")]
fn sketch_normal_appearance(world: &mut ApplicationWorld) {
    expect_normal_appearance(world, "Sketch", "Finish Sketch");
}
#[when("the user opens Ribbon Appearance from the Create panel title context menu")]
fn sketch_appearance_submenu(world: &mut ApplicationWorld) {
    open_panel_appearance(world, "Create");
}
fn open_panel_appearance(world: &mut ApplicationWorld, panel: &str) {
    let title = diagnostics::pointer_pixels(world.run(panel).center());
    world.right_click(title);
    world.click_text("Ribbon Appearance");
    assert!(world.checked("Normal"));
}
#[when("the user clicks the Tools tab while the submenu is open")]
fn tools_tab_with_appearance_open(world: &mut ApplicationWorld) {
    select_tab(world, "Tools");
}
#[when("the user selects the Sketch tab")]
fn sketch_tab(world: &mut ApplicationWorld) {
    select_tab(world, "Sketch");
}
#[then("the Sketch ribbon is displayed")]
fn sketch_ribbon_displayed(world: &mut ApplicationWorld) {
    world.expect_text("Finish Sketch");
}
#[given("the Ribbon Appearance submenu is open from a panel context menu")]
fn appearance_submenu_open(world: &mut ApplicationWorld) {
    open_panel_appearance(world, "Modify");
}
#[then("the appearance submenu and panel context menu close")]
fn appearance_menus_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Ribbon Appearance");
    world.expect_no_text("Normal");
}
#[then("the Tools ribbon is displayed")]
fn tools_ribbon_displayed(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
}
#[given("the full 3D Model ribbon shows panel titles")]
#[given("panel titles are visible")]
fn model_titles_shown(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
    world.expect_text("Modify");
    let title = world.run("Modify");
    world.modify_title_anchor = Some([title.minimum[0], title.center()[1]]);
}
#[when("the user opens the inactive Sketch tab context menu")]
fn inactive_sketch_context(world: &mut ApplicationWorld) {
    open_tab_menu(world, "Sketch");
}
#[then("3D Model remains selected and Show Panel Titles is checked")]
fn model_active_titles_checked(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
    assert!(world.checked("Show Panel Titles"));
}
#[then("the menu closes and the 3D Model panel titles are hidden")]
fn menu_closed_model_titles_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Show Panel Titles");
    model_titles_hidden(world);
}
#[then("3D Model remains selected")]
fn model_remains_selected(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
}
#[when("the user opens the 3D Model tab context menu")]
fn model_tab_menu(world: &mut ApplicationWorld) {
    open_tab_menu(world, "3D Model");
}
#[when("the user clears Show Panel Titles")]
#[when("the user selects Show Panel Titles")]
fn change_title_setting(world: &mut ApplicationWorld) {
    select_title_setting(world);
}
#[then("the ribbon panel titles are hidden")]
fn model_titles_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Modify");
}
#[then("the main panel commands remain visible")]
fn model_commands_visible(world: &mut ApplicationWorld) {
    for command in ["Start 2D Sketch", "Extrude", "Hole"] {
        world.expect_text(command);
    }
}
fn hidden_modify_arrow(world: &ApplicationWorld) -> [i32; 2] {
    let anchor = world
        .modify_title_anchor
        .expect("expected Modify title anchor");
    let arrow = world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text == "v"
                && (run.minimum[0] - anchor[0]).abs() < 15.
                && (run.center()[1] - anchor[1]).abs() < 8.
        })
        .expect("expected Modify expansion arrow with titles hidden");
    diagnostics::pointer_pixels(arrow.center())
}
#[then("the Modify panel expansion arrow remains visible")]
fn hidden_modify_arrow_visible(world: &mut ApplicationWorld) {
    hidden_modify_arrow(world);
}
#[when("the user clicks the Modify panel expansion arrow")]
fn click_hidden_modify_arrow(world: &mut ApplicationWorld) {
    world.click(hidden_modify_arrow(world));
}
#[then("the panel titles return")]
fn panel_titles_return(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
}
#[when("the user clears Show Panel Titles from the 3D Model tab context menu")]
fn hide_titles(world: &mut ApplicationWorld) {
    open_tab_menu(world, "3D Model");
    assert!(world.checked("Show Panel Titles"));
    select_title_setting(world);
}
#[when("the user selects the Tools tab")]
#[when("the user selects Tools")]
fn tools_tab(world: &mut ApplicationWorld) {
    select_tab(world, "Tools");
}
#[then("the Tools panel titles are hidden")]
#[then("the Tools panel titles are also hidden")]
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
#[then("Show Panel Titles is unchecked and Tools remains selected")]
fn tools_active_titles_unchecked(world: &mut ApplicationWorld) {
    titles_unchecked(world);
    world.expect_text("Application Options");
}
#[then("the menu closes and the Tools panel titles are visible")]
fn menu_closed_tools_titles_visible(world: &mut ApplicationWorld) {
    world.expect_no_text("Show Panel Titles");
    tools_titles_visible(world);
}
#[then("Tools remains selected")]
fn tools_remains_selected(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
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

#[test]
fn hidden_panel_titles_keep_the_modify_expansion_accessible() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    model_titles_shown(&mut world);
    hide_titles(&mut world);
    click_hidden_modify_arrow(&mut world);
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_some());
    world.expect_text("Move Bodies");
    world.save_capture("hidden-title-modify-expansion");
    world.close();
}

#[test]
fn pinned_additional_commands_survive_panel_title_changes() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let title = world.run("Modify").center();
    world.click(diagnostics::pointer_pixels(title));
    let pin = world.run("Pin");
    world.click(diagnostics::pointer_pixels([
        pin.minimum[0] + 6.,
        pin.center()[1],
    ]));
    hide_titles(&mut world);
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_some());
    open_tab_menu(&mut world, "3D Model");
    select_title_setting(&mut world);
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_some());
    world.close();
}
