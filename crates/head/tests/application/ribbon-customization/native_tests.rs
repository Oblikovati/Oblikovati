// SPDX-License-Identifier: GPL-2.0-only
//! Native checks for the ribbon customization workflow.
use super::*;

#[path = "marking_menu_native.rs"]
mod marking_menu_native;

#[path = "keyboard_native.rs"]
mod keyboard_native;

#[path = "zoom_native.rs"]
mod zoom_native;

#[test]
fn available_commands_support_keyboard_navigation_and_scrolling() {
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "2DSketches");
    world.key(ViewportPhysicalKey::End, false);
    world.run_in_window("Customize", "ZoomWindow");
    world.key(ViewportPhysicalKey::Home, false);
    assert!(world.window_text("Customize").contains("<Separator>"));
    world.key(ViewportPhysicalKey::V, false);
    world.run_in_window("Customize", "Vertical");
    assert_ne!(world.run_in_window("Customize", "Apply").alpha, u8::MAX);
    click_in_customize(&mut world, ">");
    assert!(
        world.run_in_window("Customize", "Vertical").center()[0]
            >= world.run_in_window("Customize", "Custompanel").minimum[0]
    );
    click_in_customize(&mut world, "Cancel");
    world.close();
}

#[test]
fn source_header_reverses_order_without_losing_selected_command() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "2DSketches");
    click_in_customize(&mut world, "Name");
    world.run_in_window("Customize", "ZoomWindow");
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
    click_in_customize(&mut world, ">");
    sketches_in_target(&mut world);
    assert!(world.window_text("Customize").contains("Part|Tools*"));
    click_in_customize(&mut world, "Cancel");
    world.close();
}

#[test]
fn customization_apply_adds_a_working_command_and_removal_hides_it() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.add_finished_sketch();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    toggle_row_option(&mut world, "2DSketches", "Text");
    click_in_customize(&mut world, "Apply");
    world.expect_text("User Commands");
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
    click_in_customize(&mut world, "Close");
    assert!(world.state().browser.unwrap().sketches[0].visible);
    world.click_text("2D Sketches");
    assert!(!world.state().browser.unwrap().sketches[0].visible);
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "2DSketches");
    click_in_customize(&mut world, "<");
    click_in_customize(&mut world, "Apply");
    world.expect_no_text("User Commands");
    click_in_customize(&mut world, "Close");
    world.close();
}
#[test]
fn customization_cancel_discards_a_staged_command() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "Cancel");
    world.expect_no_text("User Commands");
    open_tools_customization(&mut world);
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
    click_in_customize(&mut world, "Close");
    world.close();
}

#[test]
fn applying_a_reversed_addition_clears_the_pending_marker() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "<");
    assert!(world.window_text("Customize").contains("Part|Tools*"));
    assert_eq!(world.run_in_window("Customize", "Apply").alpha, u8::MAX);
    click_in_customize(&mut world, "Apply");
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
    assert_ne!(world.run_in_window("Customize", "Apply").alpha, u8::MAX);
    world.expect_no_text("User Commands");
    click_in_customize(&mut world, "Close");
    world.close();
}

#[test]
fn per_row_options_survive_removal_and_do_not_clear_multi_selection() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "3DAnnotations");
    click_in_customize(&mut world, ">");
    click_in_customize(&mut world, "2DSketches");
    shift_down(&mut world);
    assert!(!row_option_checked(&world, "2DSketches", "Large"));
    assert!(!row_option_checked(&world, "3DAnnotations", "Large"));
    toggle_row_option(&mut world, "2DSketches", "Large");
    assert!(row_option_checked(&world, "2DSketches", "Large"));
    assert!(!row_option_checked(&world, "3DAnnotations", "Large"));
    click_in_customize(&mut world, "<");
    both_rows_in_source(&mut world);
    click_in_customize(&mut world, ">");
    assert!(row_option_checked(&world, "2DSketches", "Large"));
    click_in_customize(&mut world, "Cancel");
    world.close();
}

#[test]
fn control_a_transfers_visible_source_rows_only_when_list_has_focus() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "2DSketches");
    world.key(oblikovati_viewport::ViewportPhysicalKey::A, true);
    click_in_customize(&mut world, ">");
    both_rows_in_target(&mut world);
    assert!(
        world.run_in_window("Customize", "3-PointArc").center()[0]
            >= world.run_in_window("Customize", "Custompanel").minimum[0]
    );
    click_in_customize(&mut world, "Cancel");
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "2DSketches");
    click_in_customize(&mut world, "Custompanel");
    world.key(oblikovati_viewport::ViewportPhysicalKey::A, true);
    click_in_customize(&mut world, ">");
    sketches_in_target(&mut world);
    assert!(
        world.run_in_window("Customize", "3DAnnotations").center()[0]
            < world.run_in_window("Customize", "Custompanel").minimum[0]
    );
    click_in_customize(&mut world, "Cancel");
    world.close();
}

#[test]
fn exported_applied_commands_can_be_imported_and_used() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.add_finished_sketch();
    let path = world.scratch_folder().join("ribbon-settings.xml");
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    toggle_row_option(&mut world, "2DSketches", "Text");
    click_in_customize(&mut world, "Apply");
    click_in_customize(&mut world, "Export");
    settings_file_choice(&mut world, "Export Customization Settings", &path, "Save");
    let xml = std::fs::read_to_string(&path).unwrap();
    assert!(xml.contains("ribbon-customization") && xml.contains("sketch-visibility"));
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
    click_in_customize(&mut world, "2DSketches");
    click_in_customize(&mut world, "<");
    click_in_customize(&mut world, "Apply");
    world.expect_no_text("User Commands");
    click_in_customize(&mut world, "Import");
    settings_file_choice(&mut world, "Import Customization Settings", &path, "Open");
    sketches_in_target(&mut world);
    assert!(row_option_checked(&world, "2DSketches", "Text"));
    assert!(world.window_text("Customize").contains("Part|Tools*"));
    world.expect_no_text("User Commands");
    click_in_customize(&mut world, "Apply");
    click_in_customize(&mut world, "Close");
    world.click_text("2D Sketches");
    assert!(!world.state().browser.unwrap().sketches[0].visible);
    world.close();
}

#[test]
fn applied_ribbon_settings_survive_restart_without_unapplied_edits() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.add_finished_sketch();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    toggle_row_option(&mut world, "2DSketches", "Text");
    click_in_customize(&mut world, "Apply");
    click_in_customize(&mut world, "Line");
    click_in_customize(&mut world, ">");
    assert!(world.window_text("Customize").contains("Part|Tools*"));
    world.reopen_application();
    world.add_part();
    world.add_finished_sketch();
    world.click_text("Tools");
    world.expect_text("User Commands");
    world.click_text("2D Sketches");
    assert!(!world.state().browser.unwrap().sketches[0].visible);
    open_tools_customization(&mut world);
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
    scroll_source_to_line(&mut world);
    assert!(
        world.run_in_window("Customize", "Line").center()[0]
            < world.run_in_window("Customize", "Custompanel").minimum[0]
    );
    click_in_customize(&mut world, "Close");
    world.close();
}

#[path = "clipboard_native.rs"]
mod clipboard_native;

#[path = "find_native.rs"]
mod find_native;

#[path = "part_assets_native.rs"]
mod part_assets_native;

#[path = "measure_native.rs"]
mod measure_native;

#[path = "reset_native.rs"]
mod reset_native;

#[test]
fn failed_ribbon_settings_save_keeps_the_draft_unapplied() {
    let mut world = ApplicationWorld::new();
    let destination = world
        .preferences_directory()
        .join("ribbon-customization.xml");
    std::fs::create_dir(&destination).unwrap();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "Apply");
    assert!(diagnostics::window_geometry("Customize").is_some());
    assert!(world.window_text("Customize").contains("Part|Tools*"));
    world.expect_no_text("User Commands");
    world.expect_text("expected a complete atomic file replacement");
    click_in_customize(&mut world, "Cancel");
    assert!(destination.is_dir());
    world.close();
}

#[test]
fn malformed_import_keeps_the_existing_draft_and_reports_an_error() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let path = world.scratch_folder().join("invalid.xml");
    std::fs::write(&path, "<ribbon-customization version=\"99\"/>").unwrap();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "Import");
    enter_settings_file(&mut world, "Import Customization Settings", &path, "Open");
    assert!(diagnostics::window_geometry("Import Customization Settings").is_none());
    world.expect_text("ribbon version Some(\"99\"): expected 1 through 8");
    sketches_in_target(&mut world);
    assert!(world.window_text("Customize").contains("Part|Tools*"));
    click_in_customize(&mut world, "Cancel");
    world.expect_no_text("User Commands");
    world.close();
}

#[test]
fn escape_dismisses_missing_file_warning_before_import_picker() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let path = world.scratch_folder().join("missing.xml");
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "Import");
    enter_settings_file(&mut world, "Import Customization Settings", &path, "Open");
    assert!(diagnostics::window_geometry("File Not Found").is_some());
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    assert!(diagnostics::window_geometry("File Not Found").is_none());
    assert!(diagnostics::window_geometry("Import Customization Settings").is_some());
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    assert!(diagnostics::window_geometry("Import Customization Settings").is_none());
    assert!(world.window_text("Customize").contains("Part|Tools*"));
    click_in_customize(&mut world, "Cancel");
    world.close();
}

#[test]
fn custom_line_command_draws_in_the_active_sketch() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "Line");
    click_in_customize(&mut world, ">");
    toggle_row_option(&mut world, "Line", "Text");
    click_in_customize(&mut world, "Apply");
    click_in_customize(&mut world, "Close");
    world.expect_text("User Commands");
    world.click_text("Line");
    world.expect_status("Select start of line");
    world.sketch_click([620, 300]);
    world.sketch_click([840, 420]);
    world.expect_segment_drawn([620, 300], [840, 420]);
    world.escape();
    world.close();
}

#[test]
fn escape_discards_customization_without_cancelling_the_active_sketch() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let geometry = world.edited_geometry();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    assert!(diagnostics::window_geometry("Customize").is_none());
    assert!(world.edited_geometry().same_recipe(&geometry));
    world.expect_no_text("User Commands");
    open_tools_customization(&mut world);
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
    click_in_customize(&mut world, "Close");
    world.close();
}

#[test]
fn escape_from_the_target_list_discards_the_pending_draft() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "Part|Tools*");
    world.expect_text("Part | Annotate");
    world.key(oblikovati_viewport::ViewportPhysicalKey::DownArrow, false);
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    assert!(diagnostics::window_geometry("Customize").is_none());
    open_tools_customization(&mut world);
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
    sketches_in_available_commands(&mut world);
    click_in_customize(&mut world, "Close");
    world.close();
}
