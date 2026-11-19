// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn member_parameter_expressions_follow_activation_and_external_edits() {
    use oblikovati_model::model_state::{
        ModelStateParameterBase, ModelStateParameterTable, ModelStateParameterValue,
    };
    let (mut workspace, id) = member_workspace();
    let second;
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        client
            .parameters()
            .add(&ParameterSetArgs {
                name: "width".into(),
                expression: "1 mm".into(),
                ..Default::default()
            })
            .unwrap();
        second = client
            .model_states()
            .create(&CreateModelStateArgs {
                name: "Second".into(),
                ..Default::default()
            })
            .unwrap()
            .model_state
            .id;
    }
    workspace
        .commands(id)
        .unwrap()
        .run("Install member expressions", |snapshot, _| {
            let parameter = snapshot.parameters.find("width").unwrap();
            let table = ModelStateParameterTable::new(
                vec![ModelStateParameterBase {
                    parameter,
                    expression: "1 mm".into(),
                }],
                vec![ModelStateParameterValue {
                    member: second,
                    parameter,
                    expression: "2 mm".into(),
                }],
            );
            snapshot
                .model_states
                .as_mut()
                .unwrap()
                .set_parameter_table(table)
        })
        .unwrap();
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        let view = client
            .model_states()
            .get_table(&EmptyObject::default())
            .unwrap();
        let evaluations = view.evaluations.unwrap();
        assert_eq!(evaluations.len(), 2);
        assert_eq!(evaluations[0].value, "1 mm");
        assert_eq!(evaluations[1].value, "2 mm");
        client
            .model_states()
            .activate(&RepRef { id: second })
            .unwrap();
        assert_eq!(
            client
                .parameters()
                .get(&ParameterNameArgs {
                    name: "width".into()
                })
                .unwrap()
                .expression,
            "2 mm"
        );
        client
            .parameters()
            .set(&ParameterSetArgs {
                name: "width".into(),
                expression: "3 mm".into(),
                ..Default::default()
            })
            .unwrap();
        client.model_states().activate(&RepRef { id: 1 }).unwrap();
        assert_eq!(
            client
                .parameters()
                .get(&ParameterNameArgs {
                    name: "width".into()
                })
                .unwrap()
                .expression,
            "1 mm"
        );
        client
            .model_states()
            .activate(&RepRef { id: second })
            .unwrap();
        assert_eq!(
            client
                .parameters()
                .get(&ParameterNameArgs {
                    name: "width".into()
                })
                .unwrap()
                .expression,
            "3 mm"
        );
    }
}

#[test]
fn inactive_member_with_unevaluable_expression_is_rejected_atomically() {
    use oblikovati_model::model_state::{
        ModelStateParameterBase, ModelStateParameterTable, ModelStateParameterValue,
    };
    let (mut workspace, id) = member_workspace();
    let second;
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        client
            .parameters()
            .add(&ParameterSetArgs {
                name: "width".into(),
                expression: "1 mm".into(),
                ..Default::default()
            })
            .unwrap();
        second = client
            .model_states()
            .create(&CreateModelStateArgs {
                name: "Second".into(),
                ..Default::default()
            })
            .unwrap()
            .model_state
            .id;
    }
    let before = workspace.session(id).unwrap().generation();
    let result = workspace
        .commands(id)
        .unwrap()
        .run("Invalid member expression", |snapshot, _| {
            let parameter = snapshot.parameters.find("width").unwrap();
            let table = ModelStateParameterTable::new(
                vec![ModelStateParameterBase {
                    parameter,
                    expression: "1 mm".into(),
                }],
                vec![ModelStateParameterValue {
                    member: second,
                    parameter,
                    expression: "missing".into(),
                }],
            );
            snapshot
                .model_states
                .as_mut()
                .unwrap()
                .set_parameter_table(table)
        });
    assert!(result.is_err());
    let session = workspace.session(id).unwrap();
    assert_eq!(session.generation(), before);
    assert!(
        session
            .snapshot()
            .model_states
            .as_ref()
            .unwrap()
            .parameter_table()
            .bases()
            .is_empty()
    );
}

#[test]
fn renaming_a_parameter_rewrites_each_member_expression_atomically() {
    use oblikovati_model::model_state::{
        ModelStateParameterBase, ModelStateParameterTable, ModelStateParameterValue,
    };
    let (mut workspace, id) = member_workspace();
    let second;
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        client
            .parameters()
            .add(&ParameterSetArgs {
                name: "width".into(),
                expression: "1 mm".into(),
                ..Default::default()
            })
            .unwrap();
        client
            .parameters()
            .add(&ParameterSetArgs {
                name: "height".into(),
                expression: "width * 2".into(),
                ..Default::default()
            })
            .unwrap();
        second = client
            .model_states()
            .create(&CreateModelStateArgs {
                name: "Second".into(),
                ..Default::default()
            })
            .unwrap()
            .model_state
            .id;
    }
    workspace
        .commands(id)
        .unwrap()
        .run("Install member expressions", |snapshot, _| {
            let parameter = snapshot.parameters.find("height").unwrap();
            let table = ModelStateParameterTable::new(
                vec![ModelStateParameterBase {
                    parameter,
                    expression: "width * 2".into(),
                }],
                vec![ModelStateParameterValue {
                    member: second,
                    parameter,
                    expression: "width * 3".into(),
                }],
            );
            snapshot
                .model_states
                .as_mut()
                .unwrap()
                .set_parameter_table(table)
        })
        .unwrap();
    let client = Client::new(SessionHost::new(&mut workspace));
    client
        .parameters()
        .rename(&ParameterRenameArgs {
            name: "width".into(),
            new_name: "span".into(),
        })
        .unwrap();
    assert_eq!(
        client
            .parameters()
            .get(&ParameterNameArgs {
                name: "height".into()
            })
            .unwrap()
            .expression,
        "span * 2"
    );
    client
        .model_states()
        .activate(&RepRef { id: second })
        .unwrap();
    assert_eq!(
        client
            .parameters()
            .get(&ParameterNameArgs {
                name: "height".into()
            })
            .unwrap()
            .expression,
        "span * 3"
    );
}
