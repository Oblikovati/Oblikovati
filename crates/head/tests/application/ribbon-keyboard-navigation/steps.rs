// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard access to the document ribbon through its native window.
use super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;

fn exact_tip(world: &ApplicationWorld, tip: &str) -> bool {
    world.frame.runs.iter().any(|run| run.text == tip)
}
#[when("the user presses Alt")]
fn alt(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::LeftAlt, false);
}
#[then("key tips appear on the ribbon tabs and quick-access controls")]
#[then("top-level key tips appear on the ribbon tabs and quick-access controls")]
#[then("top-level key tips appear")]
fn tips_shown(world: &mut ApplicationWorld) {
    assert!(
        exact_tip(world, "M"),
        "missing Model key tip: {:?}",
        world.text()
    );
    assert!(
        exact_tip(world, "1"),
        "missing quick-access key tip: {:?}",
        world.text()
    );
}
#[given("top-level ribbon key tips are visible after pressing Alt")]
fn top_level_tips_active(world: &mut ApplicationWorld) {
    alt(world);
    tips_shown(world);
}
#[then("the key tips disappear")]
#[then("the top-level key tips disappear")]
fn tips_hidden(world: &mut ApplicationWorld) {
    assert!(
        !exact_tip(world, "M"),
        "key tips still visible: {:?}",
        world.text()
    );
}

#[cfg(target_os = "linux")]
#[test]
fn native_side_docked_tab_and_quick_access_tips_toggle() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    super::ribbon_docking::docked_left_on_model(&mut world);
    super::x11_input::press_keysym(0xffe9);
    world.pump(Vec::new());
    tips_shown(&mut world);
    super::x11_input::press_keysym(0xffe9);
    world.pump(Vec::new());
    tips_hidden(&mut world);
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn native_escape_closes_side_docked_group_after_key_tips() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    super::ribbon_docking::docked_left_on_model(&mut world);
    for key in [0xffe9, u32::from(b'm'), u32::from(b'y')] {
        super::x11_input::press_keysym(key);
        world.pump(Vec::new());
    }
    modify_panel_and_tips(&mut world);
    super::x11_input::press_keysym(0xffe9);
    world.pump(Vec::new());
    nested_tips_hidden(&mut world);
    super::x11_input::press_escape();
    world.pump(Vec::new());
    assert!(super::diagnostics::window_geometry("##ribbon-temporary-panel").is_none());
    world.close();
}
#[given("the Tools ribbon tab is selected")]
fn select_tools(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.expect_text("Application Options");
    if super::diagnostics::window_geometry("##ribbon-temporary-panel").is_some() {
        world.click([1000, 600]);
        world.expect_no_text("Application Options");
    }
}
#[when("the user presses \"Z\"")]
fn unknown_tip(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Z, false);
}
#[then("the top-level key tips remain visible")]
#[then("the top-level ribbon key tips remain visible")]
fn tips_remain(world: &mut ApplicationWorld) {
    tips_shown(world);
}
#[then("the Tools ribbon remains selected")]
fn tools_remain(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
}
#[then("the 3D Model tab displays the key tip \"M\"")]
fn model_tip(world: &mut ApplicationWorld) {
    assert!(exact_tip(world, "M"));
}
#[when("the user presses \"M\"")]
fn model_key(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
}
#[then("the 3D Model ribbon is selected")]
#[then("the 3D Model ribbon tab is selected")]
fn model_selected(world: &mut ApplicationWorld) {
    world.expect_text("Extrude");
}
#[then("its command key tips are displayed")]
#[then("key tips appear on its commands")]
#[then("the 3D Model command key tips appear")]
#[then("the 3D Model command key tips appear again")]
#[then("the command key tips remain visible")]
#[then("command key tips are visible")]
#[then("the Hole command displays the key tip \"HO\"")]
fn command_tips(world: &mut ApplicationWorld) {
    assert!(exact_tip(world, "HO"), "{:?}", world.text());
}
#[given("the 3D Model ribbon command key tips are displayed")]
fn model_command_tips_active(world: &mut ApplicationWorld) {
    show_model_tips(world);
    command_tips(world);
}
#[given("the Hole command has the key tip \"HO\"")]
fn hole_key_tip_visible(world: &mut ApplicationWorld) {
    command_tips(world);
}
#[when("the user presses Alt and then \"M\"")]
fn show_model_tips(world: &mut ApplicationWorld) {
    alt(world);
    model_key(world);
}
#[when("the user presses \"H\"")]
fn prefix_h(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::H, false);
}
#[then("the visible command key tips narrow to H-prefixed choices")]
#[then("the visible command key tips narrow to H-prefixed entries")]
#[then("the visible command key tips are narrowed to matching H-prefixed entries")]
#[then("the H-prefixed command key tips remain visible")]
#[then("the displayed command key tips narrow to matching H-prefixed entries")]
fn narrowed_tips(world: &mut ApplicationWorld) {
    assert!(exact_tip(world, "HO"));
    assert!(!exact_tip(world, "SK") && !exact_tip(world, "EX"));
}
#[then("the temporary 3D Model command area opens with command key tips")]
fn temporary_model_command_tips(world: &mut ApplicationWorld) {
    temporary_model(world);
    command_tips(world);
}
#[then("the tab and quick-access key tips appear")]
fn tabs_return(world: &mut ApplicationWorld) {
    tips_shown(world);
}
#[then("the command key tips are no longer displayed")]
fn commands_hidden(world: &mut ApplicationWorld) {
    assert!(!exact_tip(world, "HO"));
}
#[when("the user clicks the Tools ribbon tab")]
fn mouse_tools(world: &mut ApplicationWorld) {
    world.click_text("Tools");
}
#[then("the Tools ribbon is selected and its commands are visible")]
fn tools_visible(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
}
#[test]
fn model_key_tips_start_the_native_sketch_tool() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    alt(&mut world);
    model_key(&mut world);
    assert!(exact_tip(&world, "SK"));
    world.key(ViewportPhysicalKey::S, false);
    world.key(ViewportPhysicalKey::K, false);
    world.expect_text("XY Plane");
    world.click_text("XY Plane");
    world.expect_text("Finish Sketch");
    world.draw_rectangle();
    world.finish_sketch();
    alt(&mut world);
    model_key(&mut world);
    world.key(ViewportPhysicalKey::E, false);
    world.key(ViewportPhysicalKey::X, false);
    assert!(super::diagnostics::window_geometry("Extrude").is_some());
    world.escape();
    world.close();
}
#[test]
fn display_key_tips_cycle_and_open_the_policy_menu() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    alt(&mut world);
    world.key(ViewportPhysicalKey::X, false);
    world.key(ViewportPhysicalKey::Alpha2, false);
    world.expect_no_text("Start 2D Sketch");
    alt(&mut world);
    world.key(ViewportPhysicalKey::X, false);
    world.key(ViewportPhysicalKey::Alpha3, false);
    world.expect_text("Minimize to Tabs");
    world.key(ViewportPhysicalKey::DownArrow, false);
    world.key(ViewportPhysicalKey::Enter, false);
    let (_, size) = super::diagnostics::window_geometry("##application-ribbon").unwrap();
    assert!(
        size[1] < 100.,
        "expected tabs-only ribbon, height {}",
        size[1]
    );
    world.close();
}
#[given("the Tools ribbon is fully displayed with \"Cycle Through All\" selected")]
fn tools_full(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.expect_text("Application Options");
    display_tip(world, ViewportPhysicalKey::Alpha3);
    world.click_text("Cycle through All");
    display_tip(world, ViewportPhysicalKey::Alpha3);
    assert!(world.checked("Cycle through All"));
    world.escape();
}
fn display_tip(world: &mut ApplicationWorld, key: ViewportPhysicalKey) {
    alt(world);
    world.key(ViewportPhysicalKey::X, false);
    world.key(key, false);
}
#[when("the user presses Alt and enters the display control key tip \"X2\"")]
#[when("the user presses Alt and enters \"X2\"")]
fn display_cycle(world: &mut ApplicationWorld) {
    display_tip(world, ViewportPhysicalKey::Alpha2);
}
#[when("the user presses Alt and enters the display menu key tip \"X3\"")]
#[when("the user reopens the display policy menu with Alt and \"X3\"")]
fn display_menu(world: &mut ApplicationWorld) {
    display_tip(world, ViewportPhysicalKey::Alpha3);
}
#[then("the ribbon displays compact panel buttons")]
fn compact(world: &mut ApplicationWorld) {
    world.expect_no_text("Application Options");
    world.expect_text("Options");
}
#[then("the ribbon display policy menu opens")]
fn policy_open(world: &mut ApplicationWorld) {
    world.expect_text("Minimize to Tabs");
}
#[when("the user presses Down and Enter")]
fn first_policy(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::DownArrow, false);
    world.key(ViewportPhysicalKey::Enter, false);
}
#[then("the ribbon displays tabs only")]
fn tabs_only(_: &mut ApplicationWorld) {
    let (_, size) = super::diagnostics::window_geometry("##application-ribbon").unwrap();
    assert!(
        size[1] < 100.,
        "expected tabs-only ribbon, height {}",
        size[1]
    );
}
#[then("\"Minimize to Tabs\" is marked as selected")]
fn tabs_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Minimize to Tabs"));
}
#[when("the user presses Down four times and Enter")]
fn all_cycle_policy(world: &mut ApplicationWorld) {
    for _ in 0..4 {
        world.key(ViewportPhysicalKey::DownArrow, false);
    }
    world.key(ViewportPhysicalKey::Enter, false);
}
#[test]
fn dismissing_key_tips_does_not_cancel_the_active_sketch_command() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.select_line_tool();
    world.sketch_click([850, 500]);
    world.expect_status("Select end of line");
    alt(&mut world);
    world.escape();
    world.expect_status("Select end of line");
    world.escape();
    world.expect_ready();
    world.close();
}
#[given("the full View ribbon is selected")]
fn view_selected(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.expect_text("Appearance");
}
#[when("the user presses \"I\"")]
fn unmatched_view_tip(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::I, false);
}
#[then("the View ribbon remains selected")]
#[then("the View ribbon and part remain visible")]
fn view_remains(world: &mut ApplicationWorld) {
    world.expect_text("Appearance");
    assert!(world.state().document.is_some());
}
#[test]
fn quick_access_key_tip_opens_new_document_chooser() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    alt(&mut world);
    world.key(ViewportPhysicalKey::Alpha1, false);
    world.expect_text("Template folders");
    world.escape();
    world.close();
}
#[test]
fn scaled_ribbon_key_tips_start_the_sketch_tool() {
    use crate::HeadWork;
    let mut world = ApplicationWorld::new();
    world.add_part();
    let scale = oblikovati_model::ui_scale::UiScaleOptions::new(1.5, 1.).unwrap();
    world.app().owner.set_ui_scale_from_gui(scale).unwrap();
    world.pump(Vec::new());
    alt(&mut world);
    assert!(exact_tip(&world, "M"));
    model_key(&mut world);
    assert!(exact_tip(&world, "SK"));
    world.key(ViewportPhysicalKey::S, false);
    world.key(ViewportPhysicalKey::K, false);
    world.expect_text("XY Plane");
    world.escape();
    world.close();
}

#[then("the temporary 3D Model command area opens")]
#[then("the temporary 3D Model command area remains visible")]
fn temporary_model(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
    assert!(super::diagnostics::window_geometry("##ribbon-temporary-panel").is_some());
}
#[then("the temporary command area closes leaving the ribbon minimized to tabs")]
#[then("the temporary command area closes and the ribbon remains minimized to tabs")]
fn temporary_closed(world: &mut ApplicationWorld) {
    assert!(super::diagnostics::window_geometry("##ribbon-temporary-panel").is_none());
    world.expect_no_text("Start 2D Sketch");
    world.expect_text("3D Model");
}

#[given("Modify is displayed as a compact group button")]
fn modify_compact(world: &mut ApplicationWorld) {
    world.run_in_window("##ribbon-column", "Modify");
}
#[then("key tips appear on the visible 3D Model commands and compact groups")]
#[then("the Modify group displays the key tip \"Y\"")]
fn modify_group_key_tip(world: &mut ApplicationWorld) {
    assert!(
        exact_tip(world, "Y"),
        "missing Modify key tip: {}",
        world.text()
    );
}
#[when("the user presses \"Y\"")]
fn modify_group_key(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Y, false);
}
#[then("the adjacent Modify command panel opens")]
#[then("its commands display key tips including \"HO\" for Hole")]
fn modify_group_command_tips(world: &mut ApplicationWorld) {
    modify_panel_and_tips(world);
}
#[given(
    "the user has pressed Alt, \"M\", and \"Y\" to open the Modify panel and its command key tips"
)]
fn open_modify_with_keys(world: &mut ApplicationWorld) {
    show_model_tips(world);
    world.key(ViewportPhysicalKey::Y, false);
    modify_panel_and_tips(world);
}
#[then("the Modify panel and its command key tips remain visible")]
fn modify_panel_and_tips(world: &mut ApplicationWorld) {
    world.expect_text("Fillet");
    assert!(super::diagnostics::window_geometry("##ribbon-temporary-panel").is_some());
    assert!(exact_tip(world, "HO"));
}
#[then("the command key tips disappear")]
fn nested_tips_hidden(world: &mut ApplicationWorld) {
    assert!(!exact_tip(world, "HO"));
    assert!(super::diagnostics::window_geometry("##ribbon-temporary-panel").is_some());
    world.expect_text("Fillet");
}

#[test]
fn side_docked_group_tips_start_sketch_and_close_the_group() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    super::ribbon_docking::docked_left_on_model(&mut world);
    show_model_tips(&mut world);
    assert!(exact_tip(&world, "K"));
    world.key(ViewportPhysicalKey::K, false);
    assert!(exact_tip(&world, "SK"));
    world.key(ViewportPhysicalKey::S, false);
    world.key(ViewportPhysicalKey::K, false);
    world.expect_text("XY Plane");
    assert!(super::diagnostics::window_geometry("##ribbon-temporary-panel").is_none());
    world.click_text("XY Plane");
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    world.close();
}
#[test]
fn compact_group_tips_open_the_sketch_command() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    super::ribbon_display::cycling(&mut world);
    super::ribbon_display::click_display_control(&mut world);
    show_model_tips(&mut world);
    assert!(exact_tip(&world, "K"));
    world.key(ViewportPhysicalKey::K, false);
    assert!(exact_tip(&world, "SK"));
    world.key(ViewportPhysicalKey::S, false);
    world.key(ViewportPhysicalKey::K, false);
    world.expect_text("XY Plane");
    assert!(super::diagnostics::window_geometry("##ribbon-temporary-panel").is_none());
    world.click_text("XY Plane");
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    world.close();
}
#[test]
fn compact_create_group_tips_open_extrude() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.draw_rectangle();
    world.finish_sketch();
    super::ribbon_display::cycling(&mut world);
    super::ribbon_display::click_display_control(&mut world);
    show_model_tips(&mut world);
    assert!(exact_tip(&world, "C"));
    world.key(ViewportPhysicalKey::C, false);
    assert!(exact_tip(&world, "EX"));
    world.key(ViewportPhysicalKey::E, false);
    world.key(ViewportPhysicalKey::X, false);
    assert!(super::diagnostics::window_geometry("Extrude").is_some());
    assert!(super::diagnostics::window_geometry("##ribbon-temporary-panel").is_none());
    world.escape();
    world.close();
}
#[path = "view_rotation.rs"]
mod view_rotation;

#[path = "browser_navigation.rs"]
mod browser_navigation;
#[path = "face_selection.rs"]
mod face_selection;
pub(super) use face_selection::visible_face_point;

#[when("the user presses \"O\"")]
fn hole_key(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::O, false);
}
#[then("the Hole properties panel opens")]
fn hole_panel_open(world: &mut ApplicationWorld) {
    assert!(super::diagnostics::window_geometry("Hole").is_some());
    world.run_in_window("Hole", "Diameter");
    let count = world.state().browser.as_ref().unwrap().features.len();
    world.observed_counts.push(count);
}
#[when("the user cancels the Hole command")]
#[when("the user clicks Cancel in the Hole properties panel")]
fn cancel_hole(world: &mut ApplicationWorld) {
    let cancel = world.run_in_window("Hole", "Cancel");
    world.click(cancel.center().map(|value| value as i32));
}
#[then("the properties panel closes without creating a hole")]
#[then("the properties panel closes")]
fn hole_panel_closed(world: &mut ApplicationWorld) {
    assert!(super::diagnostics::window_geometry("Hole").is_none());
    let count = world.state().browser.as_ref().unwrap().features.len();
    assert_eq!(Some(&count), world.observed_counts.last());
}

#[then("the ribbon key tips and adjacent Modify panel disappear")]
fn nested_hole_tips_closed(world: &mut ApplicationWorld) {
    assert!(!exact_tip(world, "HO"));
    assert!(super::diagnostics::window_geometry("##ribbon-temporary-panel").is_none());
}

#[cfg(target_os = "linux")]
#[path = "native_hole.rs"]
mod native_hole;
