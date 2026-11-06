// SPDX-License-Identifier: GPL-2.0-only
use super::{ApplicationWorld, diagnostics::window_geometry};
use crate::HeadWork;
use cucumber::{given, then, when};
const BROWSER: &str = "Model";
fn browser_geometry() -> ([f32; 2], [f32; 2]) {
    window_geometry(BROWSER).expect("model browser: expected a drawn window")
}
fn drag_header(world: &mut ApplicationWorld, to: [i32; 2]) {
    // The title bar, not the Model tab with the same words below it.
    let (position, _) = browser_geometry();
    let from = [position[0] as i32 + 40, position[1] as i32 + 9];
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
#[then("the browser remains floating over the sheet")]
fn floats_over_sheet(_world: &mut ApplicationWorld) {
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
#[then("the drawing sheet remains visible behind the browser")]
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
#[given("its model browser is docked on the left")]
#[given("the drawing browser is docked on the left")]
#[given("its Model browser is docked on the left")]
fn docked_on_left(world: &mut ApplicationWorld) {
    expect_docked_left(world);
    world.expect_text("Drawing Resources");
}
#[when("I select the browser tab's close control")]
fn close_browser_tab(world: &mut ApplicationWorld) {
    world.click_run_close_mark("Model");
}
#[then("the browser tree is hidden")]
fn tree_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Drawing Resources");
    assert!(
        window_geometry(BROWSER).is_none(),
        "expected no browser window"
    );
}
#[then("the drawing occupies the space previously used by the browser")]
fn drawing_takes_space(world: &mut ApplicationWorld) {
    // This column was inside the docked browser, and the wider sheet now reaches it.
    expect_sheet_at(world, [300, 500]);
}
#[then("a plus control remains at the upper left of the workspace")]
fn plus_remains(world: &mut ApplicationWorld) {
    let plus = world.run("+").center();
    assert!(
        plus[0] < 60. && plus[1] < 260.,
        "plus control {plus:?}: expected the upper left"
    );
}
#[when("I open the plus control")]
fn open_plus(world: &mut ApplicationWorld) {
    world.click_text("+");
    world.pump(Vec::new());
}
#[then("the panel list offers Model Browser")]
fn offers_model(world: &mut ApplicationWorld) {
    world.expect_text("Model Browser");
}
#[when("I select Model Browser")]
fn select_model_browser(world: &mut ApplicationWorld) {
    world.click_text("Model Browser");
    world.pump(Vec::new());
}
#[then("the browser returns to the left")]
fn browser_back(world: &mut ApplicationWorld) {
    expect_docked_left(world);
}
#[then("it displays the drawing resources and sheet tree")]
#[then("the drawing resources and sheet tree are displayed")]
fn drawing_tree(world: &mut ApplicationWorld) {
    world.expect_text("Drawing Resources");
    world.expect_text("Sheet:1");
}
#[given("an additional log panel is available in the browser's plus menu")]
fn log_available(world: &mut ApplicationWorld) {
    open_plus(world);
    world.expect_text("Log Panel");
    world.escape();
}
#[when("I add the log panel from the plus menu")]
fn add_log(world: &mut ApplicationWorld) {
    open_plus(world);
    world.click_text("Log Panel");
    world.pump(Vec::new());
}
fn expect_log_area(world: &mut ApplicationWorld) {
    world.expect_text("Application messages");
    world.expect_no_text("Drawing Resources");
}
#[then("the log panel becomes active in the same dock")]
#[then("the log area is displayed in the dock")]
fn log_active(world: &mut ApplicationWorld) {
    expect_log_area(world);
    expect_docked_left(world);
}
#[then("the Model browser remains available as a tab")]
fn model_tab_remains(world: &mut ApplicationWorld) {
    // The window's title and the tab both read Model.
    let tabs = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == BROWSER)
        .count();
    assert_eq!(tabs, 2, "expected the title and the Model tab");
}
#[when("I select the Model browser tab")]
fn select_model_tab(world: &mut ApplicationWorld) {
    world.click_text(BROWSER);
    world.pump(Vec::new());
}
#[when("I select the log panel tab")]
fn select_log_tab(world: &mut ApplicationWorld) {
    world.click_text("Log");
    world.pump(Vec::new());
}
#[when("I close the log panel tab")]
fn close_log_tab(world: &mut ApplicationWorld) {
    world.click_run_close_mark("Log");
    world.pump(Vec::new());
}
#[then("the Model browser is displayed")]
fn model_displayed_again(world: &mut ApplicationWorld) {
    drawing_tree(world);
    world.expect_no_text("Application messages");
}
#[then("the drawing remains visible")]
fn drawing_visible(world: &mut ApplicationWorld) {
    expect_sheet_at(world, [1000, 500]);
}
const DRAWING_BORDER: &str = "Default Border";
const SHEET_TITLE_BLOCK: &str = "Title Block";
#[given("a drawing with drawing resources and a sheet is open in the MCAD Software")]
#[given("a drawing with border entries in its resources and sheet is open in the MCAD Software")]
#[given("a drawing with browser entries is open in the MCAD Software")]
#[given("a drawing with a border and title block is open in the MCAD Software")]
#[given("a drawing with a sheet and drawing resources is open in the MCAD Software")]
#[given("a drawing with a border resource is open in the MCAD Software")]
fn drawing_open(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.add_drawing());
}
#[given("the browser has expanded branches containing nested entries")]
#[given("the sheet branch is expanded in the drawing browser")]
#[given("the sheet and drawing resources branches are expanded in the browser")]
fn drawing_branches_open(world: &mut ApplicationWorld) {
    world.expect_text(DRAWING_BORDER);
}
#[when("I open the drawing browser menu")]
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
#[when("I open the floating browser search field")]
#[given("the drawing browser search field is open")]
fn open_drawing_search(world: &mut ApplicationWorld) {
    world.click_text("Search");
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
fn diagnostics_center(run: &super::diagnostics::RenderedRun) -> [f32; 2] {
    run.center()
}
#[then("matching text is highlighted")]
#[then("the matching text is highlighted")]
fn border_marked(world: &mut ApplicationWorld) {
    assert!(
        world.end_of_text_is_marked(DRAWING_BORDER),
        "expected a marked match"
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
    world.expect_no_text("x");
    drawing_tree_back(world);
}
#[when("I enter a query that matches no drawing browser entry")]
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
fn drawing_header_right(world: &mut ApplicationWorld) {
    let width = world.extent()[0] as i32;
    drag_header(world, [width - 10, 400]);
}
#[then("the drawing browser docks on the right")]
fn drawing_docks_right(world: &mut ApplicationWorld) {
    let (position, size) = browser_geometry();
    let width = world.extent()[0] as f32;
    assert!(
        position[0] + size[0] > width - 20. && position[0] > width / 2.,
        "browser {position:?} {size:?}: expected the right edge"
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
#[when("I select the expansion control beside the sheet branch again")]
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
