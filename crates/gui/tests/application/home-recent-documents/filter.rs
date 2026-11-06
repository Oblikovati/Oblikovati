// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::GuiTemplateKind::{Assembly, Part};

fn row(name: &str, kind: crate::GuiTemplateKind, age: Option<u64>) -> GuiRecentDocument {
    GuiRecentDocument {
        path: format!("/work/{name}"),
        name: name.into(),
        location: "/work".into(),
        kind,
        modified: None,
        modified_text: String::new(),
        modified_age_days: age,
        opened: 0,
        opened_text: String::new(),
        size_bytes: None,
        size_text: String::new(),
        pinned: false,
        design_views: Vec::new(),
    }
}
fn names(rows: Vec<&GuiRecentDocument>) -> Vec<&str> {
    rows.into_iter().map(|row| row.name.as_str()).collect()
}
fn sample() -> Vec<GuiRecentDocument> {
    vec![
        row("Part26", Part, Some(0)),
        row("Bracket", Part, Some(9)),
        row("Frame", Assembly, Some(0)),
        row("Gone", Part, None),
    ]
}
#[test]
fn a_search_ignores_case_and_keeps_only_matching_names() {
    let rows = sample();
    let filter = RecentFilter::default();
    assert_eq!(names(matching_rows(&rows, "part2", &filter)), ["Part26"]);
    assert_eq!(matching_rows(&rows, "", &filter).len(), 4);
}
#[test]
fn the_badge_counts_a_date_and_each_type_and_clearing_restores_everything() {
    let rows = sample();
    let mut filter = RecentFilter {
        date: DateFilter::Today,
        ..Default::default()
    };
    assert_eq!(filter.active_count(), 1);
    assert_eq!(
        names(matching_rows(&rows, "", &filter)),
        ["Part26", "Frame"]
    );
    filter.set_type(TypeFilter::Part, true);
    assert_eq!(filter.active_count(), 2);
    assert_eq!(names(matching_rows(&rows, "", &filter)), ["Part26"]);
    filter.set_type(TypeFilter::Assembly, true);
    assert_eq!(filter.active_count(), 3);
    assert_eq!(
        names(matching_rows(&rows, "", &filter)),
        ["Part26", "Frame"]
    );
    assert_eq!(matching_rows(&rows, "", &RecentFilter::default()).len(), 4);
}
#[test]
fn a_file_with_no_readable_date_passes_only_when_dates_are_not_limited() {
    assert!(DateFilter::All.admits(None));
    assert!(!DateFilter::LastMonth.admits(None));
    assert!(DateFilter::LastWeek.admits(Some(6)));
    assert!(!DateFilter::LastWeek.admits(Some(7)));
}
