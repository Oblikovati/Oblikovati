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
                extended: None,
                kind: "Work Feature",
            })
            .collect(),
        ..Default::default()
    }
}
fn matches_of(tree: &GuiModelBrowser, query: &str) -> Vec<String> {
    let rows = matching_rows(tree, query);
    let matches = rows.into_iter().filter(|row| row.is_match);
    matches.map(|row| row.text).collect()
}
#[test]
fn a_match_keeps_the_branch_above_it_and_nothing_else() {
    let rows = matching_rows(&browser(&["Work Point1", "Work Plane1"]), "plane1");
    let shown: Vec<_> = rows
        .iter()
        .map(|row| (row.depth, row.text.as_str()))
        .collect();
    assert_eq!(shown, [(0, "Features"), (1, "Work Plane1")]);
}
#[test]
fn a_query_keeps_matching_rows_in_any_case() {
    let tree = browser(&["Work Point1", "Work Plane1", "Work Point2"]);
    assert_eq!(
        matches_of(&tree, "work point"),
        ["Work Point1", "Work Point2"]
    );
    assert_eq!(
        matches_of(&tree, "POINT"),
        ["Center Point", "Work Point1", "Work Point2"]
    );
}
#[test]
fn a_query_with_no_match_leaves_no_rows() {
    assert!(matching_rows(&browser(&[]), "zzz").is_empty());
}
#[test]
fn occurrence_search_keeps_nested_ancestors() {
    let occurrence = |name: &str, children| crate::GuiBrowserOccurrence {
        path: vec![],
        name: name.into(),
        component: String::new(),
        suppressed: false,
        virtual_: false,
        deferred: false,
        error: None,
        children,
    };
    let tree = GuiModelBrowser {
        kind: GuiBrowserKind::Assembly,
        occurrences: vec![occurrence("Housing:1", vec![occurrence("Bolt:1", vec![])])],
        ..Default::default()
    };
    let rows = matching_rows(&tree, "bolt");
    assert_eq!(
        rows.iter()
            .map(|row| (row.depth, row.text.as_str(), row.is_match))
            .collect::<Vec<_>>(),
        [(0, "Housing:1", false), (1, "Bolt:1", true)]
    );
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
#[test]
fn a_drawing_search_keeps_each_match_under_its_branch() {
    let drawing = oblikovati_model::drawing::DrawingDocument::new();
    let branches = matching_branches(Some(&drawing), "border");
    assert_eq!(
        branches,
        [
            ("Drawing Resources".into(), vec!["Default Border".into()]),
            ("Sheet:1".into(), vec!["Default Border".into()]),
        ]
    );
}
#[test]
fn a_branch_with_no_match_is_left_out() {
    let drawing = oblikovati_model::drawing::DrawingDocument::new();
    let branches = matching_branches(Some(&drawing), "formats");
    assert_eq!(
        branches,
        [("Drawing Resources".into(), vec!["Sheet Formats".into()])]
    );
    assert!(matching_branches(Some(&drawing), "zzz").is_empty());
}
#[test]
fn drawing_search_finds_a_view_in_its_authored_sheet() {
    use oblikovati_model::drawing::{
        DrawingBaseViewDefinition, DrawingDocument, DrawingSheetRecipe,
    };
    let mut drawing = DrawingDocument::new();
    drawing.set_model_reference("source.opd").unwrap();
    drawing
        .add_sheet(DrawingSheetRecipe::a3_landscape("Details"))
        .unwrap();
    drawing
        .add_base_view_definition(
            1,
            DrawingBaseViewDefinition::new(
                "Front View",
                oblikovati_api::types::BASE_VIEW_FRONT,
                1.,
                [20., 30.],
            )
            .unwrap(),
        )
        .unwrap();
    assert_eq!(
        matching_branches(Some(&drawing), "front"),
        [("Details".into(), vec!["Front View".into()])]
    );
}

#[test]
fn a_matching_parent_retains_all_children_as_unmatched_results() {
    let rows = matching_rows(&browser(&["Extrusion"]), "origin");
    assert_eq!(rows[0].text, "Origin");
    assert!(rows[0].is_match);
    assert_eq!(rows.len(), ORIGIN_ROWS.len() + 1);
    assert!(rows[1..].iter().all(|row| row.depth == 1 && !row.is_match));
    assert!(matching_rows(&browser(&[]), "FEATURES")[0].is_match);
}

#[test]
fn matching_origin_children_keep_their_original_identity() {
    let rows = matching_rows(&browser(&[]), "axis");
    let children: Vec<_> = rows.iter().skip(1).map(|row| row.text.as_str()).collect();
    assert_eq!(children, ["X Axis", "Y Axis", "Z Axis"]);
    for row in rows.iter().skip(1) {
        assert!(row.is_match);
        assert_eq!(row.text, ORIGIN_ROWS[row.origin_index.unwrap()]);
    }
}

#[test]
fn matching_a_body_folder_or_body_keeps_its_direct_children() {
    let mut tree = browser(&["Extrude1"]);
    tree.bodies.push(crate::browser::GuiBrowserBody {
        name: "Solid1".into(),
        feature: oblikovati_model::feature::FeatureId(1),
        key: "body".into(),
        index: 1,
        visible: true,
        faces: 6,
        appearance: String::new(),
        physical: None,
    });
    let folder = matching_rows(&tree, "solid bodies");
    assert_eq!(
        (folder[0].text.as_str(), folder[1].text.as_str()),
        ("Solid Bodies", "Solid1")
    );
    let body = matching_rows(&tree, "solid1");
    assert_eq!(
        body.iter().map(|row| row.text.as_str()).collect::<Vec<_>>(),
        ["Solid Bodies", "Solid1", "Extrude1"]
    );
    assert!(body[1].is_match && !body[2].is_match);
}

#[test]
fn representation_expansion_preference_can_be_changed() {
    let mut view = BrowserView::default();
    assert!(view.auto_expand_representations());
    view.set_auto_expand_representations(false);
    assert!(!view.auto_expand_representations());
    view.set_auto_expand_representations(true);
    assert!(view.auto_expand_representations());
}

#[test]
fn origin_auto_resize_is_independent_per_document_and_row() {
    let mut view = BrowserView::default();
    assert!(view.origin_auto_resize(1, 0));
    view.toggle_origin_auto_resize(1, 0);
    assert!(!view.origin_auto_resize(1, 0));
    assert!(view.origin_auto_resize(1, 1));
    assert!(view.origin_auto_resize(2, 0));
    view.toggle_origin_auto_resize(1, 0);
    assert!(view.origin_auto_resize(1, 0));
}

#[test]
fn origin_auto_resize_overrides_are_removed_when_documents_close() {
    let mut view = BrowserView::default();
    view.toggle_origin_auto_resize(1, 0);
    view.toggle_origin_auto_resize(2, 1);
    let state = GuiCommandState {
        document: Some(2),
        ..Default::default()
    };
    view.prune_origin_sizes(&state);
    assert!(view.origin_auto_resize(1, 0));
    assert!(!view.origin_auto_resize(2, 1));
}
