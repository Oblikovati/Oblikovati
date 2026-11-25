// SPDX-License-Identifier: GPL-2.0-only
//! Inspect the Save As commands shown for an active saved part.
use super::*;

#[given("a saved part is active in the MCAD Software")]
fn saved_part_active(world: &mut ApplicationWorld) {
    let id = world.add_part();
    world.previous_document = Some(id);
    world.saved_path = Some(world.save_part_to_file(id, "saved part.opd"));
}
#[when("I open the application menu")]
#[when("the user opens the application menu")]
fn open_app_menu(world: &mut ApplicationWorld) {
    world.click_text("File");
    world.pump(Vec::new());
}
#[when("I expand Save As")]
#[when("the user expands Save As")]
fn expand_save_as(world: &mut ApplicationWorld) {
    let center = world.run("Save As").center().map(|value| value as i32);
    world.hover(center);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("the menu offers Save As and Save Copy As")]
fn offers_save_as(world: &mut ApplicationWorld) {
    world.expect_text("Save As...");
    world.expect_text("Save Copy As");
}
#[then("the submenu offers Save As and Save Copy As")]
fn save_as_rows(world: &mut ApplicationWorld) {
    world.expect_text("Save As");
    world.expect_text("Save Copy As");
}
#[then("it offers Save Copy As Template and Pack and Go")]
fn offers_template(world: &mut ApplicationWorld) {
    world.expect_text("Save Copy As Template");
    world.expect_text("Pack and Go");
}
#[then("the Save Copy As description says the original file remains open")]
fn copy_description(world: &mut ApplicationWorld) {
    world.expect_text("the original file remains open");
}
