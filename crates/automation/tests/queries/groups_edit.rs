// SPDX-License-Identifier: GPL-2.0-only

use super::parameters::parameter_workspace;
use super::*;
use oblikovati_automation::{EDIT_METHODS, SessionHost};

fn group_workspace() -> (Workspace, u64) {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(7, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters
        .insert_expression(2, "height", USER_PARAM, "2 cm")
        .unwrap();
    parameter_workspace(parameters)
}

#[test]
fn typed_group_lifecycle_preserves_keys_order_and_independent_membership() {
    let (mut workspace, id) = group_workspace();
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        let group = client
            .parameters()
            .add_group(&ParameterGroupAddArgs {
                internal_name: "dimensions".into(),
                client_id: "addin".into(),
                ..Default::default()
            })
            .unwrap();
        assert_eq!(group.display_name, "dimensions");
        assert_eq!(group.client_id, "addin");
        assert_eq!(group.members, None);
        let group = client
            .parameters()
            .set_group_display_name(&ParameterGroupDisplayNameArgs {
                internal_name: "dimensions".into(),
                display_name: "Dimensions".into(),
            })
            .unwrap();
        assert_eq!(group.internal_name, "dimensions");
        assert_eq!(group.display_name, "Dimensions");
        for parameter in ["height", "width"] {
            client
                .parameters()
                .add_group_member(&ParameterGroupMemberArgs {
                    internal_name: "dimensions".into(),
                    parameter: parameter.into(),
                })
                .unwrap();
        }
        client
            .parameters()
            .add_group(&ParameterGroupAddArgs {
                internal_name: "other".into(),
                ..Default::default()
            })
            .unwrap();
        client
            .parameters()
            .add_group_member(&ParameterGroupMemberArgs {
                internal_name: "other".into(),
                parameter: "width".into(),
            })
            .unwrap();
        let groups = client.parameters().list_groups().unwrap().groups.unwrap();
        assert_eq!(
            groups
                .iter()
                .map(|g| g.internal_name.as_str())
                .collect::<Vec<_>>(),
            ["dimensions", "other"]
        );
        assert_eq!(
            groups[0].members,
            Some(vec!["width".into(), "height".into()])
        );
        let group = client
            .parameters()
            .remove_group_member(&ParameterGroupMemberArgs {
                internal_name: "dimensions".into(),
                parameter: "width".into(),
            })
            .unwrap();
        assert_eq!(group.members, Some(vec!["height".into()]));
        client
            .parameters()
            .delete_group(&ParameterGroupDeleteArgs {
                internal_name: "dimensions".into(),
                delete_parameters: false,
            })
            .unwrap();
        assert_eq!(
            client
                .parameters()
                .list()
                .unwrap()
                .parameters
                .unwrap()
                .len(),
            2
        );
        assert_eq!(
            client.parameters().list_groups().unwrap().groups.unwrap()[0].members,
            Some(vec!["width".into()])
        );
    }
    assert!(workspace.needs_save(id).unwrap());
    assert_eq!(
        workspace.session(id).unwrap().undo_labels().last(),
        Some(&"Delete Parameter Group")
    );
    workspace.commands(id).unwrap().undo().unwrap();
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .group("dimensions")
            .is_some()
    );
}

#[test]
fn membership_is_strict_and_invalid_changes_leave_no_partial_groups() {
    let (mut workspace, id) = group_workspace();
    let host = SessionHost::new(&mut workspace);
    assert!(
        host.call(
            METHOD_PARAMETERS_GROUPS_ADD_MEMBER,
            br#"{"internalName":"missing","parameter":"width"}"#
        )
        .unwrap_err()
        .to_string()
        .contains("missing")
    );
    assert!(host.call(METHOD_PARAMETERS_GROUPS_ADD, b"{}").is_err());
    host.call(METHOD_PARAMETERS_GROUPS_ADD, br#"{"internalName":"dims"}"#)
        .unwrap();
    for (method, request) in [
        (
            METHOD_PARAMETERS_GROUPS_ADD,
            br#"{"internalName":"dims"}"#.as_slice(),
        ),
        (
            METHOD_PARAMETERS_GROUPS_SET_DISPLAY_NAME,
            br#"{"internalName":"dims","displayName":""}"#,
        ),
        (
            METHOD_PARAMETERS_GROUPS_ADD_MEMBER,
            br#"{"internalName":"dims","parameter":"missing"}"#,
        ),
        (
            METHOD_PARAMETERS_GROUPS_REMOVE_MEMBER,
            br#"{"internalName":"missing","parameter":"width"}"#,
        ),
        (
            METHOD_PARAMETERS_GROUPS_DELETE,
            br#"{"internalName":"missing"}"#,
        ),
        (METHOD_PARAMETERS_GROUPS_ADD, b"[]"),
    ] {
        assert!(host.call(method, request).is_err(), "{method}");
    }
    let session = workspace.session(id).unwrap();
    assert_eq!(session.undo_labels(), ["Add Parameter Group"]);
    assert_eq!(session.snapshot().parameters.groups().len(), 1);
    assert!(
        session
            .snapshot()
            .parameters
            .group_members("dims")
            .unwrap()
            .is_empty()
    );
}

#[test]
fn cascade_refuses_surviving_readers_and_successful_deletion_is_undoable() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters
        .insert_expression(2, "area", USER_PARAM, "width * width")
        .unwrap();
    parameters.add_group("dims", "Dimensions", "").unwrap();
    parameters.add_to_group(1, "dims").unwrap();
    let (mut workspace, id) = parameter_workspace(parameters);
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        let request = ParameterGroupDeleteArgs {
            internal_name: "dims".into(),
            delete_parameters: true,
        };
        assert!(client.parameters().delete_group(&request).is_err());
        assert_eq!(
            client
                .parameters()
                .list()
                .unwrap()
                .parameters
                .unwrap()
                .len(),
            2
        );
        client
            .parameters()
            .add_group_member(&ParameterGroupMemberArgs {
                internal_name: "dims".into(),
                parameter: "area".into(),
            })
            .unwrap();
        client.parameters().delete_group(&request).unwrap();
        assert_eq!(client.parameters().list().unwrap().parameters, Some(vec![]));
        assert_eq!(client.parameters().list_groups().unwrap().groups, None);
    }
    assert_eq!(workspace.session(id).unwrap().undo_labels().len(), 2);
    workspace.commands(id).unwrap().undo().unwrap();
    assert_eq!(
        workspace.session(id).unwrap().snapshot().parameters.ids(),
        [1, 2]
    );
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .group_members("dims")
            .unwrap(),
        [1, 2]
    );
    workspace.commands(id).unwrap().redo().unwrap();
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .ids()
            .is_empty()
    );
}

#[test]
fn repeated_memberships_are_no_ops_and_grouped_edits_can_be_aborted() {
    let (mut workspace, id) = group_workspace();
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        client
            .parameters()
            .add_group(&ParameterGroupAddArgs {
                internal_name: "dims".into(),
                ..Default::default()
            })
            .unwrap();
        let request = ParameterGroupMemberArgs {
            internal_name: "dims".into(),
            parameter: "width".into(),
        };
        client.parameters().add_group_member(&request).unwrap();
        client.parameters().add_group_member(&request).unwrap();
        client
            .parameters()
            .remove_group_member(&ParameterGroupMemberArgs {
                internal_name: "dims".into(),
                parameter: "height".into(),
            })
            .unwrap();
    }
    assert_eq!(workspace.session(id).unwrap().undo_labels().len(), 2);
    workspace
        .commands(id)
        .unwrap()
        .begin("group changes")
        .unwrap();
    Client::new(SessionHost::new(&mut workspace))
        .parameters()
        .set_group_display_name(&ParameterGroupDisplayNameArgs {
            internal_name: "dims".into(),
            display_name: "Changed".into(),
        })
        .unwrap();
    workspace.commands(id).unwrap().abort().unwrap();
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .group("dims")
            .unwrap()
            .display_name,
        "dims"
    );
}

#[test]
fn group_edits_obey_veto_and_inventory_contains_no_duplicates() {
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let (mut workspace, id) = super::parameters::parameter_workspace_observer(
        ParameterCollection::new(UnitsOfMeasure::default()),
        observer,
    );
    let error = Client::new(SessionHost::new(&mut workspace))
        .parameters()
        .add_group(&ParameterGroupAddArgs {
            internal_name: "dims".into(),
            ..Default::default()
        })
        .unwrap_err();
    assert!(error.to_string().contains("settings are locked"));
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .groups()
            .is_empty()
    );
    assert!(!workspace.needs_save(id).unwrap());
    assert_eq!(
        EDIT_METHODS
            .iter()
            .collect::<std::collections::BTreeSet<_>>()
            .len(),
        24
    );
}
