// SPDX-License-Identifier: GPL-2.0-only
//! Import and export scenarios exercised through the native file chooser.
use super::*;
#[path = "interface_export.rs"]
mod interface_export;
use xml::reader::XmlEvent;

#[path = "import_drafts.rs"]
mod import_drafts;

const EXPORT: &str = "Export Customization Settings";
const IMPORT: &str = "Import Customization Settings";

#[when("the user clicks Import")]
fn open_import_chooser(world: &mut ApplicationWorld) {
    import_customization(world);
}

#[when("the user clicks Export")]
fn open_export_chooser(world: &mut ApplicationWorld) {
    export_customization(world);
}

#[then("the Import Customization Settings file chooser opens")]
fn import_chooser_open(world: &mut ApplicationWorld) {
    settings_dialog_open(world, IMPORT.into());
}

#[then("the Export Customization Settings file chooser opens")]
fn export_chooser_open(world: &mut ApplicationWorld) {
    settings_dialog_open(world, EXPORT.into());
}

#[when("the user clicks Cancel in the file chooser")]
fn cancel_file_chooser(world: &mut ApplicationWorld) {
    assert!(
        [IMPORT, EXPORT]
            .into_iter()
            .any(|title| diagnostics::window_geometry(title).is_some())
    );
    cancel_customization(world);
}

#[then("the file chooser closes")]
fn file_chooser_closed(world: &mut ApplicationWorld) {
    settings_dialog_closed(world);
}

#[then("2D Sketches remains in the Part Tools target list")]
#[then("the customization dialog still shows 2D Sketches in the target list")]
fn sketches_still_staged(world: &mut ApplicationWorld) {
    assert_eq!(
        transaction_steps::target_label(world).trim_end_matches('*'),
        "Part|Tools"
    );
    assert_eq!(transaction_steps::target_rows(world), ["2DSketches"]);
}

#[when("the user clicks Cancel in the customization dialog")]
fn cancel_customize(world: &mut ApplicationWorld) {
    click_in_customize(world, "Cancel");
}

#[given("2D Sketches has been staged with Large and Text unchecked")]
#[given("\"2D Sketches\" is staged with Large and Text unchecked")]
fn stage_small_sketches(world: &mut ApplicationWorld) {
    stage_sketch_visibility(world);
    for option in ["Large", "Text"] {
        if row_option_checked(world, "2DSketches", option) {
            toggle_row_option(world, "2DSketches", option);
        }
        assert!(!row_option_checked(world, "2DSketches", option));
    }
}

#[given("\"2D Sketches\" is added with Large and Text unchecked and applied")]
fn apply_small_sketches(world: &mut ApplicationWorld) {
    empty_tools_customization(world);
    stage_small_sketches(world);
    click_in_customize(world, "Apply");
    transaction_steps::expect_target_pending(world, false);
}

#[given("the customization is exported to a new XML file")]
fn export_applied_sketches(world: &mut ApplicationWorld) {
    export_new_file(world);
    assert_eq!(exported_tools_commands(world), ["sketch-visibility"]);
}

#[given("the command is returned to the available list and the empty panel is applied")]
fn remove_and_apply_empty_tools_panel(world: &mut ApplicationWorld) {
    click_in_customize(world, "2DSketches");
    transfer_to_source(world);
    transaction_steps::target_is_empty(world);
    click_in_customize(world, "Apply");
    no_user_commands(world);
}

#[given("the target shows a pending-change marker")]
fn staged_target_is_pending(world: &mut ApplicationWorld) {
    tools_pending(world);
}

#[when("the user exports customization to a new XML file")]
fn export_new_file(world: &mut ApplicationWorld) {
    new_xml_path(world);
    export_customization(world);
    enter_export_path(world);
    save_export(world);
}

#[then("the exported settings do not contain the staged 2D Sketches command in Part Tools")]
fn export_excludes_staged_sketches(world: &mut ApplicationWorld) {
    assert!(exported_tools_commands(world).is_empty());
}

fn exported_tools_commands(world: &ApplicationWorld) -> Vec<String> {
    let content = std::fs::File::open(world.saved_path.as_ref().unwrap()).unwrap();
    let mut root = false;
    let mut tools = false;
    let mut commands = Vec::new();
    for event in xml::EventReader::new(content) {
        match event.expect("expected well-formed XML") {
            XmlEvent::StartElement { name, .. } if !root => {
                assert_eq!(name.local_name, "ribbon-customization");
                root = true;
            }
            XmlEvent::StartElement {
                name, attributes, ..
            } if name.local_name == "tab" => {
                tools = attributes.iter().any(|attribute| {
                    attribute.name.local_name == "id" && attribute.value == "tools"
                });
            }
            XmlEvent::StartElement {
                name, attributes, ..
            } if tools && name.local_name == "command" => {
                let id = attributes
                    .into_iter()
                    .find(|attribute| attribute.name.local_name == "id")
                    .expect("expected a ribbon command id");
                commands.push(id.value);
            }
            XmlEvent::EndElement { name } if name.local_name == "tab" => tools = false,
            _ => {}
        }
    }
    assert!(root, "expected an XML root element");
    commands
}

#[test]
fn export_includes_applied_commands_staged_for_removal() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    empty_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "Apply");
    click_in_customize(&mut world, "2DSketches");
    transfer_to_source(&mut world);
    transaction_steps::target_is_empty(&world);
    export_new_file(&mut world);
    export_closed(&mut world);
    assert_eq!(exported_tools_commands(&world), ["sketch-visibility"]);
    transaction_steps::target_is_empty(&world);
    transaction_steps::expect_target_pending(&world, true);
    cancel_customize(&mut world);
    world.close();
}

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
#[then("the export chooser closes")]
fn export_closed(_world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(EXPORT).is_none());
}
#[then("a non-empty valid XML file exists at the chosen path")]
fn exported_xml_valid(world: &mut ApplicationWorld) {
    exported_tools_commands(world);
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
    export_new_file(world);
    exported_xml_valid(world);
}
#[when("the user selects that exported XML file and clicks \"Open\"")]
fn import_exported_settings(world: &mut ApplicationWorld) {
    let path = world.saved_path.clone().unwrap();
    settings_file_choice(world, IMPORT, &path, "Open");
}
#[then("the import dialog closes")]
#[then("the import chooser closes")]
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
#[then("the User Commands panel is not yet visible on the Tools ribbon")]
fn no_custom_panel(world: &mut ApplicationWorld) {
    world.expect_no_text("User Commands");
}
