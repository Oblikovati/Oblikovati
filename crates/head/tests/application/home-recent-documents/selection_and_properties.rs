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
#[when("the user checks the second part's card")]
#[when("I check the second part's card")]
fn check_card(world: &mut ApplicationWorld) {
    let at = card_check_at(world, other_recent_names(world)[0]);
    world.click(at);
    world.pump(Vec::new());
}
#[then("assembly rows are displayed and the selected part is not displayed")]
fn assembly_only(world: &mut ApplicationWorld) {
    assemblies_shown(world);
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
#[given(
    "the disposable copy of `fixtures/cad/home-recent-native/Part23.opd` is displayed in List View"
)]
fn part23_in_list(world: &mut ApplicationWorld) {
    displayed_as_list(world);
    let path = published_profile::published_profile_folder(world)
        .unwrap()
        .join("Part23.opd");
    assert!(recent_name(world, "Part23").is_some());
    assert!(
        path.is_file(),
        "expected the disposable Part23 file at {path:?}"
    );
    world.saved_path = Some(path);
}
#[given("no Recent-file properties action has been requested in this process")]
fn no_properties_requested(world: &mut ApplicationWorld) {
    assert!(world.state().documents.is_empty());
    assert!(window_geometry("Information").is_none());
    world.expect_no_text("Properties");
}
#[given("an informational notice is displayed after choosing its document properties action")]
#[given(
    "the user chooses the part's document properties action and an informational notice is displayed"
)]
fn notice_after_properties(world: &mut ApplicationWorld) {
    let name = properties_name(world);
    open_actions_of(world, &name);
    choose_properties(world);
    notice_shown(world);
}
fn properties_name(world: &ApplicationWorld) -> String {
    world
        .saved_path
        .as_deref()
        .and_then(std::path::Path::file_stem)
        .map_or_else(
            || SEARCHED.to_owned(),
            |stem| stem.to_string_lossy().into_owned(),
        )
}
#[when("the user dismisses the informational notice")]
#[when("I dismiss the informational notice")]
fn dismiss_notice(world: &mut ApplicationWorld) {
    world.click_text("OK");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("a file properties window for the part is displayed")]
fn properties_shown(world: &mut ApplicationWorld) {
    world.expect_text(&format!("{} Properties", properties_name(world)));
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
    if let Some(path) = &world.saved_path {
        let name = path.file_name().unwrap().to_string_lossy();
        let folder = path.parent().unwrap();
        let size = std::fs::metadata(path).unwrap().len();
        for field in [
            format!("File name: {name}"),
            format!("Location: {}", folder.display()),
            format!("Size: {size} bytes"),
        ] {
            world.expect_text(&field);
        }
    }
}
#[when("the user dismisses the properties window")]
#[when("I dismiss the properties window")]
fn dismiss_properties(world: &mut ApplicationWorld) {
    world.click_text("Close");
    world.pump(Vec::new());
}
#[then("the Recent list is visible again")]
fn recent_visible(world: &mut ApplicationWorld) {
    let name = properties_name(world);
    world.expect_no_text(&format!("{name} Properties"));
    assert!(
        recent_name(world, &name).is_some(),
        "expected {name} in Recent"
    );
}
// Mixed selection
#[given("a recent part and a recent assembly are checked in List View")]
fn mixed_checked(world: &mut ApplicationWorld) {
    let assembly = assembly_name(world);
    toggle_row(world, SEARCHED);
    toggle_row(world, assembly);
    assert!(row_checked(world, SEARCHED));
    assert!(row_checked(world, assembly));
}
#[when("the user opens Filter and selects Part")]
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
    let assembly = assembly_name(world);
    assert!(!row_checked(world, assembly), "expected {assembly} clear");
}
// Size sort
#[given("recent documents with different known file sizes are displayed in List View")]
fn different_sizes_are_visible(world: &mut ApplicationWorld) {
    displayed_as_list(world);
    let values = rows_by_size(world);
    assert!(
        values.windows(2).any(|pair| pair[0].1 != pair[1].1),
        "expected two different known sizes, found {values:?}"
    );
}
#[when("the user selects the Size column header")]
#[when("I select the Size column header")]
#[when("the user selects the Size column header again")]
#[when("I select the Size column header again")]
fn select_size_header(world: &mut ApplicationWorld) {
    click_home(world, "Size");
}
#[then("the visible rows with known sizes are ordered from larger to smaller")]
fn larger_first(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "Size ▾"), "expected Size descending");
    let rows = rows_by_size(world);
    assert!(
        rows.windows(2).all(|pair| pair[0].1 >= pair[1].1),
        "sizes {rows:?}"
    );
    world.save_capture("home-size-descending");
}
#[then("the visible rows with known sizes are ordered from smaller to larger")]
fn smaller_first(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "Size ▴"), "expected Size ascending");
    let rows = rows_by_size(world);
    assert!(
        rows.windows(2).all(|pair| pair[0].1 <= pair[1].1),
        "sizes {rows:?}"
    );
    world.save_capture("home-size-ascending");
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
#[when("the user chooses Reset to default")]
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
