// SPDX-License-Identifier: GPL-2.0-only
use super::{
    ApplicationWorld,
    ribbon_docking::{dock_ribbon_top, open_context_menu, undock_ribbon},
};
use cucumber::{given, then, when};
const PARAMETERS: &str = "Parameters";
#[path = "minimized_context.rs"]
mod minimized_context;
#[path = "sketch_parameters.rs"]
mod sketch_parameters;

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
fn parameters_panel_shown(world: &ApplicationWorld) -> bool {
    panel_selector::caption_is_visible(world)
}
fn expect_parameters_panel(world: &ApplicationWorld) {
    assert!(
        parameters_panel_shown(world),
        "rendered text {:?}: expected the panel",
        world.text()
    );
}
#[given("the Parameters panel is hidden")]
#[given("Parameters is hidden")]
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
#[when("I open Show Panels on the left-docked ribbon")]
#[when("I open Show Panels on the floating ribbon")]
#[when("I open Show Panels on the top ribbon")]
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
    world.expect_text("Start 2D Sketch");
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
    super::ribbon_display::click_display_menu(world);
    world.click_text("Minimize to Panel Buttons");
    world.pump(Vec::new());
    world.expect_no_text("Start 2D Sketch");
}
#[when("I return the ribbon to its expanded display")]
fn return_expanded(world: &mut ApplicationWorld) {
    while !world.shows("Start 2D Sketch") {
        world.click_text("Ribbon Display");
        world.pump(Vec::new());
    }
}
fn model_docked_left(world: &mut ApplicationWorld) {
    open_context_menu(world);
    world.click_text("Docking Positions");
    world.pump(Vec::new());
    world.click_text("Left");
    world.pump(Vec::new());
    world.expect_text("3D Model");
}
#[given("the Modify group is visible")]
#[given("the 3D Model ribbon is docked at the top with Modify visible")]
#[then("the Modify group returns to the left-docked ribbon")]
#[then("the Modify panel is visible in the top ribbon")]
#[then("the Modify group appears in the floating ribbon")]
#[then("the Modify panel is visible again")]
fn modify_visible(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
}
#[then("Modify is checked")]
fn modify_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Modify"), "expected Modify checked");
}
#[when("I select Modify in Show Panels")]
#[when("I select Modify")]
fn toggle_modify(world: &mut ApplicationWorld) {
    world.ribbon_height = world.run("Work Features").center()[1];
    toggle_panel(world, "Modify");
}
#[then("the Modify panel disappears from the ribbon")]
#[then("the floating ribbon does not show the Modify group")]
#[then("the floating ribbon no longer shows the Modify group")]
#[then("the top ribbon does not show the Modify panel")]
#[then("the Modify group remains hidden")]
fn modify_gone(world: &mut ApplicationWorld) {
    world.expect_no_text("Modify");
}
#[then("the remaining ribbon groups reflow vertically")]
fn groups_reflow(world: &mut ApplicationWorld) {
    let before = world.ribbon_height;
    let now = side_selector::side_group_y(world, "Work Features");
    assert!(
        now < before,
        "Work Features at {now}: expected it to move up from {before}"
    );
}
#[then("Modify is unchecked")]
fn modify_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Modify"), "expected Modify unchecked");
}
#[when("I hide Modify through Show Panels")]
fn hide_modify(world: &mut ApplicationWorld) {
    expand_show_panels(world);
    toggle_modify(world);
}
#[when("I dock the ribbon on the left")]
fn dock_left(world: &mut ApplicationWorld) {
    model_docked_left(world);
}
#[then("the ribbon is docked on the left")]
fn is_docked_left(world: &mut ApplicationWorld) {
    let column = super::diagnostics::window_geometry("##ribbon-column");
    let (position, _) = column.expect("expected the side column");
    assert_eq!(
        position[0], 0.,
        "column at {position:?}: expected the left edge"
    );
    world.expect_text("3D Model");
}
#[when("I undock the ribbon")]
fn undock(world: &mut ApplicationWorld) {
    undock_ribbon(world);
}
#[given("the floating 3D Model ribbon shows the Modify group")]
fn floating_with_modify(world: &mut ApplicationWorld) {
    undock_ribbon(world);
    world.expect_text("Modify");
}
#[when("I dock the ribbon at the top")]
fn dock_top(world: &mut ApplicationWorld) {
    dock_ribbon_top(world);
}
const PATTERN: &str = "Pattern";
#[given("the expanded 3D Model ribbon shows Modify and Pattern")]
fn modify_and_pattern(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
    world.expect_text(PATTERN);
}
#[when("I hide Pattern through Show Panels")]
fn hide_pattern(world: &mut ApplicationWorld) {
    expand_show_panels(world);
    toggle_panel(world, PATTERN);
}
#[then("neither Modify nor Pattern is visible in the ribbon")]
fn neither_visible(world: &mut ApplicationWorld) {
    world.expect_no_text("Modify");
    world.expect_no_text(PATTERN);
}
#[then("Modify and Pattern are both unchecked")]
fn both_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Modify"), "expected Modify unchecked");
    assert!(!world.checked(PATTERN), "expected Pattern unchecked");
}
#[then("Modify appears in the ribbon")]
fn modify_appears(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
}
#[then("Pattern remains hidden")]
fn pattern_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text(PATTERN);
}
#[then("Modify is checked and Pattern is unchecked")]
fn modify_checked_only(world: &mut ApplicationWorld) {
    assert!(world.checked("Modify"), "expected Modify checked");
    assert!(!world.checked(PATTERN), "expected Pattern unchecked");
}
#[when("I select Pattern")]
fn select_pattern(world: &mut ApplicationWorld) {
    toggle_panel(world, PATTERN);
}
#[then("both panels are visible in the ribbon")]
fn both_visible(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
    world.expect_text(PATTERN);
}
#[given("Modify is visible on the 3D Model ribbon")]
fn modify_on_model(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
    world.pump(Vec::new());
    world.expect_text("Modify");
}
#[given("Navigate is visible on the View ribbon")]
fn navigate_visible(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.pump(Vec::new());
    world.expect_text("Navigate");
    world.click_text("3D Model");
    world.pump(Vec::new());
}
#[when("I hide Modify on 3D Model through Show Panels")]
#[when("I restore Modify through Show Panels")]
fn toggle_modify_panel(world: &mut ApplicationWorld) {
    expand_show_panels(world);
    toggle_panel(world, "Modify");
    world.escape();
    world.escape();
    world.pump(Vec::new());
}
#[when("I switch to View and hide Navigate through Show Panels")]
fn hide_navigate(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.pump(Vec::new());
    expand_show_panels(world);
    toggle_panel(world, "Navigate");
    world.escape();
    world.escape();
    world.pump(Vec::new());
}
#[when("I return to 3D Model")]
fn return_model(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
    world.pump(Vec::new());
}
#[then("Modify remains hidden")]
fn modify_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Modify");
}
#[when("I switch to View")]
fn switch_view(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.pump(Vec::new());
}
#[then("Navigate remains hidden")]
fn navigate_stays_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Navigate");
}
#[when("I open Show Panels on the View ribbon")]
fn panels_on_view(world: &mut ApplicationWorld) {
    expand_show_panels(world);
}
#[when("I select Navigate")]
fn select_navigate(world: &mut ApplicationWorld) {
    toggle_panel(world, "Navigate");
    world.escape();
    world.escape();
    world.pump(Vec::new());
}
#[then("the Navigate panel appears on the View ribbon")]
fn navigate_appears(world: &mut ApplicationWorld) {
    world.expect_text("Navigate");
}
// Dismissing the list
#[given("the Create and Modify panels are visible")]
#[given("the 3D Model ribbon shows Create and Modify")]
fn shows_create_modify(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
    world.pump(Vec::new());
    world.expect_text("Create");
    world.expect_text("Modify");
}
#[given("the View ribbon is expanded with Visibility, Appearance, Windows and Navigate visible")]
#[given("the View ribbon is expanded with the Navigate panel visible")]
fn view_ribbon_expanded(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.pump(Vec::new());
    for panel in ["Visibility", "Appearance", "Windows", "Navigate"] {
        world.expect_text(panel);
    }
}
#[given("the 3D Model ribbon is expanded and docked at the top")]
#[given("the expanded 3D Model ribbon is docked at the top")]
fn model_ribbon_at_top(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
    world.pump(Vec::new());
    world.expect_text("Start 2D Sketch");
    world.expect_text("Ribbon Display");
}
#[given("it shows Create and Modify while Parameters is hidden")]
fn create_modify_without_parameters(world: &mut ApplicationWorld) {
    shows_create_modify(world);
    parameters_is_hidden(world);
}
#[then("Create and Modify are checked and Parameters is unchecked")]
fn checks_read(world: &mut ApplicationWorld) {
    for panel in ["Create", "Modify"] {
        assert!(world.checked(panel), "{panel}: expected a check mark");
    }
    assert!(!world.checked(PARAMETERS), "expected Parameters unchecked");
}
#[then("the Show Panels submenu closes")]
fn submenu_closed(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    assert!(!world.checked("Create"), "expected the list closed");
}
#[then("the ribbon context menu remains open")]
fn context_open(world: &mut ApplicationWorld) {
    world.expect_text("Show Panels");
}
#[when("I press Escape again")]
fn escape_again(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
}
#[then("the ribbon context menu closes")]
fn context_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Show Panels");
}
#[then("Create and Modify remain visible")]
fn create_modify_visible(world: &mut ApplicationWorld) {
    world.expect_text("Create");
    world.expect_text("Modify");
}
#[then("Parameters remains hidden")]
fn parameters_stays_hidden(world: &mut ApplicationWorld) {
    assert!(
        !parameters_panel_shown(world),
        "expected no Parameters panel"
    );
}
fn parameters_is_hidden(world: &mut ApplicationWorld) {
    // Whether a new part shows the panel is the ribbon's default, not this scenario's subject.
    if parameters_panel_shown(world) {
        expand_show_panels(world);
        toggle_panel(world, PARAMETERS);
        world.escape();
        world.escape();
        world.pump(Vec::new());
    }
    assert!(
        !parameters_panel_shown(world),
        "expected no Parameters panel"
    );
}

#[path = "panel_selector.rs"]
mod panel_selector;
#[path = "selector_keyboard.rs"]
mod selector_keyboard;
#[path = "side_selector.rs"]
pub(super) mod side_selector;

#[then("the other compact group buttons remain visible")]
fn other_compact_groups(world: &mut ApplicationWorld) {
    for panel in ["Sketch", "Create", "Modify"] {
        world.expect_text(panel);
    }
}
#[then("Navigate is unchecked in Show Panels")]
fn navigate_unchecked_in_menu(world: &mut ApplicationWorld) {
    expand_show_panels(world);
    navigate_unchecked(world);
}
