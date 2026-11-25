// SPDX-License-Identifier: GPL-2.0-only
use super::browser_tree::{open_browser_menu, type_in_search};
use super::*;
const DRAWING_BORDER: &str = "Default Border";
const SHEET_TITLE_BLOCK: &str = "Title Block";
#[given("a drawing with drawing resources and a sheet is open in the MCAD Software")]
#[given("a drawing with border entries in its resources and sheet is open in the MCAD Software")]
#[given("a drawing with browser entries is open in the MCAD Software")]
#[given("a drawing with a border and title block is open in the MCAD Software")]
#[given("a drawing with a sheet and drawing resources is open in the MCAD Software")]
#[given("a drawing with a border resource and a sheet border is open in the MCAD Software")]
#[given("a drawing with a border resource is open in the MCAD Software")]
fn drawing_open(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.add_drawing());
}
#[given("`fixtures/cad/blank-drawing.odd` is active in the MCAD Software")]
#[given(
    "`fixtures/cad/blank-drawing.odd` is open in the MCAD Software with drawing resources and a sheet"
)]
#[given(
    "`fixtures/cad/blank-drawing.odd` is open in the MCAD Software with border entries in its resources and sheet"
)]
#[given("`fixtures/cad/blank-drawing.odd` is open in the MCAD Software with browser entries")]
#[given(
    "`fixtures/cad/blank-drawing.odd` is open in the MCAD Software with a border and title block"
)]
#[given(
    "`fixtures/cad/blank-drawing.odd` is open in the MCAD Software with a sheet and drawing resources"
)]
#[given(
    "`fixtures/cad/blank-drawing.odd` is open in the MCAD Software with a border resource and a sheet border"
)]
fn native_drawing_open(world: &mut ApplicationWorld) {
    let path = super::super::scenario_root().join("fixtures/cad/blank-drawing.odd");
    world.open_fixture(&path);
    world.previous_document = Some(world.active_id());
    drawing_tree(world);
}

#[given("the nested title block branch is expanded")]
#[then("the nested title block branch appears expanded")]
fn title_block_open(world: &mut ApplicationWorld) {
    world.expect_text("Field Text");
}
#[given("the browser has expanded branches containing nested entries")]
#[given("the drawing browser branches are expanded")]
#[given("the sheet branch is expanded in the drawing browser")]
#[given("the sheet and drawing resources branches are expanded in the browser")]
fn drawing_branches_open(world: &mut ApplicationWorld) {
    world.expect_text(DRAWING_BORDER);
}
#[when("I open the drawing browser menu")]
#[when("the user opens the drawing browser menu")]
fn open_drawing_menu(world: &mut ApplicationWorld) {
    open_browser_menu(world);
}
#[then("the drawing resources and sheet remain visible as collapsed branches")]
fn branches_collapsed(world: &mut ApplicationWorld) {
    world.expect_text("Drawing Resources");
    world.expect_text("Sheet:1");
}
#[then("their nested entries are hidden")]
fn nested_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text(DRAWING_BORDER);
    world.expect_no_text(SHEET_TITLE_BLOCK);
}
fn expect_sheet(world: &mut ApplicationWorld) {
    expect_sheet_at(world, [1000, 500]);
}
#[then("the drawing sheet remains visible")]
#[then("the drawing sheet remains displayed")]
fn sheet_visible(world: &mut ApplicationWorld) {
    expect_sheet(world);
}
#[then("nested entries under the drawing resources and sheet are displayed")]
fn nested_shown(world: &mut ApplicationWorld) {
    world.expect_text(DRAWING_BORDER);
    world.expect_text("Sheet Formats");
    world.expect_text(SHEET_TITLE_BLOCK);
}
#[when("I open the drawing browser search field")]
#[when("the user opens the drawing browser search field")]
#[when("I open the floating browser search field")]
#[when("the user opens the floating browser search field")]
#[given("the drawing browser search field is open")]
fn open_drawing_search(world: &mut ApplicationWorld) {
    world.click_text("Search");
    world.click_text("Type Keyword");
}
#[then("matching border entries are displayed beneath their respective parent branches")]
fn borders_under_parents(world: &mut ApplicationWorld) {
    let row_of = |world: &ApplicationWorld, text: &str| world.run(text).center();
    let resources = row_of(world, "Drawing Resources");
    let sheet = row_of(world, "Sheet:1");
    let rows: Vec<[f32; 2]> = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "DefaultBorder")
        .map(diagnostics_center)
        .collect();
    assert_eq!(rows.len(), 2, "expected a border under each branch");
    assert!(
        rows[0][1] > resources[1] && rows[0][1] < sheet[1],
        "rows {rows:?}"
    );
    assert!(rows[1][1] > sheet[1], "rows {rows:?}");
    assert!(
        rows[0][0] > resources[0],
        "rows {rows:?}: expected an indent"
    );
}
fn diagnostics_center(run: &super::super::diagnostics::RenderedRun) -> [f32; 2] {
    run.center()
}
#[then("matching text is highlighted")]
#[then("the matching text is highlighted")]
fn border_marked(world: &mut ApplicationWorld) {
    let label = if world.shows("X Axis") {
        "X Axis"
    } else if world.shows("XY Plane") {
        "XY Plane"
    } else {
        DRAWING_BORDER
    };
    assert!(
        world.end_of_text_is_marked(label),
        "expected a marked match in {label}"
    );
}
#[then("unrelated browser entries are hidden")]
fn unrelated_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text(SHEET_TITLE_BLOCK);
    world.expect_no_text("Sheet Formats");
}
#[then("the unfiltered drawing browser tree is displayed")]
#[then("the unfiltered drawing browser tree is displayed in the floating panel")]
fn drawing_tree_back(world: &mut ApplicationWorld) {
    world.expect_text("Sheet Formats");
    world.expect_text(SHEET_TITLE_BLOCK);
}
#[then("the search field is hidden")]
fn drawing_search_hidden(world: &mut ApplicationWorld) {
    // Only the field's close control reads exactly this; "Field Text" holds the letter too.
    let close_shown = world.frame.runs.iter().any(|run| run.text == "x");
    assert!(!close_shown, "expected no search close control");
}
#[when("I enter a query that matches no drawing browser entry")]
#[when("the user enters a query that matches no drawing browser entry")]
fn no_drawing_match(world: &mut ApplicationWorld) {
    type_in_search(world, "zzz");
}
#[then("a no-results message is displayed in place of the browser tree")]
#[then("the floating browser displays a no-results message instead of its tree")]
fn drawing_no_results(world: &mut ApplicationWorld) {
    world.expect_text("No matching entries");
    world.expect_no_text("Sheet:1");
}
#[when("I select the sheet border entry in the drawing browser")]
#[when("the user selects the sheet border entry in the drawing browser")]
fn select_border(world: &mut ApplicationWorld) {
    world.click_text(DRAWING_BORDER);
}
#[then("the border entry is highlighted in the browser")]
fn border_highlighted(world: &mut ApplicationWorld) {
    assert!(
        world.row_highlighted(DRAWING_BORDER),
        "expected the border row accented"
    );
}
#[when("I select the sheet title block entry in the drawing browser")]
#[when("the user selects the sheet title block entry in the drawing browser")]
fn select_title_block(world: &mut ApplicationWorld) {
    world.click_text(SHEET_TITLE_BLOCK);
}
#[then("the title block entry is highlighted in the browser")]
fn title_highlighted(world: &mut ApplicationWorld) {
    assert!(
        world.row_highlighted(SHEET_TITLE_BLOCK),
        "expected the title block row accented"
    );
}
#[then("the border entry is no longer highlighted as the active selection")]
fn border_cleared(world: &mut ApplicationWorld) {
    assert!(
        !world.row_highlighted(DRAWING_BORDER),
        "expected the border row plain"
    );
}
#[given("the drawing browser floats over the drawing sheet")]
fn drawing_browser_floats(world: &mut ApplicationWorld) {
    drag_over_drawing(world);
    floats_over_sheet(world);
}
#[when("I drag the drawing browser header to the right edge of the workspace")]
#[when("the user drags the drawing browser header to the right edge of the workspace")]
fn drawing_header_right(world: &mut ApplicationWorld) {
    drag_right(world);
}
#[then("the drawing browser docks on the right")]
fn drawing_docks_right(world: &mut ApplicationWorld) {
    let (position, size) = browser_geometry();
    let width = world.extent()[0] as f32;
    let dock = super::super::diagnostics::window_dock_id("Model").unwrap_or_default();
    assert!(
        dock != 0 && position[0] + size[0] > width - 20. && position[0] > width / 2.,
        "browser {position:?} {size:?} dock {dock}: expected the right dock"
    );
}
#[then("its drawing resources and sheet tree remain visible")]
fn drawing_tree_remains(world: &mut ApplicationWorld) {
    drawing_tree(world);
}
#[then("the drawing sheet remains visible to its left")]
fn sheet_to_left(world: &mut ApplicationWorld) {
    expect_sheet_at(world, [500, 500]);
}
#[when("I drag the drawing browser header to the left edge of the workspace")]
#[when("the user drags the drawing browser header to the left edge of the workspace")]
fn drawing_header_left(world: &mut ApplicationWorld) {
    drag_left(world);
}
#[then("the drawing browser docks on the left")]
fn drawing_docks_left(world: &mut ApplicationWorld) {
    expect_docked_left(world);
}
#[then("the drawing sheet remains visible to its right")]
fn sheet_to_right(world: &mut ApplicationWorld) {
    expect_sheet_at(world, [1000, 500]);
}
/// A sheet row reads exactly so; the resources' `Default Title Block` must not count.
fn sheet_row_shown(world: &ApplicationWorld) -> bool {
    world.frame.runs.iter().any(|run| run.text == "TitleBlock")
}
#[when("I select the expansion control beside the sheet branch")]
#[when("the user selects the expansion control beside the sheet branch")]
#[when("I select the expansion control beside the sheet branch again")]
#[when("the user selects the expansion control beside the sheet branch again")]
fn toggle_sheet_branch(world: &mut ApplicationWorld) {
    world.press_disclosure("Sheet:1");
}
#[then("the sheet branch collapses and its child entries are hidden")]
fn sheet_collapsed(world: &mut ApplicationWorld) {
    world.expect_text("Sheet:1");
    assert!(!sheet_row_shown(world), "expected the sheet's rows hidden");
}
#[then("the drawing resources branch remains expanded")]
fn resources_expanded(world: &mut ApplicationWorld) {
    world.expect_text("Sheet Formats");
}
#[then("the sheet branch expands and its child entries are displayed")]
fn sheet_expanded(world: &mut ApplicationWorld) {
    assert!(sheet_row_shown(world), "expected the sheet's rows shown");
}
#[given("the drawing resources branch is collapsed in the browser")]
fn resources_collapsed(world: &mut ApplicationWorld) {
    world.press_disclosure("Drawing Resources");
    world.expect_no_text("Sheet Formats");
}
#[then("the filtered tree shows the matching border resource under its parent branches")]
fn filtered_resource(world: &mut ApplicationWorld) {
    world.expect_text("Drawing Resources");
    world.expect_text(DRAWING_BORDER);
    world.expect_no_text("Sheet Formats");
}
#[then("the filtered tree shows the matching sheet border")]
fn filtered_sheet_border(world: &mut ApplicationWorld) {
    world.expect_text("Sheet:1");
    let borders = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == DRAWING_BORDER.replace(' ', ""))
        .count();
    assert_eq!(borders, 2, "expected the resource's and the sheet's border");
}
#[then("the unfiltered tree is restored")]
fn tree_restored(world: &mut ApplicationWorld) {
    world.expect_text("Drawing Resources");
    world.expect_text("Sheet:1");
    assert!(sheet_row_shown(world), "expected the sheet's rows shown");
}
#[then("the drawing resources branch is collapsed")]
fn resources_still_collapsed(world: &mut ApplicationWorld) {
    world.expect_no_text("Sheet Formats");
}
