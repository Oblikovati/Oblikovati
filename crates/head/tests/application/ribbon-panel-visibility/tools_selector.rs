// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::{
    diagnostics, ribbon_customization::click_in_customize,
    ribbon_panel_expansion::open_blank_metric_part,
};
use super::super::{side_selector, *};

const HOME: &str = "##home-workspace";
const OPTIONS: &str = "##ribbon-additional-ToolOptions";
#[path = "face_selector.rs"]
mod face_selector;

#[given("a part and the Home workspace are open in the MCAD Software")]
fn part_and_home(world: &mut ApplicationWorld) {
    world.add_part();
    world.previous_document = Some(world.active_id());
    world.expect_tab("Home");
}

#[given(
    "`fixtures/cad/blank-metric-part.opd` is open and the Home workspace is available in the MCAD Software"
)]
fn saved_part_and_home(world: &mut ApplicationWorld) {
    open_blank_metric_part(world);
    world.expect_tab("Home");
}

#[given("the part is active with the full Tools ribbon visible")]
fn activate_tools(world: &mut ApplicationWorld) {
    assert_eq!(Some(world.active_id()), world.previous_document);
    world.click_text("Tools");
    tools_visible(world);
}

#[given("the Tools panel selector is open")]
#[when("the user opens the Tools panel selector")]
fn open_tools_selector(world: &mut ApplicationWorld) {
    side_selector::open_side_selector(world);
}

#[given("Measure, Material and Appearance, Options, Clipboard, Find and Team Web are checked")]
#[then("Measure, Material and Appearance, Options, Clipboard, Find and Team Web are checked")]
fn default_panels_checked(world: &mut ApplicationWorld) {
    assert!(side_selector::selector_popup_open());
    for panel in [
        "Measure",
        "Material and Appearance",
        "Options",
        "Clipboard",
        "Find",
        "Team Web",
    ] {
        assert!(world.checked(panel), "{panel} is unchecked");
    }
}

#[given("Tutorials is unchecked")]
fn tutorials_initially_unchecked(world: &mut ApplicationWorld) {
    assert!(side_selector::selector_popup_open());
    assert!(!world.checked("Tutorials"));
}

#[when("the user selects the Home workspace tab")]
fn home_tab(world: &mut ApplicationWorld) {
    let center = world.expect_tab("Home").center();
    world.click(diagnostics::pointer_pixels(center));
}

#[then("Home is displayed and the part panel selector is no longer visible")]
fn home_without_selector(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(HOME).is_some());
    assert!(!side_selector::selector_popup_open());
    assert!(world.document_is_open(world.previous_document.expect("expected the original Part")));
}

#[then("the Tools ribbon is visible and the panel selector is closed")]
fn tools_without_selector(world: &mut ApplicationWorld) {
    assert!(!side_selector::selector_popup_open());
    tools_visible(world);
}

#[then("the part and Tools ribbon remain visible")]
fn tools_visible(world: &mut ApplicationWorld) {
    assert_eq!(Some(world.active_id()), world.previous_document);
    world.expect_text("Application Options");
    world.expect_text("Document Settings");
}

#[when("the user clicks Customize in the Tools ribbon")]
fn click_customize(world: &mut ApplicationWorld) {
    let run = ribbon_runs(world, "Customize")
        .next()
        .expect("expected Customize");
    world.click(diagnostics::pointer_pixels(run.center()));
}

#[then("the customization dialog opens on the Ribbon tab")]
fn customize_ribbon_tab(world: &mut ApplicationWorld) {
    world.run_in_window("Customize", "Ribbon");
    world.run_in_window("Customize", "AllCommands");
}

#[then("the selected target is Part Tools")]
fn customize_tools_target(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("Part|Tools"));
}

#[when("the user closes customization without making changes")]
fn close_customization(world: &mut ApplicationWorld) {
    click_in_customize(world, "Close");
}

#[then("the display policy menu opens with Minimize to Tabs selected")]
fn tabs_policy_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Minimize to Tabs"));
}

#[then("the display policy menu closes")]
fn display_policy_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Minimize to Tabs");
}

#[then("the panel selector opens")]
fn selector_open(_: &mut ApplicationWorld) {
    assert!(side_selector::selector_popup_open());
}

#[when("the user closes the panel selector")]
fn close_selector(world: &mut ApplicationWorld) {
    world.escape();
    assert!(!side_selector::selector_popup_open());
}

#[when("the user clicks the Options panel title")]
fn click_options(world: &mut ApplicationWorld) {
    let title = ribbon_runs(world, "Options")
        .next()
        .expect("expected Options");
    world.click(diagnostics::pointer_pixels(title.center()));
}

#[then("the Options panel displays its additional commands")]
fn options_expansion(world: &mut ApplicationWorld) {
    world.run_in_window(OPTIONS, "ResetUILayout");
    world.run_in_window(OPTIONS, "iLogicConfiguration");
}
