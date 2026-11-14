// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn gui_value_lists_keep_policy_authored_order_selection_and_undo_when_cleared() {
    use oblikovati_api::wire::ExpressionListInfo;
    use oblikovati_gui::{GuiParameterChoiceEdit, GuiParameterRowOperation};
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let before = owner.gui_parameter_snapshot().unwrap().unwrap();
    let target = GuiParameterTextTarget {
        document,
        parameter: before.rows[0].parameter,
        field: GuiParameterTextField::Equation,
    };
    let edit = GuiParameterAction::Choices(GuiParameterChoiceEdit {
        target,
        request: 1,
        choices: ExpressionListInfo {
            expressions: Some(vec!["8 cm".into(), "4 cm".into()]),
            allow_custom_values: false,
            custom_order: true,
        },
    });
    owner.apply_gui_parameter(&edit).unwrap();
    let listed = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert_eq!(
        listed.rows[0]
            .detail
            .expression_list
            .as_deref()
            .unwrap()
            .expressions,
        Some(vec!["8 cm".into(), "4 cm".into()])
    );
    assert!(
        owner
            .apply_gui_parameter(&text_action(
                &owner,
                0,
                GuiParameterTextField::Equation,
                "5 cm"
            ))
            .is_err()
    );
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), listed);
    owner
        .apply_gui_parameter(&text_action(
            &owner,
            0,
            GuiParameterTextField::Equation,
            "8 cm",
        ))
        .unwrap();
    let selected = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert_eq!(selected.rows[0].detail.parameter_info.value, "80 mm");
    let clear = GuiParameterAction::Row {
        document,
        parameter: target.parameter,
        operation: GuiParameterRowOperation::ClearValueList,
    };
    owner.apply_gui_parameter(&clear).unwrap();
    let cleared = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(cleared.rows[0].detail.expression_list.is_none());
    assert_eq!(cleared.rows[0].detail.parameter_info.value, "80 mm");
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), selected);
}
#[test]
fn text_lists_allow_unicode_custom_values_and_sort_only_on_explicit_request() {
    use oblikovati_api::wire::ExpressionListInfo;
    use oblikovati_gui::GuiParameterChoiceEdit;
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let view = owner.gui_parameter_snapshot().unwrap().unwrap();
    let edit = GuiParameterAction::Choices(GuiParameterChoiceEdit {
        target: GuiParameterTextTarget {
            document,
            parameter: view.rows[1].parameter,
            field: GuiParameterTextField::Equation,
        },
        request: 1,
        choices: ExpressionListInfo {
            expressions: Some(vec!["é🙂".into(), "steel".into()]),
            allow_custom_values: true,
            custom_order: false,
        },
    });
    owner.apply_gui_parameter(&edit).unwrap();
    let listed = owner.gui_parameter_snapshot().unwrap().unwrap();
    let choices = listed.rows[1].detail.expression_list.as_deref().unwrap();
    assert_eq!(
        choices.expressions,
        Some(vec!["steel".into(), "é🙂".into()])
    );
    assert!(!choices.custom_order);
    assert!(choices.allow_custom_values);
    owner
        .apply_gui_parameter(&text_action(
            &owner,
            1,
            GuiParameterTextField::Equation,
            "木🙂",
        ))
        .unwrap();
    assert_eq!(
        owner.gui_parameter_snapshot().unwrap().unwrap().rows[1]
            .detail
            .parameter_info
            .value,
        "木🙂"
    );
}
#[test]
fn invalid_value_list_packets_and_local_open_commands_leave_history_unchanged() {
    use oblikovati_api::wire::ExpressionListInfo;
    use oblikovati_gui::{GuiParameterChoiceEdit, GuiParameterRowOperation};
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let before = owner.gui_parameter_snapshot().unwrap().unwrap();
    let id = before.rows[0].parameter;
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let edit = GuiParameterAction::Choices(GuiParameterChoiceEdit {
        target: GuiParameterTextTarget {
            document,
            parameter: id,
            field: GuiParameterTextField::Equation,
        },
        request: 1,
        choices: ExpressionListInfo {
            expressions: Some(vec!["bad\0value".into()]),
            ..Default::default()
        },
    });
    assert!(owner.apply_gui_parameter(&edit).is_err());
    let open = GuiParameterAction::Row {
        document,
        parameter: id,
        operation: GuiParameterRowOperation::EditValueList,
    };
    assert!(
        owner
            .apply_gui_parameter(&open)
            .unwrap_err()
            .contains("expected local editor dispatch")
    );
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), before);
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
}
