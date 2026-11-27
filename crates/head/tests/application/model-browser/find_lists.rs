// SPDX-License-Identifier: GPL-2.0-only
//! The Find dialog's list of criteria: its rows, their controls and saved searches.
use super::find::{
    EXTRUSION_ROW, FILLET_ROW, SUPPRESSED_ROW, add_grouped_type, add_is_enabled, add_pending,
    add_type, choose_in_list, expect_control, expect_empty_input, expect_listed, grouped_types,
};
use super::*;

const HOLE_ROW: &str = "Type equals Hole";

fn three_criteria(world: &mut ApplicationWorld) {
    grouped_types(world);
    choose_in_list(world, "Property", "Suppress status");
    add_pending(world);
}
fn select_row(world: &mut ApplicationWorld, row: &str) {
    world.click_text(row);
    settle(world);
}
fn expect_controls(world: &ApplicationWorld, property: &str, value: &str) {
    expect_control(world, "Property", property);
    expect_control(world, "Condition", "equals");
    expect_control(world, "Value", value);
}
#[given("a part is open in the MCAD Software with the Find dialog open")]
fn find_on_part(world: &mut ApplicationWorld) {
    open_block(world);
    world.previous_document = Some(world.active_id());
    super::find::open_find(world);
}
#[given("the criteria list contains Type equals Extrusion OR Type equals Fillet")]
fn listed_grouped(world: &mut ApplicationWorld) {
    add_type(world, "Extrusion");
    add_grouped_type(world, "Fillet");
}
#[given("the criteria list contains Suppress status equals Suppressed")]
fn listed_suppressed(world: &mut ApplicationWorld) {
    choose_in_list(world, "Property", "Suppress status");
    add_pending(world);
}
#[when("I choose Type equals Hole and click Add to list")]
#[when("the user chooses Type equals Hole and clicks Add to list")]
fn add_hole(world: &mut ApplicationWorld) {
    add_type(world, "Hole");
}
#[then("the prompt closes and the existing three criteria remain in the list")]
fn three_remain(world: &mut ApplicationWorld) {
    world.expect_no_text("must be grouped");
    expect_listed(world, &[EXTRUSION_ROW, FILLET_ROW, SUPPRESSED_ROW]);
}
#[then("Type equals Hole is not added to the list")]
fn hole_absent(world: &mut ApplicationWorld) {
    world.expect_no_text(HOLE_ROW);
}
#[then("the property, condition and value controls still show Type equals Hole")]
fn hole_controls(world: &mut ApplicationWorld) {
    expect_controls(world, "Type", "Hole");
}
// Reading what is listed
#[given("the MCAD Software Find dialog contains these feature criteria in order:")]
fn table_criteria(world: &mut ApplicationWorld) {
    open_block(world);
    three_criteria(world);
}
#[when("I select the Suppress status criterion")]
#[when("the user selects the Suppress status criterion")]
fn select_suppressed(world: &mut ApplicationWorld) {
    select_row(world, SUPPRESSED_ROW);
}
#[then("the criterion controls show Suppress status, equals and Suppressed")]
fn suppressed_controls(world: &mut ApplicationWorld) {
    expect_controls(world, "Suppress status", "Suppressed");
}
#[when("I select the Fillet criterion")]
#[when("the user selects the Fillet criterion")]
fn select_fillet(world: &mut ApplicationWorld) {
    select_row(world, FILLET_ROW);
}
#[then("the criterion controls show Type, equals and Fillet")]
fn fillet_controls(world: &mut ApplicationWorld) {
    expect_controls(world, "Type", "Fillet");
}
#[then("Or is selected")]
fn or_chosen(world: &mut ApplicationWorld) {
    assert!(world.radio_selected("Or"), "expected Or selected");
}
#[when("I select the Extrusion criterion")]
#[when("the user selects the Extrusion criterion")]
fn select_extrusion(world: &mut ApplicationWorld) {
    select_row(world, EXTRUSION_ROW);
}
#[then("the criterion controls show Type, equals and Extrusion")]
fn extrusion_controls(world: &mut ApplicationWorld) {
    expect_controls(world, "Type", "Extrusion");
}
#[then("the three criteria remain listed")]
fn three_listed(world: &mut ApplicationWorld) {
    expect_listed(world, &[EXTRUSION_ROW, FILLET_ROW, SUPPRESSED_ROW]);
}
// A pending value
#[given("Type equals Extrusion is selected with And selected")]
fn extrusion_chosen(world: &mut ApplicationWorld) {
    select_row(world, EXTRUSION_ROW);
    assert!(world.radio_selected("And"), "expected And selected");
}
#[when("I choose Hole in the Value list")]
#[when("the user chooses Hole in the Value list")]
fn choose_hole(world: &mut ApplicationWorld) {
    choose_in_list(world, "Value", "Hole");
}
#[then("the Value field shows Hole and Add to list is enabled")]
#[then("the Value field shows Hole and adds to list is enabled")]
fn hole_pending(world: &mut ApplicationWorld) {
    expect_control(world, "Value", "Hole");
    assert!(add_is_enabled(world), "expected Add to list enabled");
}
#[then("the listed Extrusion and Fillet criteria remain unchanged")]
fn two_unchanged(world: &mut ApplicationWorld) {
    expect_listed(world, &[EXTRUSION_ROW, FILLET_ROW]);
    world.expect_no_text(HOLE_ROW);
}
#[then("the controls show Type, equals, Fillet and Or")]
fn fillet_and_or(world: &mut ApplicationWorld) {
    expect_controls(world, "Type", "Fillet");
    assert!(world.radio_selected("Or"), "expected Or selected");
}
#[then("the controls show Type, equals, Extrusion and And")]
fn extrusion_and_and(world: &mut ApplicationWorld) {
    expect_controls(world, "Type", "Extrusion");
    assert!(world.radio_selected("And"), "expected And selected");
}
#[then("Add to list is disabled")]
#[then("adds to list is disabled")]
fn add_disabled(world: &mut ApplicationWorld) {
    assert!(!add_is_enabled(world), "expected Add to list disabled");
}
// Saved searches
#[given("the MCAD Software Find dialog contains feature search criteria")]
fn with_criteria(world: &mut ApplicationWorld) {
    open_block(world);
    grouped_types(world);
}
#[when("I choose Save Search")]
#[when("the user chooses Save Search")]
fn choose_save(world: &mut ApplicationWorld) {
    world.click_text("Save Search");
    settle(world);
}
#[then("a prompt shows an empty search-name field")]
fn save_prompt(world: &mut ApplicationWorld) {
    world.expect_text("Search name");
    expect_empty_input(world, "Search name");
}
#[then("OK is disabled and Cancel is available")]
#[then("OK is disabled and cancels is available")]
fn ok_muted(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
    assert!(world.run_is_enabled("Cancel"), "expected Cancel enabled");
}
#[then("the Find dialog remains open with its criteria unchanged")]
#[then("the Find dialog remains open with its original criteria unchanged")]
fn criteria_kept(world: &mut ApplicationWorld) {
    world.expect_text("Find Now");
    world.expect_no_text("Search name");
    expect_listed(world, &[EXTRUSION_ROW, FILLET_ROW]);
}
#[given("at least one saved search is available")]
fn one_saved(world: &mut ApplicationWorld) {
    choose_save(world);
    world.type_text("Types");
    settle(world);
    world.click_text("OK");
    settle(world);
}
#[when("I choose Open Search")]
#[when("the user chooses Open Search")]
fn choose_open_search(world: &mut ApplicationWorld) {
    world.click_text("Open Search");
    settle(world);
}
#[then("a picker shows a saved-search list and Open, Cancel and Delete actions")]
#[then("a picker shows a saved-search list and opens, Cancel and Delete actions")]
fn picker(world: &mut ApplicationWorld) {
    expect_rows(world, &["Types", "Open", "Cancel", "Delete"]);
}
