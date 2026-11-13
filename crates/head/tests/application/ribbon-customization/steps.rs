// SPDX-License-Identifier: GPL-2.0-only
//! Native tests for staged user commands and their applied ribbon actions.
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};
#[path = "category_steps.rs"]
mod category_steps;
#[path = "custom_panel_title.rs"]
mod custom_panel_title;
#[path = "file_exchange.rs"]
mod file_exchange;
#[path = "macros.rs"]
mod macros;
#[path = "missing_file.rs"]
mod missing_file;
#[path = "target_steps.rs"]
mod target_steps;
#[path = "view_category.rs"]
mod view_category;

#[given("a part document is active in The MCAD Software")]
fn active_part(world: &mut ApplicationWorld) {
    world.add_part();
}
#[when("the user right-clicks a ribbon panel title")]
fn open_panel_menu(world: &mut ApplicationWorld) {
    let center = world.run("Create").center();
    world.right_click(diagnostics::pointer_pixels(center));
}
#[when("the user chooses \"Customize User Commands\"")]
fn choose_customize(world: &mut ApplicationWorld) {
    world.click_text("Customize User Commands");
    world.pump(Vec::new());
}
#[then("the customization dialog opens on the \"Ribbon\" tab")]
fn ribbon_tab_open(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Customize").is_some());
    assert!(
        world
            .window_text("Customize")
            .contains("Customizeusercommands")
    );
}
#[then("the available commands and target custom panel lists are displayed")]
fn command_lists_visible(world: &mut ApplicationWorld) {
    let content = world.window_text("Customize");
    assert!(content.contains("Availablecommands"));
    assert!(content.contains("Custompanel"));
    assert!(content.contains("AllCommands"));
}
#[then("the target tab is \"Part | 3D Model\"")]
fn model_target(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("Part|3DModel"));
}

#[given(
    "the ribbon customization dialog is open with an empty custom panel for \"Part | 3D Model\""
)]
fn empty_model_customization(world: &mut ApplicationWorld) {
    world.add_part();
    open_panel_menu(world);
    choose_customize(world);
    model_target(world);
}
#[when("the user selects \"2D Sketches\" in the available commands")]
fn select_available_sketches(world: &mut ApplicationWorld) {
    click_in_customize(world, "2DSketches");
}
#[when("the user clicks the right transfer button")]
fn transfer_to_target(world: &mut ApplicationWorld) {
    click_in_customize(world, ">");
}
#[then("\"2D Sketches\" appears in the custom panel list")]
fn sketches_in_target(world: &mut ApplicationWorld) {
    let command = world.run_in_window("Customize", "2DSketches");
    let target = world.run_in_window("Customize", "Custompanel");
    assert!(command.center()[0] >= target.minimum[0]);
}
#[then("the target tab selector indicates a pending change")]
fn target_has_pending_change(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("Part|3DModel*"));
}
#[then("\"OK\", \"Cancel\", and \"Apply\" are available")]
fn editing_actions_are_available(world: &mut ApplicationWorld) {
    for action in ["OK", "Cancel", "Apply"] {
        assert_eq!(world.run_in_window("Customize", action).alpha, u8::MAX);
    }
}
#[given("the ribbon customization dialog displays all commands")]
fn customization_all_commands(world: &mut ApplicationWorld) {
    empty_model_customization(world);
    assert!(world.window_text("Customize").contains("AllCommands"));
}
#[when("the user chooses \"Sketch Tab\" as the command category")]
fn select_sketch_category(world: &mut ApplicationWorld) {
    click_in_customize(world, "AllCommands");
    world.click_text("Sketch Tab");
}
#[then("the available list displays sketch commands including \"Circle (Center Point)\"")]
fn sketch_category_commands(world: &mut ApplicationWorld) {
    let content = world.window_text("Customize");
    assert!(content.contains("Circle(CenterPoint)"));
    assert!(!content.contains("3DAnnotations"));
    scroll_source_to_line(world);
    assert!(world.window_text("Customize").contains("Line"));
}
#[then("the selected target remains \"Part | 3D Model\"")]
fn target_still_model(world: &mut ApplicationWorld) {
    model_target(world);
}
#[given("a part is active in the MCAD Software with the Tools ribbon visible")]
#[given("a part is active in the MCAD Software with the full Tools ribbon visible")]
fn part_with_tools(world: &mut ApplicationWorld) {
    world.add_part();
    world.click_text("Tools");
}
#[given("the Ribbon customization dialog has an initially empty Part Tools custom panel")]
#[given("the Ribbon customization dialog shows All Commands and an empty Part Tools target")]
#[given("the ribbon customization dialog has an empty Part Tools custom panel")]
#[given("the Ribbon customization dialog shows an empty Part Tools custom panel")]
#[given("ribbon customization shows an initially empty Part Tools target")]
#[given("the ribbon customization dialog shows an empty Part Tools custom panel")]
fn empty_tools_customization(world: &mut ApplicationWorld) {
    open_tools_customization(world);
}
#[when("the user stages \"2D Sketches\" in the custom panel")]
#[when("the user adds \"2D Sketches\" to the custom panel without applying")]
fn stage_sketch_command(world: &mut ApplicationWorld) {
    stage_sketch_visibility(world);
}
#[then("the target indicates a pending change")]
#[then("the target still indicates a pending change")]
#[then("the target is marked as changed")]
fn tools_pending(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("Part|Tools*"));
}
#[then("the customization dialog closes")]
fn customization_closed(_world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Customize").is_none());
}
#[then("no User Commands panel is displayed on the Tools ribbon")]
#[then("no User Commands panel appears on the Tools ribbon")]
#[then("no User Commands panel is visible on the Tools ribbon")]
fn no_user_commands(world: &mut ApplicationWorld) {
    world.expect_no_text("User Commands");
}
#[when("the user reopens ribbon customization from Tools")]
#[when("the user reopens customization from the Tools ribbon")]
#[when("the user reopens Ribbon customization")]
fn reopen_tools_customization(world: &mut ApplicationWorld) {
    open_tools_customization(world);
}
#[then("the Part Tools custom panel list is empty")]
#[then("the Part Tools target list is empty")]
#[then("the custom panel list is empty")]
fn empty_tools_target(world: &mut ApplicationWorld) {
    let command = world.run_in_window("Customize", "2DSketches");
    let target = world.run_in_window("Customize", "Custompanel");
    assert!(command.center()[0] < target.minimum[0]);
}
#[then("\"2D Sketches\" is available in the All Commands list")]
#[given("\"2D Sketches\" is available in the source list")]
#[then("\"2D Sketches\" is available in the source list again")]
fn sketches_in_available_commands(world: &mut ApplicationWorld) {
    empty_tools_target(world);
    assert!(world.window_text("Customize").contains("AllCommands"));
}
#[when("the user double-clicks \"2D Sketches\" in the source list")]
#[when("the user double-clicks \"2D Sketches\" in the available list")]
#[when("the user double-clicks \"2D Sketches\" in the custom panel list")]
fn double_click_sketches(world: &mut ApplicationWorld) {
    let center = world.run_in_window("Customize", "2DSketches").center();
    world.double_click(diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}
#[then("\"2D Sketches\" moves to the custom panel list")]
fn double_click_moved_sketches(world: &mut ApplicationWorld) {
    sketches_in_target(world);
}
#[when("the user clicks \"Cancel\"")]
#[when("the user clicks Cancel")]
fn cancel_customization(world: &mut ApplicationWorld) {
    for title in [
        "Import Customization Settings",
        "Export Customization Settings",
    ] {
        if let Some((position, size)) = diagnostics::window_geometry(title) {
            let center = world
                .frame
                .runs
                .iter()
                .rev()
                .find(|run| {
                    let point = run.center();
                    run.text == "Cancel"
                        && (0..2).all(|axis| {
                            point[axis] >= position[axis]
                                && point[axis] <= position[axis] + size[axis]
                        })
                })
                .expect("expected the file dialog's Cancel button")
                .center();
            world.click(diagnostics::pointer_pixels(center));
            world.pump(Vec::new());
            return;
        }
    }
    click_in_customize(world, "Cancel");
}
#[given("the ribbon customization dialog is open in The MCAD Software")]
fn customization_open_for_file(world: &mut ApplicationWorld) {
    empty_model_customization(world);
}
#[when("the user clicks \"Export\"")]
fn export_customization(world: &mut ApplicationWorld) {
    click_in_customize(world, "Export");
}
#[when("the user clicks \"Import\"")]
fn import_customization(world: &mut ApplicationWorld) {
    click_in_customize(world, "Import");
}
#[then(expr = "a {string} file dialog opens")]
fn settings_dialog_open(_world: &mut ApplicationWorld, title: String) {
    assert!(diagnostics::window_geometry(&title).is_some());
}
#[then("the file type is \"XML Files (*.xml)\"")]
fn xml_file_type(world: &mut ApplicationWorld) {
    world.expect_text("XML Files (*.xml)");
}
#[then(expr = "the dialog offers {string} and \"Cancel\"")]
fn settings_dialog_buttons(world: &mut ApplicationWorld, accept: String) {
    world.expect_text(&accept);
    world.expect_text("Cancel");
}
#[then("the file dialog closes")]
fn settings_dialog_closed(_world: &mut ApplicationWorld) {
    for title in [
        "Import Customization Settings",
        "Export Customization Settings",
    ] {
        assert!(diagnostics::window_geometry(title).is_none());
    }
}
#[then("the ribbon customization dialog remains open")]
fn customization_remains_open(_world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Customize").is_some());
}
#[then("the target no longer indicates a pending change")]
#[then("the target is not marked as changed")]
#[then("the target has no pending-change marker")]
#[then("the Part Tools target has no pending-change marker")]
fn tools_not_pending(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("Part|Tools"));
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
}
#[then("the dialog offers OK, Cancel, and Apply")]
fn pending_dialog_actions(world: &mut ApplicationWorld) {
    editing_actions_are_available(world);
}
#[when("the user clicks the title-bar close control")]
fn close_customization_title(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry("Customize").unwrap();
    world.click(diagnostics::pointer_pixels([
        position[0] + size[0] - 13.,
        position[1] + 10.,
    ]));
    world.pump(Vec::new());
}
#[when("the user clicks Close")]
fn close_customization_button(world: &mut ApplicationWorld) {
    click_in_customize(world, "Close");
}
#[then("\"2D Sketches\" is available in the source list")]
fn available_again(world: &mut ApplicationWorld) {
    sketches_in_available_commands(world);
}
#[given("All Commands lists \"2D Sketches\" immediately before \"3D Annotations\"")]
fn available_rows_in_order(world: &mut ApplicationWorld) {
    let first = world.run_in_window("Customize", "2DSketches");
    let second = world.run_in_window("Customize", "3DAnnotations");
    assert!(first.center()[1] < second.center()[1]);
}
#[when("the user selects \"2D Sketches\" in the available commands list")]
#[when("the user selects \"2D Sketches\" in the custom panel list")]
fn select_sketches_row(world: &mut ApplicationWorld) {
    click_in_customize(world, "2DSketches");
}
#[when("the user presses Shift+Down")]
fn shift_down(world: &mut ApplicationWorld) {
    use oblikovati_viewport::{
        ViewportEvent, ViewportInput, ViewportModifiers, ViewportPhysicalKey,
    };
    let key = ViewportPhysicalKey::DownArrow;
    let event = |pressed, modifiers| {
        ViewportEvent::Input(ViewportInput::PhysicalKey {
            key,
            layout_key: key,
            pressed,
            command: None,
            modifiers,
        })
    };
    world.pump(vec![
        event(
            true,
            ViewportModifiers {
                shift: true,
                ..Default::default()
            },
        ),
        event(false, Default::default()),
    ]);
}
#[then("both commands are selected in the available list")]
fn available_rows_selected(world: &mut ApplicationWorld) {
    world.hover([800, 900]);
    assert!(world.row_accented_beside("2D Sketches", "<Separator>"));
    assert!(world.row_accented_beside("3D Annotations", "<Separator>"));
}
#[then("the custom panel list contains \"2D Sketches\" followed by \"3D Annotations\"")]
fn target_rows_in_order(world: &mut ApplicationWorld) {
    sketches_in_target(world);
    let first = world.run_in_window("Customize", "2DSketches");
    let second = world.run_in_window("Customize", "3DAnnotations");
    assert!(first.center()[1] < second.center()[1]);
}
#[then("neither command remains in the available list")]
fn both_rows_in_target(world: &mut ApplicationWorld) {
    let target = world.run_in_window("Customize", "Custompanel").minimum[0];
    for label in ["2DSketches", "3DAnnotations"] {
        assert!(world.run_in_window("Customize", label).center()[0] >= target);
    }
}
#[when("the user clicks the left transfer button")]
fn transfer_to_source(world: &mut ApplicationWorld) {
    click_in_customize(world, "<");
}
#[then("both commands return to the available list")]
fn both_rows_in_source(world: &mut ApplicationWorld) {
    let target = world.run_in_window("Customize", "Custompanel").minimum[0];
    for label in ["2DSketches", "3DAnnotations"] {
        assert!(world.run_in_window("Customize", label).center()[0] < target);
    }
}
#[given("the user has staged \"2D Sketches\" followed by \"3D Annotations\"")]
#[given("2D Sketches and 3D Annotations are staged on Part Tools in that order")]
fn stage_two_commands(world: &mut ApplicationWorld) {
    stage_sketch_visibility(world);
    click_in_customize(world, "3DAnnotations");
    click_in_customize(world, ">");
}
#[given("the Large option is clear for both commands")]
fn both_large_clear(world: &mut ApplicationWorld) {
    assert!(!row_option_checked(world, "2DSketches", "Large"));
    assert!(!row_option_checked(world, "3DAnnotations", "Large"));
}
#[given("both commands have Large and Text unchecked")]
#[then("both commands still have Large and Text unchecked")]
fn both_options_clear(world: &mut ApplicationWorld) {
    for row in ["2DSketches", "3DAnnotations"] {
        for option in ["Large", "Text"] {
            assert!(!row_option_checked(world, row, option));
        }
    }
}
#[when("I click the 2D Sketches name in the staged target list")]
fn click_first_staged(world: &mut ApplicationWorld) {
    click_in_customize(world, "2DSketches");
}
#[when("I press Ctrl+A while the target list has focus")]
fn select_all_staged(world: &mut ApplicationWorld) {
    world.key(oblikovati_viewport::ViewportPhysicalKey::A, true);
}
#[then("both staged command rows are highlighted")]
fn staged_rows_selected(world: &mut ApplicationWorld) {
    both_target_rows_selected(world);
}
#[then("Part Tools remains marked as having pending changes")]
fn tools_remains_pending(world: &mut ApplicationWorld) {
    tools_pending(world);
}
#[when("I cancel ribbon customization")]
fn cancel_from_keyboard_scenario(world: &mut ApplicationWorld) {
    click_in_customize(world, "Cancel");
}
#[when("the user selects \"2D Sketches\" and presses Shift+Down")]
fn select_two_rows(world: &mut ApplicationWorld) {
    click_in_customize(world, "2DSketches");
    shift_down(world);
}
#[when("the user checks Large on the \"2D Sketches\" row")]
fn check_sketches_large(world: &mut ApplicationWorld) {
    toggle_row_option(world, "2DSketches", "Large");
}
#[then("Large is checked for \"2D Sketches\"")]
fn sketches_large_checked(world: &mut ApplicationWorld) {
    assert!(row_option_checked(world, "2DSketches", "Large"));
}
#[then("Large remains clear for \"3D Annotations\"")]
fn annotations_large_clear(world: &mut ApplicationWorld) {
    assert!(!row_option_checked(world, "3DAnnotations", "Large"));
}
#[then("both rows remain selected")]
fn both_target_rows_selected(world: &mut ApplicationWorld) {
    world.hover([800, 900]);
    let first = world.run_in_window("Customize", "2DSketches").clone();
    let second = world.run_in_window("Customize", "3DAnnotations").clone();
    let x = (first.minimum[0] + 100.) as usize;
    let blank = world.pixel_at([x, (second.center()[1] + 30.) as usize]);
    for row in [first, second] {
        assert_ne!(world.pixel_at([x, row.center()[1] as usize]), blank);
    }
}

fn click_in_customize(world: &mut ApplicationWorld, label: &str) {
    if label == "Line" && !world.window_text("Customize").contains("Line") {
        scroll_source_to_line(world);
    }
    let center = world.run_in_window("Customize", label).center();
    world.click(diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}
fn scroll_source_to_line(world: &mut ApplicationWorld) {
    use oblikovati_viewport::{ViewportEvent, ViewportInput};
    let at = world
        .run_in_window("Customize", "<Separator>")
        .center()
        .map(|value| value as i32);
    world.hover(at);
    world.pump(vec![ViewportEvent::Input(ViewportInput::Wheel {
        notches: -3,
        position: at,
    })]);
    world.pump(Vec::new());
}
fn open_tools_customization(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.click_text("Customize");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Customize").is_some());
    assert!(world.window_text("Customize").contains("Part|Tools"));
}
fn stage_sketch_visibility(world: &mut ApplicationWorld) {
    click_in_customize(world, "2DSketches");
    click_in_customize(world, ">");
    assert!(world.window_text("Customize").contains("Part|Tools*"));
}
fn toggle_row_option(world: &mut ApplicationWorld, row: &str, option: &str) {
    let column = world.run_in_window("Customize", option).center()[0];
    let height = world.run_in_window("Customize", row).center()[1];
    world.click(diagnostics::pointer_pixels([column, height]));
    world.pump(Vec::new());
}
fn settings_file_choice(
    world: &mut ApplicationWorld,
    title: &str,
    path: &std::path::Path,
    action: &str,
) {
    enter_settings_file(world, title, path, action);
    assert!(diagnostics::window_geometry(title).is_none());
}
fn enter_settings_file(
    world: &mut ApplicationWorld,
    title: &str,
    path: &std::path::Path,
    action: &str,
) {
    type_settings_file(world, title, path);
    click_settings_button(world, title, action);
}
fn type_settings_file(world: &mut ApplicationWorld, title: &str, path: &std::path::Path) {
    assert!(diagnostics::window_geometry(title).is_some());
    let label = world.run_in_window(title, "Filename").clone();
    world.click(diagnostics::pointer_pixels([
        label.maximum[0] + 60.,
        label.center()[1],
    ]));
    world.type_text(path.to_str().unwrap());
}
fn click_settings_button(world: &mut ApplicationWorld, title: &str, action: &str) {
    let button = world.run_in_window(title, action).center();
    world.click(diagnostics::pointer_pixels(button));
    world.pump(Vec::new());
}
fn row_option_checked(world: &ApplicationWorld, row: &str, option: &str) -> bool {
    let column = world.run_in_window("Customize", option).center()[0];
    let height = world.run_in_window("Customize", row).center()[1];
    world.frame.marks.iter().any(|mark| {
        mark.bright
            && (mark.position[0] - column).abs() < 10.
            && (mark.position[1] - height).abs() < 10.
    })
}
#[path = "native_tests.rs"]
mod native_tests;
