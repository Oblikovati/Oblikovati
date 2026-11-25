// SPDX-License-Identifier: GPL-2.0-only
//! Read visible source and target order and exercise their native controls.
use super::transaction_steps::{add_command, target_is_empty, target_rows};
use super::*;

const ASCENDING: [&str; 4] = ["<Separator>", "2-DDrawing", "2DSketches", "3DAnnotations"];
const DESCENDING: [&str; 5] = [
    "<Separator>",
    "ZoomWindow",
    "ZoomSelected",
    "ZoomAll",
    "Zoom",
];

pub(super) fn source_rows(world: &ApplicationWorld) -> Vec<&diagnostics::RenderedRun> {
    let header = world.run_in_window("Customize", "Name");
    let add = world.run_in_window("Customize", ">");
    let remove = world.run_in_window("Customize", "<");
    let x = header.minimum[0] - 10. ..add.minimum[0].min(remove.minimum[0]);
    let top = header.maximum[1];
    let mut rows: Vec<_> = world
        .frame
        .runs
        .iter()
        .filter(|run| x.contains(&run.minimum[0]) && (top..top + 200.).contains(&run.center()[1]))
        .collect();
    rows.sort_by(|a, b| a.center()[1].total_cmp(&b.center()[1]));
    rows
}

fn expect_source_prefix(world: &ApplicationWorld, expected: &[&str]) {
    let labels: Vec<_> = source_rows(world)
        .into_iter()
        .filter(|row| !row.text.is_empty())
        .map(|row| row.text.as_str())
        .collect();
    assert!(
        labels.starts_with(expected),
        "expected first source rows {expected:?}, found {labels:?}"
    );
}

#[when("the user selects \"2-D Drawing\" in the available list")]
#[when("the user selects \"2-D Drawing\" in the available commands list")]
fn select_drawing_source(world: &mut ApplicationWorld) {
    let drawing = source_rows(world)
        .into_iter()
        .find(|row| row.text == "2-DDrawing")
        .expect("expected 2-D Drawing in the available commands");
    world.click(diagnostics::pointer_pixels(drawing.center()));
}

#[given(
    "the Ribbon customization dialog shows All Commands in ascending order for an empty Part Tools custom panel"
)]
fn open_all_commands_ascending(world: &mut ApplicationWorld) {
    empty_tools_customization(world);
    let rows = source_rows(world);
    let position = |name| rows.iter().position(|row| row.text == name).unwrap();
    assert!(position("2DSketches") < position("3DAnnotations"));
}

#[given(
    "the available command list shows the separator and an unlabeled row above commands in ascending name order"
)]
fn source_starts_ascending(world: &mut ApplicationWorld) {
    expect_source_prefix(world, &ASCENDING);
    let rows = source_rows(world);
    let y = |name| rows.iter().find(|row| row.text == name).unwrap().center()[1];
    let gap = y("2-DDrawing") - y("<Separator>");
    let row_height = y("2DSketches") - y("2-DDrawing");
    assert!(
        gap > 1.5 * row_height,
        "expected an unlabeled row after the separator"
    );
}

#[when("the user clicks the available command list Name header")]
#[when("the user clicks the available command list Name header again")]
fn reverse_source(world: &mut ApplicationWorld) {
    click_in_customize(world, "Name");
}

#[then("the separator remains at the top of the visible list")]
fn separator_first(world: &mut ApplicationWorld) {
    expect_source_prefix(world, &["<Separator>"]);
}

#[then(
    "the visible command names begin in descending order with \"Zoom Window\", \"Zoom Selected\", \"Zoom All\", and \"Zoom\""
)]
#[then("the visible source list begins with the separator and descending Zoom commands")]
fn source_descending(world: &mut ApplicationWorld) {
    expect_source_prefix(world, &DESCENDING);
}

#[then(
    "the visible command names begin in ascending order with \"2-D Drawing\", \"2D Sketches\", and \"3D Annotations\""
)]
fn source_ascending(world: &mut ApplicationWorld) {
    expect_source_prefix(world, &ASCENDING);
}

#[then("the add transfer button remains enabled")]
fn add_enabled(world: &mut ApplicationWorld) {
    assert_eq!(world.run_in_window("Customize", ">").alpha, u8::MAX);
}

#[when("the user clicks the add transfer button")]
fn transfer(world: &mut ApplicationWorld) {
    transfer_to_target(world);
}

#[when("the user clicks the up ordering button")]
#[when("the user clicks the target list move-up button")]
fn move_up(world: &mut ApplicationWorld) {
    click_in_customize(world, "Up");
}

#[when("the user clicks the down ordering button")]
fn move_down(world: &mut ApplicationWorld) {
    click_in_customize(world, "Down");
}

#[then("\"2D Sketches\" remains the only command in the list")]
fn only_sketches_staged(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), ["2DSketches"]);
}

#[given("the user has staged \"3D Annotations\" followed by \"2D Sketches\"")]
fn stage_annotations_first(world: &mut ApplicationWorld) {
    for command in ["3DAnnotations", "2DSketches"] {
        add_command(world, command);
    }
}

#[when("the user clicks the custom panel list Name header twice")]
fn click_target_header(world: &mut ApplicationWorld) {
    let x = world.run_in_window("Customize", "Custompanel").minimum[0];
    let header = world
        .frame
        .runs
        .iter()
        .find(|row| row.text == "Name" && (row.minimum[0] - x).abs() < 12.)
        .expect("expected the target list Name header");
    let at = diagnostics::pointer_pixels(header.center());
    world.click(at);
    world.click(at);
}

#[then("the custom panel list remains in this order:")]
#[then("the custom panel list contains these rows in order:")]
#[then("the target list contains these rows in order:")]
fn expected_target_order(world: &mut ApplicationWorld, step: &cucumber::gherkin::Step) {
    let rows = step
        .table
        .as_ref()
        .expect("expected the ordered command names");
    let expected: Vec<_> = rows
        .rows
        .iter()
        .map(|row| {
            let [name] = row.as_slice() else {
                panic!("expected one command per row, got {row:?}");
            };
            super::super::world::squeezed(name)
        })
        .collect();
    assert_eq!(target_rows(world), expected);
}

#[when("the user clicks an empty area of the target list")]
fn click_empty_target(world: &mut ApplicationWorld) {
    target_is_empty(world);
    let header = world.run_in_window("Customize", "Custompanel");
    let large = world.run_in_window("Customize", "Large");
    world.click(diagnostics::pointer_pixels([
        header.minimum[0] + 25.,
        large.maximum[1] + 45.,
    ]));
}

#[then("\"2D Sketches\" remains highlighted in the available commands list")]
fn source_sketches_selected(world: &mut ApplicationWorld) {
    world.hover([800, 900]);
    assert!(world.row_accented_beside("2D Sketches", "<Separator>"));
}

#[then("the target list remains empty")]
fn empty_target(world: &mut ApplicationWorld) {
    target_is_empty(world);
}

#[then("Apply remains disabled")]
fn apply_disabled(world: &mut ApplicationWorld) {
    assert_ne!(world.run_in_window("Customize", "Apply").alpha, u8::MAX);
}
