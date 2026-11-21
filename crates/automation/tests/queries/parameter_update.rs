// SPDX-License-Identifier: GPL-2.0-only

use super::parameters::parameter_workspace;
use super::*;
use oblikovati_automation::SessionHost;
use oblikovati_model::property::USER_DEFINED;

fn update_parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters.set_deviation(1, 0.2, -0.1).unwrap();
    parameters
}

fn update_wire(
    workspace: &mut Workspace,
    request: &str,
) -> Result<ParameterDetail, oblikovati_api::Error> {
    let reply = SessionHost::new(workspace).call(METHOD_PARAMETERS_UPDATE, request.as_bytes())?;
    Ok(serde_json::from_slice(&reply).unwrap())
}

#[test]
fn update_fields_and_exposure_share_one_undo_step() {
    let (mut workspace, id) = parameter_workspace(update_parameters());
    let detail = update_wire(&mut workspace, r#"{"name":"width","comment":"outer width","isKey":true,"visible":false,"precision":2,"displayFormat":"fractional","exposedAsProperty":true,"modelValueType":"upper","disabledActionTypes":["delete","edit","edit"],"customPropertyFormat":{"propertyType":"number","units":"mm","precision":"twoDecimalPlaces","showTrailingZeros":true}}"#).unwrap();
    assert_eq!(detail.comment, "outer width");
    assert!(detail.is_key && detail.exposed_as_property && !detail.visible);
    assert_eq!(detail.precision, 2);
    assert_eq!(detail.display_format, "fractional");
    assert_eq!(detail.model_value, 4.2);
    assert_eq!(
        detail.disabled_action_types,
        Some(vec!["edit".into(), "delete".into()])
    );
    let format = detail.custom_property_format.unwrap();
    assert_eq!(format.property_type, "number");
    assert_eq!(format.precision, "twoDecimalPlaces");
    assert!(format.show_trailing_zeros && !format.show_leading_zeros && !format.show_units_string);
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .properties
            .get(USER_DEFINED, "width")
            .is_some()
    );
    assert_eq!(
        workspace.session(id).unwrap().undo_labels(),
        ["Edit Parameter"]
    );
    workspace.commands(id).unwrap().undo().unwrap();
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .properties
            .get(USER_DEFINED, "width")
            .is_none()
    );
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .annotations
            .comment
            .is_empty()
    );
    workspace.commands(id).unwrap().redo().unwrap();
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .properties
            .get(USER_DEFINED, "width")
            .is_some()
    );
}

#[test]
fn omitted_and_null_fields_retain_state_and_explicit_zero_values_reset_it() {
    let (mut workspace, id) = parameter_workspace(update_parameters());
    update_wire(&mut workspace, r#"{"name":"width","comment":"keep","isKey":true,"precision":4,"disabledActionTypes":["rename"]}"#).unwrap();
    let detail = update_wire(&mut workspace, r#"{"name":"width","comment":null,"isKey":null,"disabledActionTypes":null,"customPropertyFormat":null}"#).unwrap();
    assert_eq!(detail.comment, "keep");
    assert!(detail.is_key);
    assert_eq!(detail.precision, 4);
    assert_eq!(detail.disabled_action_types, Some(vec!["rename".into()]));
    assert_eq!(workspace.session(id).unwrap().undo_labels().len(), 1);
    let detail = update_wire(
        &mut workspace,
        r#"{"name":"width","comment":"","isKey":false,"precision":0,"disabledActionTypes":[]}"#,
    )
    .unwrap();
    assert!(detail.comment.is_empty() && !detail.is_key);
    assert_eq!(detail.precision, 0);
    assert!(detail.disabled_action_types.unwrap_or_default().is_empty());
}

#[test]
fn invalid_late_fields_roll_back_annotations_selection_and_properties() {
    let (mut workspace, id) = parameter_workspace(update_parameters());
    for suffix in [
        r#""displayFormat":"roman""#,
        r#""modelValueType":"midpoint""#,
        r#""disabledActionTypes":["edit","suppress"]"#,
        r#""precision":-1"#,
        r#""customPropertyFormat":{"propertyType":"blob","precision":"twoDecimalPlaces"}"#,
        r#""customPropertyFormat":{"propertyType":"text","precision":"unknown"}"#,
    ] {
        let mut request = serde_json::json!({"name": "width", "comment": "discard", "exposedAsProperty": true, "modelValueType": "upper"});
        let fields: serde_json::Value = serde_json::from_str(&format!("{{{suffix}}}")).unwrap();
        request
            .as_object_mut()
            .unwrap()
            .extend(fields.as_object().unwrap().clone());
        let request = request.to_string();
        assert!(update_wire(&mut workspace, &request).is_err(), "{request}");
    }
    assert!(workspace.session(id).unwrap().undo_labels().is_empty());
    assert!(!workspace.needs_save(id).unwrap());
    let snapshot = workspace.session(id).unwrap().snapshot();
    assert!(
        snapshot
            .parameters
            .get(1)
            .unwrap()
            .annotations
            .comment
            .is_empty()
    );
    assert_eq!(
        snapshot.parameters.get(1).unwrap().model_value_type,
        MODEL_VALUE_NOMINAL
    );
    assert!(snapshot.properties.get(USER_DEFINED, "width").is_none());
}

#[test]
fn enum_spellings_and_custom_precisions_round_trip_through_typed_client() {
    let (mut workspace, _) = parameter_workspace(update_parameters());
    let client = Client::new(SessionHost::new(&mut workspace));
    for display in ["decimal", "fractional", "architectural"] {
        let detail = client
            .parameters()
            .update(&ParameterUpdateArgs {
                name: "width".into(),
                display_format: Some(std::boxed::Box::new(display.into())),
                ..Default::default()
            })
            .unwrap();
        assert_eq!(detail.display_format, display);
    }
    for (selection, expected) in [
        ("upper", 4.2),
        ("lower", 3.9),
        ("median", 4.05),
        ("nominal", 4.0),
    ] {
        let detail = client
            .parameters()
            .update(&ParameterUpdateArgs {
                name: "width".into(),
                model_value_type: Some(std::boxed::Box::new(selection.into())),
                ..Default::default()
            })
            .unwrap();
        assert_eq!(detail.model_value, expected);
    }
    for (precision, _) in oblikovati_model::parameter::vocabulary::PRECISION_NAMES {
        let detail = client
            .parameters()
            .update(&ParameterUpdateArgs {
                name: "width".into(),
                custom_property_format: Some(std::boxed::Box::new(CustomPropertyFormatInfo {
                    property_type: "text".into(),
                    precision: (*precision).into(),
                    ..Default::default()
                })),
                ..Default::default()
            })
            .unwrap();
        assert_eq!(detail.custom_property_format.unwrap().precision, *precision);
    }
}

#[test]
fn update_veto_restores_annotations_and_exposure() {
    let (mut workspace, id) = super::parameters::parameter_workspace_observer(
        update_parameters(),
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    );
    assert!(
        update_wire(
            &mut workspace,
            r#"{"name":"width","comment":"discard","exposedAsProperty":true}"#
        )
        .is_err()
    );
    assert!(!workspace.needs_save(id).unwrap());
    assert!(workspace.session(id).unwrap().undo_labels().is_empty());
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .properties
            .get(USER_DEFINED, "width")
            .is_none()
    );
}

#[test]
fn failed_parameter_exposure_rolls_back_the_whole_update() {
    let mut parameters = update_parameters();
    parameters.set_expression(1, "missing").unwrap();
    let (mut workspace, id) = parameter_workspace(parameters);
    assert!(
        update_wire(
            &mut workspace,
            r#"{"name":"width","comment":"discard","exposedAsProperty":true}"#
        )
        .is_err()
    );
    assert!(update_wire(&mut workspace, r#"{"name":"missing","comment":"discard"}"#).is_err());
    assert!(workspace.session(id).unwrap().undo_labels().is_empty());
    let state = workspace
        .session(id)
        .unwrap()
        .snapshot()
        .parameters
        .get(1)
        .unwrap();
    assert!(state.annotations.comment.is_empty());
    assert!(!state.annotations.exposed_as_property);
    assert!(!workspace.needs_save(id).unwrap());
}
