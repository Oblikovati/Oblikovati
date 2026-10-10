// SPDX-License-Identifier: GPL-2.0-only

use super::parameters::parameter_workspace;
use super::*;
use oblikovati_automation::SessionHost;

fn update(precision: i64) -> ParameterSettingsUpdateArgs {
    ParameterSettingsUpdateArgs {
        linear_dimension_precision: Some(std::boxed::Box::new(precision)),
        ..Default::default()
    }
}

#[test]
fn typed_settings_edit_records_one_undo_step_and_query_reads_the_result() {
    let (mut workspace, id) =
        parameter_workspace(ParameterCollection::new(UnitsOfMeasure::default()));
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        let response = client.parameters().set_settings(&update(6)).unwrap();
        assert_eq!(response.linear_dimension_precision, 6);
        assert_eq!(response, client.parameters().get_settings().unwrap());
    }
    assert!(workspace.needs_save(id).unwrap());
    assert_eq!(
        workspace.session(id).unwrap().undo_labels(),
        ["Edit Parameter Settings"]
    );
    workspace.commands(id).unwrap().undo().unwrap();
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .settings()
            .linear_dimension_precision,
        3
    );
    assert!(!workspace.needs_save(id).unwrap());
    workspace.commands(id).unwrap().redo().unwrap();
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .settings()
            .linear_dimension_precision,
        6
    );
}

#[test]
fn failed_updates_do_not_partially_apply_fields_or_record_history() {
    let (mut workspace, id) =
        parameter_workspace(ParameterCollection::new(UnitsOfMeasure::default()));
    for request in [
        br#"{"linearStandardTolerance":"0.1 mm","linearDimensionPrecision":9}"#.as_slice(),
        br#"{"useStandardTolerances":true,"dimensionDisplayType":"wrong"}"#,
        br#"{"angularDimensionPrecision":-1}"#,
        b"[]",
        b"{",
        br#"{"linearDimensionPrecision":"bad"}"#,
    ] {
        let host = SessionHost::new(&mut workspace);
        assert!(
            host.call(METHOD_PARAMETERS_SET_SETTINGS, request).is_err(),
            "{}",
            String::from_utf8_lossy(request)
        );
    }
    let session = workspace.session(id).unwrap();
    assert_eq!(
        session.snapshot().parameters.settings(),
        &Default::default()
    );
    assert!(session.undo_labels().is_empty());
    assert!(!workspace.needs_save(id).unwrap());
}

#[test]
fn optional_fields_preserve_values_and_explicit_zero_false_empty_reset_them() {
    let (mut workspace, _) =
        parameter_workspace(ParameterCollection::new(UnitsOfMeasure::default()));
    let host = SessionHost::new(&mut workspace);
    let request = br#"{"linearStandardTolerance":"0.1 mm","angularStandardTolerance":"0.2 deg","useStandardTolerances":true,"exportStandardTolerances":true,"displayParameterAsExpression":true,"linearDimensionPrecision":8,"angularDimensionPrecision":7,"dimensionDisplayType":"name"}"#;
    let before = host.call(METHOD_PARAMETERS_SET_SETTINGS, request).unwrap();
    for request in [
        b"{}".as_slice(),
        b"null",
        br#"{"linearDimensionPrecision":null}"#,
    ] {
        assert_eq!(
            host.call(METHOD_PARAMETERS_SET_SETTINGS, request).unwrap(),
            before
        );
    }
    let result = host.call(METHOD_PARAMETERS_SET_SETTINGS, br#"{"linearStandardTolerance":"","angularStandardTolerance":"","useStandardTolerances":false,"exportStandardTolerances":false,"displayParameterAsExpression":false,"linearDimensionPrecision":0,"angularDimensionPrecision":0,"dimensionDisplayType":"value"}"#).unwrap();
    let settings: ParameterSettingsInfo = serde_json::from_slice(&result).unwrap();
    assert_eq!(
        settings,
        ParameterSettingsInfo {
            dimension_display_type: "value".into(),
            ..Default::default()
        }
    );
}

#[test]
fn no_op_preserves_redo_and_edits_join_an_existing_group() {
    let (mut workspace, id) =
        parameter_workspace(ParameterCollection::new(UnitsOfMeasure::default()));
    Client::new(SessionHost::new(&mut workspace))
        .parameters()
        .set_settings(&update(5))
        .unwrap();
    workspace.commands(id).unwrap().undo().unwrap();
    let before = workspace.session(id).unwrap().observer().notifications;
    Client::new(SessionHost::new(&mut workspace))
        .parameters()
        .set_settings(&ParameterSettingsUpdateArgs::default())
        .unwrap();
    assert_eq!(
        workspace.session(id).unwrap().observer().notifications,
        before
    );
    workspace.commands(id).unwrap().redo().unwrap();
    workspace.commands(id).unwrap().begin("batch").unwrap();
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        client.parameters().set_settings(&update(6)).unwrap();
        assert!(client.parameters().set_settings(&update(99)).is_err());
        assert_eq!(
            client
                .parameters()
                .get_settings()
                .unwrap()
                .linear_dimension_precision,
            6
        );
    }
    workspace.commands(id).unwrap().end().unwrap();
    assert_eq!(
        workspace.session(id).unwrap().undo_labels(),
        ["Edit Parameter Settings", "batch"]
    );
    workspace.commands(id).unwrap().undo().unwrap();
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .settings()
            .linear_dimension_precision,
        5
    );
}

#[test]
fn missing_context_and_unported_edits_are_explicit_errors() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let host = SessionHost::new(&mut workspace);
    assert!(
        host.call(METHOD_PARAMETERS_SET_SETTINGS, b"{}")
            .unwrap_err()
            .to_string()
            .contains("active document")
    );
    assert!(
        host.call(METHOD_PARAMETERS_UPDATE, b"{}")
            .unwrap_err()
            .to_string()
            .contains("supported query")
    );
}

#[test]
fn observer_veto_restores_settings_and_does_not_return_a_success_response() {
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
        .set_settings(&update(6))
        .unwrap_err();
    assert!(error.to_string().contains("settings are locked"));
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .settings()
            .linear_dimension_precision,
        3
    );
    assert!(workspace.session(id).unwrap().undo_labels().is_empty());
    assert!(!workspace.needs_save(id).unwrap());
}
