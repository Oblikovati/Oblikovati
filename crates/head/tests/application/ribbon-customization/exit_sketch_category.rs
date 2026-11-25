// SPDX-License-Identifier: GPL-2.0-only
//! Navigate the published sketch-exit customization category.
use super::{
    ApplicationWorld, click_in_customize, diagnostics, empty_tools_customization,
    list_steps::source_rows,
};
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;

#[given("ribbon customization is open in the MCAD Software for an empty Part Tools custom panel")]
fn open_empty_tools_draft(world: &mut ApplicationWorld) {
    empty_tools_customization(world);
}

#[given("the source category is \"Exit 2D Sketch Tab\"")]
fn select_exit_sketch_filter(world: &mut ApplicationWorld) {
    click_in_customize(world, "AllCommands");
    world.click_text("Exit 2D Sketch Tab");
    world.run_in_window("Customize", "Exit2DSketchTab");
}

#[given("the source list shows the separator, \"Cancel Sketch\", and \"Finish Sketch\"")]
fn exit_sketch_source_order(world: &mut ApplicationWorld) {
    let rows: Vec<_> = source_rows(world)
        .into_iter()
        .map(|run| run.text.as_str())
        .collect();
    assert_eq!(rows, ["<Separator>", "CancelSketch", "FinishSketch"]);
}

#[when("the user selects \"Cancel Sketch\" and presses Page Down")]
fn page_down_from_cancel(world: &mut ApplicationWorld) {
    let row = source_rows(world)
        .into_iter()
        .find(|run| run.text == "CancelSketch")
        .unwrap();
    world.click(diagnostics::pointer_pixels(row.center()));
    world.key(ViewportPhysicalKey::PageDown, false);
}

#[then("\"Finish Sketch\" is highlighted")]
fn finish_highlighted(world: &mut ApplicationWorld) {
    source_highlighted(world, "Finish Sketch");
}

#[then("the separator is highlighted")]
fn separator_highlighted(world: &mut ApplicationWorld) {
    source_highlighted(world, "<Separator>");
}

fn source_highlighted(world: &mut ApplicationWorld, expected: &str) {
    world.hover([1250, 780]);
    assert!(world.row_accented_beside(expected, "Cancel Sketch"));
}
