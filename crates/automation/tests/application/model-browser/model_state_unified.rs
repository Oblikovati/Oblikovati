// SPDX-License-Identifier: GPL-2.0-only
//! Mixed model-state changes are one generation-checked document transaction.
use super::*;
use oblikovati_api::{
    json::EmptyObject,
    types::{
        MODEL_STATE_TABLE_PARAMETER_COLUMN, MODEL_STATE_TABLE_PROPERTY_COLUMN,
        MODEL_STATE_TABLE_SUPPRESSION_COLUMN, ModelStatePropertyKey, Variant,
    },
};
use oblikovati_automation::SessionHost;
use oblikovati_model::feature::{
    AuthoredFeature,
    work_geometry::{DatumLength, WorkGeometryDefinition},
};
use std::boxed::Box as OwnedBox;

fn mixed_workspace() -> (Workspace, u64, u64, u64, u64) {
    let mut workspace = WorkspaceSession::new(QueryObserver::default());
    let document = register(
        &mut workspace,
        UnitsOfMeasure::default(),
        &mut QueryGuids(0),
    );
    workspace
        .commands(document)
        .unwrap()
        .run("Initialize part", |snapshot, _| {
            snapshot.model_states = Some(Default::default());
            Ok::<_, String>(())
        })
        .unwrap();
    let client = Client::new(SessionHost::new(&mut workspace));
    client
        .parameters()
        .add(&ParameterSetArgs {
            name: "width".into(),
            expression: "1 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let second = client
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Second".into(),
            ..Default::default()
        })
        .unwrap()
        .model_state
        .id;
    let parameter = workspace
        .session(document)
        .unwrap()
        .snapshot()
        .parameters
        .find("width")
        .unwrap();
    let feature = workspace
        .commands(document)
        .unwrap()
        .run("Create point", |snapshot, ids| {
            let feature = ids.allocate_feature().map_err(|cause| cause.to_string())?;
            snapshot
                .work_geometry
                .append(AuthoredFeature::datum(
                    feature,
                    "Point",
                    WorkGeometryDefinition::Point([
                        DatumLength::Parameter(parameter),
                        DatumLength::Constant(0.),
                        DatumLength::Constant(0.),
                    ]),
                ))
                .map_err(|cause| cause.to_string())?;
            Ok::<_, String>(feature.0)
        })
        .unwrap();
    (workspace, document, second, parameter, feature)
}
fn mixed_request(
    view: ModelStateTableView,
    second: u64,
    parameter: u64,
    feature: u64,
) -> SetModelStateTableArgs {
    let primary = ModelStateTableRowRef {
        member_id: 1,
        ..Default::default()
    };
    let other = ModelStateTableRowRef {
        member_id: second,
        ..Default::default()
    };
    let draft = ModelStateTableRowRef {
        draft_key: "copy".into(),
        ..Default::default()
    };
    let key = ModelStatePropertyKey {
        set: "Summary Information".into(),
        name: "Category".into(),
    };
    let property = ModelStateTableProperty {
        property: key.clone(),
        base: None,
        cells: Some(vec![
            ModelStateTablePropertyCell {
                row: primary.clone(),
                value: None,
            },
            ModelStateTablePropertyCell {
                row: other.clone(),
                value: Some(OwnedBox::new(Variant {
                    value: VariantValue::String("Review".into()),
                    unit: String::new(),
                })),
            },
            ModelStateTablePropertyCell {
                row: draft.clone(),
                value: None,
            },
        ]),
    };
    let suppression = ModelStateTableSuppression {
        feature_id: feature,
        base: false,
        cells: Some(vec![
            ModelStateTableSuppressionCell {
                row: primary.clone(),
                suppressed: Some(OwnedBox::new(true)),
            },
            ModelStateTableSuppressionCell {
                row: other.clone(),
                suppressed: None,
            },
            ModelStateTableSuppressionCell {
                row: draft.clone(),
                suppressed: Some(OwnedBox::new(true)),
            },
        ]),
    };
    let parameter_column = ModelStateTableParameter {
        parameter_id: parameter,
        base: "1 cm".into(),
        cells: Some(vec![
            ModelStateTableParameterCell {
                row: primary.clone(),
                expression: None,
            },
            ModelStateTableParameterCell {
                row: other.clone(),
                expression: Some(OwnedBox::new("2 cm".into())),
            },
            ModelStateTableParameterCell {
                row: draft.clone(),
                expression: Some(OwnedBox::new("3 cm".into())),
            },
        ]),
    };
    SetModelStateTableArgs {
        expected_generation: view.generation,
        rows: Some(vec![
            ModelStateTableRow {
                ref_: primary,
                name: "Primary".into(),
                source: None,
            },
            ModelStateTableRow {
                ref_: other.clone(),
                name: "Second".into(),
                source: None,
            },
            ModelStateTableRow {
                ref_: draft,
                name: "Copy".into(),
                source: Some(OwnedBox::new(other)),
            },
        ]),
        columns: Some(vec![
            ModelStateTableColumnRef {
                kind: MODEL_STATE_TABLE_PROPERTY_COLUMN,
                property: Some(OwnedBox::new(key)),
                ..Default::default()
            },
            ModelStateTableColumnRef {
                kind: MODEL_STATE_TABLE_SUPPRESSION_COLUMN,
                feature_id: feature,
                ..Default::default()
            },
            ModelStateTableColumnRef {
                kind: MODEL_STATE_TABLE_PARAMETER_COLUMN,
                parameter_id: parameter,
                ..Default::default()
            },
        ]),
        properties: Some(vec![property]),
        suppressions: Some(vec![suppression]),
        parameters: Some(vec![parameter_column]),
    }
}
#[test]
fn mixed_table_is_one_transaction_and_invalid_draft_changes_nothing() {
    let (mut workspace, document, second, parameter, feature) = mixed_workspace();
    let request;
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        let view = client
            .model_states()
            .get_table(&EmptyObject::default())
            .unwrap();
        request = mixed_request(view, second, parameter, feature);
        let mut invalid = request.clone();
        invalid.parameters.as_mut().unwrap()[0]
            .cells
            .as_mut()
            .unwrap()[1]
            .expression = Some(OwnedBox::new("missing".into()));
        assert!(client.model_states().set_table(&invalid).is_err());
        assert_eq!(
            client
                .model_states()
                .get_table(&EmptyObject::default())
                .unwrap()
                .generation,
            request.expected_generation
        );
        let result = client.model_states().set_table(&request).unwrap();
        assert_eq!(result.created_rows.unwrap().len(), 1);
        assert_eq!(result.table.rows.unwrap().len(), 3);
        assert_eq!(result.table.columns.unwrap().len(), 3);
    }
    let session = workspace.session(document).unwrap();
    assert!(
        session
            .snapshot()
            .feature_suppressed(oblikovati_model::feature::FeatureId(feature))
            .unwrap()
    );
    assert_eq!(
        *session.undo_labels().last().unwrap(),
        "Edit Model State Table"
    );
    Client::new(SessionHost::new(&mut workspace))
        .transactions()
        .undo()
        .unwrap();
    let snapshot = workspace.session(document).unwrap().snapshot();
    assert_eq!(snapshot.model_states.as_ref().unwrap().list().len(), 2);
    assert!(
        !snapshot
            .feature_suppressed(oblikovati_model::feature::FeatureId(feature))
            .unwrap()
    );
    assert!(
        snapshot
            .model_states
            .as_ref()
            .unwrap()
            .table_columns()
            .is_empty()
    );
    Client::new(SessionHost::new(&mut workspace))
        .transactions()
        .redo()
        .unwrap();
    let snapshot = workspace.session(document).unwrap().snapshot();
    assert_eq!(snapshot.model_states.as_ref().unwrap().list().len(), 3);
    assert_eq!(
        snapshot
            .model_states
            .as_ref()
            .unwrap()
            .table_columns()
            .len(),
        3
    );
}

#[test]
fn verify_table_identifies_invalid_member_cell_without_reserving_ids() {
    let (mut workspace, document, second, parameter, feature) = mixed_workspace();
    let before_generation = workspace.session(document).unwrap().generation();
    let before_high_water = workspace.session(document).unwrap().high_water();
    let before_labels = workspace.session(document).unwrap().undo_labels().len();
    let client = Client::new(SessionHost::new(&mut workspace));
    let view = client
        .model_states()
        .get_table(&EmptyObject::default())
        .unwrap();
    let mut request = mixed_request(view, second, parameter, feature);
    request.parameters.as_mut().unwrap()[0]
        .cells
        .as_mut()
        .unwrap()[1]
        .expression = Some(OwnedBox::new("abc".into()));
    let report = client.model_states().verify_table(&request).unwrap();
    assert!(!report.valid);
    let diagnostics = report.diagnostics.unwrap();
    assert!(diagnostics.iter().any(|item| {
        item.parameter_id == parameter
            && item
                .row
                .as_deref()
                .is_some_and(|row| row.member_id == second)
            && item.expression == "abc"
    }));
    assert!(client.model_states().set_table(&request).is_err());
    let session = workspace.session(document).unwrap();
    assert_eq!(session.generation(), before_generation);
    assert_eq!(session.high_water(), before_high_water);
    assert_eq!(session.undo_labels().len(), before_labels);
}

#[test]
fn verify_table_reports_both_ends_of_a_cross_parameter_cycle() {
    let (mut workspace, document, second, parameter, feature) = mixed_workspace();
    let client = Client::new(SessionHost::new(&mut workspace));
    client
        .parameters()
        .add(&ParameterSetArgs {
            name: "height".into(),
            expression: "1 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let height = workspace
        .session(document)
        .unwrap()
        .snapshot()
        .parameters
        .find("height")
        .unwrap();
    let before = workspace.session(document).unwrap().generation();
    let client = Client::new(SessionHost::new(&mut workspace));
    let view = client
        .model_states()
        .get_table(&EmptyObject::default())
        .unwrap();
    let mut request = mixed_request(view, second, parameter, feature);
    let rows = request.rows.as_ref().unwrap();
    request.parameters.as_mut().unwrap()[0]
        .cells
        .as_mut()
        .unwrap()[0]
        .expression = Some(OwnedBox::new("height".into()));
    request
        .parameters
        .as_mut()
        .unwrap()
        .push(ModelStateTableParameter {
            parameter_id: height,
            base: "1 cm".into(),
            cells: Some(
                rows.iter()
                    .enumerate()
                    .map(|(index, row)| ModelStateTableParameterCell {
                        row: row.ref_.clone(),
                        expression: (index == 0).then(|| OwnedBox::new("width".into())),
                    })
                    .collect(),
            ),
        });
    request
        .columns
        .as_mut()
        .unwrap()
        .push(ModelStateTableColumnRef {
            kind: MODEL_STATE_TABLE_PARAMETER_COLUMN,
            parameter_id: height,
            ..Default::default()
        });
    let report = client.model_states().verify_table(&request).unwrap();
    assert!(!report.valid);
    let diagnostics = report.diagnostics.unwrap();
    assert!(diagnostics.iter().any(|item| item.parameter_id == parameter
        && item.row.as_deref().is_some_and(|row| row.member_id == 1)));
    assert!(diagnostics.iter().any(|item| item.parameter_id == height
        && item.row.as_deref().is_some_and(|row| row.member_id == 1)));
    assert!(client.model_states().set_table(&request).is_err());
    assert_eq!(workspace.session(document).unwrap().generation(), before);
}

#[test]
fn typed_jump_crosses_table_edits_and_ordinary_label_collisions() {
    use oblikovati_model::history::HistoryTransitionKind;
    let (mut workspace, document, second, parameter, feature) = mixed_workspace();
    let client = Client::new(SessionHost::new(&mut workspace));
    let before = client
        .transactions()
        .history(&TransactionHistoryArgs { document })
        .unwrap()
        .position;
    let view = client
        .model_states()
        .get_table(&EmptyObject::default())
        .unwrap();
    client
        .model_states()
        .set_table(&mixed_request(view, second, parameter, feature))
        .unwrap();
    workspace
        .commands(document)
        .unwrap()
        .run("Edit Model State Table", |snapshot, _| {
            snapshot.properties.put(
                "Summary Information",
                "Title",
                VariantValue::String("collision".into()),
            )
        })
        .unwrap();
    assert_eq!(
        workspace.session(document).unwrap().undo_transition_kind(),
        Some(HistoryTransitionKind::Ordinary)
    );
    let client = Client::new(SessionHost::new(&mut workspace));
    let moved = client
        .transactions()
        .jump_to(&TransactionJumpToArgs {
            document,
            position: before,
        })
        .unwrap();
    assert_eq!(moved.position, before);
    let table = client
        .model_states()
        .get_table(&EmptyObject::default())
        .unwrap();
    assert_eq!(table.rows.unwrap().len(), 2);
    let moved = client
        .transactions()
        .jump_to(&TransactionJumpToArgs {
            document,
            position: before + 2,
        })
        .unwrap();
    assert_eq!(moved.position, before + 2);
    assert_eq!(
        client
            .model_states()
            .get_table(&EmptyObject::default())
            .unwrap()
            .rows
            .unwrap()
            .len(),
        3
    );
}
