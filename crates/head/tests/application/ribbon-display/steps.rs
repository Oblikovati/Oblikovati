// SPDX-License-Identifier: GPL-2.0-only
use super::{ApplicationWorld, diagnostics::window_geometry};
use cucumber::{given, then, when};
const VIEW_GROUPS: [&str; 4] = ["Visibility", "Appearance", "Windows", "Navigate"];
const WINDOWS_COMMAND: &str = "User Interface";
fn band_height() -> f32 {
    window_geometry("##application-ribbon")
        .expect("ribbon: expected a drawn band")
        .1[1]
}
fn activate_display_control(world: &mut ApplicationWorld) {
    world.ribbon_height = band_height();
    world.click_text("Ribbon Display");
}
fn open_part_on_view(world: &mut ApplicationWorld) {
    use crate::HeadWork;
    world.app().owner.create_part_document().unwrap();
    world.previous_document = world.app().owner.gui_command_state().unwrap().document;
    world.pump(Vec::new());
    world.click_text("View");
}
fn expect_shorter_band(world: &ApplicationWorld) {
    assert!(
        band_height() < world.ribbon_height,
        "ribbon {} after {}: expected a shorter band",
        band_height(),
        world.ribbon_height
    );
}
fn expect_compact_groups(world: &ApplicationWorld) {
    for group in VIEW_GROUPS {
        world.expect_text(group);
    }
    world.expect_no_text(WINDOWS_COMMAND);
}
#[given("the ribbon display control is configured to cycle through all modes")]
fn cycling(world: &mut ApplicationWorld) {
    world.click_text("v");
    assert!(
        world.checked("Cycle through All"),
        "expected the cycling default"
    );
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
}
#[given("the View ribbon tab displays its full groups of commands")]
#[given("the View ribbon is expanded")]
fn view_expanded(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.expect_text(WINDOWS_COMMAND);
}
#[when("I activate the ribbon display control once")]
#[when("I activate the ribbon display control again")]
fn activate(world: &mut ApplicationWorld) {
    activate_display_control(world);
}
#[then("the View ribbon shows compact buttons for Visibility, Appearance, Windows and Navigate")]
fn compact_buttons(world: &mut ApplicationWorld) {
    expect_compact_groups(world);
}
#[then("the viewport has more vertical space")]
fn more_space(world: &mut ApplicationWorld) {
    expect_shorter_band(world);
}
#[when("I select the Windows group button")]
#[when("I select the Windows group title")]
fn open_windows_group(world: &mut ApplicationWorld) {
    world.click_text("Windows");
}
#[then("a temporary panel displays that group's commands")]
#[then("a temporary panel displays the Windows commands")]
fn windows_panel(world: &mut ApplicationWorld) {
    world.expect_text(WINDOWS_COMMAND);
}
#[given("the View ribbon displays compact group buttons")]
fn compact_state(world: &mut ApplicationWorld) {
    open_part_on_view(world);
    activate_display_control(world);
    expect_compact_groups(world);
}
#[then("the View ribbon displays group titles in a narrow strip")]
fn title_strip(world: &mut ApplicationWorld) {
    expect_compact_groups(world);
    expect_shorter_band(world);
}
#[given("the View ribbon displays its group titles")]
fn title_state(world: &mut ApplicationWorld) {
    compact_state(world);
    activate_display_control(world);
    expect_compact_groups(world);
}
#[then("the ribbon shows its tabs without the group strip")]
fn tabs_only(world: &mut ApplicationWorld) {
    world.expect_text("3D Model");
    world.expect_text("View");
    world.expect_no_text("Windows");
    expect_shorter_band(world);
}
#[when("I select the View tab")]
fn select_view_tab(world: &mut ApplicationWorld) {
    world.click_text("View");
}
#[then("the View commands appear in a temporary panel below the tabs")]
fn view_commands_below(world: &mut ApplicationWorld) {
    world.expect_text(WINDOWS_COMMAND);
    let below = band_height();
    assert!(
        world.run(WINDOWS_COMMAND).minimum[1] > below,
        "expected the panel below the tabs"
    );
}
#[when("I click in an empty area of the viewport")]
#[when("I click an empty area of the viewport")]
fn click_viewport(world: &mut ApplicationWorld) {
    let extent = world.extent();
    world.click([extent[0] as i32 - 100, extent[1] as i32 - 100]);
}
#[then("the expanded View ribbon remains visible")]
fn expanded_remains(world: &mut ApplicationWorld) {
    world.expect_text(WINDOWS_COMMAND);
    assert!(
        world.run(WINDOWS_COMMAND).minimum[1] < band_height(),
        "expected the full ribbon"
    );
}
#[when("I open the ribbon display control menu")]
fn open_menu(world: &mut ApplicationWorld) {
    world.click_text("v");
}
#[then("the menu offers minimization to tabs, panel titles and panel buttons")]
fn menu_choices(world: &mut ApplicationWorld) {
    for choice in [
        "Minimize to Tabs",
        "Minimize to Panel Titles",
        "Minimize to Panel Buttons",
    ] {
        world.expect_text(choice);
    }
}
#[then("it offers a cycle-through-all choice")]
fn cycle_choice(world: &mut ApplicationWorld) {
    world.expect_text("Cycle through All");
}
#[when("I choose Minimize to Tabs")]
fn choose_tabs(world: &mut ApplicationWorld) {
    world.click_text("Minimize to Tabs");
}
#[then("the command groups are hidden and the ribbon tabs remain visible")]
fn groups_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Navigate");
    world.expect_text("3D Model");
}
#[when("I select the 3D Model tab")]
fn select_model(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
}
#[then("its command groups appear temporarily below the tabs")]
fn model_commands(world: &mut ApplicationWorld) {
    world.expect_text("Sketch XY");
}
#[then("the temporary command groups close")]
fn groups_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Sketch XY");
}
#[then("the 3D Model tab remains selected")]
fn model_selected(world: &mut ApplicationWorld) {
    // A selected tab has its own fill, so the pixel left of each label tells them apart.
    let selected = world.pixel_left_of("3D Model");
    let other = world.pixel_left_of("View");
    assert_ne!(selected, other, "expected the 3D Model tab to be selected");
}
#[then("Minimize to Tabs is checked")]
fn tabs_checked(world: &mut ApplicationWorld) {
    assert!(
        world.checked("Minimize to Tabs"),
        "expected a checked choice"
    );
}
