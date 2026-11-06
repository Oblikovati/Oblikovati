// SPDX-License-Identifier: GPL-2.0-only
use super::{ApplicationWorld, ribbon_docking::open_context_menu};
use cucumber::{given, then, when};
const PARAMETERS: &str = "Parameters";

fn expand_show_panels(world: &mut ApplicationWorld) {
    world.hover([700, 500]);
    open_context_menu(world);
    world.click_text("Show Panels");
    world.pump(Vec::new());
}
fn toggle_panel(world: &mut ApplicationWorld, panel: &str) {
    world.click_text(panel);
    world.pump(Vec::new());
    world.hover([700, 500]);
}
/// The browser also has a Parameters row, so the ribbon's panel is the second such run.
fn parameters_panel_shown(world: &ApplicationWorld) -> bool {
    let runs = world.frame.runs.iter().filter(|run| run.text == PARAMETERS);
    runs.count() > 1
}
fn expect_parameters_panel(world: &ApplicationWorld) {
    assert!(
        parameters_panel_shown(world),
        "rendered text {:?}: expected the panel",
        world.text()
    );
}
#[given("the Parameters panel is hidden")]
#[then("the Parameters panel disappears from the ribbon")]
#[then("the Parameters group button disappears")]
#[then("the Parameters panel is absent from the expanded ribbon")]
fn parameters_hidden(world: &mut ApplicationWorld) {
    assert!(
        !parameters_panel_shown(world),
        "expected no Parameters panel"
    );
}
#[when("I open the ribbon context menu and expand Show Panels")]
#[when("I open the compact ribbon context menu and expand Show Panels")]
#[when("I reopen Show Panels")]
#[when("I open the View ribbon context menu and expand Show Panels")]
#[when("I reopen Show Panels on the View ribbon")]
fn show_panels(world: &mut ApplicationWorld) {
    expand_show_panels(world);
}
#[then("Parameters is unchecked")]
fn parameters_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked(PARAMETERS), "expected Parameters unchecked");
}
#[then("visible panels including Create and Modify are checked")]
fn visible_checked(world: &mut ApplicationWorld) {
    for panel in ["Create", "Modify"] {
        assert!(world.checked(panel), "{panel}: expected a check mark");
    }
}
#[when("I select Parameters in Show Panels")]
#[when("I select Parameters again")]
fn toggle_parameters(world: &mut ApplicationWorld) {
    toggle_panel(world, PARAMETERS);
}
#[then("the Parameters panel appears in the ribbon")]
#[then("the Parameters panel is still visible")]
#[then("its 3D Model ribbon also shows the Parameters panel")]
#[then("its Parameters panel remains visible")]
fn parameters_shown(world: &mut ApplicationWorld) {
    expect_parameters_panel(world);
}
#[then("Parameters is checked")]
fn parameters_checked(world: &mut ApplicationWorld) {
    assert!(world.checked(PARAMETERS), "expected Parameters checked");
}
#[given("two part documents are open in the MCAD Software")]
fn two_parts(world: &mut ApplicationWorld) {
    world.add_part();
    world.add_part();
}
#[given("the first part is active with the 3D Model ribbon expanded")]
fn first_active_expanded(world: &mut ApplicationWorld) {
    let first = world.parts[0];
    world.activate_through_tab(first);
    world.expect_text("Sketch XY");
}
#[given("the Parameters panel is visible")]
#[given("the Parameters group button is visible")]
fn parameters_made_visible(world: &mut ApplicationWorld) {
    expand_show_panels(world);
    toggle_panel(world, PARAMETERS);
    expect_parameters_panel(world);
}
#[when("I return to the 3D Model ribbon tab")]
fn return_to_model(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
    world.pump(Vec::new());
}
#[when("I activate the second part document")]
fn activate_second(world: &mut ApplicationWorld) {
    let second = world.parts[1];
    world.click_tab(second);
    world.pump(Vec::new());
}
#[when("I return to the first part document")]
fn activate_first(world: &mut ApplicationWorld) {
    let first = world.parts[0];
    world.click_tab(first);
    world.pump(Vec::new());
}
#[given("the View ribbon is expanded with the Navigate panel visible")]
fn view_expanded_with_navigate(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.pump(Vec::new());
    world.expect_text("Navigate");
    world.previous_document = Some(world.active_id());
}
#[then("Visibility, Appearance, Windows and Navigate are checked")]
fn view_panels_checked(world: &mut ApplicationWorld) {
    for panel in ["Visibility", "Appearance", "Windows", "Navigate"] {
        assert!(world.checked(panel), "{panel}: expected a check mark");
    }
}
#[when("I select Navigate in Show Panels")]
fn toggle_navigate(world: &mut ApplicationWorld) {
    toggle_panel(world, "Navigate");
}
#[then("the Navigate panel disappears from the View ribbon")]
#[then("the Navigate panel remains hidden")]
fn navigate_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Navigate");
}
#[when("I switch to 3D Model and return to View")]
fn switch_and_return(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
    world.pump(Vec::new());
    world.click_text("View");
    world.pump(Vec::new());
}
#[then("Navigate is unchecked")]
fn navigate_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Navigate"), "expected Navigate unchecked");
}
#[then("the Navigate panel reappears on the View ribbon")]
fn navigate_back(world: &mut ApplicationWorld) {
    world.expect_text("Navigate");
}
#[given("the 3D Model ribbon shows compact group buttons")]
fn model_compact(world: &mut ApplicationWorld) {
    world.click_text("Ribbon Display");
    world.pump(Vec::new());
    world.expect_no_text("Sketch XY");
}
#[then("the other compact group buttons remain visible")]
fn other_groups_remain(world: &mut ApplicationWorld) {
    for group in ["Sketch", "Create", "Modify"] {
        world.expect_text(group);
    }
}
#[when("I return the ribbon to its expanded display")]
fn return_expanded(world: &mut ApplicationWorld) {
    while !world.shows("Sketch XY") {
        world.click_text("Ribbon Display");
        world.pump(Vec::new());
    }
}
