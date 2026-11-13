// SPDX-License-Identifier: GPL-2.0-only
//! Native checks for the ribbon customization workflow.
use super::*;

#[test]
fn api_marking_menu_overflow_runs_its_sketch_command() {
    use oblikovati_api::{
        client::Client,
        types,
        wire::{MarkingMenuView, SetMarkingMenuArgs},
    };
    let mut world = ApplicationWorld::new();
    world.add_part();
    let menu = MarkingMenuView {
        environment: types::SKETCH_ENVIRONMENT,
        quadrants: None,
        overflow: Some(vec!["Sketch.Rectangle".into()]),
    };
    Client::new(world.app().owner.document_host())
        .ui()
        .set_marking_menu(&SetMarkingMenuArgs { menu })
        .unwrap();
    world.start_sketch();
    world.right_click([900, 450]);
    world.pump(Vec::new());
    let entry = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Rectangle" && run.center()[0] > 700. && run.center()[1] > 250.)
        .unwrap_or_else(|| panic!("expected Rectangle in menu overflow: {:?}", world.text()))
        .center();
    world.click(diagnostics::pointer_pixels(entry));
    world.click([760, 380]);
    world.click([1060, 560]);
    assert_eq!(world.edited_geometry().lines().len(), 4);
    world.close();
}

#[test]
fn marking_menu_editor_applies_and_restores_a_slot() {
    use oblikovati_api::{client::Client, types, wire::GetMarkingMenuArgs};
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "MarkingMenu");
    let north = world.run_in_window("Customize", "North").clone();
    world.click([(north.minimum[0] - 70.) as i32, north.center()[1] as i32]);
    world.pump(Vec::new());
    world.click_text("Rectangle");
    world.pump(Vec::new());
    click_in_customize(&mut world, "Apply");
    let menu = Client::new(world.app().owner.document_host())
        .ui()
        .marking_menu(&GetMarkingMenuArgs {
            environment: types::SKETCH_ENVIRONMENT,
        })
        .unwrap();
    assert!(menu.quadrants.unwrap().iter().any(
        |item| item.quadrant == types::QUADRANT_NORTH && item.command_id == "Sketch.Rectangle"
    ));
    click_in_customize(&mut world, "Close");
    world.reopen_application();
    world.add_part();
    world.start_sketch();
    world.right_click([900, 450]);
    world.pump(Vec::new());
    let entry = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Rectangle" && run.center()[0] > 700. && run.center()[1] > 250.)
        .unwrap_or_else(|| panic!("expected Rectangle in canvas menu: {:?}", world.text()))
        .center();
    world.click(diagnostics::pointer_pixels(entry));
    world.click([760, 380]);
    world.click([1060, 560]);
    assert_eq!(world.edited_geometry().lines().len(), 4);
    world.close();
}

#[test]
fn saved_classic_menu_keeps_the_sketch_context_menu_linear() {
    let mut world = ApplicationWorld::new();
    std::fs::write(
        world.preferences_directory().join("markingmenu.yaml"),
        "classic: true\n",
    )
    .unwrap();
    world.reopen_application();
    world.add_part();
    world.start_sketch();
    assert!(world.state().sketch_classic_menu);
    world.right_click([900, 450]);
    world.expect_text("Measure");
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| run.text == "Rectangle" && run.center()[0] > 700. && run.center()[1] > 250.)
    );
    world.close();
}

#[test]
fn api_marking_menu_drives_the_sketch_canvas() {
    use oblikovati_api::{
        client::Client,
        types,
        wire::{MarkingMenuItem, MarkingMenuView, SetMarkingMenuArgs},
    };
    let mut world = ApplicationWorld::new();
    world.add_part();
    let menu = MarkingMenuView {
        environment: types::SKETCH_ENVIRONMENT,
        quadrants: Some(vec![MarkingMenuItem {
            quadrant: types::QUADRANT_NORTH,
            command_id: "Sketch.Rectangle".into(),
        }]),
        overflow: None,
    };
    Client::new(world.app().owner.document_host())
        .ui()
        .set_marking_menu(&SetMarkingMenuArgs { menu })
        .unwrap();
    world.reopen_application();
    world.add_part();
    world.start_sketch();
    world.right_click([900, 450]);
    world.pump(Vec::new());
    let entry = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Rectangle" && run.center()[0] > 700. && run.center()[1] > 250.)
        .unwrap_or_else(|| panic!("expected Rectangle in canvas menu: {:?}", world.text()))
        .center();
    world.click(diagnostics::pointer_pixels(entry));
    world.click([760, 380]);
    world.click([1060, 560]);
    assert_eq!(world.edited_geometry().lines().len(), 4);
    world.close();
}

#[test]
fn external_keymap_change_blocks_a_stale_dialog_draft() {
    use oblikovati_api::{client::Client, wire::SetChordArgs};
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "Keyboard");
    let search = world.run_in_window("Customize", "Searchcommands").clone();
    world.click([(search.minimum[0] - 70.) as i32, search.center()[1] as i32]);
    world.type_text("Undo");
    click_in_customize(&mut world, "Ctrl+Z");
    let chord = world.run_in_window("Customize", "Shortcut").clone();
    world.click([(chord.minimum[0] - 70.) as i32, chord.center()[1] as i32]);
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("Ctrl+U");
    click_in_customize(&mut world, "Setbinding");
    Client::new(world.app().owner.document_host())
        .keymap()
        .set_chord(&SetChordArgs {
            action_id: "edit.redo".into(),
            chord: "Ctrl+R".into(),
        })
        .unwrap();
    world.pump(Vec::new());
    assert!(
        world
            .window_text("Customize")
            .contains("Keyboardsettingschangedoutsidethisdialog")
    );
    assert_ne!(world.run_in_window("Customize", "Apply").alpha, u8::MAX);
    click_in_customize(&mut world, "Cancel");
    let bindings = world.state().keymap_bindings;
    assert_eq!(
        bindings
            .iter()
            .find(|entry| entry.action_id == "edit.undo")
            .unwrap()
            .chord,
        "Ctrl+Z"
    );
    assert_eq!(
        bindings
            .iter()
            .find(|entry| entry.action_id == "edit.redo")
            .unwrap()
            .chord,
        "Ctrl+R"
    );
    world.close();
}

#[test]
fn rectangle_alias_is_edited_and_used_from_the_dialog() {
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "Keyboard");
    let search = world.run_in_window("Customize", "Searchcommands").clone();
    world.click([(search.minimum[0] - 70.) as i32, search.center()[1] as i32]);
    world.type_text("Rect");
    click_in_customize(&mut world, "Rectangle");
    let alias = world.run_in_window("Customize", "Alias").clone();
    world.click([(alias.minimum[0] - 70.) as i32, alias.center()[1] as i32]);
    world.type_text("RECT");
    click_in_customize(&mut world, "Setbinding");
    assert_eq!(world.state().keymap.unwrap().resolve_alias("rect"), None);
    click_in_customize(&mut world, "Apply");
    assert_eq!(
        world.state().keymap.unwrap().resolve_alias("rect"),
        Some("sketch-rectangle")
    );
    click_in_customize(&mut world, "Close");
    world.click_text("3D Model");
    world.start_sketch();
    world.type_text("rect");
    world.key(ViewportPhysicalKey::Enter, false);
    world.click([760, 380]);
    world.click([1060, 560]);
    assert_eq!(world.edited_geometry().lines().len(), 4);
    world.close();
}

#[test]
fn typed_rectangle_alias_starts_the_sketch_tool() {
    use oblikovati_api::{client::Client, wire::SetAliasArgs};
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.add_part();
    Client::new(world.app().owner.document_host())
        .keymap()
        .set_alias(&SetAliasArgs {
            action_id: "sketch-rectangle".into(),
            alias: "RECT".into(),
        })
        .unwrap();
    world.start_sketch();
    world.type_text("rect");
    assert_eq!(world.state().sketch_alias.as_deref(), Some("rect"));
    world.key(ViewportPhysicalKey::Enter, false);
    world.click([760, 380]);
    world.click([1060, 560]);
    assert_eq!(world.edited_geometry().lines().len(), 4);
    world.close();
}

#[test]
fn bound_dimension_shortcut_starts_the_dimension_tool() {
    use oblikovati_api::{client::Client, wire::SetChordArgs};
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.add_part();
    Client::new(world.app().owner.document_host())
        .keymap()
        .set_chord(&SetChordArgs {
            action_id: "dimension".into(),
            chord: "Ctrl+G".into(),
        })
        .unwrap();
    world.start_sketch();
    assert!(!world.state().dimension_tool_active);
    world.key(ViewportPhysicalKey::G, true);
    assert!(world.state().dimension_tool_active);
    world.close();
}

#[test]
fn text_input_owns_a_bound_sketch_shortcut() {
    use oblikovati_api::{client::Client, wire::SetChordArgs};
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.add_part();
    Client::new(world.app().owner.document_host())
        .keymap()
        .set_chord(&SetChordArgs {
            action_id: "sketch-rectangle".into(),
            chord: "Ctrl+R".into(),
        })
        .unwrap();
    world.start_sketch();
    world.click_text("Tools");
    world.click_text("Customize");
    world.pump(Vec::new());
    click_in_customize(&mut world, "Keyboard");
    let label = world.run_in_window("Customize", "Searchcommands").clone();
    world.click([(label.minimum[0] - 70.) as i32, label.center()[1] as i32]);
    world.key(ViewportPhysicalKey::R, true);
    assert!(!world.state().sketch_tool_keyboard);
    click_in_customize(&mut world, "Close");
    world.key(ViewportPhysicalKey::R, true);
    assert!(world.state().sketch_tool_keyboard);
    world.close();
}

#[test]
fn bound_rectangle_shortcut_runs_the_ribbon_tool() {
    use oblikovati_api::{client::Client, wire::SetChordArgs};
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.add_part();
    Client::new(world.app().owner.document_host())
        .keymap()
        .set_chord(&SetChordArgs {
            action_id: "sketch-rectangle".into(),
            chord: "Ctrl+,".into(),
        })
        .unwrap();
    world.start_sketch();
    world.key(ViewportPhysicalKey::Comma, true);
    world.click([760, 380]);
    world.click([1060, 560]);
    assert_eq!(world.edited_geometry().lines().len(), 4);
    world.close();
}

#[test]
fn keyboard_filter_and_reset_are_staged_until_applied() {
    use oblikovati_api::{client::Client, wire::SetChordArgs};
    let mut world = ApplicationWorld::new();
    world.add_part();
    Client::new(world.app().owner.document_host())
        .keymap()
        .set_chord(&SetChordArgs {
            action_id: "edit.undo".into(),
            chord: "Ctrl+U".into(),
        })
        .unwrap();
    world.pump(Vec::new());
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "Keyboard");
    let label = world.run_in_window("Customize", "Searchcommands").clone();
    world.click([(label.minimum[0] - 70.) as i32, label.center()[1] as i32]);
    world.type_text("home");
    let text = world.window_text("Customize");
    assert!(text.contains("HomeView"));
    assert!(!text.contains("PreviousView"));
    click_in_customize(&mut world, "Clear");
    assert!(world.window_text("Customize").contains("Circle"));
    let label = world.run_in_window("Customize", "Searchcommands").clone();
    world.click([(label.minimum[0] - 70.) as i32, label.center()[1] as i32]);
    world.type_text("Undo");
    click_in_customize(&mut world, "ResetAllKeys");
    assert!(world.window_text("Customize").contains("Ctrl+Z"));
    click_in_customize(&mut world, "Cancel");
    assert_eq!(undo_shortcut(&mut world), "Ctrl+U");
    world.click_text("Customize");
    world.pump(Vec::new());
    if !world.window_text("Customize").contains("ResetAllKeys") {
        click_in_customize(&mut world, "Keyboard");
    }
    click_in_customize(&mut world, "ResetAllKeys");
    click_in_customize(&mut world, "Apply");
    assert_eq!(undo_shortcut(&mut world), "Ctrl+Z");
    click_in_customize(&mut world, "Close");
    world.close();
}
fn undo_shortcut(world: &mut ApplicationWorld) -> String {
    world
        .state()
        .keymap_bindings
        .into_iter()
        .find(|entry| entry.action_id == "edit.undo")
        .unwrap()
        .chord
}

#[test]
fn keyboard_shortcut_edit_is_staged_and_applied_from_the_dialog() {
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "Keyboard");
    let search = world.run_in_window("Customize", "Searchcommands").clone();
    world.click([(search.minimum[0] - 70.) as i32, search.center()[1] as i32]);
    world.type_text("Undo");
    click_in_customize(&mut world, "Ctrl+Z");
    let label = world.run_in_window("Customize", "Shortcut").clone();
    world.click([(label.minimum[0] - 70.) as i32, label.center()[1] as i32]);
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("Ctrl+U");
    click_in_customize(&mut world, "Setbinding");
    let binding = |world: &mut ApplicationWorld| {
        world
            .state()
            .keymap_bindings
            .into_iter()
            .find(|entry| entry.action_id == "edit.undo")
            .unwrap()
            .chord
    };
    assert_eq!(binding(&mut world), "Ctrl+Z");
    click_in_customize(&mut world, "Apply");
    assert_eq!(binding(&mut world), "Ctrl+U");
    click_in_customize(&mut world, "Close");
    world.click_text("Customize");
    world.pump(Vec::new());
    assert!(world.window_text("Customize").contains("Keyboardshortcuts"));
    click_in_customize(&mut world, "Close");
    world.reopen_application();
    assert_eq!(binding(&mut world), "Ctrl+U");
    world.add_part();
    world.click_text("3D Model");
    world.start_sketch();
    world.draw_rectangle();
    world.escape();
    let before_undo = world.edited_geometry();
    assert_eq!(before_undo.lines().len(), 4);
    world.key(ViewportPhysicalKey::Z, true);
    assert!(world.edited_geometry().same_recipe(&before_undo));
    world.key(ViewportPhysicalKey::U, true);
    assert!(world.edited_geometry().lines().is_empty());
    world.key(ViewportPhysicalKey::Y, true);
    assert_eq!(world.edited_geometry().lines().len(), 4);
    oblikovati_api::client::Client::new(world.app().owner.document_host())
        .keymap()
        .set_alias(&oblikovati_api::wire::SetAliasArgs {
            action_id: "edit.undo".into(),
            alias: "BACK".into(),
        })
        .unwrap();
    world.type_text("back");
    world.key(ViewportPhysicalKey::Enter, false);
    assert!(world.edited_geometry().lines().is_empty());
    world.close();
}

#[test]
fn keyboard_switches_targets_without_losing_staged_commands() {
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "Part|Tools*");
    world.key(ViewportPhysicalKey::DownArrow, false);
    world.key(ViewportPhysicalKey::Enter, false);
    assert!(world.window_text("Customize").contains("Part|Manage"));
    sketches_in_available_commands(&mut world);
    click_in_customize(&mut world, "Part|Manage");
    world.key(ViewportPhysicalKey::UpArrow, false);
    world.key(ViewportPhysicalKey::Enter, false);
    assert!(world.window_text("Customize").contains("Part|Tools*"));
    sketches_in_target(&mut world);
    click_in_customize(&mut world, "Cancel");
    world.close();
}

#[test]
fn escape_from_command_category_discards_the_pending_draft() {
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "AllCommands");
    world.expect_text("Macros");
    world.key(ViewportPhysicalKey::DownArrow, false);
    world.key(ViewportPhysicalKey::Escape, false);
    assert!(diagnostics::window_geometry("Customize").is_none());
    world.expect_no_text("User Commands");
    open_tools_customization(&mut world);
    assert!(world.window_text("Customize").contains("AllCommands"));
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
    sketches_in_available_commands(&mut world);
    click_in_customize(&mut world, "Close");
    world.close();
}

#[test]
fn command_category_can_be_selected_with_keyboard() {
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "AllCommands");
    world.key(ViewportPhysicalKey::DownArrow, false);
    world.key(ViewportPhysicalKey::Enter, false);
    assert!(world.window_text("Customize").contains("Macros"));
    assert!(!world.window_text("Customize").contains("2DSketches"));
    click_in_customize(&mut world, "Close");
    world.close();
}

#[test]
fn available_commands_support_keyboard_navigation_and_scrolling() {
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "2DSketches");
    world.key(ViewportPhysicalKey::End, false);
    assert!(world.window_text("Customize").contains("Vertical"));
    world.key(ViewportPhysicalKey::Home, false);
    assert!(world.window_text("Customize").contains("<Separator>"));
    world.key(ViewportPhysicalKey::V, false);
    assert!(world.window_text("Customize").contains("Vertical"));
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
    assert!(world.window_text("Customize").contains("Vertical"));
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
fn exported_draft_can_be_imported_applied_and_used() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.add_finished_sketch();
    let path = world.scratch_folder().join("ribbon-settings.xml");
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    toggle_row_option(&mut world, "2DSketches", "Text");
    click_in_customize(&mut world, "Export");
    settings_file_choice(&mut world, "Export Customization Settings", &path, "Save");
    let xml = std::fs::read_to_string(&path).unwrap();
    assert!(xml.contains("ribbon-customization") && xml.contains("sketch-visibility"));
    assert!(world.window_text("Customize").contains("Part|Tools*"));
    click_in_customize(&mut world, "Cancel");
    open_tools_customization(&mut world);
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
    assert!(diagnostics::window_geometry("Import Customization Settings").is_some());
    world.expect_text("expected version 1 or 2 ribbon settings");
    cancel_customization(&mut world);
    sketches_in_target(&mut world);
    assert!(world.window_text("Customize").contains("Part|Tools*"));
    click_in_customize(&mut world, "Cancel");
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
