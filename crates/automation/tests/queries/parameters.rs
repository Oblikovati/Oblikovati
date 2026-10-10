// SPDX-License-Identifier: GPL-2.0-only

use super::*;

pub(super) fn parameter_workspace(parameters: ParameterCollection<u64>) -> (Workspace, u64) {
    parameter_workspace_observer(parameters, QueryObserver::default())
}

pub(super) fn parameter_workspace_observer(
    parameters: ParameterCollection<u64>,
    observer: QueryObserver,
) -> (Workspace, u64) {
    let ids = parameters.ids().to_vec();
    let high = ids.iter().max().copied().unwrap_or(0);
    let mut properties = DocumentProperties::default();
    properties.sync_exposed_parameters(&parameters).unwrap();
    let snapshot = DocumentSnapshot {
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
        parameters,
        properties,
    };
    let history = DocumentEditHistory::new(
        snapshot,
        RecipeIdAllocator::restore(Some(high), &ids).unwrap(),
    )
    .unwrap();
    let mut workspace = Workspace::new(QueryObserver::default());
    let mut guids = QueryGuids(0);
    let id = workspace
        .register(
            history,
            FileIdentity::new("test", &mut guids).unwrap(),
            observer,
            &mut guids,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id;
    (workspace, id)
}

#[test]
fn list_and_get_keep_insertion_order_typed_literals_and_nominal_values() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(8, "width", MODEL_PARAM, "4 cm")
        .unwrap();
    parameters.set_deviation(8, 1.0, -1.0).unwrap();
    parameters
        .set_model_value_type(8, MODEL_VALUE_UPPER)
        .unwrap();
    parameters.insert_text(2, "material", "steel").unwrap();
    parameters.insert_boolean(5, "vented", true).unwrap();
    let (workspace, id) = parameter_workspace(parameters);
    let notices = workspace.observer().notifications;
    let client = Client::new(SessionQueryHost::new(&workspace));
    let rows = client.parameters().list().unwrap().parameters.unwrap();
    assert_eq!(
        rows.iter().map(|row| row.name.as_str()).collect::<Vec<_>>(),
        ["width", "material", "vented"]
    );
    assert_eq!(
        rows.iter()
            .map(|row| row.value.as_str())
            .collect::<Vec<_>>(),
        ["40 mm", "steel", "true"]
    );
    assert_eq!(rows[0].kind, "model");
    assert_eq!(rows[0].expression, "4 cm");
    assert!(rows.iter().all(|row| row.health.is_empty()));
    for row in rows {
        assert_eq!(
            client
                .parameters()
                .get(&ParameterNameArgs {
                    name: row.name.clone()
                })
                .unwrap(),
            row
        );
    }
    assert_eq!(workspace.session(id).unwrap().generation(), 0);
    assert!(!workspace.needs_save(id).unwrap());
    assert_eq!(workspace.observer().notifications, notices);
}

#[test]
fn failed_parameters_report_health_and_retain_last_good_values() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters.set_expression(1, "missing").unwrap();
    parameters
        .insert_expression(2, "dependent", USER_PARAM, "width * 2")
        .unwrap();
    let (workspace, _) = parameter_workspace(parameters);
    let client = Client::new(SessionQueryHost::new(&workspace));
    let rows = client.parameters().list().unwrap().parameters.unwrap();
    assert_eq!(rows[0].value, "40 mm");
    assert_eq!(rows[0].expression, "missing");
    assert!(rows[0].health.contains("missing"));
    assert_eq!(rows[1].value, "0");
    assert!(rows[1].health.contains("healthy"));
}

#[test]
fn neighbor_queries_report_only_direct_links_and_preserve_graph_order() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    for (id, name, expression) in [
        (7, "width", "4 cm"),
        (2, "depth", "2 cm"),
        (9, "area", "depth * width"),
        (3, "volume", "area * width"),
    ] {
        parameters
            .insert_expression(id, name, USER_PARAM, expression)
            .unwrap();
    }
    let (workspace, _) = parameter_workspace(parameters);
    let client = Client::new(SessionQueryHost::new(&workspace));
    let area = ParameterNameArgs {
        name: "area".into(),
    };
    assert_eq!(
        client.parameters().driven_by(&area).unwrap().names,
        Some(vec!["width".into(), "depth".into()])
    );
    assert_eq!(
        client.parameters().dependents(&area).unwrap().names,
        Some(vec!["volume".into()])
    );
    let width = ParameterNameArgs {
        name: "width".into(),
    };
    assert_eq!(
        client.parameters().driven_by(&width).unwrap().names,
        None // The canonical DTO omits an empty neighbor list.
    );
    assert_eq!(
        client.parameters().dependents(&width).unwrap().names,
        Some(vec!["area".into(), "volume".into()])
    );
}

#[test]
fn empty_lists_are_arrays_and_missing_names_or_context_are_errors() {
    let (workspace, _) = parameter_workspace(ParameterCollection::new(UnitsOfMeasure::default()));
    let client = Client::new(SessionQueryHost::new(&workspace));
    assert_eq!(client.parameters().list().unwrap().parameters, Some(vec![]));
    let missing = ParameterNameArgs {
        name: "missing".into(),
    };
    assert!(
        client
            .parameters()
            .get(&missing)
            .unwrap_err()
            .to_string()
            .contains("missing")
    );
    assert!(client.parameters().driven_by(&missing).is_err());
    assert!(client.parameters().dependents(&missing).is_err());
    let empty = Workspace::new(QueryObserver::default());
    let host = SessionQueryHost::new(&empty);
    for method in [
        METHOD_PARAMETERS_LIST,
        METHOD_PARAMETERS_GET,
        METHOD_PARAMETERS_DRIVEN_BY,
        METHOD_PARAMETERS_DEPENDENTS,
    ] {
        assert!(
            host.call(method, b"{}")
                .unwrap_err()
                .to_string()
                .contains("active document")
        );
    }
}
