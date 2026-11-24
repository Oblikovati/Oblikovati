// SPDX-License-Identifier: GPL-2.0-only
//! Import and export scenarios exercised through the native file chooser.
use super::*;
use xml::reader::XmlEvent;

const EXPORT: &str = "Export Customization Settings";
const IMPORT: &str = "Import Customization Settings";

#[given("a new XML file path is available in a writable directory")]
fn new_xml_path(world: &mut ApplicationWorld) {
    let path = world.scratch_folder().join("customization.xml");
    assert!(!path.exists());
    world.saved_path = Some(path);
}
#[when("the user enters the new file path")]
fn enter_export_path(world: &mut ApplicationWorld) {
    let path = world.saved_path.clone().unwrap();
    type_settings_file(world, EXPORT, &path);
}
#[when("the user clicks \"Save\"")]
fn save_export(world: &mut ApplicationWorld) {
    click_settings_button(world, EXPORT, "Save");
}
#[then("the export dialog closes")]
fn export_closed(_world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(EXPORT).is_none());
}
#[then("a non-empty valid XML file exists at the chosen path")]
fn exported_xml_valid(world: &mut ApplicationWorld) {
    let path = world.saved_path.as_ref().unwrap();
    let content = std::fs::read(path).unwrap();
    assert!(!content.is_empty());
    let mut root = false;
    for event in xml::EventReader::new(content.as_slice()) {
        if let XmlEvent::StartElement { name, .. } = event.expect("expected well-formed XML")
            && !root
        {
            assert_eq!(name.local_name, "ribbon-customization");
            root = true;
        }
    }
    assert!(root, "expected an XML root element");
}
#[given("the Part 3D Model custom panel list is empty with no pending changes")]
fn initially_empty_model(world: &mut ApplicationWorld) {
    model_target(world);
    assert!(!world.window_text("Customize").contains("Part|3DModel*"));
    assert!(
        world.run_in_window("Customize", "2DSketches").center()[0]
            < world.run_in_window("Customize", "Custompanel").minimum[0]
    );
}
#[given("the user has exported the current customization settings to a new XML file")]
fn export_empty_settings(world: &mut ApplicationWorld) {
    new_xml_path(world);
    export_customization(world);
    let path = world.saved_path.clone().unwrap();
    settings_file_choice(world, EXPORT, &path, "Save");
    exported_xml_valid(world);
}
#[when("the user selects that exported XML file and clicks \"Open\"")]
fn import_exported_settings(world: &mut ApplicationWorld) {
    let path = world.saved_path.clone().unwrap();
    settings_file_choice(world, IMPORT, &path, "Open");
}
#[then("the import dialog closes")]
#[then("the file picker closes")]
fn import_closed(_world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(IMPORT).is_none());
}
#[given("an XML file contains the incomplete text \"<invalid\"")]
fn malformed_settings_file(world: &mut ApplicationWorld) {
    let path = world.scratch_folder().join("malformed-customization.xml");
    std::fs::write(&path, b"<invalid").unwrap();
    world.saved_path = Some(path);
}
#[when("the user selects the file through the customization Import dialog")]
fn import_malformed_settings(world: &mut ApplicationWorld) {
    import_customization(world);
    let path = world.saved_path.clone().unwrap();
    enter_settings_file(world, IMPORT, &path, "Open");
}

#[test]
fn malformed_import_reports_the_error_without_discarding_a_staged_command() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    empty_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    malformed_settings_file(&mut world);
    import_malformed_settings(&mut world);
    import_closed(&mut world);
    world.expect_text("Cannot import ribbon customization");
    sketches_in_target(&mut world);
    transaction_steps::expect_target_pending(&world, true);
    cancel_customization(&mut world);
    no_user_commands(&mut world);
    world.close();
}
#[then("the Part 3D Model target indicates a pending change")]
fn imported_target_pending(world: &mut ApplicationWorld) {
    target_has_pending_change(world);
}
#[then("the Part 3D Model custom panel list remains empty")]
fn imported_model_empty(world: &mut ApplicationWorld) {
    assert!(
        world.run_in_window("Customize", "2DSketches").center()[0]
            < world.run_in_window("Customize", "Custompanel").minimum[0]
    );
}
#[then("no custom panel appears on the active ribbon")]
fn no_custom_panel(world: &mut ApplicationWorld) {
    world.expect_no_text("User Commands");
}
