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
        title.text.ends_with('v')
            || world.frame.runs.iter().any(|run| {
                run.text == "v"
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
