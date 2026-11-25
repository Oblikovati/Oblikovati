// SPDX-License-Identifier: GPL-2.0-only
//! Native checks for keyboard customization.
use super::*;
#[path = "keyboard_initial.rs"]
mod keyboard_initial;

#[test]
fn default_sketch_aliases_are_staged_and_reload_before_dispatch() {
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let original = world.edited_geometry();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "Keyboard");
    let label = "Use default multi-character command aliases";
    assert!(!world.checked(label));
    assert_eq!(world.state().keymap.unwrap().resolve_alias("rectang"), None);
    world.click_text(label);
    assert!(world.checked(label));
    assert_eq!(world.state().keymap.unwrap().resolve_alias("rectang"), None);
    click_in_customize(&mut world, "Cancel");
    assert!(world.edited_geometry().same_recipe(&original));
    world.click_text("Customize");
    world.pump(Vec::new());
    assert!(!world.checked(label));
    world.click_text(label);
    click_in_customize(&mut world, "Apply");
    assert_eq!(
        world.state().keymap.unwrap().resolve_alias("rectang"),
        Some("sketch-rectangle")
    );
    click_in_customize(&mut world, "Close");

    world.reopen_application();
    world.start_empty_sketch();
    assert!(world.state().keymap.unwrap().default_aliases());
    world.type_text("rectang");
    world.key(ViewportPhysicalKey::Enter, false);
    world.sketch_click([760, 380]);
    world.sketch_click([1060, 560]);
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
    let search = keyboard_initial::search_label(&world).clone();
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
    let search = keyboard_initial::search_label(&world).clone();
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
    let label = keyboard_initial::search_label(&world).clone();
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
    let label = keyboard_initial::search_label(&world).clone();
    world.click([(label.minimum[0] - 70.) as i32, label.center()[1] as i32]);
    world.type_text("home");
    let text = world.window_text("Customize");
    assert!(text.contains("HomeView"));
    assert!(!text.contains("PreviousView"));
    clear_keyboard_search(&mut world);
    assert!(world.window_text("Customize").contains("Circle"));
    let label = keyboard_initial::search_label(&world).clone();
    world.click([(label.minimum[0] - 70.) as i32, label.center()[1] as i32]);
    world.type_text("Undo");
    click_in_customize(&mut world, "ResetAllKeys");
    assert!(world.window_text("Customize").contains("Ctrl+Z"));
    click_in_customize(&mut world, "Cancel");
    assert_eq!(undo_shortcut(&mut world), "Ctrl+U");
    world.click_text("Customize");
    world.pump(Vec::new());
    keyboard_initial::keyword_search_is_empty(&mut world);
    keyboard_initial::displayed_all(&world);
    click_in_customize(&mut world, "ResetAllKeys");
    click_in_customize(&mut world, "Apply");
    assert_eq!(undo_shortcut(&mut world), "Ctrl+Z");
    click_in_customize(&mut world, "Close");
    world.close();
}
fn clear_keyboard_search(world: &mut ApplicationWorld) {
    let label = keyboard_initial::search_label(world);
    let clear = world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.list == label.list
                && run.text.ends_with("Clear")
                && (run.center()[1] - label.center()[1]).abs() < 3.
        })
        .expect("expected Clear beside the keyboard search field");
    world.click(diagnostics::pointer_pixels([
        clear.maximum[0] - 6.,
        clear.center()[1],
    ]));
    world.pump(Vec::new());
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
    let search = keyboard_initial::search_label(&world).clone();
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
