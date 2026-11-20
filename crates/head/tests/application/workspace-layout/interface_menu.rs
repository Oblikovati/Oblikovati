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
