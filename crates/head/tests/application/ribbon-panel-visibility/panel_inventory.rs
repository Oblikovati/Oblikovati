// SPDX-License-Identifier: GPL-2.0-only
//! Inspect the published Show Panels inventories on the actual ribbon tabs.
use super::{tools_panels::TOOLS_PANELS, *};

const MANAGE_PANELS: [&str; 10] = [
    "Update",
    "Parameters",
    "Manage",
    "Styles and Standards",
    "Insert",
    "Point Cloud",
    "Layout",
    "Author",
    "iLogic",
    "Content Center",
];
const MODEL_PANELS: [(&str, bool); 19] = [
    ("Sketch", true),
    ("Primitives", false),
    ("Create", true),
    ("Modify", true),
    ("Explore", true),
    ("Work Features", true),
    ("Pattern", true),
    ("Create Freeform", true),
    ("Surface", true),
    ("Parameters", false),
    ("Simplification", true),
    ("Plastic Part", false),
    ("Measure", false),
    ("Insert", false),
    ("Harness", false),
    ("iPart/Assembly", false),
    ("Simulation", true),
    ("Convert", true),
    ("3D Print", true),
];

#[given("the full Manage ribbon is active")]
fn activate_manage(world: &mut ApplicationWorld) {
    world.click_text("Manage");
    manage_active(world);
}

#[then("the Manage ribbon remains active")]
fn manage_active(world: &mut ApplicationWorld) {
    assert!(ribbon_runs(world, "Parameters").next().is_some());
    assert!(ribbon_runs(world, "Insert").next().is_some());
    world.expect_no_text("Start 2D Sketch");
}

#[when("the user opens Show Panels from the Manage ribbon context menu")]
fn manage_panel_menu(world: &mut ApplicationWorld) {
    open_show_panels_at(world, "Manage");
}

#[then(
    "the list contains Update, Parameters, Manage, Styles and Standards, Insert, Point Cloud, Layout, Author, iLogic and Content Center"
)]
fn manage_inventory(world: &mut ApplicationWorld) {
    expect_panel_menu_order(world, &MANAGE_PANELS);
}

#[then("every listed Manage panel is checked")]
fn manage_panels_checked(world: &mut ApplicationWorld) {
    expect_checks(world, &MANAGE_PANELS, true);
}

#[test]
fn manage_ribbon_opens_active_part_properties_and_updates() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    activate_manage(&mut world);
    world.click_text("Update");
    world.expect_active(part);
    world.click_text("iProperties");
    let (position, size) =
        super::super::diagnostics::window_geometry("iProperties").expect("expected iProperties");
    world.expect_text("Document properties");
    let close = [position[0] + size[0] - 13., position[1] + 10.];
    world.click(super::super::diagnostics::pointer_pixels(close));
    assert!(super::super::diagnostics::window_geometry("iProperties").is_none());
    world.expect_active(part);
    world.close();
}

#[given("the 3D Model ribbon is active and docked at the left side")]
fn dock_model_left(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
    model_docked_left(world);
    model_docked_active(world);
}

#[then("the 3D Model ribbon remains active and docked at the left side")]
fn model_docked_active(world: &mut ApplicationWorld) {
    is_docked_left(world);
    side_selector::side_group_y(world, "Modify");
}

#[test]
fn side_model_measure_visibility_is_shared_with_the_direct_selector() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    dock_model_left(&mut world);
    side_selector::open_side_selector(&mut world);
    assert!(!world.checked("Measure"));
    world.click_text("Measure");
    assert!(world.checked("Measure"));
    side_selector::click_selector(&mut world);
    assert!(!side_selector::selector_popup_open());
    side_selector::side_group_y(&world, "Measure");
    side_selector::side_group_y(&world, "Explore");
    world.close();
}

#[when("the user opens Show Panels from the ribbon context menu")]
#[when("the user opens Show Panels in the context menu")]
#[when("the user opens the \"Show Panels\" submenu")]
fn open_panel_submenu(world: &mut ApplicationWorld) {
    world.click_text("Show Panels");
    world.pump(Vec::new());
    assert!(side_selector::popup_depth() >= 2, "expected Show Panels");
}

#[then(
    "the submenu lists Sketch, Primitives, Create, Modify, Explore, Work Features, Pattern, Create Freeform, Surface, Parameters, Simplification, Plastic Part, Measure, Insert, Harness, iPart/Assembly, Simulation, Convert and 3D Print"
)]
fn model_inventory(world: &mut ApplicationWorld) {
    expect_panel_menu_order(world, &MODEL_PANELS.map(|(name, _)| name));
}

#[then("the submenu lists panels from the active 3D Model ribbon")]
fn active_model_inventory(world: &mut ApplicationWorld) {
    expect_panel_menu_order(world, &["Sketch", "Create", "Modify"]);
}

#[then(
    "Sketch, Create, Modify, Explore, Work Features, Pattern, Create Freeform, Surface, Simplification, Simulation, Convert and 3D Print are checked"
)]
fn model_panels_checked(world: &mut ApplicationWorld) {
    check_model_visibility(world, true);
}

#[then(
    "Primitives, Parameters, Plastic Part, Measure, Insert, Harness and iPart/Assembly are unchecked"
)]
fn model_panels_unchecked(world: &mut ApplicationWorld) {
    check_model_visibility(world, false);
}

fn check_model_visibility(world: &ApplicationWorld, visible: bool) {
    for (panel, _) in MODEL_PANELS
        .iter()
        .filter(|(_, checked)| *checked == visible)
    {
        assert_eq!(world.checked(panel), visible, "{panel}: wrong check state");
    }
}

#[given("the Primitives panel is hidden and the Modify panel is visible")]
fn model_panel_defaults(world: &mut ApplicationWorld) {
    assert!(ribbon_runs(world, "Primitives").next().is_none());
    assert!(ribbon_runs(world, "Modify").next().is_some());
}

#[when(regex = r"^the user right-clicks the inactive (View|Tools) ribbon tab$")]
fn right_click_inactive_tab(world: &mut ApplicationWorld, tab: String) {
    let at = ribbon_runs(world, &tab)
        .next()
        .expect("expected tab")
        .center();
    world.right_click(super::super::diagnostics::pointer_pixels(at));
}

#[then("the 3D Model ribbon remains active")]
fn model_ribbon_active(world: &mut ApplicationWorld) {
    model_panel_defaults(world);
    world.expect_text("Start 2D Sketch");
}

#[then("a ribbon context menu opens")]
fn context_menu_opens(world: &mut ApplicationWorld) {
    world.expect_text("Show Panels");
    assert!(side_selector::popup_depth() > 0);
}

#[then("Primitives is unchecked and Modify is checked")]
#[given("Modify is checked and Primitives is unchecked")]
#[then("Modify is still checked and Primitives is still unchecked")]
fn model_panel_states(world: &mut ApplicationWorld) {
    assert!(side_selector::popup_depth() > 0, "expected a panel menu");
    expect_checks(world, &["Primitives"], false);
    expect_checks(world, &["Modify"], true);
}

#[then("Explore is checked in the submenu")]
fn explore_is_checked(world: &mut ApplicationWorld) {
    expect_checks(world, &["Explore"], true);
}

fn expect_panel_menu_order(world: &ApplicationWorld, panels: &[&str]) {
    assert!(side_selector::popup_depth() >= 2, "expected Show Panels");
    let mut previous = world.run(panels[0]);
    for panel in &panels[1..] {
        let row = world.run(panel);
        assert_eq!(row.list, previous.list, "{panel}: expected the submenu");
        assert!(
            row.center()[1] > previous.center()[1],
            "{panel}: wrong order"
        );
        previous = row;
    }
}

fn expect_checks(world: &ApplicationWorld, panels: &[&str], expected: bool) {
    for panel in panels {
        assert_eq!(world.checked(panel), expected, "{panel}: wrong check state");
    }
}

#[then(
    "the selector lists Measure, Material and Appearance, Options, Clipboard, Find, Team Web, and Tutorials"
)]
fn direct_tools_panel_entries(world: &mut ApplicationWorld) {
    assert!(side_selector::selector_popup_open());
    let list = world.run("Tutorials").list;
    for label in TOOLS_PANELS {
        assert_eq!(
            world.run(label).list,
            list,
            "{label}: expected the selector row"
        );
    }
}

#[then("the selector does not list User Commands")]
fn direct_selector_excludes_custom_panel(world: &mut ApplicationWorld) {
    let list = world.run("Tutorials").list;
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|row| { row.list == list && row.text == "UserCommands" }),
        "User Commands appears in the Tools panel selector"
    );
}

#[then("the User Commands panel remains visible on the ribbon")]
fn custom_panel_remains_visible(world: &mut ApplicationWorld) {
    assert!(ribbon_runs(world, "User Commands").next().is_some());
}
