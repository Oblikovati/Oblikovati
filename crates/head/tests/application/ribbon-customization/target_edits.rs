// SPDX-License-Identifier: GPL-2.0-only
//! Exercise native panel edits, including repeated rows with the same caption.
use super::list_steps::source_rows;
use super::transaction_steps::{
    add_command, choose_target, expect_target_pending, target_is_empty, target_row_runs,
    target_rows,
};
use super::*;

const CENTER_CIRCLE: &str = "Circle(CenterPoint)";

#[given(
    "the ribbon customization dialog has staged \"2D Sketches\" followed by \"3D Annotations\""
)]
fn stage_model_pair(world: &mut ApplicationWorld) {
    empty_model_customization(world);
    for command in ["2DSketches", "3DAnnotations"] {
        add_command(world, command);
    }
    assert_eq!(target_rows(world), ["2DSketches", "3DAnnotations"]);
}

#[when("the user selects \"3D Annotations\" and clicks the up arrow")]
fn move_annotations_up(world: &mut ApplicationWorld) {
    click_target_row(world, "3DAnnotations");
    click_in_customize(world, "Up");
}

#[then("\"3D Annotations\" appears above \"2D Sketches\"")]
fn annotations_precede_sketches(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), ["3DAnnotations", "2DSketches"]);
}

#[when("the user selects the \"Large\" checkbox for \"3D Annotations\"")]
fn select_annotations_large(world: &mut ApplicationWorld) {
    assert!(!row_option_checked(world, "3DAnnotations", "Large"));
    toggle_row_option(world, "3DAnnotations", "Large");
}

#[then("its \"Large\" checkbox is checked")]
fn annotations_large_is_checked(world: &mut ApplicationWorld) {
    assert!(row_option_checked(world, "3DAnnotations", "Large"));
}

#[when("the user removes \"3D Annotations\" with the left transfer button")]
fn remove_annotations(world: &mut ApplicationWorld) {
    remove_target_row(world, "3DAnnotations");
}

#[then("\"3D Annotations\" returns to the available commands list")]
fn annotations_return_to_source(world: &mut ApplicationWorld) {
    source_row(world, "3DAnnotations");
}

#[then("\"2D Sketches\" remains in the custom panel list")]
fn sketches_are_the_only_target_row(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), ["2DSketches"]);
}

#[when("the user adds \"Circle (Center Point)\" and a separator to the custom panel list")]
fn add_circle_and_separator(world: &mut ApplicationWorld) {
    click_in_customize(world, CENTER_CIRCLE);
    transfer_to_target(world);
    click_source_separator(world);
    transfer_to_target(world);
}

#[then("the custom panel list contains the circle command followed by a separator")]
fn circle_before_separator(world: &mut ApplicationWorld) {
    let rows = target_rows(world);
    assert_eq!(rows, [circle_target_label(world), "<Separator>"]);
    assert!(
        !source_rows(world)
            .iter()
            .any(|row| row.text == CENTER_CIRCLE),
        "the staged circle should leave the source list"
    );
}

#[when("the user removes the separator and the circle command from the custom panel list")]
fn remove_separator_and_circle(world: &mut ApplicationWorld) {
    remove_target_row(world, "<Separator>");
    let circle = circle_target_label(world).to_owned();
    remove_target_row(world, &circle);
    source_row(world, CENTER_CIRCLE);
}

fn circle_target_label(world: &ApplicationWorld) -> &str {
    // The target Name column can clip the end of the circle caption.
    target_rows(world)
        .into_iter()
        .find(|row| row.starts_with("Circle(Center") && CENTER_CIRCLE.starts_with(row))
        .expect("expected the center-point circle in the target list")
}

#[then("the \"User Commands\" panel appears on the active ribbon")]
fn user_panel_is_visible(world: &mut ApplicationWorld) {
    world.expect_text("User Commands");
}

#[given(
    "the ribbon customization dialog has empty custom panel lists for \"Part | Sketch\" and \"Part | Annotate\""
)]
fn empty_sketch_and_annotate(world: &mut ApplicationWorld) {
    empty_model_customization(world);
    for tab in ["Sketch", "Annotate"] {
        choose_target(world, tab);
        target_is_empty(world);
        expect_target_pending(world, false);
    }
}

#[when("the user selects \"Part | Sketch\" as the target")]
#[when("the user returns to the \"Part | Sketch\" target")]
fn select_sketch_target(world: &mut ApplicationWorld) {
    choose_target(world, "Sketch");
}

#[when("the user selects \"Part | Annotate\" as the target")]
fn select_annotate_target(world: &mut ApplicationWorld) {
    choose_target(world, "Annotate");
}

#[when("the user adds \"2D Sketches\" to the custom panel list")]
fn add_sketches_to_current_target(world: &mut ApplicationWorld) {
    add_command(world, "2DSketches");
}

#[then("\"2D Sketches\" is still in its custom panel list")]
#[then("\"2D Sketches\" remains in the Part Tools target list")]
fn target_retains_sketches(world: &mut ApplicationWorld) {
    sketches_in_target(world);
}

#[then("the target selector indicates a pending change")]
fn current_target_pending(world: &mut ApplicationWorld) {
    expect_target_pending(world, true);
}

#[then("the target selector no longer indicates a pending change")]
#[then("the target selector does not indicate a pending change")]
fn current_target_clean(world: &mut ApplicationWorld) {
    expect_target_pending(world, false);
}

#[when("the user clicks \"OK\"")]
fn apply_and_close(world: &mut ApplicationWorld) {
    click_in_customize(world, "OK");
}

#[when("the user selects \"<Separator>\" in the available list")]
fn click_source_separator(world: &mut ApplicationWorld) {
    let at = source_row(world, "<Separator>");
    world.click(at);
    world.pump(Vec::new());
}

#[when("the user double-clicks \"<Separator>\" in the available commands list")]
#[when("the user double-clicks \"<Separator>\" in the available commands list again")]
fn double_click_source_separator(world: &mut ApplicationWorld) {
    double_click_source(world, "<Separator>");
}

#[then("the custom panel list contains one separator")]
#[then("the Part Tools custom panel list contains one separator")]
#[then("one separator row remains in the custom panel list")]
fn single_separator(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), ["<Separator>"]);
}

#[then("the custom panel list contains two separator rows")]
fn two_separators(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), ["<Separator>", "<Separator>"]);
}

#[then("\"<Separator>\" remains in the available commands list")]
#[then("the separator remains available in the source list")]
fn reusable_separator(world: &mut ApplicationWorld) {
    source_row(world, "<Separator>");
}

#[when("the user double-clicks the first separator row in the custom panel list")]
#[when("the user double-clicks the separator in the custom panel list")]
fn double_click_first_target_separator(world: &mut ApplicationWorld) {
    let at = target_row(world, "<Separator>");
    world.double_click(at);
    world.pump(Vec::new());
}

#[when("the user double-clicks \"2D Sketches\" and then \"3D Annotations\" in the available list")]
fn double_click_source_pair(world: &mut ApplicationWorld) {
    for label in ["2DSketches", "3DAnnotations"] {
        double_click_source(world, label);
    }
}

fn double_click_source(world: &mut ApplicationWorld, label: &str) {
    let at = source_row(world, label);
    world.double_click(at);
    world.pump(Vec::new());
}

fn source_row(world: &ApplicationWorld, label: &str) -> [i32; 2] {
    let row = source_rows(world)
        .into_iter()
        .find(|row| row.text == label)
        .unwrap_or_else(|| panic!("expected {label:?} in the source list"));
    diagnostics::pointer_pixels(row.center())
}

fn target_row(world: &ApplicationWorld, label: &str) -> [i32; 2] {
    let row = target_row_runs(world)
        .into_iter()
        .find(|row| row.text == label)
        .unwrap_or_else(|| panic!("expected {label:?} in the target list"));
    diagnostics::pointer_pixels(row.center())
}

fn click_target_row(world: &mut ApplicationWorld, label: &str) {
    let at = target_row(world, label);
    world.click(at);
    world.pump(Vec::new());
}

fn remove_target_row(world: &mut ApplicationWorld, label: &str) {
    click_target_row(world, label);
    click_in_customize(world, "<");
}
