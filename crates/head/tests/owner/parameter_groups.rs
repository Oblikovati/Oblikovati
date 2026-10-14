// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::ParameterGroupKey;
use oblikovati_gui::{GuiParameterGroupAction, GuiParameterGroupEdit, GuiParameterRowOperation};
fn group_action(owner: &TestOwner, row: usize, key: &str) -> GuiParameterAction {
    let view = owner.gui_parameter_snapshot().unwrap().unwrap();
    GuiParameterAction::Group(GuiParameterGroupEdit {
        target: GuiParameterTextTarget {
            document: view.document,
            parameter: view.rows[row].parameter,
            field: GuiParameterTextField::Name,
        },
        request: 1,
        key: ParameterGroupKey(key.to_owned().into()),
    })
}
#[test]
fn gui_group_changes_create_once_preserve_other_memberships_and_support_undo() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let before = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(!before.rows[0].has_groups);
    owner
        .apply_gui_parameter(&group_action(&owner, 0, "ratios"))
        .unwrap();
    owner
        .apply_gui_parameter(&group_action(&owner, 0, "other"))
        .unwrap();
    owner
        .apply_gui_parameter(&group_action(&owner, 1, "ratios"))
        .unwrap();
    let session = owner.files.workspace().session(document).unwrap();
    assert_eq!(session.snapshot().parameters.groups().len(), 2);
    assert_eq!(
        session
            .snapshot()
            .parameters
            .group_members("ratios")
            .unwrap()
            .len(),
        2
    );
    let grouped = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(grouped.rows[0].has_groups && grouped.rows[1].has_groups);
    let remove = GuiParameterAction::Row {
        document,
        parameter: grouped.rows[0].parameter,
        operation: GuiParameterRowOperation::RemoveFromGroups,
    };
    owner.apply_gui_parameter(&remove).unwrap();
    let removed = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(!removed.rows[0].has_groups);
    assert!(removed.rows[1].has_groups);
    assert_eq!(removed.rows[0].detail, before.rows[0].detail);
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .parameters
            .groups()
            .len(),
        2
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), grouped);
}
#[test]
fn existing_group_identity_display_name_and_client_are_not_replaced_by_gui_assignment() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    owner
        .files
        .commands(document)
        .unwrap()
        .run("Add Group", |snapshot, _| {
            snapshot
                .parameters
                .add_group("ratios", "Friendly name", "addin")
        })
        .unwrap();
    owner
        .apply_gui_parameter(&group_action(&owner, 0, "ratios"))
        .unwrap();
    let group = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .parameters
        .group("ratios")
        .unwrap();
    assert_eq!(group.display_name, "Friendly name");
    assert_eq!(group.client_id, "addin");
    assert_eq!(&*group.key.0, "ratios");
    let view = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert_eq!(view.groups[0].display_name, "Friendly name");
    assert_eq!(view.groups[0].client_id, "addin");
    assert_eq!(view.groups[0].members, vec![view.rows[0].parameter]);
}
#[test]
fn group_management_uses_api_actions_and_preserves_member_parameters_without_cascade() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let parameter_count = owner.gui_parameter_snapshot().unwrap().unwrap().rows.len();
    let create = GuiParameterAction::GroupManagement(GuiParameterGroupAction::Create {
        document,
        internal_name: "ratios".into(),
        display_name: "Gear ratios".into(),
        client_id: String::new(),
    });
    owner.apply_gui_parameter(&create).unwrap();
    let rename = GuiParameterAction::GroupManagement(GuiParameterGroupAction::Rename {
        document,
        internal_name: "ratios".into(),
        display_name: "Ratios".into(),
    });
    owner.apply_gui_parameter(&rename).unwrap();
    let assigned = group_action(&owner, 0, "ratios");
    owner.apply_gui_parameter(&assigned).unwrap();
    let remove = GuiParameterAction::GroupManagement(GuiParameterGroupAction::Delete {
        document,
        internal_name: "ratios".into(),
        delete_parameters: false,
    });
    owner.apply_gui_parameter(&remove).unwrap();
    let view = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(view.groups.is_empty());
    assert_eq!(view.rows.len(), parameter_count);
    assert!(!view.rows[0].has_groups);
}
#[test]
fn group_management_rejects_stale_and_invalid_packets_before_mutation() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let invalid = GuiParameterAction::GroupManagement(GuiParameterGroupAction::Create {
        document,
        internal_name: "bad\0key".into(),
        display_name: "Bad".into(),
        client_id: String::new(),
    });
    assert!(owner.apply_gui_parameter(&invalid).is_err());
    let stale = GuiParameterAction::GroupManagement(GuiParameterGroupAction::Create {
        document,
        internal_name: "ratios".into(),
        display_name: "Ratios".into(),
        client_id: String::new(),
    });
    create(&mut owner, &awake);
    assert!(
        owner
            .apply_gui_parameter(&stale)
            .unwrap_err()
            .contains("current document")
    );
}
#[test]
fn invalid_and_stale_group_packets_do_not_create_groups_or_change_history() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let before = owner.gui_parameter_snapshot().unwrap().unwrap();
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    for key in ["", "bad\0group"] {
        assert!(
            owner
                .apply_gui_parameter(&group_action(&owner, 0, key))
                .is_err()
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
    let stale = group_action(&owner, 0, "ratios");
    create(&mut owner, &awake);
    let active = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(
        owner
            .apply_gui_parameter(&stale)
            .unwrap_err()
            .contains("expected current document")
    );
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), active);
}
