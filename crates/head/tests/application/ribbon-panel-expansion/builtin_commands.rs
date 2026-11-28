// SPDX-License-Identifier: GPL-2.0-only
//! Move an unavailable built-in command through the native ribbon.
use super::*;

const CLIPBOARD: &str = "##ribbon-additional-ToolClipboard";
#[given("the full Tools ribbon shows an unavailable Copy command in the Clipboard main panel")]
fn clipboard_main(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.expect_text("Clipboard");
    assert!(!world.run_is_enabled("Copy"));
}
#[when("the user right-clicks Copy and selects \"Move to Expanded Panel\"")]
fn move_copy_out(world: &mut ApplicationWorld) {
    right_click_copy(world);
    world.click_text("Move to Expanded Panel");
}
#[then("Copy is no longer visible in the Clipboard main panel")]
fn copy_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Copy");
}
#[then("the Clipboard panel has an expansion arrow")]
fn clipboard_arrow(world: &mut ApplicationWorld) {
    let title = world.run("Clipboard");
    assert!(
        title.text.ends_with(['v', '▼'])
            || world.frame.runs.iter().any(|run| {
                matches!(run.text.as_str(), "v" | "▼")
                    && run.minimum[0] > title.maximum[0]
                    && (run.center()[1] - title.center()[1]).abs() < 8.
            })
    );
}
#[when("the user opens the Clipboard expanded panel")]
fn open_clipboard(world: &mut ApplicationWorld) {
    click_expansion_arrow(world, "Clipboard");
    assert!(diagnostics::window_geometry(CLIPBOARD).is_some());
    world.save_capture("clipboard-expanded");
}
#[then("Copy appears there as an unavailable command")]
#[then("the unavailable Copy command appears in the Clipboard main panel again")]
fn copy_unavailable(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("Copy"));
}
#[when("the user clicks Copy in the expanded panel")]
fn click_unavailable_copy(world: &mut ApplicationWorld) {
    world.click_text("Copy");
}
#[then("the expanded panel remains open and no command opens")]
fn no_command_opened(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(CLIPBOARD).is_some());
    assert!(!world.state().transaction_open);
}
#[when("the user right-clicks Copy in the expanded panel")]
fn right_click_copy(world: &mut ApplicationWorld) {
    let position = diagnostics::pointer_pixels(world.run("Copy").center());
    world.right_click(position);
}
#[then("the context menu offers \"Move to Main Panel\" and \"Add to Quick Access Toolbar\"")]
fn copy_context(world: &mut ApplicationWorld) {
    world.expect_text("Move to Main Panel");
    world.expect_text("Add to Quick Access Toolbar");
}

#[cfg(target_os = "linux")]
#[test]
fn x11_plane_group_moves_and_keeps_its_main_action() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let part = world.active_id();
    native_plane_context(&mut world);
    native_click(&mut world, "Move to Expanded Panel");
    assert!(!world.window_text("##application-ribbon").contains("Plane"));
    let expanded = "##ribbon-additional-PartWorkFeatures";
    if diagnostics::window_geometry(expanded).is_none() {
        click_expansion_arrow(&mut world, "Work Features");
    }
    let plane = world.run_in_window(expanded, "Plane").center();
    super::super::x11_input::click(diagnostics::pointer_pixels(plane));
    world.pump(Vec::new());
    world.expect_text("Offset Plane");
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    world.expect_no_text("Offset Plane");
    if diagnostics::window_geometry(expanded).is_none() {
        click_expansion_arrow(&mut world, "Work Features");
    }
    let plane = world.run_in_window(expanded, "Plane").center();
    super::super::x11_input::right_click(diagnostics::pointer_pixels(plane));
    world.pump(Vec::new());
    world.expect_text("Ungroup Commands from Drop-Down Menu");
    native_click(&mut world, "Move to Main Panel");
    assert!(diagnostics::window_geometry(expanded).is_none());
    world.expect_text("Plane");
    world.expect_active(part);
    assert!(world.state().browser.unwrap().features.is_empty());
    world.close();
}
#[cfg(target_os = "linux")]
#[test]
fn x11_plane_group_can_ungroup_and_regroup() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.app().window.native.view.resize([2400, 900]).unwrap();
    world.pump(Vec::new());
    let plane = world.run("Plane");
    let arrow = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            matches!(run.text.as_str(), "v" | "▼")
                && run.minimum[0] > plane.maximum[0]
                && (run.center()[1] - plane.center()[1]).abs() < 18.
        })
        .min_by(|left, right| left.minimum[0].total_cmp(&right.minimum[0]))
        .expect("expected the Plane drop-down arrow")
        .center();
    super::super::x11_input::click(diagnostics::pointer_pixels(arrow));
    world.pump(Vec::new());
    assert!(world.run_is_enabled("Midplane between Two Planes"));
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    native_plane_context(&mut world);
    native_click(&mut world, "Ungroup Commands from Drop-Down Menu");
    assert!(world.run_is_enabled("Midplane between Two Planes"));
    world.expect_text("Three Points");
    native_plane_context(&mut world);
    native_click(&mut world, "Group Commands into Drop-Down Menu");
    world.expect_no_text("Midplane");
    native_click(&mut world, "Plane");
    world.expect_text("Offset Plane");
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    world.expect_no_text("Offset Plane");
    world.close();
}
#[cfg(target_os = "linux")]
fn native_plane_context(world: &mut ApplicationWorld) {
    let plane = diagnostics::pointer_pixels(world.run("Plane").center());
    super::super::x11_input::right_click(plane);
    world.pump(Vec::new());
}
#[cfg(target_os = "linux")]
fn native_click(world: &mut ApplicationWorld, label: &str) {
    let point = diagnostics::pointer_pixels(world.run(label).center());
    super::super::x11_input::click(point);
    world.pump(Vec::new());
}
