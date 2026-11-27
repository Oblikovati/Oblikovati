// SPDX-License-Identifier: GPL-2.0-only
//! Verify Customize resize behavior using its floating window and rendered lists.
use super::super::*;

#[given(
    "`fixtures/cad/blank-metric-part.opd` is open and ribbon customization is open with an empty Part Tools target"
)]
#[given(
    "`fixtures/cad/blank-metric-part.opd` is open and ribbon customization is open with no pending edits"
)]
fn open_blank_customize(world: &mut ApplicationWorld) {
    super::super::super::ribbon_panel_expansion::open_blank_metric_part(world);
    empty_tools_customization(world);
    assert!(world.run_in_window("Customize", "Apply").alpha < u8::MAX);
}

#[given("ribbon customization is open in the MCAD Software with an empty Part Tools target")]
fn open_empty_tools_target(world: &mut ApplicationWorld) {
    world.previous_document = world.state().document.or_else(|| Some(world.add_part()));
    empty_tools_customization(world);
}

#[given("All Commands is shown in the available-command list")]
fn all_commands_visible(world: &mut ApplicationWorld) {
    transaction_steps::all_commands(world);
}

#[when("the user drags the lower-right dialog corner outward")]
fn grow(world: &mut ApplicationWorld) {
    resize(world, [72, 72]);
}

#[when("the user drags the lower-right dialog corner inward")]
fn shrink(world: &mut ApplicationWorld) {
    resize(world, [-60, -60]);
}

fn resize(world: &mut ApplicationWorld, delta: [i32; 2]) {
    world.customize_size_before_drag = Some(sizes(world));
    world.observed_counts = vec![list_steps::source_rows(world).len()];
    let (origin, size) = diagnostics::window_geometry("Customize").unwrap();
    let from =
        diagnostics::pointer_pixels(std::array::from_fn(|axis| origin[axis] + size[axis] - 4.));
    let to = std::array::from_fn(|axis| from[axis] + delta[axis]);
    world.drag(from, to);
    world.pump(Vec::new());
}

#[then("the dialog and both command lists become wider and taller")]
#[then("the dialog and its command lists become wider and taller")]
fn grew(world: &mut ApplicationWorld) {
    expect_resize(world, 1.);
}

#[then("the dialog and both command lists become smaller")]
fn shrank(world: &mut ApplicationWorld) {
    expect_resize(world, -1.);
}

fn expect_resize(world: &ApplicationWorld, direction: f32) {
    let before = world.customize_size_before_drag.unwrap();
    let after = sizes(world);
    for (before, after) in before.into_iter().zip(after) {
        assert!(
            (0..2).all(|axis| (after[axis] - before[axis]) * direction > 12.),
            "expected both dimensions to change with the drag: {before:?} to {after:?}"
        );
    }
}

#[then("more available-command rows are visible")]
fn more_source_rows(world: &mut ApplicationWorld) {
    assert!(list_steps::source_rows(world).len() > world.observed_counts[0]);
}

#[when("the user adds 2D Sketches to the custom-panel target")]
fn stage_sketches(world: &mut ApplicationWorld) {
    transaction_steps::add_command(world, "2DSketches");
}

#[then("the target shows 2D Sketches with Large and Text unchecked")]
#[then("the staged 2D Sketches row and its unchecked options remain visible")]
fn staged_row_remains(world: &mut ApplicationWorld) {
    assert_eq!(transaction_steps::target_rows(world), ["2DSketches"]);
    for option in ["Large", "Text"] {
        assert!(!row_option_checked(world, "2DSketches", option));
    }
}

#[then("the target selector still indicates a pending change")]
fn edit_remains_pending(world: &mut ApplicationWorld) {
    transaction_steps::expect_target_pending(world, true);
}

#[then("the dialog closes without applying the staged command")]
fn draft_discarded(world: &mut ApplicationWorld) {
    customization_closed(world);
    world.expect_no_text("User Commands");
    assert_eq!(world.state().document, world.previous_document);
}

#[when("the user reopens customization from the Tools ribbon in the same session")]
fn reopen(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Customize").is_none());
    open_tools_customization(world);
}

#[then("the dialog reopens at its default size")]
fn default_size_restored(world: &mut ApplicationWorld) {
    assert_eq!(sizes(world), world.customize_size_before_drag.unwrap());
}

#[then("the available-command and custom-panel lists remain visible")]
fn lists_visible(world: &mut ApplicationWorld) {
    assert_eq!(world.state().document, world.previous_document);
    assert!(
        sizes(world)
            .iter()
            .flatten()
            .all(|dimension| *dimension > 90.)
    );
}

fn sizes(world: &ApplicationWorld) -> [[f32; 2]; 3] {
    let child_size = |header: &str| {
        let header = world.run_in_window("Customize", header);
        let point = [header.center()[0] + 12., header.maximum[1] + 24.];
        diagnostics::child_size_at("Customize", point)
    };
    [
        diagnostics::window_geometry("Customize").unwrap().1,
        child_size("Name"),
        child_size("Large"),
    ]
}
