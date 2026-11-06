// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::browser::GuiBrowserFeature;
fn browser(features: &[&str]) -> GuiModelBrowser {
    GuiModelBrowser {
        has_origin: true,
        features: features
            .iter()
            .enumerate()
            .map(|(index, name)| GuiBrowserFeature {
                id: oblikovati_model::feature::FeatureId(index as u64 + 1),
                name: (*name).into(),
                suppressed: false,
            })
            .collect(),
        ..Default::default()
    }
}
#[test]
fn a_query_keeps_matching_rows_in_any_case() {
    let tree = browser(&["Work Point1", "Work Plane1", "Work Point2"]);
    assert_eq!(
        matching_rows(&tree, "work point"),
        ["Work Point1", "Work Point2"]
    );
    assert_eq!(
        matching_rows(&tree, "POINT"),
        ["Center Point", "Work Point1", "Work Point2"]
    );
}
#[test]
fn a_query_with_no_match_leaves_no_rows() {
    assert!(matching_rows(&browser(&[]), "zzz").is_empty());
}
#[test]
fn a_document_without_an_origin_does_not_search_origin_rows() {
    let mut tree = browser(&[]);
    tree.has_origin = false;
    assert!(matching_rows(&tree, "plane").is_empty());
}
#[test]
fn the_match_starts_where_the_query_begins() {
    assert_eq!(match_start("Work Point1", "point"), Some(5));
    assert_eq!(match_start("Work Point1", "axis"), None);
}
#[test]
fn the_close_control_clears_first_and_closes_second() {
    let mut view = BrowserView::default();
    view.open_search();
    assert_eq!(view.query(), None);
    view.query = Some("work".into());
    assert_eq!(view.query(), Some("work"));
    view.close_or_clear();
    assert_eq!(view.query.as_deref(), Some(""));
    view.close_or_clear();
    assert!(view.query.is_none());
}
#[test]
fn an_expansion_applies_to_one_frame_only() {
    let mut view = BrowserView {
        expansion: Some(false),
        ..Default::default()
    };
    assert_eq!(view.take_expansion(), Some(false));
    assert_eq!(view.take_expansion(), None);
}
