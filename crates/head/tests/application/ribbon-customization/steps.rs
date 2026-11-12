// SPDX-License-Identifier: GPL-2.0-only
//! Native tests for staged user commands and their applied ribbon actions.
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};

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
    assert!(content.contains("Line"));
    assert!(!content.contains("3DAnnotations"));
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
fn reopen_tools_customization(world: &mut ApplicationWorld) {
    open_tools_customization(world);
}
#[then("the Part Tools custom panel list is empty")]
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
    click_in_customize(world, "Cancel");
}
#[then("the target no longer indicates a pending change")]
#[then("the target is not marked as changed")]
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
    assert!(world.row_accented_beside("2D Sketches", "Line"));
    assert!(world.row_accented_beside("3D Annotations", "Line"));
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
    let center = world.run_in_window("Customize", label).center();
    world.click(diagnostics::pointer_pixels(center));
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
fn row_option_checked(world: &ApplicationWorld, row: &str, option: &str) -> bool {
    let column = world.run_in_window("Customize", option).center()[0];
    let height = world.run_in_window("Customize", row).center()[1];
    world.frame.marks.iter().any(|mark| {
        mark.bright
            && (mark.position[0] - column).abs() < 10.
            && (mark.position[1] - height).abs() < 10.
    })
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
        world.run_in_window("Customize", "Line").center()[0]
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
fn escape_closes_the_target_list_before_the_customization_dialog() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "Part|Tools*");
    world.expect_text("Part | Annotate");
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    assert!(diagnostics::window_geometry("Customize").is_some());
    assert!(world.window_text("Customize").contains("Part|Tools*"));
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    assert!(diagnostics::window_geometry("Customize").is_none());
    world.close();
}
