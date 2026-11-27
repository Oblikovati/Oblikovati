// SPDX-License-Identifier: GPL-2.0-only
use super::super::diagnostics::RenderedRun;
use super::*;
#[path = "tools_selector.rs"]
mod tools_selector;

fn caption<'a>(world: &'a ApplicationWorld, name: &str) -> Option<&'a RenderedRun> {
    let options = ribbon_runs(world, "Options")
        .next()
        .expect("expected Options caption");
    ribbon_runs(world, name).find(|run| (run.center()[1] - options.center()[1]).abs() < 5.)
}

fn shown<'a>(world: &'a ApplicationWorld, panel: &str) -> &'a RenderedRun {
    caption(world, panel)
        .unwrap_or_else(|| panic!("expected Tools panel {panel:?}: {:?}", world.text()))
}

#[given("the Team Web panel is visible")]
fn initial_team_web(world: &mut ApplicationWorld) {
    team_web_returns(world);
}

#[given("the Find panel is visible")]
#[given("the Find and Tutorials panels are both visible")]
fn initial_find(world: &mut ApplicationWorld) {
    enable_tutorials(world);
    shown(world, "Find");
}

#[given("the Tutorials panel is visible after Team Web")]
#[given("the Tutorials panel is visible")]
fn enable_tutorials(world: &mut ApplicationWorld) {
    if caption(world, "Tutorials").is_none() {
        tools_tab_show_panels(world);
        assert!(!world.checked("Tutorials"));
        click_panel_row(world, "Tutorials");
    }
    tutorials_after_team_web(world);
}

#[when("the user opens Show Panels from the Options panel title menu")]
#[when("the user opens Show Panels from the Options panel title context menu")]
#[when("the user opens Show Panels from a ribbon panel title context menu")]
#[when("the user reopens Show Panels from the Options panel title menu")]
#[when("the user opens Show Panels again")]
fn options_show_panels(world: &mut ApplicationWorld) {
    open_show_panels_at(world, "Options");
}

#[when("the user opens Show Panels from the Tools tab context menu")]
#[when("the user reopens Show Panels")]
#[when("the user opens Show Panels on the Tools tab")]
fn tools_tab_show_panels(world: &mut ApplicationWorld) {
    open_show_panels_at(world, "Tools");
}

#[then(regex = r"^(Team Web|Tutorials) is (checked|unchecked)$")]
fn panel_check(world: &mut ApplicationWorld, name: String, expected: String) {
    assert_eq!(world.checked(&name), expected == "checked", "{name}");
}

#[when(regex = r"^the user selects (Team Web|Tutorials|Find)(?: again)?$")]
fn choose_panel(world: &mut ApplicationWorld, panel: String) {
    click_panel_row(world, &panel);
}

fn click_panel_row(world: &mut ApplicationWorld, panel: &str) {
    assert!(
        super::side_selector::popup_depth() >= 2,
        "expected Show Panels"
    );
    toggle_panel(world, panel);
    world.expect_no_text("Show Panels");
}

#[then(regex = r"^the (Team Web|Tutorials|Find) panel disappears from the (?:Tools )?ribbon$")]
fn panel_hidden(world: &mut ApplicationWorld, panel: String) {
    assert!(caption(world, &panel).is_none(), "{panel} remains visible");
}

#[then("Tutorials is checked in the panel list")]
fn tutorials_checked_in_list(world: &mut ApplicationWorld) {
    if side_selector::popup_depth() < 2 {
        options_show_panels(world);
    }
    assert!(world.checked("Tutorials"));
}

#[then("Tutorials is unchecked in the panel list")]
fn tutorials_unchecked_in_list(world: &mut ApplicationWorld) {
    options_show_panels(world);
    assert!(!world.checked("Tutorials"));
}

#[then("the other visible Tools panels remain available")]
fn other_tools_remain(world: &mut ApplicationWorld) {
    for panel in [
        "Measure",
        "Material and Appearance",
        "Options",
        "Clipboard",
        "Find",
    ] {
        shown(world, panel);
    }
}

#[then("the Team Web panel appears on the Tools ribbon")]
fn team_web_returns(world: &mut ApplicationWorld) {
    shown(world, "Team Web");
}

#[then("the Tutorials panel reappears after Team Web")]
fn tutorials_after_team_web(world: &mut ApplicationWorld) {
    assert!(shown(world, "Team Web").center()[0] < shown(world, "Tutorials").center()[0]);
}

#[then(regex = r"^Show Panels marks (Team Web|Tutorials) as visible$")]
fn restored_panel_is_checked(world: &mut ApplicationWorld, panel: String) {
    tools_tab_show_panels(world);
    assert!(world.checked(&panel), "{panel} is unchecked");
    world.escape();
    world.escape();
}

#[when("the user hides Find from Show Panels on the Tools tab")]
fn hide_find(world: &mut ApplicationWorld) {
    tools_tab_show_panels(world);
    assert!(world.checked("Find"));
    click_panel_row(world, "Find");
}

#[when("the user switches to 3D Model and returns to Tools")]
fn round_trip_to_model(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
    world.expect_text("Start 2D Sketch");
    world.click_text("Tools");
    world.expect_text("Application Options");
}

#[then("the Find panel remains hidden")]
fn find_still_hidden(world: &mut ApplicationWorld) {
    panel_hidden(world, "Find".into());
}

#[then("Find is unchecked and Tutorials remains checked")]
fn independent_tools_choices(world: &mut ApplicationWorld) {
    assert!(!world.checked("Find"));
    assert!(world.checked("Tutorials"));
}

#[then("the Find panel reappears before Team Web")]
fn find_before_team_web(world: &mut ApplicationWorld) {
    assert!(shown(world, "Find").center()[0] < shown(world, "Team Web").center()[0]);
}

#[then("Find and Tutorials are both visible")]
fn find_and_tutorials_visible(world: &mut ApplicationWorld) {
    shown(world, "Find");
    shown(world, "Tutorials");
}

#[then("Tutorials remains visible")]
fn tutorials_remains_visible(world: &mut ApplicationWorld) {
    shown(world, "Tutorials");
}

#[then("Find is unchecked while Tutorials remains checked")]
fn confirm_find_is_hidden(world: &mut ApplicationWorld) {
    options_show_panels(world);
    independent_tools_choices(world);
    world.escape();
    world.escape();
}

#[then("Find and Tutorials are both checked")]
fn confirm_both_panels_are_checked(world: &mut ApplicationWorld) {
    options_show_panels(world);
    for panel in ["Find", "Tutorials"] {
        assert!(world.checked(panel), "{panel} is unchecked");
    }
    world.escape();
    world.escape();
}

#[given("the full Tools ribbon is active")]
fn full_tools_with_tutorials(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    tools_remains_active(world);
    enable_tutorials(world);
}

#[when("the user right-clicks the inactive 3D Model tab")]
fn right_click_model_tab(world: &mut ApplicationWorld) {
    let tab = ribbon_runs(world, "3D Model")
        .next()
        .expect("expected 3D Model tab");
    world.right_click(super::super::diagnostics::pointer_pixels(tab.center()));
}

#[then("the Tools ribbon remains active")]
#[then("the 3D Model tab remains inactive")]
fn tools_remains_active(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
    world.expect_no_text("Start 2D Sketch");
}

#[then("the ribbon context menu is visible")]
fn context_menu_visible(world: &mut ApplicationWorld) {
    world.expect_text("Show Panel Titles");
}

#[when("the user opens Show Panels from that menu")]
fn open_current_panel_menu(world: &mut ApplicationWorld) {
    world.click_text("Show Panels");
}

pub(super) const TOOLS_PANELS: [&str; 7] = [
    "Measure",
    "Material and Appearance",
    "Options",
    "Clipboard",
    "Find",
    "Team Web",
    "Tutorials",
];

#[then(
    "the submenu lists Measure, Material and Appearance, Options, Clipboard, Find, Team Web and Tutorials"
)]
fn active_tools_entries(world: &mut ApplicationWorld) {
    assert!(side_selector::popup_depth() >= 2, "expected Show Panels");
    for label in TOOLS_PANELS {
        world.expect_text(label);
    }
    world.expect_no_text("Modify");
}

#[then("each listed Tools panel is checked")]
fn active_tools_checked(world: &mut ApplicationWorld) {
    for label in TOOLS_PANELS {
        assert!(world.checked(label), "expected {label} checked");
    }
}
