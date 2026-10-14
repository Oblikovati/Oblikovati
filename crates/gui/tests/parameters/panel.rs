// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{GuiFrameEngine, GuiFrameRequest};
use oblikovati_api::wire::{ParameterDetail, ParameterInfo, ToleranceInfo};
fn parameter_view() -> GuiParameterSnapshot {
    let mut numeric = parameter_row(11, "width", "user", "4 cm", "mm");
    numeric.detail.expression_list = Some(Box::new(oblikovati_api::wire::ExpressionListInfo {
        expressions: Some(vec!["4 cm".into(), "8 cm".into()]),
        ..Default::default()
    }));
    numeric.detail.tolerance = Some(Box::new(ToleranceInfo {
        upper: 0.2,
        lower: -0.1,
        ..Default::default()
    }));
    GuiParameterSnapshot {
        linked_tables: Vec::new(),
        link_sources: Vec::new(),
        groups: Vec::new(),
        document: 7,
        rows: vec![
            numeric,
            parameter_row(12, "vented", "user", "false", "boolean"),
            parameter_row(13, "material", "user", "steel", "text"),
            parameter_row(14, "reference", "reference", "2 cm", "mm"),
        ],
    }
}
fn parameter_row(
    parameter: u64,
    name: &str,
    kind: &str,
    expression: &str,
    units: &str,
) -> GuiParameterRow {
    GuiParameterRow {
        tolerance_text: String::new(),
        has_groups: false,
        parameter,
        detail: ParameterDetail {
            parameter_info: ParameterInfo {
                name: name.into(),
                kind: kind.into(),
                expression: expression.into(),
                value: expression.into(),
                health: if kind == "reference" {
                    "Unresolved reference".into()
                } else {
                    String::new()
                },
            },
            units: units.into(),
            comment: "A searchable comment".into(),
            ..Default::default()
        },
    }
}
fn panel_request() -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [1000, 700],
        events: Vec::new(),
        commands: Default::default(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
    }
}
#[test]
fn parameter_table_builds_owned_draws_with_each_value_family_and_model_user_sections() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let closed = engine.frame(&panel_request()).unwrap();
    assert!(
        engine
            .set_parameter_snapshot(Some(parameter_view()))
            .unwrap()
    );
    engine.set_parameters_open(true);
    assert!(engine.parameters_open());
    engine.frame(&panel_request()).unwrap();
    let open = engine.frame(&panel_request()).unwrap();
    assert_ne!(open.draws, closed.draws);
    assert!(open.parameter_actions.is_empty());
    let retained = open.draws.clone();
    engine.set_parameters_open(false);
    engine.frame(&panel_request()).unwrap();
    assert_eq!(open.draws, retained);
    assert!(!engine.parameters_open());
}
#[test]
fn search_matches_name_equation_and_comment_without_changing_section_identity() {
    let view = parameter_view();
    assert!(parameter_matches(&view.rows[0], true, "width"));
    assert!(parameter_matches(&view.rows[0], true, "4 cm"));
    assert!(parameter_matches(&view.rows[0], true, "searchable"));
    assert!(!parameter_matches(&view.rows[0], false, ""));
    assert!(parameter_matches(&view.rows[3], false, "reference"));
    assert!(!parameter_matches(&view.rows[0], true, "missing"));
}
#[test]
fn invalid_parameter_projection_does_not_replace_the_previous_panel() {
    let mut panel = ParameterPanel::new().unwrap();
    let view = parameter_view();
    panel.replace_snapshot(Some(view.clone())).unwrap();
    assert!(!panel.replace_snapshot(Some(view.clone())).unwrap());
    let mut invalid = view.clone();
    invalid.rows[0].detail.comment = "a\0b".into();
    assert!(panel.replace_snapshot(Some(invalid)).is_err());
    assert_eq!(panel.snapshot, Some(view));
    panel.replace_snapshot(None).unwrap();
    assert_eq!(panel.snapshot, None);
}

#[test]
fn toggle_failures_remain_visible_until_an_owner_success() {
    let mut panel = ParameterPanel::new().unwrap();
    let action = GuiParameterAction::Toggle {
        document: 7,
        parameter: 11,
        field: GuiParameterToggleField::Export,
        value: true,
    };
    panel.acknowledge_action(
        &action,
        Err("parameter 11: expected an editable parameter".into()),
    );
    assert_eq!(
        panel.issue.as_deref(),
        Some("parameter 11: expected an editable parameter")
    );
    panel.acknowledge_action(&action, Ok(()));
    assert!(panel.issue.is_none());
}
#[test]
fn value_list_open_actions_stay_local_and_stale_editor_targets_are_named() {
    let mut panel = ParameterPanel::new().unwrap();
    panel.replace_snapshot(Some(parameter_view())).unwrap();
    let edit = GuiParameterAction::Row {
        document: 7,
        parameter: 11,
        operation: GuiParameterRowOperation::EditValueList,
    };
    let toggle = GuiParameterAction::Toggle {
        document: 7,
        parameter: 11,
        field: GuiParameterToggleField::Key,
        value: true,
    };
    let mut actions = vec![edit, toggle.clone()];
    panel.consume_editor_actions(&mut actions);
    assert_eq!(actions, vec![toggle]);
    assert!(panel.issue.is_none());
    let mut stale = vec![GuiParameterAction::Row {
        document: 8,
        parameter: 11,
        operation: GuiParameterRowOperation::EditValueList,
    }];
    panel.consume_editor_actions(&mut stale);
    assert!(stale.is_empty());
    assert!(panel.issue.as_deref().unwrap().contains("8/11"));
}
#[test]
fn tolerance_editor_actions_stay_local_and_refuse_nonnumeric_targets() {
    let mut panel = ParameterPanel::new().unwrap();
    panel.replace_snapshot(Some(parameter_view())).unwrap();
    let mut actions = vec![GuiParameterAction::Row {
        document: 7,
        parameter: 11,
        operation: GuiParameterRowOperation::EditTolerance,
    }];
    panel.consume_editor_actions(&mut actions);
    assert!(actions.is_empty());
    assert!(panel.issue.is_none());
    let mut boolean = vec![GuiParameterAction::Row {
        document: 7,
        parameter: 12,
        operation: GuiParameterRowOperation::EditTolerance,
    }];
    panel.consume_editor_actions(&mut boolean);
    assert!(boolean.is_empty());
    assert!(
        panel
            .issue
            .as_deref()
            .unwrap()
            .contains("expected a numeric parameter")
    );
}
#[test]
fn metadata_failures_retire_old_targets_and_repeat_without_forcing_another_frame() {
    let mut panel = ParameterPanel::new().unwrap();
    let snapshot = parameter_view();
    panel.replace_snapshot(Some(snapshot.clone())).unwrap();
    assert!(
        panel
            .report_metadata_issue("parameter 11: expected display-safe text".into())
            .unwrap()
    );
    assert!(panel.snapshot.is_none());
    assert!(panel.metadata_issue.is_some());
    assert!(
        !panel
            .report_metadata_issue("parameter 11: expected display-safe text".into())
            .unwrap()
    );
    assert!(panel.replace_snapshot(Some(snapshot)).unwrap());
    assert!(panel.metadata_issue.is_none());
}
#[test]
fn group_editor_open_actions_stay_local_for_read_only_rows() {
    let mut panel = ParameterPanel::new().unwrap();
    panel.replace_snapshot(Some(parameter_view())).unwrap();
    let mut actions = vec![GuiParameterAction::Row {
        document: 7,
        parameter: 14,
        operation: GuiParameterRowOperation::AddToGroup,
    }];
    panel.consume_editor_actions(&mut actions);
    assert!(actions.is_empty());
    assert!(panel.issue.is_none());
}
#[test]
fn group_management_actions_require_safe_document_and_group_names() {
    let valid = GuiParameterGroupAction::Create {
        document: 7,
        internal_name: "ratios".into(),
        display_name: "Ratios".into(),
        client_id: String::new(),
    };
    valid.validate().unwrap();
    let invalid = GuiParameterGroupAction::Rename {
        document: 7,
        internal_name: "ratios".into(),
        display_name: String::new(),
    };
    assert!(
        invalid
            .validate()
            .unwrap_err()
            .contains("presentation text")
    );
}
