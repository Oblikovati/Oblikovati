// SPDX-License-Identifier: GPL-2.0-only
use super::*;

// Sort menu
#[given("recent documents with different names are shown in Grid View")]
fn names_in_grid(world: &mut ApplicationWorld) {
    click_home(world, "Grid View");
}
#[when("the user opens the Sort by menu")]
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
#[when("the user chooses Name")]
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
#[when("the user activates the sort direction control")]
#[when("I activate the sort direction control")]
fn sort_direction(world: &mut ApplicationWorld) {
    click_home(world, "▾");
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
#[when("the user opens the actions menu on a recent part row")]
#[when("I open the actions menu on a recent part row")]
#[when("the user opens a recent row's actions menu")]
#[when("I open a recent row's actions menu")]
#[when("the user opens the part row's actions menu")]
#[when("I open the part row's actions menu")]
fn open_actions(world: &mut ApplicationWorld) {
    open_row_actions(world);
}
#[then("the menu offers Open and opens with options")]
#[then("the menu offers Open and Open with options")]
fn menu_open(world: &mut ApplicationWorld) {
    for label in ["Open", "Openwithoptions"] {
        assert!(
            world
                .frame
                .runs
                .iter()
                .any(|row| row.text == label && world.frame.is_popup_row(row)),
            "expected {label:?} in the actions menu"
        );
    }
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
#[given("the first part's row is checked")]
#[given("one part row is checked and no other rows are checked")]
fn one_checked(world: &mut ApplicationWorld) {
    toggle_row(world, SEARCHED);
    assert!(row_checked(world, SEARCHED), "expected {SEARCHED} checked");
}
#[then("the menu offers Open Selected and Remove Selected")]
fn menu_selected(world: &mut ApplicationWorld) {
    world.expect_text("Open Selected");
    world.expect_text("Remove Selected");
}
#[when("the user checks a second recent-document row")]
#[when("I check a second recent-document row")]
fn check_second(world: &mut ApplicationWorld) {
    let second = other_recent_names(world)[0];
    toggle_row(world, second);
}
#[then("both rows are checked and highlighted")]
#[then("both parts' rows are checked and highlighted")]
fn both_checked(world: &mut ApplicationWorld) {
    let other = other_recent_names(world);
    for name in [SEARCHED, other[0]] {
        assert!(row_checked(world, name), "expected {name} checked");
        assert!(
            world.row_accented_beside(name, other[1]),
            "expected {name} highlighted"
        );
    }
}
#[then("Open Selected and Remove Selected appear above the list")]
#[then("the selected-item actions remain visible")]
#[then("Open Selected and Remove Selected are displayed")]
#[then("Open Selected and Remove Selected remain displayed")]
fn selected_shown(world: &mut ApplicationWorld) {
    expect_selected_actions(world);
}
#[when("the user clears the second row's checkbox")]
#[when("I clear the second row's checkbox")]
fn clear_second(world: &mut ApplicationWorld) {
    let second = other_recent_names(world)[0];
    toggle_row(world, second);
}
#[then("the first row remains checked")]
fn first_checked(world: &mut ApplicationWorld) {
    assert!(row_checked(world, SEARCHED), "expected {SEARCHED} checked");
    let second = other_recent_names(world)[0];
    assert!(!row_checked(world, second), "expected {second} clear");
}
#[when("the user clears the first row's checkbox")]
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
#[when("the user hovers over the part's card")]
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
    expect_pin_state(world, false);
}
#[when("the user activates Pin")]
#[when("I activate Pin")]
fn activate_pin(world: &mut ApplicationWorld) {
    click_home(world, "Pin");
}
#[then("the control offers Unpin")]
fn unpin_offered(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "Unpin"), "expected Unpin");
    expect_pin_state(world, true);
}
fn expect_pin_state(world: &mut ApplicationWorld, pinned: bool) {
    let rows = world.state().recent_documents;
    let row = rows.iter().find(|row| row.name == SEARCHED).unwrap();
    assert_eq!(row.pinned, pinned);
}
#[then("the part remains in the recent grid")]
fn stays_in_grid(world: &mut ApplicationWorld) {
    assert!(home_shows(world, SEARCHED), "expected {SEARCHED}");
}
#[when("the user activates Unpin")]
#[when("I activate Unpin")]
fn activate_unpin(world: &mut ApplicationWorld) {
    click_home(world, "Unpin");
}
#[when("the user opens Column Settings and clears Location")]
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
#[then("Location is unchecked in Available Columns")]
fn location_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Location"), "expected Location clear");
}
#[when("the user selects Location again")]
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
#[when("the user drags Location above Date Modified in Displayed Columns")]
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
