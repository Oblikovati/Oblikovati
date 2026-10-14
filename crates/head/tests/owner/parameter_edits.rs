// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_gui::{
    GuiParameterAction, GuiParameterTextEdit, GuiParameterTextField, GuiParameterTextTarget,
    GuiParameterToggleField,
};
fn text_action(
    owner: &TestOwner,
    row: usize,
    field: GuiParameterTextField,
    value: &str,
) -> GuiParameterAction {
    let view = owner.gui_parameter_snapshot().unwrap().unwrap();
    GuiParameterAction::Text(GuiParameterTextEdit {
        target: GuiParameterTextTarget {
            document: view.document,
            parameter: view.rows[row].parameter,
            field,
        },
        request: 1,
        value: value.into(),
    })
}
fn toggle_action(
    owner: &TestOwner,
    row: usize,
    field: GuiParameterToggleField,
    value: bool,
) -> GuiParameterAction {
    let view = owner.gui_parameter_snapshot().unwrap().unwrap();
    GuiParameterAction::Toggle {
        document: view.document,
        parameter: view.rows[row].parameter,
        field,
        value,
    }
}
#[test]
fn parameter_ui_edits_keep_units_literals_identity_and_undo_history() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let initial = owner.gui_parameter_snapshot().unwrap().unwrap();
    let numeric = text_action(&owner, 0, GuiParameterTextField::Equation, "7");
    owner.apply_gui_parameter(&numeric).unwrap();
    assert_eq!(
        owner.gui_parameter_snapshot().unwrap().unwrap().rows[0]
            .detail
            .parameter_info
            .value,
        "7 mm"
    );
    let empty_text = text_action(&owner, 1, GuiParameterTextField::Equation, "");
    owner.apply_gui_parameter(&empty_text).unwrap();
    assert_eq!(
        owner.gui_parameter_snapshot().unwrap().unwrap().rows[1]
            .detail
            .parameter_info
            .value,
        ""
    );
    let boolean = toggle_action(&owner, 2, GuiParameterToggleField::Boolean, true);
    owner.apply_gui_parameter(&boolean).unwrap();
    assert_eq!(
        owner.gui_parameter_snapshot().unwrap().unwrap().rows[2]
            .detail
            .parameter_info
            .value,
        "true"
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(
        owner.gui_parameter_snapshot().unwrap().unwrap().rows[2]
            .detail
            .parameter_info
            .value,
        "false"
    );
    assert_eq!(
        owner.gui_parameter_snapshot().unwrap().unwrap().rows[0].parameter,
        initial.rows[0].parameter
    );
}
#[test]
fn rename_comment_key_and_export_use_the_same_parameter_identity() {
    let (mut owner, awake) = owner();
    create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    for action in [
        text_action(&owner, 0, GuiParameterTextField::Name, "span"),
        text_action(&owner, 0, GuiParameterTextField::Comment, "é🙂"),
        toggle_action(&owner, 0, GuiParameterToggleField::Key, true),
        toggle_action(&owner, 0, GuiParameterToggleField::Export, true),
    ] {
        owner.apply_gui_parameter(&action).unwrap();
    }
    let detail = owner.gui_parameter_snapshot().unwrap().unwrap().rows[0]
        .detail
        .clone();
    assert_eq!(detail.parameter_info.name, "span");
    assert_eq!(detail.comment, "é🙂");
    assert!(detail.is_key && detail.exposed_as_property);
}
#[test]
fn malformed_formula_and_stale_document_actions_leave_history_and_values_unchanged() {
    let (mut owner, awake) = owner();
    let original_document = create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let before = owner.gui_parameter_snapshot().unwrap();
    let generation = owner
        .files
        .workspace()
        .session(original_document)
        .unwrap()
        .generation();
    let invalid = text_action(&owner, 0, GuiParameterTextField::Equation, "(");
    assert!(owner.apply_gui_parameter(&invalid).is_err());
    assert_eq!(owner.gui_parameter_snapshot().unwrap(), before);
    assert_eq!(
        owner
            .files
            .workspace()
            .session(original_document)
            .unwrap()
            .generation(),
        generation
    );
    let stale = text_action(&owner, 0, GuiParameterTextField::Equation, "9 cm");
    create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let current = owner.gui_parameter_snapshot().unwrap();
    assert!(
        owner
            .apply_gui_parameter(&stale)
            .unwrap_err()
            .contains("expected current document")
    );
    assert_eq!(owner.gui_parameter_snapshot().unwrap(), current);
}
#[test]
fn value_lists_reject_unlisted_text_and_deleted_targets_cannot_bind_to_another_row() {
    let (mut owner, awake) = owner();
    create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let client = Client::new(owner.document_host());
    client
        .parameters()
        .set_expression_list(&ParameterExpressionListArgs {
            name: "material".into(),
            expressions: Some(vec!["é🙂".into(), "steel".into()]),
            ..Default::default()
        })
        .unwrap();
    let invalid = text_action(&owner, 1, GuiParameterTextField::Equation, "wood");
    assert!(owner.apply_gui_parameter(&invalid).is_err());
    let valid = text_action(&owner, 1, GuiParameterTextField::Equation, "steel");
    owner.apply_gui_parameter(&valid).unwrap();
    assert_eq!(
        owner.gui_parameter_snapshot().unwrap().unwrap().rows[1]
            .detail
            .parameter_info
            .value,
        "steel"
    );
    Client::new(owner.document_host())
        .parameters()
        .delete(&ParameterNameArgs {
            name: "material".into(),
        })
        .unwrap();
    assert!(
        owner
            .apply_gui_parameter(&valid)
            .unwrap_err()
            .contains("expected a registered parameter")
    );
}
#[test]
fn gui_parameter_creation_uses_session_identity_all_value_families_and_undo() {
    use oblikovati_gui::{GuiParameterCreation, GuiParameterInitialValue};
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    for (name, value, expected) in [
        (
            "width",
            GuiParameterInitialValue::Numeric("4 cm".into()),
            "40 mm",
        ),
        (
            "material",
            GuiParameterInitialValue::Text("é🙂".into()),
            "é🙂",
        ),
        ("vented", GuiParameterInitialValue::Boolean(true), "true"),
    ] {
        let action = GuiParameterAction::Add(GuiParameterCreation {
            document,
            request: 1,
            name: name.into(),
            value,
        });
        owner.apply_gui_parameter(&action).unwrap();
        let view = owner.gui_parameter_snapshot().unwrap().unwrap();
        let row = view.rows.last().unwrap();
        assert_eq!(row.detail.parameter_info.name, name);
        assert_eq!(row.detail.parameter_info.value, expected);
        assert_eq!(row.detail.parameter_info.kind, "user");
    }
    let before = owner.gui_parameter_snapshot().unwrap().unwrap();
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(
        owner.gui_parameter_snapshot().unwrap().unwrap().rows.len(),
        2
    );
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), before);
}
#[test]
fn gui_creation_declines_do_not_change_the_document_or_reuse_reserved_identity() {
    use oblikovati_gui::{GuiParameterCreation, GuiParameterInitialValue};
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let mut creation = GuiParameterCreation {
        document,
        request: 1,
        name: "width".into(),
        value: GuiParameterInitialValue::Numeric("(".into()),
    };
    let before = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(
        owner
            .apply_gui_parameter(&GuiParameterAction::Add(creation.clone()))
            .is_err()
    );
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), before);
    creation.value = GuiParameterInitialValue::Numeric("4 cm".into());
    owner
        .apply_gui_parameter(&GuiParameterAction::Add(creation.clone()))
        .unwrap();
    let created = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(created.rows[0].parameter > 0);
    assert!(
        owner
            .apply_gui_parameter(&GuiParameterAction::Add(creation.clone()))
            .is_err()
    );
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), created);
    create(&mut owner, &awake);
    let active = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(
        owner
            .apply_gui_parameter(&GuiParameterAction::Add(creation))
            .unwrap_err()
            .contains("expected current document")
    );
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), active);
}
#[test]
fn parameter_row_copy_and_delete_preserve_identity_history_and_refuse_live_dependencies() {
    use oblikovati_gui::GuiParameterRowOperation;
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let original = owner.gui_parameter_snapshot().unwrap().unwrap();
    let source = original.rows[0].parameter;
    let row_action = GuiParameterAction::Row {
        document,
        parameter: source,
        operation: GuiParameterRowOperation::CopyToUser,
    };
    owner.apply_gui_parameter(&row_action).unwrap();
    let copied = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert_eq!(
        copied.rows.last().unwrap().detail.parameter_info.name,
        "width_copy"
    );
    assert_ne!(copied.rows.last().unwrap().parameter, source);
    assert_eq!(
        copied.rows.last().unwrap().detail.parameter_info.value,
        original.rows[0].detail.parameter_info.value
    );
    let delete = GuiParameterAction::Row {
        document,
        parameter: copied.rows.last().unwrap().parameter,
        operation: GuiParameterRowOperation::Delete,
    };
    owner.apply_gui_parameter(&delete).unwrap();
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), original);
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), copied);
    native_parameter_dependency(&mut owner);
    let dependent = owner.gui_parameter_snapshot().unwrap().unwrap();
    let delete_source = GuiParameterAction::Row {
        document,
        parameter: source,
        operation: GuiParameterRowOperation::Delete,
    };
    assert!(owner.apply_gui_parameter(&delete_source).is_err());
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), dependent);
}
fn native_parameter_dependency(owner: &mut TestOwner) {
    Client::new(owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "double_width".into(),
            expression: "2 * width".into(),
            ..Default::default()
        })
        .unwrap();
}
#[path = "parameter_choices.rs"]
mod choices;
#[path = "parameter_derived.rs"]
mod derived;
#[path = "parameter_groups.rs"]
mod groups;
#[path = "parameter_tolerance.rs"]
mod tolerance;
