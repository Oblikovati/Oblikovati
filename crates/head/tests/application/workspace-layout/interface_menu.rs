// SPDX-License-Identifier: GPL-2.0-only
//! Read visibility options and check marks from the open View ribbon menu (#3897).
use super::{ApplicationWorld, browser_feature};
use cucumber::{given, then, when};

const CHECKED: [&str; 6] = [
    "ViewCube",
    "Navigation Bar",
    "Status Bar",
    "Document Tabs",
    "Marking Menu",
    "Model Browser",
];
const UNCHECKED: [&str; 4] = ["Mini-Toolbar", "Rule Log", "Rule Automation", "Favorites"];

#[given("the View ribbon tab is active")]
fn activate_view(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.expect_text("Visibility");
    world.expect_text("User Interface");
    // FUNCTIONAL.md describes a captured session with the toolbar already hidden.
    show_visibility_options(world);
    if world.checked("Mini-Toolbar") {
        world.click_text("Mini-Toolbar");
    }
    world.escape();
}

#[when("the user opens the User Interface menu")]
fn show_visibility_options(world: &mut ApplicationWorld) {
    world.click_text("User Interface");
    world.expect_text("Model Browser");
}

#[then(
    "the menu lists the viewport orientation control, navigation bar, contextual command toolbar, status bar, document tabs, marking menu, model browser, rule log, rule automation and favorites"
)]
fn lists_workspace_options(world: &mut ApplicationWorld) {
    let absent: Vec<_> = CHECKED
        .into_iter()
        .chain(UNCHECKED)
        .filter(|label| !world.shows(label))
        .collect();
    assert!(absent.is_empty(), "missing visibility options {absent:?}");
}

#[then(
    "the viewport orientation control, navigation bar, status bar, document tabs, marking menu and model browser are checked"
)]
fn visible_regions_are_checked(world: &mut ApplicationWorld) {
    for label in CHECKED {
        assert!(
            world.checked(label),
            "{label}: expected a checked visibility option"
        );
    }
}

#[then("the contextual command toolbar, rule log, rule automation and favorites are unchecked")]
fn hidden_regions_are_unchecked(world: &mut ApplicationWorld) {
    for label in UNCHECKED {
        world.expect_text(label);
        assert!(
            !world.checked(label),
            "{label}: expected an unchecked visibility option"
        );
    }
}

#[then("the part remains visible in the viewport")]
fn part_visible_behind_menu(world: &mut ApplicationWorld) {
    browser_feature::expect_solid_visible(world);
}

#[cfg(target_os = "linux")]
#[test]
fn native_rule_panel_choices_are_visible_and_disabled() {
    let mut world = ApplicationWorld::new();
    super::native_part::open_named_extrusion(&mut world);
    click_visible(&mut world, "View");
    click_visible(&mut world, "User Interface");
    assert_disabled_rule_choices(&mut world);
    world.close();
}

#[cfg(target_os = "linux")]
fn assert_disabled_rule_choices(world: &mut ApplicationWorld) {
    let available = world.run("Log Panel").alpha;
    for label in ["Rule Log", "Rule Automation"] {
        assert!(
            world.run(label).alpha < available,
            "{label}: expected a disabled option"
        );
        click_visible(world, label);
        assert!(!world.checked(label));
        assert!(super::window_geometry("Log").is_none());
        assert!(super::window_geometry(label).is_none());
    }
}

#[cfg(target_os = "linux")]
#[test]
fn native_view_menu_toggles_the_mini_toolbar_without_clearing_selection() {
    let mut world = ApplicationWorld::new();
    super::native_part::open_named_extrusion(&mut world);
    let name = super::native_part::single_extrusion_name(&mut world);
    click_visible(&mut world, &name);
    let selected = world.state().browser.unwrap().selected_feature;
    assert!(selected.is_some());
    assert!(super::window_geometry("##selection-toolbar").is_some());
    click_visible(&mut world, "View");
    toggle_mini_toolbar(&mut world, true);
    assert!(super::window_geometry("##selection-toolbar").is_none());
    assert_eq!(world.state().browser.unwrap().selected_feature, selected);
    toggle_mini_toolbar(&mut world, false);
    assert!(super::window_geometry("##selection-toolbar").is_some());
    assert_eq!(world.state().browser.unwrap().selected_feature, selected);
    world.close();
}

#[cfg(target_os = "linux")]
fn click_visible(world: &mut ApplicationWorld, label: &str) {
    let center = world.run(label).center();
    super::drawing_browser_steps::click_panel_action(world, center);
}

#[cfg(target_os = "linux")]
fn toggle_mini_toolbar(world: &mut ApplicationWorld, checked: bool) {
    click_visible(world, "User Interface");
    world.expect_text("Mini-Toolbar");
    assert_eq!(world.checked("Mini-Toolbar"), checked);
    click_visible(world, "Mini-Toolbar");
    world.expect_text("Mini-Toolbar");
    assert_eq!(world.checked("Mini-Toolbar"), !checked);
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
}
