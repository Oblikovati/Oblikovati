// SPDX-License-Identifier: GPL-2.0-only
use super::*;

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
fn choose_properties(world: &mut ApplicationWorld) {
    world.click_text("Document Properties");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[given("an informational notice is displayed after choosing its document properties action")]
fn notice_after_properties(world: &mut ApplicationWorld) {
    open_row_actions(world);
    choose_properties(world);
    notice_shown(world);
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
fn notice_shown(world: &mut ApplicationWorld) {
    world.expect_text("These properties come from the file on disk.");
}
