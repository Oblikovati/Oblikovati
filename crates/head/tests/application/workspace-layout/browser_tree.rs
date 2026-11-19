// SPDX-License-Identifier: GPL-2.0-only
use super::*;
const RIBBON_BAND_BOTTOM: f32 = 160.;
#[given("a part with nested model browser branches is open in the MCAD Software")]
fn nested_part(world: &mut ApplicationWorld) {
    world.add_part();
    world.start_sketch();
    world.draw_rectangle();
    world.finish_sketch();
}
#[given("some browser branches are expanded")]
fn branches_expanded(world: &mut ApplicationWorld) {
    world.expect_text("XY Plane");
}
#[when("I open the model browser menu")]
#[when("the user opens the model browser menu")]
pub(in super::super) fn open_browser_menu(world: &mut ApplicationWorld) {
    world.click_text("Menu");
    world.pump(Vec::new());
}
#[when("I choose Collapse All")]
#[when("the user chooses Collapse All")]
fn collapse_all(world: &mut ApplicationWorld) {
    world.click_text("Collapse All");
    world.pump(Vec::new());
}
#[when("I choose Expand All")]
#[when("the user chooses Expand All")]
fn expand_all(world: &mut ApplicationWorld) {
    world.click_text("Expand All");
    world.pump(Vec::new());
}
#[then("the expanded branches collapse to their parent rows")]
fn collapsed(world: &mut ApplicationWorld) {
    for parent in ["Origin", "Sketches", "Features"] {
        world.expect_text(parent);
    }
    world.expect_no_text("XY Plane");
}
#[then("nested browser rows are displayed")]
fn nested_rows(world: &mut ApplicationWorld) {
    world.expect_text("XY Plane");
    world.expect_browser_sketch();
}
#[then("the model remains displayed in the viewport")]
fn model_displayed(world: &mut ApplicationWorld) {
    world.expect_drawn_region();
}
const WORK_POINT_NAMES: usize = 3;
/// Setup only: work points have no ribbon command yet, so the API creates them.
fn add_work_points(world: &mut ApplicationWorld) {
    use oblikovati_api::{client::Client, wire::CreateWorkPointArgs};
    let host = world.app().owner.document_host();
    let client = Client::new(host);
    for index in 0..WORK_POINT_NAMES {
        client
            .work_points()
            .create(&CreateWorkPointArgs {
                at: Some(vec![index as f64, 0., 0.]),
                ..Default::default()
            })
            .unwrap();
    }
    world.pump(Vec::new());
}
#[given("a part with multiple work points and other features is open in the MCAD Software")]
#[given("a populated part is open in the MCAD Software")]
fn populated_part(world: &mut ApplicationWorld) {
    nested_part(world);
    add_work_points(world);
}
#[given("the model browser is visible")]
fn browser_visible(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
}
#[when("I open the browser search field")]
#[when("the user opens the browser search field")]
#[given("the model browser search field is open")]
fn open_search(world: &mut ApplicationWorld) {
    world.click_text("Search");
    world.click_text("Search");
}
pub(in super::super) fn type_in_search(world: &mut ApplicationWorld, query: &str) {
    // The empty field shows its hint, drawn after the button that opened it.
    world.click_text("Search");
    world.type_text(query);
}
#[when(expr = "I enter {string}")]
fn enter_query(world: &mut ApplicationWorld, query: String) {
    type_in_search(world, &query);
}
#[then("matching work point rows are shown in the browser")]
fn work_point_rows(world: &mut ApplicationWorld) {
    let shown = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text.starts_with("WorkPoint") && run.text != "WorkPoint")
        .count();
    assert_eq!(
        shown,
        WORK_POINT_NAMES,
        "rendered text {:?}: expected every work point",
        world.text()
    );
}
#[then("the matching text is highlighted in those rows")]
fn matches_highlighted(world: &mut ApplicationWorld) {
    assert!(
        world.text_is_selected("WorkPoint1"),
        "expected a highlight behind the match"
    );
}
#[then("unrelated visible feature rows are filtered from the browser")]
fn unrelated_filtered(world: &mut ApplicationWorld) {
    // Whole rows: the ribbon's Work Features group holds one of these words too.
    // Matching work points remain under their Features ancestor.
    world.expect_text("Features");
    for unrelated in ["Origin", "Sketches", "EndofPart"] {
        let shown = world.frame.runs.iter().any(|run| run.text == unrelated);
        assert!(
            !shown,
            "rendered text {:?}: expected no {unrelated:?} row",
            world.text()
        );
    }
    world.expect_no_text("XY Plane");
    let browser = world
        .state()
        .browser
        .expect("expected the populated browser");
    let (position, size) = super::browser_geometry();
    for sketch in browser.sketches {
        let name: String = sketch
            .name
            .chars()
            .filter(|ch| !ch.is_whitespace())
            .collect();
        let shown = world.frame.runs.iter().any(|run| {
            let [x, y] = run.center();
            run.text == name
                && x >= position[0]
                && x < position[0] + size[0]
                && y >= position[1]
                && y < position[1] + size[1]
        });
        assert!(
            !shown,
            "filtered sketch {:?}: expected no browser row",
            sketch.name
        );
    }
}
#[when("I close the empty search field")]
fn press_search_close(world: &mut ApplicationWorld) {
    world.click_text("x");
}
#[then("the unfiltered browser tree is shown again")]
#[then("the unfiltered model browser tree is shown again")]
fn unfiltered(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    world.expect_text("End of Part");
}
#[then("the browser search field is hidden")]
fn search_hidden(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    let (position, size) = browser_geometry();
    let hint = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Search")
        .count();
    assert_eq!(
        hint, 1,
        "expected only the Search button in {position:?} {size:?}"
    );
}
#[when("I enter a query that matches no browser entry")]
#[when("the user enters a query that matches no browser entry")]
fn enter_no_match(world: &mut ApplicationWorld) {
    type_in_search(world, "zzz");
}
#[then("the browser shows a no-results message instead of feature rows")]
fn no_results(world: &mut ApplicationWorld) {
    world.expect_text("No matching entries");
    world.expect_no_text("Origin");
}
// While a sketch is being edited
fn expect_sketch_editing(world: &mut ApplicationWorld) {
    assert!(
        world.state().authoring == oblikovati_gui::GuiAuthoringContext::Sketch,
        "expected the sketch still under edit"
    );
    world.expect_text("Finish Sketch");
}
/// The browser's row, below the band where the ribbon's Center Point toggle also reads so.
fn origin_row_shown(world: &ApplicationWorld) -> bool {
    world
        .frame
        .runs
        .iter()
        .any(|run| run.text == "CenterPoint" && run.center()[1] > RIBBON_BAND_BOTTOM)
}
fn close_control_shown(world: &ApplicationWorld) -> bool {
    world.frame.runs.iter().any(|run| run.text == "x")
}
#[given("a part document is open in the MCAD Software with a sketch being edited")]
fn part_with_sketch_under_edit(world: &mut ApplicationWorld) {
    world.add_part();
    world.start_sketch();
    expect_sketch_editing(world);
}
#[given("the model browser shows an expanded Origin branch and other document entries")]
#[given("the model browser shows an expanded Origin branch")]
#[given("the Origin branch is expanded in the model browser")]
fn origin_expanded(world: &mut ApplicationWorld) {
    // Outside a sketch, the scenarios go on to watch a solid: this fixture part has one.
    if world.state().authoring == oblikovati_gui::GuiAuthoringContext::Part {
        super::browser_feature::open_extruded_part(world);
    }
    world.expect_text("Center Point");
    world.expect_text("Sketches");
    world.expect_text("Features");
}
#[when("I select the expansion control beside Origin")]
#[when("I select the expansion control beside Origin again")]
fn toggle_origin(world: &mut ApplicationWorld) {
    world.press_disclosure("Origin");
}
#[then("the Origin branch collapses and its child entries are hidden")]
fn origin_collapsed(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    assert!(
        !origin_row_shown(world),
        "expected the branch's rows hidden"
    );
}
#[then("the Origin branch expands and its child entries are displayed")]
fn origin_open(world: &mut ApplicationWorld) {
    assert!(origin_row_shown(world), "expected the branch's rows shown");
}
#[then("the other document entries remain visible in the browser")]
#[then("the other document entries remain visible")]
fn other_entries_remain(world: &mut ApplicationWorld) {
    world.expect_text("Sketches");
    world.expect_text("Features");
}
#[then("the sketch remains displayed in the viewport")]
#[then("the sketch remains visible in the viewport")]
#[then("the sketch viewport remains visible")]
fn sketch_remains(world: &mut ApplicationWorld) {
    expect_sketch_editing(world);
}
#[given("the model browser is visible with its search field closed")]
fn search_closed(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    assert!(!close_control_shown(world), "expected no search field");
}
#[when("I select the browser search control")]
fn select_search_control(world: &mut ApplicationWorld) {
    open_search(world);
}
#[then("an empty search field appears beneath the browser header")]
fn empty_field(world: &mut ApplicationWorld) {
    assert!(close_control_shown(world), "expected the search field");
    let control = world.run("Menu").center()[1];
    let close = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "x")
        .map_or(0., |run| run.center()[1]);
    assert!(close > control, "expected the field beneath the header");
}
#[then("the model browser tree and sketch viewport remain visible")]
#[then("the browser tree and sketch viewport remain visible")]
fn tree_and_viewport(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    expect_sketch_editing(world);
}
#[when("I select the close control on the empty search field")]
fn close_empty_field(world: &mut ApplicationWorld) {
    press_search_close(world);
}
// Origin rows and Display Preferences
#[when("I select the X Axis entry in the browser")]
fn select_x_axis(world: &mut ApplicationWorld) {
    world.click_text("X Axis");
    world.pump(Vec::new());
}
#[then("the X Axis entry is highlighted in the browser")]
fn x_axis_highlighted(world: &mut ApplicationWorld) {
    assert!(
        world.row_highlighted("X Axis"),
        "expected X Axis highlighted"
    );
}
pub(in super::super) fn expand_display_preferences(world: &mut ApplicationWorld) {
    open_browser_menu(world);
    world.click_text("Display Preferences");
    world.pump(Vec::new());
}
#[when("I open the browser header menu")]
fn open_header_menu(world: &mut ApplicationWorld) {
    open_browser_menu(world);
}
#[when("I expand Display Preferences")]
fn expand_preferences(world: &mut ApplicationWorld) {
    world.click_text("Display Preferences");
    world.pump(Vec::new());
}
fn expect_rows(world: &ApplicationWorld, rows: &[&str]) {
    for row in rows {
        world.expect_text(row);
    }
}
#[then(
    "the submenu offers Hide UCS, Hide Work Features, Hide Notes, Hide Documents and Hide Warnings"
)]
fn hide_rows(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &[
            "Hide UCS",
            "Hide Work Features",
            "Hide Notes",
            "Hide Documents",
            "Hide Warnings",
        ],
    );
}
#[then("it offers Show Extended Names and Hide Blue Dot Indicator")]
fn name_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &["Show Extended Names", "Hide Blue Dot Indicator"]);
}
#[then("it offers Auto expand Representations Folder on Open and Design View Expansion")]
fn folder_rows(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &[
            "Auto expand Representations Folder on Open",
            "Design View Expansion",
        ],
    );
}
#[given("Hide Work Features is unchecked in browser Display Preferences")]
fn work_features_unchecked(world: &mut ApplicationWorld) {
    expand_display_preferences(world);
    assert!(
        !world.checked("Hide Work Features"),
        "expected the row unchecked"
    );
}
#[when("I select Hide Work Features")]
#[when("I select Hide Work Features again")]
fn select_hide_work_features(world: &mut ApplicationWorld) {
    world.click_text("Hide Work Features");
    world.pump(Vec::new());
}
#[then("the Origin branch is hidden from the browser")]
fn origin_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Origin");
}
#[then("the sketch entries and sketch viewport remain visible")]
fn sketch_entries_remain(world: &mut ApplicationWorld) {
    world.expect_text("Sketches");
    expect_sketch_editing(world);
}
#[when("I reopen browser Display Preferences")]
fn reopen_preferences(world: &mut ApplicationWorld) {
    expand_display_preferences(world);
}
#[then("Hide Work Features is checked")]
fn work_features_checked(world: &mut ApplicationWorld) {
    assert!(
        world.checked("Hide Work Features"),
        "expected the row checked"
    );
}
#[then("the Origin branch returns to the browser in a collapsed state")]
fn origin_back_collapsed(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    assert!(!origin_row_shown(world), "expected the branch closed");
}
