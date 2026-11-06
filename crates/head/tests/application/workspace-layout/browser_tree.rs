// SPDX-License-Identifier: GPL-2.0-only
use super::*;
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
pub(super) fn open_browser_menu(world: &mut ApplicationWorld) {
    world.click_text("Menu");
    world.pump(Vec::new());
}
#[when("I choose Collapse All")]
fn collapse_all(world: &mut ApplicationWorld) {
    world.click_text("Collapse All");
    world.pump(Vec::new());
}
#[when("I choose Expand All")]
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
#[given("the model browser search field is open")]
fn open_search(world: &mut ApplicationWorld) {
    world.click_text("Search");
}
pub(super) fn type_in_search(world: &mut ApplicationWorld, query: &str) {
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
    for unrelated in ["Origin", "Features", "End of Part"] {
        world.expect_no_text(unrelated);
    }
    world.expect_no_text("XY Plane");
}
#[when("I clear the search text")]
#[when("I clear the search query")]
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
fn enter_no_match(world: &mut ApplicationWorld) {
    type_in_search(world, "zzz");
}
#[then("the browser shows a no-results message instead of feature rows")]
fn no_results(world: &mut ApplicationWorld) {
    world.expect_text("No matching entries");
    world.expect_no_text("Origin");
}
