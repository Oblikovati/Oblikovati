// SPDX-License-Identifier: GPL-2.0-only
use super::{ApplicationWorld, diagnostics::window_geometry};
use crate::HeadWork;
use cucumber::{given, then, when};
const BROWSER: &str = "Model";
fn browser_geometry() -> ([f32; 2], [f32; 2]) {
    window_geometry(BROWSER).expect("model browser: expected a drawn window")
}
fn drag_header(world: &mut ApplicationWorld, to: [i32; 2]) {
    let from = world.run(BROWSER).center().map(|value| value as i32);
    world.drag(from, to);
}
fn expect_docked_left(world: &mut ApplicationWorld) {
    let (position, size) = browser_geometry();
    assert_eq!(position[0], 8., "browser: expected the left dock");
    let height = world.extent()[1] as f32;
    assert!(
        size[1] > height / 2.,
        "browser {size:?}: expected full height"
    );
}
#[given("a part document is open in the MCAD Software")]
fn part_open(world: &mut ApplicationWorld) {
    world.app().owner.create_part_document().unwrap();
    world.previous_document = world.app().owner.gui_command_state().unwrap().document;
    world.pump(Vec::new());
}
#[given("the 3D Model ribbon tab is active")]
fn model_tab_active(world: &mut ApplicationWorld) {
    world.expect_text("3D Model");
    world.expect_text("Sketch XY");
}
#[when("I select the View ribbon tab")]
fn select_view(world: &mut ApplicationWorld) {
    world.click_text("View");
}
#[then("the ribbon displays view, appearance, window and navigation controls")]
fn view_groups(world: &mut ApplicationWorld) {
    for group in ["Visibility", "Appearance", "Windows", "Navigate"] {
        world.expect_text(group);
    }
}
#[then("the active document remains displayed")]
fn document_displayed(world: &mut ApplicationWorld) {
    let active = world.app().owner.gui_command_state().unwrap().document;
    assert!(active.is_some() && active == world.previous_document);
    world.expect_text("End of Part");
}
#[given("the model browser is visible at the left of the workspace")]
#[given("the model browser is docked to the left edge of the workspace")]
fn browser_left(world: &mut ApplicationWorld) {
    expect_docked_left(world);
}
#[when("I open the User Interface menu from the View ribbon tab")]
fn open_user_interface(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.click_text("User Interface");
}
#[when("I clear the Model Browser option")]
#[when("I select the Model Browser option again")]
fn toggle_browser(world: &mut ApplicationWorld) {
    world.click_text("Model Browser");
}
#[then("the model browser is hidden")]
fn browser_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Origin");
}
#[then("the drawing area extends into the space it occupied")]
#[then("the viewport occupies the space previously used by the docked browser")]
fn left_space_free(world: &mut ApplicationWorld) {
    let height = world.extent()[1] as usize;
    world.expect_blank_region([20, 290], [height - 200, height - 20]);
}
#[then("the Model Browser option is unchecked")]
fn option_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Model Browser"), "expected a cleared option");
}
#[then("the Model Browser option is checked")]
fn option_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Model Browser"), "expected a checked option");
}
#[then("the model browser returns to the left of the workspace")]
#[then("the model browser docks at the left edge")]
fn browser_returned(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    expect_docked_left(world);
}
#[when("I drag the model browser by its header into the viewport")]
fn drag_into_viewport(world: &mut ApplicationWorld) {
    drag_header(world, [800, 300]);
}
#[then("the model browser becomes a floating panel over the viewport")]
fn browser_floating(world: &mut ApplicationWorld) {
    let (position, size) = browser_geometry();
    let extent = world.extent();
    assert!(
        position[0] > 400.,
        "browser {position:?}: expected to float over the view"
    );
    assert!(
        size[1] < extent[1] as f32 / 2.,
        "browser {size:?}: expected a compact panel"
    );
    world.expect_text("Origin");
}
#[when("I drag the floating browser by its header to the left edge of the workspace")]
#[when("I drag its header to the left edge of the workspace")]
#[when("I drag the floating browser header to the left edge of the workspace")]
fn drag_left(world: &mut ApplicationWorld) {
    drag_header(world, [10, 400]);
}
#[when("I drag its header to the right edge of the workspace")]
fn drag_right(world: &mut ApplicationWorld) {
    let width = world.extent()[0] as i32;
    drag_header(world, [width - 10, 400]);
}
#[then("the viewport occupies the remaining workspace")]
fn right_space_free(world: &mut ApplicationWorld) {
    let extent = world.extent();
    world.expect_blank_region(
        [330, extent[0] as usize - 20],
        [250, extent[1] as usize - 20],
    );
}
#[then("the model browser docks at the right edge")]
fn browser_right(world: &mut ApplicationWorld) {
    let (position, size) = browser_geometry();
    let width = world.extent()[0] as f32;
    assert_eq!(
        position[0] + size[0],
        width - 8.,
        "browser: expected the right dock"
    );
}
#[then("the viewport occupies the workspace to its left")]
fn view_left_of_browser(world: &mut ApplicationWorld) {
    let extent = world.extent();
    world.expect_blank_region(
        [20, extent[0] as usize - 330],
        [250, extent[1] as usize - 20],
    );
}
const TAB_STRIP: &str = "##open-document-tabs";
#[given("the model browser is docked to the left of the viewport")]
fn docked_left_of_viewport(world: &mut ApplicationWorld) {
    expect_docked_left(world);
    world.browser_width = Some(browser_geometry().1[0]);
    world.strip_left = window_geometry(TAB_STRIP).map(|(position, _)| position[0]);
}
/// Press the boundary the Model window's right edge draws, and drag it sideways.
fn drag_boundary(world: &mut ApplicationWorld, delta: i32) {
    let (position, size) = browser_geometry();
    let from = [
        (position[0] + size[0]) as i32 + 3,
        (position[1] + size[1] / 2.) as i32,
    ];
    world.drag(from, [from[0] + delta, from[1]]);
    world.pump(Vec::new());
}
#[when("I drag the boundary between the model browser and viewport to the right")]
fn drag_right_boundary(world: &mut ApplicationWorld) {
    drag_boundary(world, 90);
}
#[then("the model browser becomes wider")]
fn browser_wider(world: &mut ApplicationWorld) {
    let before = world.browser_width.expect("expected a starting width");
    let now = browser_geometry().1[0];
    assert!(
        now > before + 40.,
        "browser width {now}: expected more than {before}"
    );
}
#[then("the viewport becomes narrower")]
fn viewport_narrower(world: &mut ApplicationWorld) {
    let before = world.strip_left.expect("expected a starting strip");
    let now = window_geometry(TAB_STRIP)
        .expect("expected the tab strip")
        .0[0];
    assert!(
        now > before + 40.,
        "document area starts at {now}: expected past {before}"
    );
}
#[then("the document and model tree remain displayed")]
fn tree_remains(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    let id = world.active_id();
    world.expect_browser_of(id);
}
#[when("I drag the boundary back to its original position")]
fn drag_back(world: &mut ApplicationWorld) {
    let before = world.browser_width.expect("expected a starting width");
    let now = browser_geometry().1[0];
    drag_boundary(world, (before - now) as i32);
}
#[then("the model browser and viewport return to their previous widths")]
fn widths_back(world: &mut ApplicationWorld) {
    let before = world.browser_width.expect("expected a starting width");
    let now = browser_geometry().1[0];
    assert!(
        (now - before).abs() <= 1.,
        "browser width {now}: expected {before}"
    );
    let strip = world.strip_left.expect("expected a starting strip");
    let left = window_geometry(TAB_STRIP)
        .expect("expected the tab strip")
        .0[0];
    assert!(
        (left - strip).abs() <= 1.,
        "document area {left}: expected {strip}"
    );
}
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
fn open_browser_menu(world: &mut ApplicationWorld) {
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
fn type_in_search(world: &mut ApplicationWorld, query: &str) {
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
fn expect_sheet_at(world: &mut ApplicationWorld, at: [usize; 2]) {
    let sheet = world.pixel_at(at);
    assert!(
        sheet[..3].iter().all(|channel| *channel > 200),
        "pixel {at:?} {sheet:?}: expected the light drawing sheet"
    );
}
#[given("its browser is docked to the left of the drawing")]
fn drawing_browser_docked(world: &mut ApplicationWorld) {
    expect_docked_left(world);
}
#[when("I drag the browser header into the drawing area")]
fn drag_over_drawing(world: &mut ApplicationWorld) {
    drag_header(world, [800, 300]);
}
#[then("the browser floats over the drawing sheet")]
fn floats_over_sheet(world: &mut ApplicationWorld) {
    let (position, _) = browser_geometry();
    assert!(
        position[0] > 400.,
        "browser {position:?}: expected it over the sheet"
    );
}
#[then("the browser displays drawing resources and the sheet tree")]
fn drawing_tree_shown(world: &mut ApplicationWorld) {
    world.expect_text("Drawing Resources");
    world.expect_text("Sheet:1");
}
#[then("the drawing remains visible behind the browser")]
fn sheet_behind(world: &mut ApplicationWorld) {
    expect_sheet_at(world, [500, 500]);
}
#[then("the browser docks at the left edge")]
fn drawing_browser_docks(world: &mut ApplicationWorld) {
    expect_docked_left(world);
}
#[then("the drawing remains visible in the remaining workspace")]
fn sheet_remains(world: &mut ApplicationWorld) {
    expect_sheet_at(world, [1000, 500]);
}
