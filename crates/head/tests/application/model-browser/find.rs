// SPDX-License-Identifier: GPL-2.0-only
//! The Find dialog: its lists, adding criteria and searching.
use super::*;

pub(super) const EXTRUSION_ROW: &str = "Type equals Extrusion";
pub(super) const FILLET_ROW: &str = "OR Type equals Fillet";
pub(super) const SUPPRESSED_ROW: &str = "Suppress status equals Suppressed";

pub(super) fn open_find(world: &mut ApplicationWorld) {
    open_browser_menu(world);
    choose_menu_row(world, "Find");
    settle(world);
    world.expect_text("Find Now");
}
/// A combo's button sits left of its label.
pub(super) fn open_list(world: &mut ApplicationWorld, label: &str) {
    let field = world.run(label).clone();
    world.click([field.minimum[0] as i32 - 40, field.center()[1] as i32]);
    settle(world);
}
pub(super) fn choose_in_list(world: &mut ApplicationWorld, label: &str, row: &str) {
    open_list(world, label);
    world.click_text(row);
    settle(world);
}
pub(super) fn add_pending(world: &mut ApplicationWorld) {
    world.click_text("Add to list");
    settle(world);
}
pub(super) fn add_type(world: &mut ApplicationWorld, value: &str) {
    choose_in_list(world, "Property", "Type");
    choose_in_list(world, "Value", value);
    add_pending(world);
}
pub(super) fn add_grouped_type(world: &mut ApplicationWorld, value: &str) {
    add_type(world, value);
    world.expect_text("must be grouped");
    world.click_text("Yes");
    settle(world);
}
/// The list of the dialog's fixtures: the grouped Extrusion and Fillet.
pub(super) fn grouped_types(world: &mut ApplicationWorld) {
    open_find(world);
    add_type(world, "Extrusion");
    add_grouped_type(world, "Fillet");
}
pub(super) fn expect_listed(world: &ApplicationWorld, rows: &[&str]) {
    let text = all_text(world);
    for row in rows {
        assert!(
            text.contains(&row.replace(' ', "")),
            "expected {row:?} in {text:?}"
        );
    }
}
/// A control's text sits on its label's line, to the left.
pub(super) fn expect_control(world: &ApplicationWorld, label: &str, value: &str) {
    let field = world.run(label).clone();
    let line: String = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            (run.center()[1] - field.center()[1]).abs() < 8. && run.maximum[0] <= field.maximum[0]
        })
        .map(|run| run.text.as_str())
        .collect();
    assert!(
        line.contains(&value.replace(' ', "")),
        "{label} line {line:?}: expected {value}"
    );
}
pub(super) fn add_is_enabled(world: &ApplicationWorld) -> bool {
    world.run_is_enabled("Add to list")
}
#[when("I choose Find from the model browser menu")]
#[when("the user chooses Find from the model browser menu")]
fn choose_find(world: &mut ApplicationWorld) {
    open_find(world);
}
#[then("a search criteria dialog opens over the workspace")]
fn dialog_up(world: &mut ApplicationWorld) {
    world.expect_text("Find: Features");
    expect_rows(
        world,
        &["Property", "Condition", "Value", "Find Now", "Close"],
    );
}
fn close_lists(world: &mut ApplicationWorld) {
    world.escape();
    settle(world);
}
#[then("its object type list offers Sketches and Features")]
fn object_rows(world: &mut ApplicationWorld) {
    open_list(world, "Find");
    expect_rows(world, &["Sketches", "Features"]);
    close_lists(world);
}
#[when("I select Features and open the Property list")]
#[when("the user selects Features and opens the Property list")]
fn features_property(world: &mut ApplicationWorld) {
    choose_in_list(world, "Find", "Features");
    open_list(world, "Property");
}
#[then("the list offers Name, Note contents, Note title, Suppress status and Type")]
fn feature_properties(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &[
            "Name",
            "Note contents",
            "Note title",
            "Suppress status",
            "Type",
        ],
    );
}
#[when("I select Name and open the Condition list")]
#[when("the user selects Name and opens the Condition list")]
fn name_condition(world: &mut ApplicationWorld) {
    world.click_text("Name");
    settle(world);
    open_list(world, "Condition");
}
#[then("the list offers any value, begins with, ends with, includes, is exactly and is not")]
fn text_conditions(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &[
            "any value",
            "begins with",
            "ends with",
            "includes",
            "is exactly",
            "is not",
        ],
    );
}
#[when("I dismiss the list and close the dialog")]
#[when("the user dismisses the list and closes the dialog")]
fn dismiss_and_close(world: &mut ApplicationWorld) {
    close_lists(world);
    world.click_text("Close");
    settle(world);
}
#[then("the model browser and part remain visible")]
#[then("the model browser and part are visible")]
#[then("the part workspace is visible")]
fn part_stays(world: &mut ApplicationWorld) {
    world.expect_no_text("Find Now");
    world.expect_text("Origin");
    expect_solid_visible(world);
}
// Sketch criteria
#[when("I choose Sketches in the object type list")]
fn choose_sketches(world: &mut ApplicationWorld) {
    choose_in_list(world, "Find", "Sketches");
    world.expect_text("Find: Sketches");
}
#[when("I open the Property list")]
fn open_property(world: &mut ApplicationWorld) {
    open_list(world, "Property");
}
#[then("the list offers Name, Note contents, Note title and Sketch Parameter Name")]
fn sketch_properties(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &[
            "Name",
            "Note contents",
            "Note title",
            "Sketch Parameter Name",
        ],
    );
}
#[when("I choose Sketch Parameter Name")]
fn choose_parameter(world: &mut ApplicationWorld) {
    world.click_text("Sketch Parameter Name");
    settle(world);
}
#[then("the Condition field displays contains")]
fn condition_contains(world: &mut ApplicationWorld) {
    expect_control(world, "Condition", "contains");
}
#[then("the Value field is empty and Add to list is disabled")]
fn value_empty(world: &mut ApplicationWorld) {
    assert!(!add_is_enabled(world), "expected Add to list disabled");
}
#[when("I open the Condition list")]
fn open_condition(world: &mut ApplicationWorld) {
    open_list(world, "Condition");
}
#[then("it offers contains and does not contain")]
fn parameter_conditions(world: &mut ApplicationWorld) {
    expect_rows(world, &["contains", "does not contain"]);
}
// Searching by type
#[given("the Find criteria list is empty")]
fn nothing_listed(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    world.expect_no_text(EXTRUSION_ROW);
}
#[when("I select Features as the object type")]
fn select_features(world: &mut ApplicationWorld) {
    choose_in_list(world, "Find", "Features");
}
#[when("I select Type as the property and Extrusion as the value")]
fn select_type_extrusion(world: &mut ApplicationWorld) {
    choose_in_list(world, "Property", "Type");
    choose_in_list(world, "Value", "Extrusion");
}
#[then("the condition displays equals")]
fn condition_equals(world: &mut ApplicationWorld) {
    expect_control(world, "Condition", "equals");
}
#[when("I click Add to list")]
fn click_add(world: &mut ApplicationWorld) {
    add_pending(world);
}
#[then("the criteria list contains Type equals Extrusion")]
#[then("the criteria list shows Type equals Extrusion")]
fn extrusion_listed(world: &mut ApplicationWorld) {
    expect_listed(world, &[EXTRUSION_ROW]);
}
#[when("I click Find Now")]
fn click_find_now(world: &mut ApplicationWorld) {
    world.click_text("Find Now");
    settle(world);
}
#[then("the extrusion row is selected in the model browser")]
fn extrusion_row(world: &mut ApplicationWorld) {
    assert!(row_accent(world, FEATURE), "expected the row's accent");
}
#[then("the Find dialog remains open")]
fn find_open(world: &mut ApplicationWorld) {
    world.expect_text("Find Now");
}
#[when("I close the dialog")]
#[when("I close the Find dialog")]
fn close_find(world: &mut ApplicationWorld) {
    world.click_text("Close");
    settle(world);
}
#[then("the extrusion remains selected and the block edges remain highlighted")]
#[then("the extrusion row is selected and the block edges are highlighted")]
fn extrusion_stays(world: &mut ApplicationWorld) {
    assert!(row_accent(world, FEATURE), "expected the row's accent");
    expect_outline(world, true);
}
// Grouping
#[given("the Find dialog contains the criterion Type equals Extrusion")]
fn extrusion_criterion(world: &mut ApplicationWorld) {
    open_find(world);
    add_type(world, "Extrusion");
}
#[given("Features is the selected object type and And is selected for additional criteria")]
#[given("And is selected for additional criteria")]
#[then("And is selected")]
fn and_chosen(world: &mut ApplicationWorld) {
    assert!(world.radio_selected("And"), "expected And selected");
}
#[when("I choose Type equals Fillet and click Add to list")]
fn add_fillet(world: &mut ApplicationWorld) {
    add_type(world, "Fillet");
}
#[then("a prompt says criteria using the same property must be grouped")]
fn prompt_says(world: &mut ApplicationWorld) {
    world.expect_text("must be grouped");
}
#[then("the prompt offers to add the criterion using the OR operator")]
#[then("a prompt offers to group the additional type criterion using OR")]
fn prompt_offers(world: &mut ApplicationWorld) {
    world.expect_text("OR operator");
    expect_rows(world, &["Yes", "No"]);
}
#[when("I choose Yes")]
fn choose_yes(world: &mut ApplicationWorld) {
    world.click_text("Yes");
    settle(world);
}
#[then("the next line shows OR Type equals Fillet")]
fn fillet_below(world: &mut ApplicationWorld) {
    let first = world.run(EXTRUSION_ROW).center()[1];
    let second = world.run(FILLET_ROW).center()[1];
    assert!(second > first, "fillet row {second} beneath {first}");
}
#[when("I close and reopen the Find dialog")]
fn reopen_find(world: &mut ApplicationWorld) {
    close_find(world);
    open_find(world);
}
#[then("both criteria remain in the list")]
fn both_listed(world: &mut ApplicationWorld) {
    expect_listed(world, &[EXTRUSION_ROW, FILLET_ROW]);
}
#[when("I close the Find dialog and click empty viewport space")]
fn close_and_click(world: &mut ApplicationWorld) {
    close_find(world);
    click_empty_viewport(world);
}
#[then("the highlight disappears and the part remains visible")]
fn highlight_gone(world: &mut ApplicationWorld) {
    expect_outline(world, false);
    expect_solid_visible(world);
}
// No match
#[given("the Find dialog contains Type equals Extrusion OR Type equals Fillet")]
fn grouped_list(world: &mut ApplicationWorld) {
    grouped_types(world);
}
#[given("the MCAD Software Find dialog lists Type equals Extrusion and OR Type equals Fillet")]
fn part_with_grouped_list(world: &mut ApplicationWorld) {
    open_block(world);
    grouped_types(world);
}
#[when("I select Suppress status as the property")]
fn select_suppress(world: &mut ApplicationWorld) {
    choose_in_list(world, "Property", "Suppress status");
}
#[then("the condition is equals and the value is Suppressed")]
fn suppress_controls(world: &mut ApplicationWorld) {
    expect_control(world, "Condition", "equals");
    expect_control(world, "Value", "Suppressed");
}
#[then("the criteria list includes Suppress status equals Suppressed")]
fn suppressed_listed(world: &mut ApplicationWorld) {
    expect_listed(world, &[SUPPRESSED_ROW]);
}
#[then("an information message says that no objects matched the criteria")]
fn no_match(world: &mut ApplicationWorld) {
    world.expect_text("No objects matched");
}
#[when("I acknowledge the message")]
fn acknowledge(world: &mut ApplicationWorld) {
    world.click_text("OK");
    settle(world);
}
#[then("the Find dialog remains open with the criteria displayed")]
fn open_with_criteria(world: &mut ApplicationWorld) {
    world.expect_no_text("No objects matched");
    expect_listed(world, &[EXTRUSION_ROW, FILLET_ROW, SUPPRESSED_ROW]);
}
