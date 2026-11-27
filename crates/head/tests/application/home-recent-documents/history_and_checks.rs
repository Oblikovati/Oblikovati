// SPDX-License-Identifier: GPL-2.0-only
use super::*;

// History recall
#[given(
    "the user searches for \"Part26\", clears the field and activates Search to restore the broader list"
)]
fn search_and_restore(world: &mut ApplicationWorld) {
    search_for(world, SEARCHED);
    clear_field(world);
    click_home(world, "Search");
    list_contains(world);
}
#[given(regex = r#"^the recent search history contains "Part26"$"#)]
fn history_contains(world: &mut ApplicationWorld) {
    if published_profile::uses_published_profile(world) {
        focus_search(world);
        history_includes(world);
    } else {
        search_for(world, SEARCHED);
    }
}
#[given("the search field is empty and the broader recent list is displayed")]
fn empty_field_broad(world: &mut ApplicationWorld) {
    if !published_profile::uses_published_profile(world) {
        clear_field(world);
        click_home(world, "Search");
    }
    field_empty(world);
    list_contains(world);
}
#[when("the user focuses Search recent")]
#[when("I focus Search recent")]
fn focus_search(world: &mut ApplicationWorld) {
    click_home(world, "Search recent");
}
#[when("the user chooses \"Part26\" from Search History")]
#[when(regex = r#"^I choose "Part26" from Search History$"#)]
fn choose_term(world: &mut ApplicationWorld) {
    let panel = home_run(world, "Search History")
        .expect("expected the panel")
        .center()[1];
    let term = world
        .frame
        .runs
        .iter()
        .find(|run| in_home(run) && run.text.ends_with(SEARCHED) && run.center()[1] > panel)
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
    let search = home_run(world, "Search").expect("expected the Search button");
    assert!(
        world.frame.runs.iter().any(|run| {
            in_home(run)
                && run.text == SEARCHED
                && run.maximum[0] < search.minimum[0]
                && (run.center()[1] - search.center()[1]).abs() < ROW_REACH
        }),
        "expected {SEARCHED} in the search field"
    );
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
#[when("the user checks the list header checkbox")]
#[when("I check the list header checkbox")]
#[when("the user activates the partially selected header checkbox")]
#[when("I activate the partially selected header checkbox")]
#[when("the user clears the header checkbox")]
#[when("I clear the header checkbox")]
fn header_box(world: &mut ApplicationWorld) {
    toggle_header(world);
}
#[then("all visible recent-document rows are checked and highlighted")]
#[then("all visible recent-document rows are checked again")]
fn all_checked(world: &mut ApplicationWorld) {
    for name in std::iter::once(SEARCHED).chain(other_recent_names(world).iter().copied()) {
        assert!(row_checked(world, name), "expected {name} checked");
    }
}
#[when("the user clears one visible row's checkbox")]
#[when("I clear one visible row's checkbox")]
fn clear_one(world: &mut ApplicationWorld) {
    toggle_row(world, SEARCHED);
}
#[then("that row is unchecked while the other visible rows remain checked")]
fn others_checked(world: &mut ApplicationWorld) {
    assert!(!row_checked(world, SEARCHED), "expected {SEARCHED} clear");
    for name in other_recent_names(world) {
        assert!(row_checked(world, name), "expected {name} checked");
    }
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
    for name in std::iter::once(SEARCHED).chain(other_recent_names(world).iter().copied()) {
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
#[when("the user selects the Name column header")]
#[when("I select the Name column header")]
#[when("the user selects the Name column header again")]
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
#[when("the user enters \"zzq_no_match_7319\" in Search recent")]
#[when(regex = r#"^I enter "zzq_no_match_7319" in Search recent$"#)]
fn enter_no_match(world: &mut ApplicationWorld) {
    enter_search(world, NO_MATCH);
}
#[then("the recent-document rows are replaced by a No results message")]
#[then("No results is displayed")]
fn no_results(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "No results"), "expected No results");
    assert!(
        !matched_part_is_visible(world),
        "expected no matching Part row"
    );
    others_hidden(world);
}
#[then("the page offers Clear search and filters")]
fn offers_clear(world: &mut ApplicationWorld) {
    assert!(
        home_shows(world, "Clear search and filters"),
        "expected the button"
    );
}
#[when("the user activates Clear search and filters")]
#[when("I activate Clear search and filters")]
fn activate_clear(world: &mut ApplicationWorld) {
    click_home(world, "Clear search and filters");
}
#[given(regex = r#"^no assembly matches "Part26"$"#)]
fn no_assembly_match(world: &mut ApplicationWorld) {
    assert!(
        world.state().recent_documents.into_iter().all(|row| {
            row.kind != oblikovati_gui::GuiTemplateKind::Assembly || !row.name.contains(SEARCHED)
        }),
        "expected the fixture's names apart"
    );
}
#[when("the user searches recent documents for \"Part26\"")]
#[when(regex = r#"^I search recent documents for "Part26"$"#)]
fn search_part26(world: &mut ApplicationWorld) {
    search_for(world, SEARCHED);
}
#[when("the user opens Filter")]
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
    filtered_remains(world);
}
#[when("the user activates the clear control beside the active filter indicator")]
#[when("I activate the clear control beside the active filter indicator")]
#[when("the user clears the active filter")]
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
