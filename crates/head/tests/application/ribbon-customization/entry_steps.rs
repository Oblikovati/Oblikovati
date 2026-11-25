// SPDX-License-Identifier: GPL-2.0-only
//! Open Customize from the visible ribbon without replacing its active workspace.
use super::super::transaction_steps::{
    all_commands, expect_target_pending, target_is_empty, target_label,
};
use super::*;

const NORMAL_PROFILE: &str = "fixtures/preferences/ribbon-normal";
const RIBBON_SETTINGS_FILE: &str = "ribbon-customization.xml";

#[given(
    "the MCAD Software starts with a writable copy of `fixtures/preferences/ribbon-normal` as its configuration directory"
)]
fn start_with_normal_profile(world: &mut ApplicationWorld) {
    assert!(world.state().documents.is_empty());
    let source = super::super::super::scenario_root().join(NORMAL_PROFILE);
    let preferences = tempfile::tempdir().unwrap();
    for file in std::fs::read_dir(source).unwrap() {
        let file = file.unwrap();
        assert!(file.file_type().unwrap().is_file());
        let bytes = std::fs::read(file.path()).unwrap();
        std::fs::write(preferences.path().join(file.file_name()), bytes).unwrap();
    }
    assert!(preferences.path().join(RIBBON_SETTINGS_FILE).is_file());
    world.restart_with_preferences(preferences);
}

#[given("ribbon customization has not been opened earlier in the captured session")]
fn first_customization_opening(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Customize").is_none());
    let settings = world.preferences_directory().join(RIBBON_SETTINGS_FILE);
    if settings.exists() {
        let seed = super::super::super::scenario_root()
            .join(NORMAL_PROFILE)
            .join(RIBBON_SETTINGS_FILE);
        assert_eq!(
            std::fs::read(settings).unwrap(),
            std::fs::read(seed).unwrap()
        );
    }
}

#[when("the user clicks \"Customize\" in the Options panel")]
#[when("the user opens Customize from the Options panel")]
#[when("the user opens Customize from the Tools ribbon")]
fn customize_from_options(world: &mut ApplicationWorld) {
    world.expect_text("Options");
    open_tools_customization(world);
}

#[then("the Customize dialog opens on the Ribbon tab")]
fn ribbon_tab_displayed(world: &mut ApplicationWorld) {
    ribbon_tab_open(world);
}

#[then("the ribbon customization dialog opens")]
fn customization_open(world: &mut ApplicationWorld) {
    customization_remains_open(world);
}

#[then("the target selector shows \"Part | Tools\"")]
fn tools_target_displayed(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world), "Part|Tools");
}

#[then("the available-command category is \"All Commands\"")]
#[then("the command category is All Commands")]
fn all_commands_displayed(world: &mut ApplicationWorld) {
    all_commands(world);
}

#[then("the Part | Tools custom panel list is empty with no pending-change marker")]
#[then("the target is Part Tools with an empty custom panel list")]
fn tools_target_is_clean(world: &mut ApplicationWorld) {
    tools_target_displayed(world);
    target_is_empty(world);
    expect_target_pending(world, false);
}

#[then(
    "the Part | Tools custom panel list is empty beneath the \"Large\", \"Text\" and \"Name\" columns"
)]
fn empty_tools_columns(world: &mut ApplicationWorld) {
    tools_target_is_clean(world);
    for label in ["Large", "Text", "Name"] {
        world.run_in_window("Customize", label);
    }
}

#[then("\"Import\", \"Export\" and \"Close\" are available")]
fn clean_file_controls(world: &mut ApplicationWorld) {
    for label in ["Import", "Export", "Close"] {
        assert_eq!(world.run_in_window("Customize", label).alpha, u8::MAX);
    }
}

#[then("Close is available while Cancel and Apply are unavailable")]
#[then("Close is available while Cancel and Apply are disabled")]
#[then("\"Cancel\" and \"Apply\" are unavailable")]
#[then("cancels and Apply are unavailable because no edits are staged")]
fn clean_dialog_controls(world: &mut ApplicationWorld) {
    let title = if diagnostics::window_geometry("Customize").is_some() {
        "Customize"
    } else {
        "Document Settings"
    };
    assert_eq!(world.run_in_window(title, "Close").alpha, u8::MAX);
    for label in ["Cancel", "Apply"] {
        assert_ne!(world.run_in_window(title, label).alpha, u8::MAX);
    }
}

#[then("the dialog closes and the Tools ribbon remains visible")]
fn tools_remains_visible(world: &mut ApplicationWorld) {
    customization_closed(world);
    world.expect_text("Options");
    world.expect_text("Customize");
}

#[given("the View ribbon tab is selected")]
fn select_view(world: &mut ApplicationWorld) {
    world.click_text("View");
    view_remains_visible(world);
}

#[given("the Part | View custom target is empty")]
fn view_has_no_custom_panel(world: &mut ApplicationWorld) {
    view_remains_visible(world);
    no_user_commands(world);
}

#[when("the user opens the context menu on the Windows panel title")]
fn open_windows_title_menu(world: &mut ApplicationWorld) {
    let point = world.run("Windows").center();
    world.right_click(diagnostics::pointer_pixels(point));
    world.expect_text("Customize User Commands");
}

#[when("the user opens Customize User Commands from the Windows panel context menu")]
#[when("the user reopens Customize User Commands from the Windows panel context menu")]
fn customize_from_view(world: &mut ApplicationWorld) {
    open_windows_title_menu(world);
    choose_customize(world);
    view_target_displayed(world);
}

#[then("the target is \"Part | View\"")]
fn view_target_displayed(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world).trim_end_matches('*'), "Part|View");
}

#[then("the Part | View target is empty")]
fn view_target_empty(world: &mut ApplicationWorld) {
    view_target_displayed(world);
    target_is_empty(world);
    expect_target_pending(world, false);
}

#[when("the user double-clicks \"2D Sketches\" in All Commands")]
fn double_click_sketches_from_all(world: &mut ApplicationWorld) {
    all_commands(world);
    sketches_in_available_commands(world);
    double_click_sketches(world);
}

#[then("\"2D Sketches\" appears in the Part | View target")]
fn sketches_staged_on_view(world: &mut ApplicationWorld) {
    view_target_displayed(world);
    sketches_in_target(world);
}

#[then("\"2D Sketches\" is available in All Commands")]
fn sketches_available_in_all(world: &mut ApplicationWorld) {
    sketches_in_all_commands(world);
}

fn view_remains_visible(world: &mut ApplicationWorld) {
    world.expect_text("Windows");
    world.expect_text("User Interface");
}

#[when("the user opens Customize User Commands from the Create panel title context menu")]
fn customize_from_sketch(world: &mut ApplicationWorld) {
    super::super::sketch_target_steps::open_create_menu(world);
    choose_customize(world);
}

#[then("Ribbon, Keyboard and Marking Menu tabs are available")]
fn customization_tabs(world: &mut ApplicationWorld) {
    for tab in ["Ribbon", "Keyboard", "MarkingMenu"] {
        world.run_in_window("Customize", tab);
    }
}

#[when("the user closes the dialog without making changes")]
fn close_clean_dialog(world: &mut ApplicationWorld) {
    clean_dialog_controls(world);
    close_customization_button(world);
    customization_closed(world);
}

#[given("a part with a solid body is active in the MCAD Software")]
fn fixture_solid_remains_active(world: &mut ApplicationWorld) {
    assert_eq!(world.state().document, world.previous_document);
    super::super::super::workspace_layout::browser_feature::expect_solid_visible(world);
}

#[given("one face is selected")]
fn select_fixture_face(world: &mut ApplicationWorld) {
    let part = world.active_id();
    world.parts.push(part);
    super::super::super::viewport_entity_selection::click_visible_face(world);
    face_selection_displayed(world);
}

#[then("the selection indicator reports one face")]
fn face_selection_displayed(world: &mut ApplicationWorld) {
    assert!(world.state().model_face_chosen);
    assert!(world.frame.runs.iter().any(|run| run.text == "1Face"));
}
