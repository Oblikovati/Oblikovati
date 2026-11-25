// SPDX-License-Identifier: GPL-2.0-only
//! Track the native Shell image as its shortcut is added and removed.
use super::super::*;
use super::{RIBBON, icon_images_inside, shell_standard};

const ADD: &str = "Add to Quick Access Toolbar";
const REMOVE: &str = "Remove from Quick Access Toolbar";

#[given("\"Shell\" is visible in the 3D Model Modify panel")]
fn shell_on_ribbon(world: &mut ApplicationWorld) {
    shell_standard(world);
}

#[given("\"Shell\" is absent from the quick-access toolbar")]
#[then("the added Shell icon disappears from the toolbar")]
fn shortcut_absent(world: &mut ApplicationWorld) {
    assert!(shortcut_center(world).is_none());
}

#[when("the user right-clicks \"Shell\" and selects \"Add to Quick Access Toolbar\"")]
fn add_shell_shortcut(world: &mut ApplicationWorld) {
    open_shell_context(world);
    assert!(world.run_is_enabled(ADD));
    world.click_text(ADD);
}

#[when("the user reopens the ribbon command context menu")]
fn open_shell_context(world: &mut ApplicationWorld) {
    let shell = world.run_in_window(RIBBON, "Shell");
    world.right_click(diagnostics::pointer_pixels(shell.center()));
    world.expect_text(ADD);
}

#[then("a Shell icon appears in the quick-access toolbar")]
fn shortcut_appears(world: &mut ApplicationWorld) {
    assert!(shortcut_center(world).is_some());
}

#[then("\"Shell\" remains in the Modify panel")]
fn shell_stays_on_ribbon(world: &mut ApplicationWorld) {
    shell_standard(world);
}

#[then("\"Add to Quick Access Toolbar\" is unavailable")]
fn shortcut_cannot_be_added_twice(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled(ADD));
}

#[when("the user right-clicks the added Shell icon in the quick-access toolbar")]
fn open_shortcut_context(world: &mut ApplicationWorld) {
    let center = shortcut_center(world).expect("expected the Shell shortcut");
    world.right_click(diagnostics::pointer_pixels(center));
}

#[then("its menu offers \"Remove from Quick Access Toolbar\"")]
fn shortcut_offers_removal(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled(REMOVE));
}

#[then("the ribbon command context menu offers \"Add to Quick Access Toolbar\" again")]
fn shortcut_available_again(world: &mut ApplicationWorld) {
    open_shell_context(world);
    assert!(world.run_is_enabled(ADD));
    assert_eq!(world.state().document, world.previous_document);
}

fn shortcut_center(world: &ApplicationWorld) -> Option<[f32; 2]> {
    let shell = world
        .shell_button_texture
        .expect("expected the Modify Shell image");
    icon_images_inside(([0., 0.], [f32::MAX, 32.]))
        .into_iter()
        .find_map(|((first, last), texture)| {
            (texture == shell).then(|| std::array::from_fn(|axis| (first[axis] + last[axis]) / 2.))
        })
}
