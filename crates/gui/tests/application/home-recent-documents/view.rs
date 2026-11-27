// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn row(name: &str, opened: u64, size: Option<u64>, pinned: bool) -> GuiRecentDocument {
    GuiRecentDocument {
        path: format!("/work/{name}"),
        name: name.into(),
        location: "/work".into(),
        kind: crate::GuiTemplateKind::Part,
        modified: None,
        modified_text: String::new(),
        modified_age_days: None,
        opened,
        opened_text: String::new(),
        size_bytes: size,
        size_text: String::new(),
        pinned,
        design_views: Vec::new(),
    }
}
fn names(sort: RecentSort, rows: &[GuiRecentDocument]) -> Vec<String> {
    let mut shown: Vec<&GuiRecentDocument> = rows.iter().collect();
    sort.apply(&mut shown);
    shown.iter().map(|row| row.name.clone()).collect()
}
#[test]
fn a_new_field_starts_descending_and_choosing_it_again_reverses() {
    let rows = [
        row("A", 1, Some(9), false),
        row("C", 2, Some(1), false),
        row("B", 3, None, false),
    ];
    let mut sort = RecentSort::default();
    assert_eq!(names(sort, &rows), ["B", "C", "A"]);
    sort.choose(SortField::Name);
    assert_eq!(names(sort, &rows), ["C", "B", "A"]);
    assert_eq!(sort.mark(), "▾");
    sort.choose(SortField::Name);
    assert_eq!(names(sort, &rows), ["A", "B", "C"]);
    assert_eq!(sort.mark(), "▴");
    sort.choose(SortField::Size);
    assert_eq!(names(sort, &rows), ["A", "C", "B"]);
}
#[test]
fn pinned_rows_lead_whatever_the_sort() {
    let rows = [
        row("A", 1, None, false),
        row("B", 2, None, true),
        row("C", 3, None, false),
    ];
    assert_eq!(names(RecentSort::default(), &rows), ["B", "C", "A"]);
}
#[test]
fn a_column_added_back_goes_last_and_can_move_up() {
    let mut columns = RecentColumns::default();
    columns.set_shown(SortField::Location, false);
    assert!(!columns.shown.contains(&SortField::Location));
    columns.set_shown(SortField::Location, true);
    assert_eq!(columns.shown.last(), Some(&SortField::Location));
    columns.move_to(3, 0);
    assert_eq!(columns.shown[0], SortField::Location);
    columns.move_to(0, 99);
    assert_eq!(
        columns.shown[0],
        SortField::Location,
        "an out-of-range move changes nothing"
    );
}
#[test]
fn the_header_reads_none_partial_all_and_selects_from_partial() {
    let rows = [row("A", 1, None, false), row("B", 2, None, false)];
    let shown: Vec<&GuiRecentDocument> = rows.iter().collect();
    let mut checks = RecentChecks::default();
    assert_eq!(checks.header(&shown), HeaderCheck::None);
    checks.set("/work/A", true);
    assert_eq!(checks.header(&shown), HeaderCheck::Partial);
    checks.toggle_header(&shown);
    assert_eq!(checks.header(&shown), HeaderCheck::All);
    checks.toggle_header(&shown);
    assert!(checks.paths.is_empty());
}
#[test]
fn a_hidden_row_loses_its_check() {
    let rows = [row("A", 1, None, false)];
    let shown: Vec<&GuiRecentDocument> = rows.iter().collect();
    let mut checks = RecentChecks::default();
    checks.set("/work/A", true);
    checks.set("/work/B", true);
    checks.keep_only(&shown);
    assert_eq!(checks.paths, ["/work/A"]);
}
