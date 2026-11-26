// SPDX-License-Identifier: GPL-2.0-only
use super::{ApplicationWorld, diagnostics::window_geometry};
use cucumber::{given, then, when};
mod compact_options;
#[cfg(target_os = "linux")]
mod context_control;
#[cfg(target_os = "linux")]
mod document_settings;
mod double_click;
mod drawing;
mod home_transition;
#[cfg(target_os = "linux")]
pub(super) mod layout_contract;
#[cfg(target_os = "linux")]
mod manage_insert;
mod measure;
mod mode_policy;
#[cfg(target_os = "linux")]
mod overflow;
mod sketch_exit;
mod tabs_only;
#[cfg(target_os = "linux")]
mod view_compaction;
#[path = "wheel_tools.rs"]
mod wheel_tools;
#[cfg(target_os = "linux")]
mod window_compaction;
use wheel_tools::full_tools;
const VIEW_GROUPS: [&str; 4] = ["Visibility", "Appearance", "Windows", "Navigate"];
const WINDOWS_COMMAND: &str = "User Interface";
fn sketch_visibility_label(world: &mut ApplicationWorld) -> &'static str {
    let sketches_visible = world
        .state()
        .browser
        .is_some_and(|browser| browser.sketches.iter().any(|sketch| sketch.visible));
    if sketches_visible {
        "Hide Sketches"
    } else {
        "Show Sketches"
    }
}

fn band_height() -> f32 {
    window_geometry("##application-ribbon")
        .expect("ribbon: expected a drawn band")
        .1[1]
}
pub(super) fn activate_display_control(world: &mut ApplicationWorld) {
    world.ribbon_height = band_height();
    click_display_control(world);
}
fn open_part_on_view(world: &mut ApplicationWorld) {
    super::workspace_layout::part_open(world);
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
pub(super) fn compact_panel_point(world: &ApplicationWorld, name: &str) -> [i32; 2] {
    super::diagnostics::pointer_pixels(world.run(name).center())
}
pub(super) fn display_control_point(world: &ApplicationWorld) -> [i32; 2] {
    let (position, size) = window_geometry("##application-ribbon").unwrap();
    let expected_x = position[0] + size[0] - 72.;
    let expected_y = world.run("File").center()[1];
    let bounds = (
        [expected_x - 12., expected_y - 13.],
        [expected_x + 12., expected_y + 13.],
    );
    let icons = super::ribbon_appearance::icon_quads::icon_quads_inside(bounds);
    let (min, max) = icons.first().expect("the Ribbon Display icon is visible");
    super::diagnostics::pointer_pixels(std::array::from_fn(|axis| (min[axis] + max[axis]) / 2.))
}
pub(super) fn click_display_control(world: &mut ApplicationWorld) {
    world.click(display_control_point(world));
}
pub(super) fn click_display_menu(world: &mut ApplicationWorld) {
    let [x, y] = display_control_point(world);
    world.click([x + 22, y]);
}
#[given("the ribbon display control is configured to cycle through all modes")]
#[given("the ribbon display control is set to cycle through all modes")]
pub(super) fn cycling(world: &mut ApplicationWorld) {
    click_display_menu(world);
    if !world.checked("Cycle through All") {
        world.click_text("Cycle through All");
    } else {
        world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    }
}
#[then("the full View command groups remain visible")]
fn view_groups_remain(world: &mut ApplicationWorld) {
    world.expect_text(WINDOWS_COMMAND);
}
#[given("the View ribbon is expanded")]
fn view_expanded(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.expect_text(WINDOWS_COMMAND);
}
#[given("the View ribbon tab displays its full groups of commands")]
#[given("the full View ribbon is visible")]
fn view_expanded_with_panel_buttons_policy(world: &mut ApplicationWorld) {
    view_expanded(world);
    click_display_menu(world);
    if world.checked("Cycle through All") {
        world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
        return;
    }
    world.click_text("Minimize to Panel Buttons");
    activate_display_control(world);
    world.expect_text(WINDOWS_COMMAND);
}
#[when("I activate the ribbon display control")]
#[when("the user activates the ribbon display control")]
#[when("I activate the ribbon display control once")]
#[when("the user activates the ribbon display control once")]
#[when("I activate the ribbon display control again")]
#[when("the user activates the ribbon display control again")]
fn activate(world: &mut ApplicationWorld) {
    activate_display_control(world);
}
#[then("the View groups are shown as compact buttons")]
#[then("the View ribbon displays compact group buttons")]
#[then("the ribbon displays compact group buttons")]
#[then("the View ribbon shows compact buttons for Visibility, Appearance, Windows and Navigate")]
fn compact_buttons(world: &mut ApplicationWorld) {
    expect_compact_groups(world);
}
#[then("the viewport has more vertical space")]
fn more_space(world: &mut ApplicationWorld) {
    expect_shorter_band(world);
}
#[when("I select the Windows group button")]
#[when("the user selects the Windows group button")]
#[when("I select the Windows group title")]
#[when("the user selects the Windows group title")]
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
#[then("the View group titles appear in a narrow strip")]
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
#[then("the tabs remain visible without a permanent group strip")]
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
#[then("its commands appear in a temporary panel")]
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
#[when("the user clicks in an empty area of the viewport")]
#[when("I click an empty area of the viewport")]
fn click_viewport(world: &mut ApplicationWorld) {
    let extent = world.extent();
    world.click([extent[0] as i32 - 100, extent[1] as i32 - 100]);
}
#[then("the expanded View ribbon remains visible")]
#[then("the expanded View ribbon is restored")]
#[then("the full View ribbon remains visible")]
#[then("the full View ribbon is restored")]
fn expanded_remains(world: &mut ApplicationWorld) {
    world.expect_text(WINDOWS_COMMAND);
    assert!(
        world.run(WINDOWS_COMMAND).minimum[1] < band_height(),
        "expected the full ribbon"
    );
}
#[when("the user dismisses the menu with Escape")]
fn dismiss_display_menu(world: &mut ApplicationWorld) {
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    world.expect_no_text("Minimize to Panel Buttons");
}
#[then("the temporary Windows panel closes")]
fn windows_panel_closed(world: &mut ApplicationWorld) {
    assert!(window_geometry("##ribbon-temporary-panel").is_none());
    world.expect_text(WINDOWS_COMMAND);
}
#[when("I open the ribbon display control menu")]
#[when("the user opens the ribbon display control menu")]
#[when("the user opens the ribbon display menu")]
#[when("the user opens the ribbon display menu again")]
fn open_menu(world: &mut ApplicationWorld) {
    click_display_menu(world);
}
#[when("the user chooses Minimize to Panel Buttons from the ribbon display menu")]
fn choose_panel_buttons_from_menu(world: &mut ApplicationWorld) {
    click_display_menu(world);
    world.click_text("Minimize to Panel Buttons");
}
#[when("the user activates the ribbon display control twice")]
fn activate_display_twice(world: &mut ApplicationWorld) {
    activate_display_control(world);
    activate_display_control(world);
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
#[when("the user chooses Minimize to Tabs")]
fn choose_tabs(world: &mut ApplicationWorld) {
    world.click_text("Minimize to Tabs");
}
#[then("the command groups are hidden and the ribbon tabs remain visible")]
fn groups_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Navigate");
    world.expect_text("3D Model");
}
#[when("I select the 3D Model tab")]
#[when("the user selects the 3D Model tab")]
fn select_model(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
}
#[then("its command groups appear temporarily below the tabs")]
fn model_commands(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
}
#[then("the temporary command groups close")]
fn groups_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Start 2D Sketch");
    assert!(window_geometry("##ribbon-temporary-panel").is_none());
}
#[then("the 3D Model tab remains selected")]
fn model_selected(world: &mut ApplicationWorld) {
    let tabs = ["3DModel", "Tools", "View"];
    assert_eq!(
        ribbon_tab_underlines(world, tabs),
        [true, false, false],
        "active underline for {tabs:?}"
    );
    world.save_capture("3d-model-tab-selected");
}

fn ribbon_tab_underlines<const COUNT: usize>(
    world: &mut ApplicationWorld,
    tabs: [&str; COUNT],
) -> [bool; COUNT] {
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let [width, height] = capture.extent.map(|pixels| pixels as usize);
    let tab_strip = super::diagnostics::child_scroll("##application-ribbon", "##ribbon-tab-region")
        .expect("expected the ribbon tab strip");
    // Commands below the tab strip can also use the accent.
    let bottom = super::diagnostics::pointer_pixels([0., tab_strip.origin[1] + tab_strip.size[1]])
        [1] as usize;
    // The live GUI context provides the accent for both light and dark themes.
    let accent =
        unsafe { imgui::sys::igGetColorU32_Col(imgui::sys::ImGuiCol_NavHighlight as i32, 1.) }
            .to_le_bytes();
    tabs.map(|label| {
        let label = super::world::squeezed(label);
        let run = world
            .frame
            .runs
            .iter()
            .find(|run| run.list == tab_strip.list && super::world::squeezed(&run.text) == label)
            .unwrap_or_else(|| panic!("expected {label:?} in the ribbon tab strip"));
        let [x, top] = super::diagnostics::pointer_pixels([run.center()[0], run.maximum[1]])
            .map(|pixel| pixel as usize);
        (top..bottom.min(height)).any(|y| {
            let at = (y * width + x) * 4;
            capture.rgba.get(at..at + 3) == Some(&accent[..3])
        })
    })
}
#[then("Minimize to Tabs is checked")]
fn tabs_checked(world: &mut ApplicationWorld) {
    assert!(
        world.checked("Minimize to Tabs"),
        "expected a checked choice"
    );
}
#[when("I select the Annotate tab")]
#[when("the user selects the Annotate tab")]
fn select_annotate(world: &mut ApplicationWorld) {
    world.click_text("Annotate");
    world.pump(Vec::new());
}
#[then("the temporary Create panel closes")]
fn temporary_create_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Base View");
}
#[then("the ribbon displays compact Annotate group buttons")]
fn compact_annotate(world: &mut ApplicationWorld) {
    world.expect_no_text("Baseline");
    world.expect_no_text("Sheets");
}
#[then("those buttons include Dimension, Text and Symbols")]
#[then("it includes Dimension, Text and Symbols commands")]
fn annotate_groups(world: &mut ApplicationWorld) {
    for group in ["Dimension", "Text", "Symbols"] {
        world.expect_text(group);
    }
}
#[then("the drawing remains active")]
fn drawing_stays(world: &mut ApplicationWorld) {
    drawing::drawing_remains(world);
}
#[given("the Annotate ribbon is expanded")]
fn annotate_expanded(world: &mut ApplicationWorld) {
    select_annotate(world);
    world.expect_text("Baseline");
    world.ribbon_height = band_height();
}
#[when("I choose Minimize to Panel Titles")]
#[when("the user chooses Minimize to Panel Titles")]
#[when("the user selects Minimize to Panel Titles")]
fn minimize_to_titles(world: &mut ApplicationWorld) {
    world.click_text("Minimize to Panel Titles");
}
fn expect_annotate_titles(world: &mut ApplicationWorld) {
    annotate_groups(world);
    world.expect_no_text("Baseline");
    assert!(
        band_height() < world.ribbon_height,
        "expected a narrower ribbon"
    );
}
#[then("the Annotate command groups are reduced to a narrow strip of titles")]
#[then("the ribbon displays the group-title strip")]
fn annotate_titles(world: &mut ApplicationWorld) {
    expect_annotate_titles(world);
}
#[then("the full Annotate command groups are visible")]
#[then("the full Annotate ribbon is visible")]
fn full_annotate(world: &mut ApplicationWorld) {
    world.expect_text("Baseline");
    world.expect_text("Leader");
}
#[then("Minimize to Panel Titles is checked")]
#[then("Minimize to Panel Titles is selected")]
fn titles_checked(world: &mut ApplicationWorld) {
    assert!(
        world.checked("Minimize to Panel Titles"),
        "expected the chosen mode checked"
    );
}
#[when("I choose Minimize to Panel Buttons")]
#[when("the user chooses Minimize to Panel Buttons")]
#[when("the user selects Minimize to Panel Buttons")]
fn minimize_to_buttons(world: &mut ApplicationWorld) {
    world.click_text("Minimize to Panel Buttons");
}
#[then("the Annotate command groups appear as compact buttons")]
fn annotate_buttons(world: &mut ApplicationWorld) {
    annotate_groups(world);
    world.expect_no_text("Baseline");
}
#[then("Minimize to Panel Buttons is checked")]
#[then("Minimize to Panel Buttons is selected")]
fn buttons_checked(world: &mut ApplicationWorld) {
    assert!(
        world.checked("Minimize to Panel Buttons"),
        "expected the chosen mode checked"
    );
}
fn choose_tabs_mode(world: &mut ApplicationWorld) {
    click_display_menu(world);
    world.click_text("Minimize to Tabs");
    world.expect_no_text("Start 2D Sketch");
}
#[given("the ribbon displays tabs without permanent command groups")]
#[given("Minimize to Tabs is the selected ribbon display mode")]
fn tabs_only_mode(world: &mut ApplicationWorld) {
    choose_tabs_mode(world);
}
#[when("the pointer rests over the ribbon display control")]
fn rest_on_control(world: &mut ApplicationWorld) {
    let at = display_control_point(world);
    world.hover(at);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("a tooltip identifies the action as Show Full Ribbon")]
fn full_ribbon_tooltip(world: &mut ApplicationWorld) {
    world.expect_text("Show Full Ribbon");
}
#[then("the part viewport remains visible")]
fn viewport_visible(world: &mut ApplicationWorld) {
    world.expect_blank_region([400, 1400], [250, 650]);
}
#[given("the 3D Model ribbon is expanded")]
fn model_ribbon_expanded(world: &mut ApplicationWorld) {
    // After Minimize to Tabs, the control expands it; a full ribbon already is.
    if !world.shows("Start 2D Sketch") {
        activate_display_control(world);
    }
    world.expect_text("Start 2D Sketch");
}
#[then("the permanent command groups disappear and the ribbon tabs remain visible")]
#[then("the ribbon is minimized to tabs")]
#[then("only the ribbon tabs remain visible")]
fn groups_gone(world: &mut ApplicationWorld) {
    world.expect_no_text("Start 2D Sketch");
    world.expect_text("3D Model");
}
#[then("the full 3D Model ribbon is displayed")]
#[then("the full 3D Model command groups are visible")]
#[then("the full 3D Model ribbon is visible")]
#[given("the full 3D Model ribbon is visible")]
fn model_groups_visible(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
    world.expect_text("Extrude");
}
#[then("the ribbon returns to tabs-only display")]
#[then("the ribbon remains in tabs-only display")]
fn back_to_tabs_only(world: &mut ApplicationWorld) {
    groups_gone(world);
}
