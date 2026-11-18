// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard access to full-width ribbon expansions (#3856).
use super::{ApplicationWorld, diagnostics};
use cucumber::{then, when};
use oblikovati_viewport::ViewportPhysicalKey;

#[when("the user presses Alt, then M, then Y")]
fn open_modify_by_keys(world: &mut ApplicationWorld) {
    for key in [
        ViewportPhysicalKey::LeftAlt,
        ViewportPhysicalKey::M,
        ViewportPhysicalKey::Y,
    ] {
        world.key(key, false);
    }
}
#[then("the Modify additional command area opens below the ribbon over the browser area")]
fn keyboard_expansion_placement(_: &mut ApplicationWorld) {
    let (panel, _) =
        diagnostics::window_geometry("##ribbon-additional-PartModify").expect("Modify expansion");
    let (ribbon, size) = diagnostics::window_geometry("##application-ribbon").unwrap();
    assert!(panel[1] >= ribbon[1] + size[1]);
    let (browser, browser_size) = diagnostics::window_geometry("Model").expect("model browser");
    assert!(
        panel[0] < browser[0] + browser_size[0],
        "expansion {panel:?}, browser {browser:?} {browser_size:?}: expected overlap"
    );
}
#[then("the solid body and model browser remain visible")]
fn body_and_browser(world: &mut ApplicationWorld) {
    super::expect_solid_visible(world);
    world.expect_text("Extrude1");
}
#[then("the additional command area closes and keyboard tips disappear")]
fn expansion_and_tips_close(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_none());
    assert!(!world.frame.runs.iter().any(|run| run.text == "HO"));
}

#[cfg(target_os = "linux")]
#[test]
fn x11_keyboard_opens_the_full_modify_expansion_over_the_browser() {
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    for key in [0xffe9, u32::from(b'm'), u32::from(b'y')] {
        super::super::x11_input::press_keysym(key);
        world.pump(Vec::new());
    }
    keyboard_expansion_placement(&mut world);
    let title = world
        .run_in_window("##ribbon-additional-PartModify", "Modify")
        .center();
    super::super::x11_input::click(diagnostics::pointer_pixels(title));
    world.pump(Vec::new());
    expansion_and_tips_close(&mut world);
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn x11_move_body_tip_opens_and_cancels_the_dialog() {
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    for key in [
        0xffe9,
        u32::from(b'm'),
        u32::from(b'y'),
        u32::from(b'o'),
        u32::from(b'd'),
    ] {
        super::super::x11_input::press_keysym(key);
        world.pump(Vec::new());
    }
    assert!(diagnostics::window_geometry("Move Bodies").is_some());
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_none());
    world.click_text("Cancel");
    assert!(diagnostics::window_geometry("Move Bodies").is_none());
    super::expect_solid_visible(&mut world);
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn x11_pinned_move_body_cancel_keeps_the_expansion() {
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    super::opened_unpinned(&mut world);
    super::toggle_pin(&mut world);
    let button = world
        .run_in_window("##ribbon-additional-PartModify", "MoveBodies")
        .center();
    super::super::x11_input::click(diagnostics::pointer_pixels(button));
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Move Bodies").is_some());
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_some());
    world.click_text("Cancel");
    assert!(diagnostics::window_geometry("Move Bodies").is_none());
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_some());
    super::expect_solid_visible(&mut world);
    world.close();
}

#[when("the user enters \"OD\"")]
fn enter_move_body_tip(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::O, false);
    world.key(ViewportPhysicalKey::D, false);
}
#[then("the Modify additional command area opens with key tips for its commands")]
fn modify_tips_visible(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_some());
    world.expect_text("OD");
}
#[then("the Move Bodies dialog opens")]
fn motion_dialog_visible(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Move Bodies").is_some());
}
#[then("the part remains visible without applying the preview")]
#[then("the dialog closes without applying the move")]
fn motion_cancelled(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Move Bodies").is_none());
    super::expect_solid_visible(world);
    world.expect_no_text("Move1");
}
#[when("the user selects \"Move Bodies\" from the additional command area")]
fn select_move_body(world: &mut ApplicationWorld) {
    let button = world
        .run_in_window("##ribbon-additional-PartModify", "MoveBodies")
        .center();
    world.click(diagnostics::pointer_pixels(button));
}
#[when("the user cancels the Move Bodies dialog")]
fn cancel_motion(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
}

#[cucumber::given("the full ribbon is displayed")]
fn full_ribbon(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
}
#[then("the 3D Model ribbon displays command key tips")]
fn model_tips(world: &mut ApplicationWorld) {
    world.expect_text("HO");
}

#[cucumber::given(
    "the 3D Model Modify additional command area is open using Alt, \"M\", and \"Y\""
)]
fn keyboard_open_modify(world: &mut ApplicationWorld) {
    open_modify_by_keys(world);
    modify_tips_visible(world);
}
#[cucumber::given("its command key tips are visible")]
fn expansion_tips_visible(world: &mut ApplicationWorld) {
    modify_tips_visible(world);
}
#[then("the additional command area and its key tips remain visible")]
fn expansion_tips_remain(world: &mut ApplicationWorld) {
    modify_tips_visible(world);
}
#[then("the additional command area and key tips close")]
fn expansion_tips_closed(world: &mut ApplicationWorld) {
    expansion_and_tips_close(world);
    assert!(!world.frame.runs.iter().any(|run| run.text == "OD"));
}

#[cfg(target_os = "linux")]
#[test]
fn move_body_context_placement_round_trips_and_keeps_command_live() {
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    super::opened_unpinned(&mut world);
    let row = world
        .run_in_window("##ribbon-additional-PartModify", "MoveBodies")
        .center();
    world.right_click(diagnostics::pointer_pixels(row));
    world.click_text("Move to Main Panel");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_none());
    let main = world
        .run_in_window("##application-ribbon", "MoveBodies")
        .center();
    world.click_text("Tools");
    world.click_text("3D Model");
    let returned = world
        .run_in_window("##application-ribbon", "MoveBodies")
        .center();
    assert_eq!(main, returned);
    world.click(diagnostics::pointer_pixels(returned));
    assert!(diagnostics::window_geometry("Move Bodies").is_some());
    world.click_text("Cancel");
    let main = world
        .run_in_window("##application-ribbon", "MoveBodies")
        .center();
    world.right_click(diagnostics::pointer_pixels(main));
    world.click_text("Move to Expanded Panel");
    world.pump(Vec::new());
    super::opened_unpinned(&mut world);
    world.run_in_window("##ribbon-additional-PartModify", "MoveBodies");
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn move_body_context_adds_a_working_quick_access_command() {
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    super::opened_unpinned(&mut world);
    let row = world
        .run_in_window("##ribbon-additional-PartModify", "MoveBodies")
        .center();
    world.right_click(diagnostics::pointer_pixels(row));
    world.click_text("Add to Quick Access Toolbar");
    world.pump(Vec::new());
    world.click_text("Tools");
    world.click_text("3D Model");
    let shortcut = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "MoveBodies" && run.center()[1] < 34.)
        .expect("quick access Move Bodies")
        .center();
    world.click(diagnostics::pointer_pixels(shortcut));
    assert!(diagnostics::window_geometry("Move Bodies").is_some());
    world.click_text("Cancel");
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn move_body_context_menu_accepts_keyboard_selection() {
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    super::opened_unpinned(&mut world);
    let row = world
        .run_in_window("##ribbon-additional-PartModify", "MoveBodies")
        .center();
    super::super::x11_input::right_click(diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    world.expect_text("Move to Main Panel");
    for key in [0xff54, 0xff54, 0xff0d] {
        super::super::x11_input::press_keysym(key);
        world.pump(Vec::new());
    }
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_none());
    let main = world
        .run_in_window("##application-ribbon", "MoveBodies")
        .center();
    super::super::x11_input::right_click(diagnostics::pointer_pixels(main));
    world.pump(Vec::new());
    world.expect_text("Move to Expanded Panel");
    super::super::x11_input::press_keysym(0xff0d);
    world.pump(Vec::new());
    super::opened_unpinned(&mut world);
    world.run_in_window("##ribbon-additional-PartModify", "MoveBodies");
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn move_body_context_adds_quick_access_with_keyboard() {
    let mut world = ApplicationWorld::new();
    super::solid_part(&mut world);
    super::opened_unpinned(&mut world);
    let row = world
        .run_in_window("##ribbon-additional-PartModify", "MoveBodies")
        .center();
    super::super::x11_input::right_click(diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    world.expect_text("Add to Quick Access Toolbar");
    for key in [0xff54, 0xff0d] {
        super::super::x11_input::press_keysym(key);
        world.pump(Vec::new());
    }
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| run.text == "MoveBodies" && run.center()[1] < 34.)
    );
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_some());
    world.close();
}
