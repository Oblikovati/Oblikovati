// SPDX-License-Identifier: GPL-2.0-only
//! Recover from a missing customization import through the native dialogs.
use super::*;

#[given("the ribbon customization dialog is open with no pending changes")]
fn empty_customization(world: &mut ApplicationWorld) {
    empty_model_customization(world);
}
#[given("the customization import file picker is open")]
fn import_picker(world: &mut ApplicationWorld) {
    import_customization(world);
}
#[when("the user enters a path to a nonexistent XML file and clicks \"Open\"")]
fn open_missing_file(world: &mut ApplicationWorld) {
    let path = world.scratch_folder().join("missing.xml");
    world.saved_path = Some(path.clone());
    enter_settings_file(world, "Import Customization Settings", &path, "Open");
}
#[then("a warning says the file was not found and asks the user to check its name")]
fn missing_warning(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("File Not Found").is_some());
    world.expect_text("was not found");
    world.expect_text("Check the file name");
}
#[when("the user acknowledges the warning")]
fn acknowledge_warning(world: &mut ApplicationWorld) {
    let center = world.run_in_window("File Not Found", "OK").center();
    world.click(diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}
#[then("the import file picker remains open with the filename selected")]
fn filename_selected(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Import Customization Settings").is_some());
    let path = world.saved_path.as_ref().unwrap().display().to_string();
    assert!(world.text_is_selected(&path));
}
#[when("the user cancels the file picker")]
fn cancel_picker(world: &mut ApplicationWorld) {
    cancel_customization(world);
}
#[then("the customization dialog remains open without a pending-change marker")]
fn customization_unchanged(world: &mut ApplicationWorld) {
    customization_remains_open(world);
    assert!(world.window_text("Customize").contains("Part|3DModel"));
    assert!(!world.window_text("Customize").contains("Part|3DModel*"));
}
#[then("\"Apply\" is unavailable")]
fn apply_disabled(world: &mut ApplicationWorld) {
    assert_ne!(world.run_in_window("Customize", "Apply").alpha, u8::MAX);
}
