// SPDX-License-Identifier: GPL-2.0-only
//! Drive the published Object Visibility checklist through the View ribbon.
use super::{ApplicationWorld, diagnostics, scenario_root, world::squeezed};
use cucumber::{given, then, when};

const MASTER: &str = "All Workfeatures";
const ORIGIN_PLANES: &str = "Origin Planes";

#[given("the MCAD Software has a part document open")]
#[given("the MCAD Software has a solid part document open")]
fn open_part(world: &mut ApplicationWorld) {
    let path = scenario_root().join("fixtures/cad/docking-visible-solid-current.opd");
    world.open_fixture(&path);
    super::workspace_layout::browser_feature::expect_solid_visible(world);
}

#[given("the View ribbon is selected")]
fn view_selected(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.expect_text("Object Visibility");
}

fn click(world: &mut ApplicationWorld, label: &str) {
    let at = diagnostics::pointer_pixels(world.run(label).center());
    #[cfg(target_os = "linux")]
    super::desktop_input::click(world, at);
    #[cfg(not(target_os = "linux"))]
    world.click(at);
}

#[when("I open Object Visibility")]
fn open_checklist(world: &mut ApplicationWorld) {
    click(world, "Object Visibility");
}

fn entries(world: &ApplicationWorld) -> Vec<&diagnostics::RenderedRun> {
    let master = world.run(MASTER);
    assert!(world.frame.is_popup_row(master));
    world
        .frame
        .runs
        .iter()
        .filter(|row| row.list == master.list)
        .collect()
}

#[then("a checklist opens beneath the ribbon control")]
fn opens_beneath_trigger(world: &mut ApplicationWorld) {
    let trigger = world.run("Object Visibility");
    let first = entries(world)[0];
    assert!(first.minimum[1] > trigger.maximum[1]);
    assert!((trigger.minimum[0] - first.minimum[0]).abs() < 40.);
}

#[then("the viewport and model browser remain visible behind the checklist")]
fn workspace_remains(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Model").is_some());
    super::workspace_layout::browser_feature::expect_solid_visible(world);
}

fn table_entries(step: &cucumber::gherkin::Step) -> impl Iterator<Item = String> + '_ {
    step.table
        .as_ref()
        .expect("expected checklist entries")
        .rows
        .iter()
        .flatten()
        .map(|name| squeezed(name))
}

#[then("the checklist contains the following entries in order:")]
fn lists_in_order(world: &mut ApplicationWorld, step: &cucumber::gherkin::Step) {
    let shown: Vec<_> = entries(world).iter().map(|row| row.text.clone()).collect();
    assert_eq!(shown, table_entries(step).collect::<Vec<_>>());
}

#[then("a separator appears before the UCS entries")]
fn separates_ucs(world: &mut ApplicationWorld) {
    let center = |label| world.run(label).center()[1];
    let normal = center("3D Annotations") - center("Sketch Dimensions");
    let separator = center("UCS Triad") - center("3D Annotations");
    assert!(
        separator > normal + 1.,
        "UCS gap {separator}, row pitch {normal}"
    );
}

fn expect_checks(world: &ApplicationWorld, want: impl Fn(&str) -> bool) {
    for row in entries(world) {
        assert_eq!(world.checked_run(row), want(&row.text), "{}", row.text);
    }
}

#[given("all object visibility entries are checked")]
fn starts_checked(world: &mut ApplicationWorld) {
    open_checklist(world);
    expect_checks(world, |_| true);
}

#[given("All Workfeatures and its related categories are unchecked")]
fn master_starts_unchecked(world: &mut ApplicationWorld) {
    starts_checked(world);
    click(world, MASTER);
    assert!(!world.checked(MASTER));
}

#[when("I uncheck All Workfeatures")]
#[when("I check All Workfeatures")]
fn toggle_master(world: &mut ApplicationWorld) {
    click(world, MASTER);
}

#[then("the menu remains open")]
fn menu_open(world: &mut ApplicationWorld) {
    entries(world);
    assert!(!world.checked(MASTER));
}

#[then("the following entries are unchecked:")]
fn listed_unchecked(world: &mut ApplicationWorld, step: &cucumber::gherkin::Step) {
    for name in table_entries(step) {
        assert!(!world.checked(&name), "{name}");
    }
}

#[then("the following entries remain checked:")]
fn listed_checked(world: &mut ApplicationWorld, step: &cucumber::gherkin::Step) {
    for name in table_entries(step) {
        assert!(world.checked(&name), "{name}");
    }
}

#[then("the related origin, user work-feature, and UCS entries are checked")]
#[then("the other category checks remain unchanged")]
fn restored(world: &mut ApplicationWorld) {
    expect_checks(world, |_| true);
}

#[when("I uncheck Origin Planes")]
fn toggle_planes(world: &mut ApplicationWorld) {
    click(world, ORIGIN_PLANES);
}

#[then("Origin Planes is unchecked")]
fn planes_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked(ORIGIN_PLANES));
}

#[then("All Workfeatures remains checked")]
fn master_stays_checked(world: &mut ApplicationWorld) {
    assert!(world.checked(MASTER));
}

#[then("every other entry remains checked")]
fn only_planes_unchecked(world: &mut ApplicationWorld) {
    expect_checks(world, |name| name != "OriginPlanes");
}

#[test]
fn object_visibility_hides_and_restores_origin_axis_and_sketch_dimensions() {
    use super::workspace_layout::browser_feature::colored_pixels;
    let mut world = ApplicationWorld::new();
    open_part(&mut world);
    view_selected(&mut world);
    world.click_text("X Axis");
    open_checklist(&mut world);
    let blue_axis = [64, 153, 255];
    assert!(colored_pixels(&mut world, blue_axis) > 0);
    click(&mut world, "Origin Axes");
    assert_eq!(colored_pixels(&mut world, blue_axis), 0);
    click(&mut world, "Origin Axes");
    assert!(colored_pixels(&mut world, blue_axis) > 0);
    world.expect_text("40 mm");
    click(&mut world, "Sketch Dimensions");
    world.expect_no_text("40 mm");
    click(&mut world, "Sketch Dimensions");
    world.expect_text("40 mm");
    workspace_remains(&mut world);
    world.close();
}
