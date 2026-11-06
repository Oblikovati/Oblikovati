// SPDX-License-Identifier: GPL-2.0-only
use super::*;

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
#[given("the broader list includes visible recent documents that do not match the search")]
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
            .find(|run| in_home(run) && run.text.ends_with(term) && run.center()[1] > panel)
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
