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
#[then("the Location header and values disappear from the list")]
fn location_gone(world: &mut ApplicationWorld) {
    let list_row = row_run(world, SEARCHED);
    assert!(!home_shows(world, "/tmp"), "expected no location values");
    assert!(list_row.center()[0] > 0., "expected the row to remain");
}
/// Read known file sizes in screen order, for example before asserting a Size sort.
pub(super) fn rows_by_size(world: &mut ApplicationWorld) -> Vec<(f32, u64)> {
    let mut rows: Vec<(f32, u64)> = world
        .state()
        .recent_documents
        .iter()
        .filter_map(|row| {
            row.size_bytes
                .map(|size| (row_run(world, &row.name).center()[1], size))
        })
        .collect();
    rows.sort_by(|a, b| a.0.total_cmp(&b.0));
    rows
}
#[path = "history_and_checks.rs"]
mod history_and_checks;
#[path = "names_and_cards.rs"]
mod names_and_cards;
#[path = "open_options_and_metadata.rs"]
mod open_options_and_metadata;
#[path = "selection_and_properties.rs"]
mod selection_and_properties;
#[path = "sort_and_rows.rs"]
mod sort_and_rows;
