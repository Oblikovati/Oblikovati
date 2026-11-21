// SPDX-License-Identifier: GPL-2.0-only

use super::*;
fn fixture() -> ModelStateTableView {
    ModelStateTableView {
        rows: Some(vec![ModelStateTableRow {
            ref_: ModelStateTableRowRef {
                member_id: 1,
                ..Default::default()
            },
            name: "Primary".into(),
            source: None,
        }]),
        columns: Some(vec![]),
        suppressions: Some(vec![]),
        parameters: Some(vec![]),
        available_features: Some(vec![ModelStateTableFeatureChoice {
            id: 3,
            name: "Extrusion".into(),
            base_suppressed: false,
        }]),
        available_parameters: Some(vec![ModelStateTableParameterChoice {
            id: 4,
            name: "d0".into(),
            base_expression: "10 mm".into(),
            ..Default::default()
        }]),
        ..Default::default()
    }
}
#[test]
fn parameter_tree_uses_the_active_member_expression() {
    let mut view = fixture();
    view.active_member_id = 1;
    let selection = ExtraSelection {
        parameter_available: Some(4),
        ..Default::default()
    };
    assert!(add(&mut view, &selection, ExtraKind::Parameters));
    assert_eq!(active_expression(&view, 4), Some("10 mm"));
    view.parameters.as_mut().unwrap()[0].cells.as_mut().unwrap()[0].expression =
        Some(Box::new("20 mm".into()));
    assert_eq!(active_expression(&view, 4), Some("20 mm"));
}
#[test]
fn chosen_columns_keep_member_overrides_when_hidden_and_restored() {
    let mut view = fixture();
    let mut selection = ExtraSelection {
        feature_available: Some(3),
        parameter_available: Some(4),
        ..Default::default()
    };
    assert!(add(&mut view, &selection, ExtraKind::Suppression));
    assert!(add(&mut view, &selection, ExtraKind::Parameters));
    view.suppressions.as_mut().unwrap()[0]
        .cells
        .as_mut()
        .unwrap()[0]
        .suppressed = Some(Box::new(true));
    view.parameters.as_mut().unwrap()[0].cells.as_mut().unwrap()[0].expression =
        Some(Box::new("20 mm".into()));
    selection.feature_chosen = Some(3);
    selection.parameter_chosen = Some(4);
    assert!(remove(&mut view, &mut selection, ExtraKind::Suppression));
    assert!(remove(&mut view, &mut selection, ExtraKind::Parameters));
    assert!(view.columns.as_ref().unwrap().is_empty());
    assert!(add(&mut view, &selection, ExtraKind::Suppression));
    assert!(add(&mut view, &selection, ExtraKind::Parameters));
    assert_eq!(
        view.suppressions.as_ref().unwrap()[0]
            .cells
            .as_ref()
            .unwrap()[0]
            .suppressed
            .as_deref(),
        Some(&true)
    );
    assert_eq!(
        view.parameters.as_ref().unwrap()[0].cells.as_ref().unwrap()[0]
            .expression
            .as_deref()
            .map(String::as_str),
        Some("20 mm")
    );
}

#[test]
fn deleting_a_parameter_column_keeps_its_overrides_for_restoration() {
    let mut view = fixture();
    view.active_member_id = 1;
    let mut selection = ExtraSelection {
        parameter_available: Some(4),
        parameter_chosen: Some(4),
        ..Default::default()
    };
    assert!(add(&mut view, &selection, ExtraKind::Parameters));
    view.parameters.as_mut().unwrap()[0].cells.as_mut().unwrap()[0].expression =
        Some(Box::new("20 mm".into()));
    assert!(!remove_parameter(&mut view, &mut selection, 99));
    assert_eq!(selection.parameter_chosen, Some(4));
    assert!(remove_parameter(&mut view, &mut selection, 4));
    assert_eq!(selection.parameter_chosen, None);
    assert!(view.columns.as_ref().unwrap().is_empty());
    assert!(add(&mut view, &selection, ExtraKind::Parameters));
    assert_eq!(active_expression(&view, 4), Some("20 mm"));
}
