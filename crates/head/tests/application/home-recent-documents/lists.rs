// SPDX-License-Identifier: GPL-2.0-only
//! Sorting, row checks and menus, pins, columns and the empty state of the recent list.
use super::*;

/// The row's box lies this far left of its name: a kind mark sits between them.
const CHECK_REACH: f32 = 52.;
const ROW_BUCKET: f32 = 20.;
const CARD_CHECK_ABOVE_NAME: i32 = 58;
fn row_run(world: &ApplicationWorld, name: &str) -> RenderedRun {
    home_run(world, name).unwrap_or_else(|| panic!("expected a row {name}"))
}
/// The row's box sits at the row's start, left of everything else on it.
fn row_check_at(world: &ApplicationWorld, name: &str) -> [i32; 2] {
    let run = row_run(world, name);
    [
        (run.minimum[0] - CHECK_REACH + 8.) as i32,
        run.center()[1] as i32,
    ]
}
fn toggle_row(world: &mut ApplicationWorld, name: &str) {
    let at = row_check_at(world, name);
    world.click(at);
    world.pump(Vec::new());
}
fn row_checked(world: &ApplicationWorld, name: &str) -> bool {
    let run = row_run(world, name);
    world.frame.marks.iter().any(|mark| {
        mark.bright
            && mark.position[0] < run.minimum[0]
            && (mark.position[1] - run.center()[1]).abs() < 6.
            && run.minimum[0] - mark.position[0] < CHECK_REACH + 8.
    })
}
/// A card's box sits at its top-left corner, above its kind button and name.
fn card_check_at(world: &ApplicationWorld, name: &str) -> [i32; 2] {
    let card = row_run(world, name);
    [
        card.minimum[0] as i32 + 10,
        card.center()[1] as i32 - CARD_CHECK_ABOVE_NAME,
    ]
}
fn order_of(world: &ApplicationWorld, names: [&str; 3]) -> Vec<String> {
    // Glyphs differ in height, so a row's position is bucketed before comparing across it.
    let mut found: Vec<(f32, f32, String)> = names
        .iter()
        .map(|name| {
            let center = row_run(world, name).center();
            (
                (center[1] / ROW_BUCKET).floor(),
                center[0],
                (*name).to_owned(),
            )
        })
        .collect();
    found.sort_by(|a, b| a.0.total_cmp(&b.0).then(a.1.total_cmp(&b.1)));
    found.into_iter().map(|row| row.2).collect()
}
const NAMES: [&str; 3] = [OTHERS[0], OTHERS[1], SEARCHED];
fn expect_descending_names(world: &ApplicationWorld) {
    let order = order_of(world, NAMES);
    assert_eq!(
        order,
        [SEARCHED, OTHERS[1], OTHERS[0]],
        "descending, found {order:?}"
    );
}
fn expect_ascending_names(world: &ApplicationWorld) {
    let order = order_of(world, NAMES);
    assert_eq!(
        order,
        [OTHERS[0], OTHERS[1], SEARCHED],
        "ascending, found {order:?}"
    );
}
fn expect_selected_actions(world: &ApplicationWorld) {
    for action in SELECTED_ACTIONS {
        assert!(home_shows(world, action), "expected {action}");
    }
}
fn expect_no_selected_actions(world: &ApplicationWorld) {
    for action in SELECTED_ACTIONS {
        assert!(!home_shows(world, action), "expected no {action}");
    }
}
// Sort menu
#[given("recent documents with different names are shown in Grid View")]
fn names_in_grid(world: &mut ApplicationWorld) {
    click_home(world, "Grid View");
}
#[given("Sort by is set to Last Opened")]
fn sort_last_opened(world: &mut ApplicationWorld) {
    assert!(
        home_shows(world, "Sort by Last Opened"),
        "expected the default sort"
    );
}
#[when("I open the Sort by menu")]
fn open_sort(world: &mut ApplicationWorld) {
    click_home(world, "Sort by");
}
#[then("it offers Name, Location, Date Modified, Last Opened, Size and Area")]
fn sort_offers(world: &mut ApplicationWorld) {
    for field in [
        "Name",
        "Location",
        "Date Modified",
        "Last Opened",
        "Size",
        "Area",
    ] {
        world.expect_text(field);
    }
}
#[when("I choose Name")]
fn choose_name(world: &mut ApplicationWorld) {
    world.click_text("Name");
    world.pump(Vec::new());
}
#[then("the sort field shows Name")]
fn field_name(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "Sort by Name"), "expected the Name sort");
}
#[then("the visible cards are ordered by name in descending order")]
#[then("the visible rows are ordered by name in descending order")]
fn cards_descending(world: &mut ApplicationWorld) {
    expect_descending_names(world);
}
#[when("I activate the sort direction control")]
fn sort_direction(world: &mut ApplicationWorld) {
    click_home(world, "v");
}
#[then("the visible cards are ordered by name in ascending order")]
#[then("the visible rows are ordered by name in ascending order")]
#[then("the visible cards follow the ascending name order")]
fn cards_ascending(world: &mut ApplicationWorld) {
    expect_ascending_names(world);
}
// Row menus
#[given("no recent-document checkboxes are selected")]
#[given("no recent-document rows are checked")]
fn no_checks(world: &mut ApplicationWorld) {
    expect_no_selected_actions(world);
}
fn open_row_actions(world: &mut ApplicationWorld) {
    open_actions_of(world, SEARCHED);
}
/// The control shows only on a row under the pointer.
fn open_actions_of(world: &mut ApplicationWorld, name: &str) {
    let row_at = row_run(world, name).center();
    world.hover(row_at.map(|value| value as i32));
    world.pump(Vec::new());
    let row = row_at[1];
    let control = world
        .frame
        .runs
        .iter()
        .find(|run| in_home(run) && run.text == "..." && (run.center()[1] - row).abs() < 12.)
        .cloned()
        .expect("expected the row's actions control");
    world.click(control.center().map(|value| value as i32));
    world.pump(Vec::new());
}
#[when("I open the actions menu on a recent part row")]
#[when("I open a recent row's actions menu")]
#[when("I open the part row's actions menu")]
fn open_actions(world: &mut ApplicationWorld) {
    open_row_actions(world);
}
#[then("the menu offers Open and Open with options")]
fn menu_open(world: &mut ApplicationWorld) {
    world.expect_text("Open with options");
}
#[then("the menu offers Remove from Recent")]
fn menu_remove(world: &mut ApplicationWorld) {
    world.expect_text("Remove from Recent");
}
#[then("the menu offers commands to open or explore the containing folder")]
fn menu_folder(world: &mut ApplicationWorld) {
    world.expect_text("Open Containing Folder");
    world.expect_text("Explore Containing Folder");
}
#[then("the menu offers document properties")]
fn menu_properties(world: &mut ApplicationWorld) {
    world.expect_text("Document Properties");
}
#[given("a recent-document checkbox is selected")]
#[given("one recent-document row is checked")]
#[given("the first part's row is checked")]
#[given("one part row is checked")]
fn one_checked(world: &mut ApplicationWorld) {
    toggle_row(world, SEARCHED);
    assert!(row_checked(world, SEARCHED), "expected {SEARCHED} checked");
}
#[then("the menu offers Open Selected and Remove Selected")]
fn menu_selected(world: &mut ApplicationWorld) {
    world.expect_text("Open Selected");
    world.expect_text("Remove Selected");
}
#[when("I check a second recent-document row")]
fn check_second(world: &mut ApplicationWorld) {
    toggle_row(world, OTHERS[0]);
}
#[then("both rows are checked and highlighted")]
#[then("both parts' rows are checked and highlighted")]
fn both_checked(world: &mut ApplicationWorld) {
    assert!(row_checked(world, SEARCHED), "expected {SEARCHED} checked");
    assert!(
        row_checked(world, OTHERS[0]),
        "expected {} checked",
        OTHERS[0]
    );
    assert!(
        world.row_accented_beside(SEARCHED, OTHERS[1]),
        "expected a highlighted row"
    );
}
#[then("Open Selected and Remove Selected appear above the list")]
#[then("the selected-item actions remain visible")]
#[then("Open Selected and Remove Selected are displayed")]
#[then("Open Selected and Remove Selected remain displayed")]
fn selected_shown(world: &mut ApplicationWorld) {
    expect_selected_actions(world);
}
#[when("I clear the second row's checkbox")]
fn clear_second(world: &mut ApplicationWorld) {
    toggle_row(world, OTHERS[0]);
}
#[then("the first row remains checked")]
fn first_checked(world: &mut ApplicationWorld) {
    assert!(row_checked(world, SEARCHED), "expected {SEARCHED} checked");
    assert!(
        !row_checked(world, OTHERS[0]),
        "expected {} clear",
        OTHERS[0]
    );
}
#[when("I clear the first row's checkbox")]
fn clear_first(world: &mut ApplicationWorld) {
    toggle_row(world, SEARCHED);
}
#[then("no recent-document rows are checked")]
fn none_checked(world: &mut ApplicationWorld) {
    assert!(!row_checked(world, SEARCHED), "expected {SEARCHED} clear");
}
#[then("Open Selected and Remove Selected are no longer displayed above the list")]
#[then("Open Selected and Remove Selected are no longer displayed")]
#[then("Open Selected and Remove Selected remain absent")]
fn selected_gone(world: &mut ApplicationWorld) {
    expect_no_selected_actions(world);
}
// Pins
#[given("an unpinned recent part is displayed in Grid View")]
fn unpinned_in_grid(world: &mut ApplicationWorld) {
    click_home(world, "Grid View");
    assert!(home_shows(world, SEARCHED), "expected {SEARCHED}");
}
#[when("I hover over the part's card")]
fn hover_card(world: &mut ApplicationWorld) {
    let at = row_run(world, SEARCHED).center().map(|value| value as i32);
    world.hover(at);
    world.pump(Vec::new());
}
#[then("a Pin control appears on the card")]
#[then("the control offers Pin again")]
fn pin_offered(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "Pin"), "expected Pin");
}
#[when("I activate Pin")]
fn activate_pin(world: &mut ApplicationWorld) {
    click_home(world, "Pin");
}
#[then("the control offers Unpin")]
fn unpin_offered(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "Unpin"), "expected Unpin");
}
#[then("the part remains in the recent grid")]
fn stays_in_grid(world: &mut ApplicationWorld) {
    assert!(home_shows(world, SEARCHED), "expected {SEARCHED}");
}
#[when("I activate Unpin")]
fn activate_unpin(world: &mut ApplicationWorld) {
    click_home(world, "Unpin");
}
// Columns
fn column_x(world: &ApplicationWorld, column: &str) -> f32 {
    home_run(world, column)
        .expect("expected the column")
        .center()[0]
}
#[given("Location is displayed immediately after Name")]
fn location_after_name(world: &mut ApplicationWorld) {
    assert!(
        column_x(world, "Name") < column_x(world, "Location"),
        "column order"
    );
    assert!(
        column_x(world, "Location") < column_x(world, "Date Modified"),
        "column order"
    );
}
#[when("I open Column Settings and clear Location")]
fn clear_location(world: &mut ApplicationWorld) {
    click_home(world, "Column Settings");
    world.pump(Vec::new());
    let available = world.run("Available Columns").center()[1];
    let row = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| run.text == "Location" && run.center()[1] > available)
        .cloned()
        .expect("expected Location in Available Columns");
    world.click(row.center().map(|value| value as i32));
    world.pump(Vec::new());
}
#[then("the Location header and values disappear from the list")]
fn location_gone(world: &mut ApplicationWorld) {
    let list_row = row_run(world, SEARCHED);
    assert!(!home_shows(world, "/tmp"), "expected no location values");
    assert!(list_row.center()[0] > 0., "expected the row to remain");
}
#[then("Location is unchecked in Available Columns")]
fn location_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Location"), "expected Location clear");
}
#[when("I select Location again")]
fn select_location(world: &mut ApplicationWorld) {
    world.click_text("Location");
    world.pump(Vec::new());
}
#[then("Location appears after the other adjustable columns")]
fn location_last(world: &mut ApplicationWorld) {
    let displayed = world.run("Displayed Columns").center()[1];
    let available = world.run("Available Columns").center()[1];
    let order: Vec<f32> = ["Date Modified", "Last Opened", "Size", "Location"]
        .iter()
        .map(|name| displayed_row(world, name, displayed, available).center()[1])
        .collect();
    assert!(
        order.windows(2).all(|pair| pair[0] < pair[1]),
        "displayed rows at {order:?}"
    );
}
fn displayed_row(world: &ApplicationWorld, name: &str, above: f32, below: f32) -> RenderedRun {
    let wanted = name.replace(' ', "");
    world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| run.text == wanted && run.center()[1] > above && run.center()[1] < below)
        .cloned()
        .unwrap_or_else(|| panic!("expected {name} in Displayed Columns"))
}
#[when("I drag Location above Date Modified in Displayed Columns")]
fn drag_location(world: &mut ApplicationWorld) {
    let displayed = world.run("Displayed Columns").center()[1];
    let available = world.run("Available Columns").center()[1];
    let from = displayed_row(world, "Location", displayed, available).center();
    let to = displayed_row(world, "Date Modified", displayed, available).center();
    world.drag(from.map(|value| value as i32), to.map(|value| value as i32));
    world.pump(Vec::new());
}
#[then("Location appears immediately after Name in the list")]
fn location_back(world: &mut ApplicationWorld) {
    assert!(
        column_x(world, "Location") < column_x(world, "Date Modified"),
        "column order"
    );
}
// History recall
#[given(regex = r#"^the recent search history contains "Part26"$"#)]
fn history_contains(world: &mut ApplicationWorld) {
    search_for(world, SEARCHED);
}
#[given("the search field is empty and the broader recent list is displayed")]
fn empty_field_broad(world: &mut ApplicationWorld) {
    let run = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| in_home(run) && run.text == "x")
        .cloned()
        .expect("expected a clear control");
    world.click(run.center().map(|value| value as i32));
    world.pump(Vec::new());
    click_home(world, "Search");
    assert!(home_shows(world, OTHERS[0]), "expected the broader list");
}
#[when("I focus Search recent")]
fn focus_search(world: &mut ApplicationWorld) {
    click_home(world, "Search recent");
}
#[when(regex = r#"^I choose "Part26" from Search History$"#)]
fn choose_term(world: &mut ApplicationWorld) {
    let panel = home_run(world, "Search History")
        .expect("expected the panel")
        .center()[1];
    let term = world
        .frame
        .runs
        .iter()
        .find(|run| in_home(run) && run.text == SEARCHED && run.center()[1] > panel)
        .cloned()
        .expect("expected the term");
    world.click(term.center().map(|value| value as i32));
    world.pump(Vec::new());
}
#[then(regex = r#"^the search field contains "Part26"$"#)]
#[then(regex = r#"^the search field still contains "Part26"$"#)]
fn field_contains(world: &mut ApplicationWorld) {
    assert!(
        !home_shows(world, "Search recent"),
        "expected text in place of the hint"
    );
    assert!(home_shows(world, SEARCHED), "expected the term");
}
// Header checkbox
fn header_box_at(world: &ApplicationWorld) -> [i32; 2] {
    let name = home_run(world, "Name").expect("expected the Name column");
    [
        (name.minimum[0] - CHECK_REACH + 8.) as i32,
        name.center()[1] as i32,
    ]
}
fn toggle_header(world: &mut ApplicationWorld) {
    let at = header_box_at(world);
    world.click(at);
    world.pump(Vec::new());
}
#[when("I check the list header checkbox")]
#[when("I activate the partially selected header checkbox")]
#[when("I clear the header checkbox")]
fn header_box(world: &mut ApplicationWorld) {
    toggle_header(world);
}
#[then("all visible recent-document rows are checked and highlighted")]
#[then("all visible recent-document rows are checked again")]
fn all_checked(world: &mut ApplicationWorld) {
    for name in NAMES {
        assert!(row_checked(world, name), "expected {name} checked");
    }
}
#[when("I clear one visible row's checkbox")]
fn clear_one(world: &mut ApplicationWorld) {
    toggle_row(world, SEARCHED);
}
#[then("that row is unchecked while the other visible rows remain checked")]
fn others_checked(world: &mut ApplicationWorld) {
    assert!(!row_checked(world, SEARCHED), "expected {SEARCHED} clear");
    assert!(
        row_checked(world, OTHERS[0]),
        "expected {} checked",
        OTHERS[0]
    );
}
#[then("the header checkbox shows a partial selection")]
fn header_partial(world: &mut ApplicationWorld) {
    let at = header_box_at(world).map(|value| value as usize);
    let [x, y] = at;
    let lit = (0..14).any(|offset| {
        let color = world.pixel_at([x - 7 + offset, y]);
        color[..3].iter().any(|channel| *channel > 110)
    });
    assert!(lit, "expected a partial mark in the header box");
}
#[then("the visible recent-document rows are unchecked")]
fn rows_unchecked(world: &mut ApplicationWorld) {
    for name in NAMES {
        assert!(!row_checked(world, name), "expected {name} clear");
    }
}
// Column header sort
#[given("Last Opened is the active descending sort")]
fn opened_descending(world: &mut ApplicationWorld) {
    assert!(
        home_shows(world, "Last Opened v"),
        "expected the default sort mark"
    );
}
#[when("I select the Name column header")]
#[when("I select the Name column header again")]
fn select_name_header(world: &mut ApplicationWorld) {
    click_home(world, "Name");
}
#[then("a downward sort indicator appears in the Name header")]
fn name_down(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "Name v"), "expected a downward mark");
}
#[then("the Name header shows an upward sort indicator")]
fn name_up(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "Name ^"), "expected an upward mark");
}
#[then("Sort by shows Name with an upward direction indicator")]
fn sort_by_up(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "Sort by Name"), "expected the Name sort");
    assert!(home_shows(world, "^"), "expected an upward direction");
}
// Empty state
#[given(regex = r#"^no recent document matches "zzq_no_match_7319"$"#)]
fn nothing_matches(world: &mut ApplicationWorld) {
    assert!(!home_shows(world, NO_MATCH), "expected no match");
}
#[when(regex = r#"^I enter "zzq_no_match_7319" in Search recent$"#)]
fn enter_no_match(world: &mut ApplicationWorld) {
    enter_search(world, NO_MATCH);
}
#[then("the recent-document rows are replaced by a No results message")]
#[then("No results is displayed")]
fn no_results(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "No results"), "expected No results");
    assert!(!home_shows(world, OTHERS[0]), "expected no rows");
}
#[then("the page offers Clear search and filters")]
fn offers_clear(world: &mut ApplicationWorld) {
    assert!(
        home_shows(world, "Clear search and filters"),
        "expected the button"
    );
}
#[when("I activate Clear search and filters")]
fn activate_clear(world: &mut ApplicationWorld) {
    click_home(world, "Clear search and filters");
}
#[given(regex = r#"^no assembly matches "Part26"$"#)]
fn no_assembly_match(_: &mut ApplicationWorld) {
    assert!(
        !ASSEMBLY.contains(SEARCHED),
        "expected the fixture's names apart"
    );
}
#[when(regex = r#"^I search recent documents for "Part26"$"#)]
fn search_part26(world: &mut ApplicationWorld) {
    search_for(world, SEARCHED);
}
#[when("I open Filter")]
fn when_open_filter(world: &mut ApplicationWorld) {
    open_filter(world);
}
#[then("Assembly is unchecked")]
fn assembly_clear(world: &mut ApplicationWorld) {
    assert!(!world.checked("Assembly"), "expected Assembly clear");
}
#[given(regex = r#"^searching for "Part26" displays a matching recent part$"#)]
fn searching_shows(world: &mut ApplicationWorld) {
    search_for(world, SEARCHED);
    assert!(!home_shows(world, OTHERS[0]), "expected only the match");
}
#[when("I activate the clear control beside the active filter indicator")]
#[when("I clear the active filter")]
fn clear_indicator(world: &mut ApplicationWorld) {
    let filter = home_run(world, "Filter (").expect("expected an active filter");
    let control = world
        .frame
        .runs
        .iter()
        .find(|run| {
            in_home(run)
                && run.text == "x"
                && run.minimum[0] > filter.maximum[0]
                && (run.center()[1] - filter.center()[1]).abs() < 4.
        })
        .cloned()
        .expect("expected the clear control beside the indicator");
    world.click(control.center().map(|value| value as i32));
    world.pump(Vec::new());
}
#[then("the active-filter count is removed")]
fn count_gone(world: &mut ApplicationWorld) {
    world.expect_no_text("Filter (");
}
// Selection across views and filters
#[then("the first part's card is checked and highlighted")]
fn card_checked(world: &mut ApplicationWorld) {
    let at = card_check_at(world, SEARCHED);
    let mark = world.frame.marks.iter().any(|mark| {
        mark.bright
            && (mark.position[0] - at[0] as f32).abs() < 12.
            && (mark.position[1] - at[1] as f32).abs() < 12.
    });
    assert!(mark, "expected a check on the card near {at:?}");
}
#[when("I check the second part's card")]
fn check_card(world: &mut ApplicationWorld) {
    let at = card_check_at(world, OTHERS[0]);
    world.click(at);
    world.pump(Vec::new());
}
#[then("assembly rows are displayed and the selected part is not displayed")]
fn assembly_only(world: &mut ApplicationWorld) {
    assert!(home_shows(world, ASSEMBLY), "expected {ASSEMBLY}");
    assert!(!home_shows(world, SEARCHED), "expected no {SEARCHED}");
}
#[then("the part row is displayed again without a checkmark")]
fn part_unchecked(world: &mut ApplicationWorld) {
    assert!(!row_checked(world, SEARCHED), "expected {SEARCHED} clear");
}
// Properties
#[when("I choose the document properties action")]
fn choose_properties(world: &mut ApplicationWorld) {
    world.click_text("Document Properties");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("an informational notice is displayed")]
fn notice_shown(world: &mut ApplicationWorld) {
    world.expect_text("These properties come from the file on disk.");
}
#[when("I dismiss the informational notice")]
fn dismiss_notice(world: &mut ApplicationWorld) {
    world.click_text("OK");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("a file properties window for the part is displayed")]
fn properties_shown(world: &mut ApplicationWorld) {
    world.expect_text(&format!("{SEARCHED} Properties"));
}
#[then("its General page shows the file name, type, location, sizes, dates and attributes")]
fn general_shown(world: &mut ApplicationWorld) {
    for field in [
        "File name:",
        "Type:",
        "Location:",
        "Size:",
        "Size on disk:",
        "Created:",
        "Modified:",
        "Attributes:",
    ] {
        world.expect_text(field);
    }
}
#[when("I dismiss the properties window")]
fn dismiss_properties(world: &mut ApplicationWorld) {
    world.click_text("Close");
    world.pump(Vec::new());
}
#[then("the Recent list is visible again")]
fn recent_visible(world: &mut ApplicationWorld) {
    world.expect_no_text(&format!("{SEARCHED} Properties"));
    assert!(home_shows(world, SEARCHED), "expected the list");
}
// Mixed selection
#[given("a recent part and a recent assembly are checked in List View")]
fn mixed_checked(world: &mut ApplicationWorld) {
    toggle_row(world, SEARCHED);
    toggle_row(world, ASSEMBLY);
}
#[when("I open Filter and select Part")]
fn filter_part(world: &mut ApplicationWorld) {
    open_filter(world);
    world.click_text("Part");
    world.pump(Vec::new());
}
#[then("the selected part remains checked and highlighted")]
fn part_still_checked(world: &mut ApplicationWorld) {
    assert!(row_checked(world, SEARCHED), "expected {SEARCHED} checked");
}
#[then("the part remains checked")]
fn part_remains_checked(world: &mut ApplicationWorld) {
    assert!(row_checked(world, SEARCHED), "expected {SEARCHED} checked");
}
#[then("the assembly is displayed again unchecked")]
fn assembly_back_clear(world: &mut ApplicationWorld) {
    assert!(!row_checked(world, ASSEMBLY), "expected {ASSEMBLY} clear");
}
// Size sort
#[when("I select the Size column header")]
#[when("I select the Size column header again")]
fn select_size_header(world: &mut ApplicationWorld) {
    click_home(world, "Size");
}
fn rows_by_size(world: &ApplicationWorld) -> Vec<(f32, u64)> {
    let state = world
        .app
        .as_ref()
        .map(|app| crate::HeadWork::gui_command_state(&app.owner).unwrap())
        .expect("expected an application");
    let mut rows: Vec<(f32, u64)> = state
        .recent_documents
        .iter()
        .map(|row| {
            (
                row_run(world, &row.name).center()[1],
                row.size_bytes.unwrap_or(0),
            )
        })
        .collect();
    rows.sort_by(|a, b| a.0.total_cmp(&b.0));
    rows
}
#[then("the visible rows with known sizes are ordered from larger to smaller")]
fn larger_first(world: &mut ApplicationWorld) {
    let rows = rows_by_size(world);
    assert!(
        rows.windows(2).all(|pair| pair[0].1 >= pair[1].1),
        "sizes {rows:?}"
    );
}
#[then("the visible rows with known sizes are ordered from smaller to larger")]
fn smaller_first(world: &mut ApplicationWorld) {
    let rows = rows_by_size(world);
    assert!(
        rows.windows(2).all(|pair| pair[0].1 <= pair[1].1),
        "sizes {rows:?}"
    );
}
// Views keep the search and filter
#[given(regex = r#"^a recent part named "Part26" is displayed in List View$"#)]
fn part26_listed(world: &mut ApplicationWorld) {
    assert!(home_shows(world, SEARCHED), "expected {SEARCHED}");
    assert!(home_shows(world, "Location"), "expected the list's columns");
}
#[given(regex = r#"^the search for "Part26" and the Part filter are active$"#)]
fn search_and_filter(world: &mut ApplicationWorld) {
    search_for(world, SEARCHED);
    filter_part(world);
    expect_filter_count(world, 1);
}
#[then("the matching part is displayed as a card")]
fn shown_as_card(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "Sort by"), "expected the grid's controls");
    assert!(home_shows(world, SEARCHED), "expected {SEARCHED}");
}
#[then("the filter control still indicates one active filter")]
fn still_one(world: &mut ApplicationWorld) {
    expect_filter_count(world, 1);
}
#[then("the matching part is displayed as a row")]
fn shown_as_row(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "Location"), "expected the list's columns");
    assert!(home_shows(world, SEARCHED), "expected {SEARCHED}");
}
// Reset columns
#[then("the Location column disappears")]
fn location_column_gone(world: &mut ApplicationWorld) {
    location_gone(world);
}
#[when("I choose Reset to default")]
fn reset_columns(world: &mut ApplicationWorld) {
    world.click_text("Reset to default");
    world.pump(Vec::new());
}
#[then("Location is checked in Available Columns")]
fn location_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Location"), "expected Location checked");
}
#[then("Location is displayed immediately after Name again")]
fn location_after_name_again(world: &mut ApplicationWorld) {
    location_after_name(world);
}
#[then("Column Settings remains open")]
fn columns_open(world: &mut ApplicationWorld) {
    world.expect_text("Available Columns");
}
#[then("the Filter panel remains open")]
fn filter_open(world: &mut ApplicationWorld) {
    world.expect_text("File Types");
}
// Partial names
const PARTIAL: &str = "part2";
fn add_partial_files(world: &mut ApplicationWorld) {
    for name in ["Part2", "Part23"] {
        let part = world.add_part();
        world.save_part_to_file(part, &format!("{name}.opd"));
    }
    world.click_text("Home");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[given(regex = r#"^the recent list includes parts named "Part2", "Part23" and "Part26"$"#)]
fn partial_named(world: &mut ApplicationWorld) {
    add_partial_files(world);
}
#[when(regex = r#"^I enter "part2" in Search recent$"#)]
fn enter_partial(world: &mut ApplicationWorld) {
    enter_search(world, PARTIAL);
}
const PARTIAL_NAMES: [&str; 3] = ["Part2", "Part23", SEARCHED];
#[then(regex = r#"^the results include "Part2", "Part23" and "Part26"$"#)]
fn partial_results(world: &mut ApplicationWorld) {
    for name in PARTIAL_NAMES {
        assert!(home_shows(world, name), "expected {name}");
    }
    assert!(!home_shows(world, OTHERS[0]), "expected no {}", OTHERS[0]);
}
#[then("the matching portion of each displayed name is highlighted")]
fn portion_lit(world: &mut ApplicationWorld) {
    for name in PARTIAL_NAMES {
        assert!(name_is_lit(world, name), "expected {name} lit");
    }
}
#[given(regex = r#"^a search for "part2" displays several matching part rows in List View$"#)]
fn partial_search_rows(world: &mut ApplicationWorld) {
    add_partial_files(world);
    search_for(world, PARTIAL);
    partial_results(world);
}
#[given("other recent documents do not match the search")]
fn others_do_not_match(world: &mut ApplicationWorld) {
    assert!(!home_shows(world, ASSEMBLY), "expected no {ASSEMBLY}");
}
#[then("all displayed search-result rows are checked and highlighted")]
fn results_checked(world: &mut ApplicationWorld) {
    for name in PARTIAL_NAMES {
        assert!(row_checked(world, name), "expected {name} checked");
    }
}
#[when("I clear the search field and activate Search")]
fn clear_and_search(world: &mut ApplicationWorld) {
    clear_field(world);
    click_home(world, "Search");
}
#[then("the visible previously selected matching rows remain checked")]
fn matches_checked(world: &mut ApplicationWorld) {
    results_checked(world);
}
#[then("the visible nonmatching rows are unchecked")]
fn nonmatching_clear(world: &mut ApplicationWorld) {
    assert!(
        !row_checked(world, OTHERS[0]),
        "expected {} clear",
        OTHERS[0]
    );
    assert!(!row_checked(world, ASSEMBLY), "expected {ASSEMBLY} clear");
}
/// The row's own name in Home, read against the plain button beyond it.
fn name_is_lit(world: &mut ApplicationWorld, name: &str) -> bool {
    let run = row_run(world, name);
    let y = run.center()[1] as usize;
    let plain = world.pixel_at([run.minimum[0] as usize - 3, y]);
    let inside = world.pixel_at([run.minimum[0] as usize + 1, run.minimum[1] as usize]);
    plain != inside
}
// History order
fn search_history_order(world: &ApplicationWorld) -> bool {
    let panel = home_run(world, "Search History")
        .expect("expected the panel")
        .center()[1];
    let term_y = |term: &str| {
        world
            .frame
            .runs
            .iter()
            .find(|run| in_home(run) && run.text == term && run.center()[1] > panel)
            .map(|run| run.center()[1])
    };
    matches!((term_y(PARTIAL), term_y(SEARCHED)), (Some(first), Some(second)) if first < second)
}
#[given(regex = r#"^Search History displays "part2" above "Part26"$"#)]
fn history_order_given(world: &mut ApplicationWorld) {
    search_for(world, SEARCHED);
    clear_field(world);
    click_home(world, "Search");
    search_for(world, PARTIAL);
    clear_field(world);
    assert!(
        search_history_order(world),
        "expected {PARTIAL} above {SEARCHED}"
    );
}
#[given("the search field is empty")]
fn field_empty_given(world: &mut ApplicationWorld) {
    assert!(
        home_shows(world, "Search recent"),
        "expected the field's hint"
    );
}
#[when("I clear the search field")]
fn clear_search_field(world: &mut ApplicationWorld) {
    clear_field(world);
}
#[then(regex = r#"^Search History displays "part2" above "Part26"$"#)]
fn history_order_then(world: &mut ApplicationWorld) {
    assert!(
        search_history_order(world),
        "expected {PARTIAL} above {SEARCHED}"
    );
}
// Card menu
#[given("recent documents are displayed in Grid View")]
fn in_grid(world: &mut ApplicationWorld) {
    click_home(world, "Grid View");
}
#[given("no recent-document cards are checked")]
fn no_cards_checked(world: &mut ApplicationWorld) {
    expect_no_selected_actions(world);
}
#[when("I hover over a recent part card")]
fn hover_recent_card(world: &mut ApplicationWorld) {
    let at = row_run(world, SEARCHED).center().map(|value| value as i32);
    world.hover(at);
    world.pump(Vec::new());
}
#[when("I open its actions menu")]
fn open_card_menu(world: &mut ApplicationWorld) {
    let name = row_run(world, SEARCHED);
    let control = world
        .frame
        .runs
        .iter()
        .find(|run| {
            in_home(run)
                && run.text == "..."
                && (run.minimum[0] - name.minimum[0]).abs() < 180.
                && run.center()[1] < name.center()[1]
        })
        .cloned()
        .expect("expected the card's actions control");
    world.click(control.center().map(|value| value as i32));
    world.pump(Vec::new());
}
#[when("I click outside the menu")]
fn click_outside(world: &mut ApplicationWorld) {
    world.click([900, 800]);
    world.pump(Vec::new());
}
#[then("the actions menu closes")]
fn actions_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Remove from Recent");
}
// Open options
#[given("a recent part with a Front design view is displayed in List View")]
fn front_view_part(world: &mut ApplicationWorld) {
    let part = world.parts[0];
    world
        .app()
        .owner
        .document_commands(part)
        .create_design_view("Front")
        .unwrap();
    world.save_part_to_file(part, &format!("{SEARCHED}.opd"));
    world.pump(Vec::new());
    assert!(home_shows(world, SEARCHED), "expected the part on the list");
}
#[when("I choose Open with options")]
fn choose_open_options(world: &mut ApplicationWorld) {
    world.click_text("Open with options");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("a File Open Options dialog is displayed")]
fn options_shown(world: &mut ApplicationWorld) {
    world.expect_text("File Open Options");
}
#[then("Model State shows the primary state")]
fn primary_state(world: &mut ApplicationWorld) {
    world.expect_text("Model State: Primary");
}
#[then("Design View shows Last Active")]
fn last_active(world: &mut ApplicationWorld) {
    world.expect_text("Design View: Last Active");
}
#[when("I open the Design View list")]
fn open_view_list(world: &mut ApplicationWorld) {
    world.click_text("Design View");
    world.pump(Vec::new());
}
#[then("the list includes the primary view and the Front view")]
fn view_list_rows(world: &mut ApplicationWorld) {
    world.expect_text("Primary");
    world.expect_text("Front");
}
#[when("I choose Front")]
fn choose_front(world: &mut ApplicationWorld) {
    world.click_text("Front");
    world.pump(Vec::new());
}
#[then("Design View shows Front")]
fn shows_front(world: &mut ApplicationWorld) {
    world.expect_text("Design View: Front");
}
#[then("the dialog closes and the Recent list is visible")]
fn dialog_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("File Open Options");
    assert!(home_shows(world, SEARCHED), "expected the list");
}
#[given("a recent assembly is displayed in List View")]
fn assembly_listed(world: &mut ApplicationWorld) {
    assert!(home_shows(world, ASSEMBLY), "expected {ASSEMBLY}");
}
#[when("I open the assembly row's actions menu")]
fn assembly_actions(world: &mut ApplicationWorld) {
    open_actions_of(world, ASSEMBLY);
}
#[then("the menu offers Open Express")]
fn offers_express(world: &mut ApplicationWorld) {
    world.expect_text("Open Express");
}
#[then("File Open Options shows Full as the opening mode")]
fn mode_full(world: &mut ApplicationWorld) {
    world.expect_text("Mode: Full");
}
#[then("the dialog shows Model State, Design View and Position View fields")]
fn assembly_fields(world: &mut ApplicationWorld) {
    for field in ["Model State:", "Design View:", "Position View:"] {
        world.expect_text(field);
    }
}
#[then("Skip all unresolved files is unchecked")]
fn skip_unchecked(world: &mut ApplicationWorld) {
    assert!(
        !world.checked("Skip all unresolved files"),
        "expected it clear"
    );
}
fn change_mode(world: &mut ApplicationWorld, mode: &str) {
    world.click_text("Mode: Full");
    world.pump(Vec::new());
    world.click_text(mode);
    world.pump(Vec::new());
}
#[when("I change the opening mode to Express")]
fn mode_express(world: &mut ApplicationWorld) {
    change_mode(world, "Express");
}
#[then("the opening mode shows Express")]
fn shows_express(world: &mut ApplicationWorld) {
    world.expect_text("Mode: Express");
}
#[then("the Design View and Position View fields are disabled")]
fn views_disabled(world: &mut ApplicationWorld) {
    assert!(
        !world.run_is_enabled("Design View:"),
        "expected Design View disabled"
    );
    assert!(
        !world.run_is_enabled("Position View:"),
        "expected Position View disabled"
    );
}
fn save_recent_drawing(world: &mut ApplicationWorld) {
    let drawing = world.add_drawing();
    world.save_part_to_file(drawing, "Sheet.odd");
    world.click_text("Home");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[given("a recent drawing is displayed in List View")]
fn drawing_listed(world: &mut ApplicationWorld) {
    save_recent_drawing(world);
    assert!(
        home_shows(world, "Sheet"),
        "expected the drawing on the list"
    );
}
#[when("I open the drawing row's actions menu")]
fn drawing_actions(world: &mut ApplicationWorld) {
    open_actions_of(world, "Sheet");
}
#[then("the menu offers Defer Open")]
fn offers_defer(world: &mut ApplicationWorld) {
    world.expect_text("Defer Open");
}
#[when("I change the opening mode to Defer")]
fn mode_defer(world: &mut ApplicationWorld) {
    change_mode(world, "Defer");
}
#[then("the opening mode shows Defer")]
fn shows_defer(world: &mut ApplicationWorld) {
    world.expect_text("Mode: Defer");
}
#[then("Skip all unresolved files is disabled")]
fn skip_disabled(world: &mut ApplicationWorld) {
    assert!(
        !world.run_is_enabled("Skip all unresolved files"),
        "expected it disabled"
    );
}
// Blank sizes
const BLANK: &str = "Gone";
#[given("one displayed entry has no size value")]
fn entry_without_size(world: &mut ApplicationWorld) {
    let part = world.add_part();
    let path = world.save_part_to_file(part, &format!("{BLANK}.opd"));
    std::fs::remove_file(&path).unwrap();
    // A later save refreshes the list, which reads each file's facts again.
    let other = world.add_part();
    world.save_part_to_file(other, "Later.opd");
    world.click_text("Home");
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(home_shows(world, BLANK), "expected the entry on the list");
}
#[given("other displayed entries have known file sizes")]
fn others_have_sizes(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "KB"), "expected displayed sizes");
}
#[when("I select the Size column header twice")]
fn size_twice(world: &mut ApplicationWorld) {
    click_home(world, "Size");
    click_home(world, "Size");
}
#[then("the entry with no displayed size appears before the entries with known sizes")]
fn blank_first(world: &mut ApplicationWorld) {
    let blank = row_run(world, BLANK).center()[1];
    let known = row_run(world, SEARCHED).center()[1];
    assert!(
        blank < known,
        "blank at {blank}: expected above the known size at {known}"
    );
}
#[then("the entries with known sizes are ordered from smaller to larger")]
fn known_ascending(world: &mut ApplicationWorld) {
    let mut rows = rows_by_size(world);
    rows.retain(|row| row.1 > 0);
    assert!(
        rows.windows(2).all(|pair| pair[0].1 <= pair[1].1),
        "sizes {rows:?}"
    );
}
// Row hover
#[given("a recent part row is visible")]
fn part_row_visible(world: &mut ApplicationWorld) {
    assert!(home_shows(world, SEARCHED), "expected {SEARCHED}");
}
#[when("I hover over the part row")]
fn hover_row(world: &mut ApplicationWorld) {
    let at = row_run(world, SEARCHED).center().map(|value| value as i32);
    world.observed_row_plain = Some(row_ground(world, SEARCHED));
    world.hover(at);
    world.pump(Vec::new());
}
/// The row's colour well past its text, where only an accent reaches.
fn row_ground(world: &mut ApplicationWorld, name: &str) -> [u8; 4] {
    let run = row_run(world, name);
    world.pixel_at([run.minimum[0] as usize - 20, run.minimum[1] as usize - 1])
}
#[then("the row is visually highlighted")]
fn row_lit(world: &mut ApplicationWorld) {
    let plain = world
        .observed_row_plain
        .expect("expected the row's plain colour");
    assert_ne!(row_ground(world, SEARCHED), plain, "expected the row lit");
}
#[then("the row shows a selection checkbox and an actions control")]
fn row_controls(world: &mut ApplicationWorld) {
    let row = row_run(world, SEARCHED).center()[1];
    let control = world
        .frame
        .runs
        .iter()
        .any(|run| in_home(run) && run.text == "..." && (run.center()[1] - row).abs() < 12.);
    assert!(control, "expected the row's actions control");
}
#[when("I open the row's actions control")]
fn open_row_control(world: &mut ApplicationWorld) {
    open_row_actions(world);
}
#[then("the individual document actions are displayed")]
fn individual_actions(world: &mut ApplicationWorld) {
    world.expect_text("Remove from Recent");
    world.expect_text("Document Properties");
}
// Metadata
#[given("the default columns are displayed")]
fn default_columns(world: &mut ApplicationWorld) {
    location_after_name(world);
}
#[given("the recent list contains a part and a drawing")]
fn part_and_drawing(world: &mut ApplicationWorld) {
    save_recent_drawing(world);
    assert!(
        home_shows(world, SEARCHED) && home_shows(world, "Sheet"),
        "expected both rows"
    );
}
#[then("the list has Name, Location, Date Modified, Last Opened and Size headers")]
fn all_headers(world: &mut ApplicationWorld) {
    for column in ["Name", "Location", "Date Modified", "Last Opened", "Size"] {
        assert!(home_shows(world, column), "expected the {column} header");
    }
}
#[then("the part row shows its name, location, dates and size")]
fn part_row_facts(world: &mut ApplicationWorld) {
    let row = row_run(world, SEARCHED).center()[1];
    let on_row: Vec<String> = world
        .frame
        .runs
        .iter()
        .filter(|run| in_home(run) && (run.center()[1] - row).abs() < 8.)
        .map(|run| run.text.clone())
        .collect();
    let joined = on_row.join(" ");
    for part in ["/tmp", "UTC", "KB"] {
        assert!(
            joined.contains(part),
            "expected {part} on the row {joined:?}"
        );
    }
}
#[then("the part and drawing rows show different document-type symbols")]
fn different_symbols(world: &mut ApplicationWorld) {
    let mark = |name: &str| {
        let row = row_run(world, name);
        world
            .frame
            .runs
            .iter()
            .find(|run| {
                in_home(run)
                    && run.text.len() == 1
                    && run.maximum[0] < row.minimum[0]
                    && (run.center()[1] - row.center()[1]).abs() < 8.
            })
            .map(|run| run.text.clone())
    };
    let (part, drawing) = (mark(SEARCHED), mark("Sheet"));
    assert!(
        part.is_some() && part != drawing,
        "marks {part:?} and {drawing:?}"
    );
}
