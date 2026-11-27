// SPDX-License-Identifier: GPL-2.0-only

use super::parameters::parameter_workspace;
use super::*;
use oblikovati_automation::SessionHost;

fn linked_workspace() -> (Workspace, u64) {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(7, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters
        .insert_expression(2, "area", USER_PARAM, "width * width")
        .unwrap();
    parameters.add_group("dims", "Dimensions", "addin").unwrap();
    parameters.add_to_group(7, "dims").unwrap();
    parameter_workspace(parameters)
}

#[test]
fn rename_preserves_identity_and_rewrites_dependencies_with_undo() {
    let (mut workspace, id) = linked_workspace();
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        let renamed = client
            .parameters()
            .rename(&ParameterRenameArgs {
                name: "width".into(),
                new_name: "span".into(),
            })
            .unwrap();
        assert_eq!(renamed.name, "span");
        assert_eq!(renamed.value, "40 mm");
        assert_eq!(
            client
                .parameters()
                .get(&ParameterNameArgs {
                    name: "area".into()
                })
                .unwrap()
                .expression,
            "span * span"
        );
        assert_eq!(
            client.parameters().list_groups().unwrap().groups.unwrap()[0].members,
            Some(vec!["span".into()])
        );
        assert!(
            client
                .parameters()
                .get(&ParameterNameArgs {
                    name: "width".into()
                })
                .is_err()
        );
    }
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .find("span"),
        Some(7)
    );
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .drivers(2)
            .unwrap(),
        [7]
    );
    workspace.commands(id).unwrap().undo().unwrap();
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .find("width"),
        Some(7)
    );
    assert!(!workspace.needs_save(id).unwrap());
    workspace.commands(id).unwrap().redo().unwrap();
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .find("span"),
        Some(7)
    );
}

#[test]
fn conversion_preserves_identity_and_rejects_nonconvertible_targets() {
    let (mut workspace, id) = linked_workspace();
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        for (input, expected) in [
            (" Model ", "model"),
            ("REFERENCE", "reference"),
            ("user", "user"),
        ] {
            let response = client
                .parameters()
                .convert(&ParameterConvertArgs {
                    name: "width".into(),
                    target_kind: input.into(),
                })
                .unwrap();
            assert_eq!(response.kind, expected);
            assert_eq!(response.expression, "4 cm");
            assert_eq!(response.value, "40 mm");
        }
        for target in ["", "table", "derived", "unknown"] {
            assert!(
                client
                    .parameters()
                    .convert(&ParameterConvertArgs {
                        name: "width".into(),
                        target_kind: target.into()
                    })
                    .is_err()
            );
        }
    }
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .find("width"),
        Some(7)
    );
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .drivers(2)
            .unwrap(),
        [7]
    );
    assert_eq!(workspace.session(id).unwrap().undo_labels().len(), 3);
    workspace.commands(id).unwrap().undo().unwrap();
    let state = &workspace.session(id).unwrap().snapshot().parameters;
    assert_eq!(state.get(7).unwrap().kind, REFERENCE_PARAM);
}

#[test]
fn deletion_refuses_live_readers_and_restores_membership_on_undo() {
    let (mut workspace, id) = linked_workspace();
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        assert!(
            client
                .parameters()
                .delete(&ParameterNameArgs {
                    name: "width".into()
                })
                .is_err()
        );
        client
            .parameters()
            .delete(&ParameterNameArgs {
                name: "area".into(),
            })
            .unwrap();
        client
            .parameters()
            .delete(&ParameterNameArgs {
                name: "width".into(),
            })
            .unwrap();
        assert_eq!(client.parameters().list().unwrap().parameters, Some(vec![]));
        assert_eq!(
            client.parameters().list_groups().unwrap().groups.unwrap()[0].members,
            None
        );
    }
    assert_eq!(workspace.session(id).unwrap().undo_labels().len(), 2);
    workspace.commands(id).unwrap().undo().unwrap();
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .group_members("dims")
            .unwrap(),
        [7]
    );
    workspace.commands(id).unwrap().undo().unwrap();
    assert_eq!(
        workspace.session(id).unwrap().snapshot().parameters.ids(),
        [7, 2]
    );
    assert!(!workspace.needs_save(id).unwrap());
}

#[test]
fn invalid_rename_and_noop_edits_do_not_change_history() {
    let (mut workspace, id) = linked_workspace();
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        for name in ["", "area", "two words", " width "] {
            assert!(
                client
                    .parameters()
                    .rename(&ParameterRenameArgs {
                        name: "width".into(),
                        new_name: name.into()
                    })
                    .is_err()
            );
        }
        client
            .parameters()
            .rename(&ParameterRenameArgs {
                name: "width".into(),
                new_name: "width".into(),
            })
            .unwrap();
        client
            .parameters()
            .convert(&ParameterConvertArgs {
                name: "width".into(),
                target_kind: "user".into(),
            })
            .unwrap();
        assert!(
            client
                .parameters()
                .delete(&ParameterNameArgs {
                    name: "missing".into()
                })
                .is_err()
        );
    }
    assert!(workspace.session(id).unwrap().undo_labels().is_empty());
    assert!(!workspace.needs_save(id).unwrap());
}

#[test]
fn built_in_conversion_is_refused_and_veto_rolls_back_rename() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters.insert_auto_model(1, "3 cm").unwrap();
    let (mut workspace, id) = parameter_workspace(parameters);
    assert!(
        Client::new(SessionHost::new(&mut workspace))
            .parameters()
            .convert(&ParameterConvertArgs {
                name: "d0".into(),
                target_kind: "user".into()
            })
            .is_err()
    );
    assert!(!workspace.needs_save(id).unwrap());
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "3 cm")
        .unwrap();
    let (mut workspace, id) = super::parameters::parameter_workspace_observer(
        parameters,
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    );
    assert!(
        Client::new(SessionHost::new(&mut workspace))
            .parameters()
            .rename(&ParameterRenameArgs {
                name: "width".into(),
                new_name: "span".into()
            })
            .is_err()
    );
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .find("width"),
        Some(1)
    );
    assert!(workspace.session(id).unwrap().undo_labels().is_empty());
}

#[test]
fn rename_and_delete_refresh_exposed_properties_in_the_same_undo_step() {
    use oblikovati_model::{parameter::ParameterAnnotations, property::USER_DEFINED};
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters
        .set_annotations(
            1,
            ParameterAnnotations {
                exposed_as_property: true,
                ..Default::default()
            },
        )
        .unwrap();
    let (mut workspace, id) = parameter_workspace(parameters);
    Client::new(SessionHost::new(&mut workspace))
        .parameters()
        .rename(&ParameterRenameArgs {
            name: "width".into(),
            new_name: "span".into(),
        })
        .unwrap();
    let properties = &workspace.session(id).unwrap().snapshot().properties;
    assert!(properties.get(USER_DEFINED, "width").is_none());
    assert!(properties.get(USER_DEFINED, "span").is_some());
    Client::new(SessionHost::new(&mut workspace))
        .parameters()
        .delete(&ParameterNameArgs {
            name: "span".into(),
        })
        .unwrap();
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .properties
            .get(USER_DEFINED, "span")
            .is_none()
    );
    workspace.commands(id).unwrap().undo().unwrap();
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .properties
            .get(USER_DEFINED, "span")
            .is_some()
    );
    workspace.commands(id).unwrap().undo().unwrap();
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .properties
            .get(USER_DEFINED, "width")
            .is_some()
    );
    assert!(!workspace.needs_save(id).unwrap());
}
