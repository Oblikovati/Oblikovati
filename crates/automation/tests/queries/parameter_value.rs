// SPDX-License-Identifier: GPL-2.0-only

use super::parameters::parameter_workspace;
use super::*;
use oblikovati_automation::SessionHost;

fn numeric(name: &str, expression: &str) -> ParameterSetArgs {
    ParameterSetArgs {
        name: name.into(),
        expression: expression.into(),
        ..Default::default()
    }
}

#[test]
fn creation_allocates_ids_and_undo_does_not_reuse_them() {
    let (mut workspace, id) =
        parameter_workspace(ParameterCollection::new(UnitsOfMeasure::default()));
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        let value = client.parameters().add(&numeric("width", "4 cm")).unwrap();
        assert_eq!(
            (
                value.name.as_str(),
                value.kind.as_str(),
                value.value.as_str()
            ),
            ("width", "user", "40 mm")
        );
        client
            .parameters()
            .add(&numeric("area", "width * width"))
            .unwrap();
    }
    let original = workspace
        .session(id)
        .unwrap()
        .snapshot()
        .parameters
        .find("area")
        .unwrap();
    workspace.commands(id).unwrap().undo().unwrap();
    Client::new(SessionHost::new(&mut workspace))
        .parameters()
        .add(&numeric("height", "2 cm"))
        .unwrap();
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .find("height")
            .unwrap()
            > original
    );
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .find("area")
            .is_none()
    );
}

#[test]
fn creation_supports_text_boolean_and_user_created_model_parameters() {
    let (mut workspace, _) =
        parameter_workspace(ParameterCollection::new(UnitsOfMeasure::default()));
    let client = Client::new(SessionHost::new(&mut workspace));
    let text = client
        .parameters()
        .add(&ParameterSetArgs {
            value_type: " Text ".into(),
            kind: "model".into(),
            ..numeric("material", "")
        })
        .unwrap();
    assert_eq!(text.kind, "user");
    assert_eq!(text.value, "");
    assert_eq!(text.expression, "\"\"");
    for (index, input) in [
        "true", "True", "TRUE", "t", "T", "1", "false", "False", "FALSE", "f", "F", "0",
    ]
    .iter()
    .enumerate()
    {
        let result = client
            .parameters()
            .add(&ParameterSetArgs {
                value_type: "bool".into(),
                ..numeric(&format!("flag{index}"), input)
            })
            .unwrap();
        assert_eq!(result.value, if index < 6 { "true" } else { "false" });
    }
    client
        .parameters()
        .add(&ParameterSetArgs {
            kind: " Model ".into(),
            ..numeric("depth", "2 cm")
        })
        .unwrap();
    let detail = client
        .parameters()
        .get_detail(&ParameterNameArgs {
            name: "depth".into(),
        })
        .unwrap();
    assert_eq!(detail.parameter_info.kind, "model");
    assert!(!detail.built_in);
    assert!(detail.in_use); // Go InUse includes model parameters even without parameter readers.
}

#[test]
fn value_edits_qualify_bare_numbers_and_recompute_dependents_with_undo() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters
        .insert_expression(2, "area", USER_PARAM, "width * width")
        .unwrap();
    let (mut workspace, id) = parameter_workspace(parameters);
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        let result = client.parameters().set(&numeric("width", "7")).unwrap();
        assert_eq!(result.expression, "7 mm");
        assert_eq!(result.value, "7 mm");
        let area = client
            .units()
            .get_database_units_from_expression(&ExpressionArgs {
                expression: "area".into(),
            })
            .unwrap();
        assert!((area.value - 0.49).abs() < 1e-14);
        let repaired = client
            .parameters()
            .set(&numeric("width", "missing"))
            .unwrap();
        assert!(!repaired.health.is_empty());
        assert!(
            !client
                .parameters()
                .get(&ParameterNameArgs {
                    name: "area".into()
                })
                .unwrap()
                .health
                .is_empty()
        );
        assert!(
            client
                .parameters()
                .set(&numeric("width", "8"))
                .unwrap()
                .health
                .is_empty()
        );
    }
    workspace.commands(id).unwrap().undo().unwrap();
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .is_none()
    );
    workspace.commands(id).unwrap().undo().unwrap();
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .source,
        "7 mm"
    );
}

#[test]
fn invalid_creation_and_value_edits_preserve_document_state() {
    let (mut workspace, id) =
        parameter_workspace(ParameterCollection::new(UnitsOfMeasure::default()));
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        for request in [
            numeric("", "1"),
            numeric("width", ""),
            numeric("width", "1 +"),
            ParameterSetArgs {
                value_type: "unknown".into(),
                ..numeric("width", "1")
            },
            ParameterSetArgs {
                value_type: "bool".into(),
                ..numeric("width", "yes")
            },
        ] {
            assert!(client.parameters().add(&request).is_err());
        }
        client.parameters().add(&numeric("width", "4 cm")).unwrap();
        assert!(client.parameters().add(&numeric("width", "2 cm")).is_err());
        for request in [
            numeric("width", ""),
            numeric("width", "1 +"),
            numeric("width", "width * 2"),
            numeric("missing", "2"),
        ] {
            assert!(client.parameters().set(&request).is_err());
        }
        client
            .parameters()
            .convert(&ParameterConvertArgs {
                name: "width".into(),
                target_kind: "reference".into(),
            })
            .unwrap();
        assert!(client.parameters().set(&numeric("width", "3 cm")).is_err());
    }
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .ids()
            .len(),
        1
    );
    assert_eq!(workspace.session(id).unwrap().undo_labels().len(), 2);
}

#[test]
fn creation_and_updates_join_groups_and_roll_back_on_veto() {
    let (mut workspace, id) =
        parameter_workspace(ParameterCollection::new(UnitsOfMeasure::default()));
    workspace.commands(id).unwrap().begin("dimensions").unwrap();
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        client.parameters().add(&numeric("width", "4 cm")).unwrap();
        client.parameters().set(&numeric("width", "5 cm")).unwrap();
    }
    workspace.commands(id).unwrap().abort().unwrap();
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .ids()
            .is_empty()
    );
    let (mut workspace, id) = super::parameters::parameter_workspace_observer(
        ParameterCollection::new(UnitsOfMeasure::default()),
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    );
    assert!(
        Client::new(SessionHost::new(&mut workspace))
            .parameters()
            .add(&numeric("width", "4 cm"))
            .is_err()
    );
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .ids()
            .is_empty()
    );
    // The rejected edit still reserves an ID that storage must retain.
    assert!(workspace.needs_save(id).unwrap());
}
