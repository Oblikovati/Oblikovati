// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::{
    json::Bytes,
    types::{MODEL_VALUE_UPPER, USER_PARAM, VariantValue},
};
use oblikovati_model::{parameter::ParameterCollection, property::*, units::UnitsOfMeasure};

#[test]
fn default_sets_are_stable_and_custom_sets_follow_creation_order() {
    let mut properties = DocumentProperties::default();
    let names = properties
        .sets()
        .iter()
        .map(|set| set.name.as_str())
        .collect::<Vec<_>>();
    assert_eq!(
        names,
        [
            SUMMARY_INFORMATION,
            DOCUMENT_SUMMARY,
            DESIGN_TRACKING,
            USER_DEFINED
        ]
    );
    assert!(
        properties
            .set(USER_DEFINED)
            .unwrap()
            .properties()
            .is_empty()
    );
    assert!(properties.set("missing").is_none());
    properties
        .put("second", "B", VariantValue::Boolean(true))
        .unwrap();
    properties
        .put("first", "A", VariantValue::Integer(1))
        .unwrap();
    assert_eq!(properties.sets()[4].name, "second");
    assert_eq!(properties.sets()[5].name, "first");
}

#[test]
fn replacement_retains_position_and_remove_then_insert_moves_to_the_end() {
    let mut properties = DocumentProperties::default();
    properties
        .put(USER_DEFINED, "A", VariantValue::Integer(1))
        .unwrap();
    properties
        .put(USER_DEFINED, "B", VariantValue::Integer(2))
        .unwrap();
    properties
        .put(USER_DEFINED, "A", VariantValue::String("replaced".into()))
        .unwrap();
    assert_eq!(
        properties.set(USER_DEFINED).unwrap().properties()[0].name,
        "A"
    );
    assert_eq!(
        properties.get(USER_DEFINED, "A").unwrap().value,
        VariantValue::String("replaced".into())
    );
    assert!(!properties.remove("missing", "A"));
    assert!(!properties.remove(USER_DEFINED, "missing"));
    assert!(properties.remove(USER_DEFINED, "A"));
    properties
        .put(USER_DEFINED, "A", VariantValue::Boolean(false))
        .unwrap();
    assert_eq!(
        properties.set(USER_DEFINED).unwrap().properties()[1].name,
        "A"
    );
    assert!(properties.remove(USER_DEFINED, "B"));
    assert!(properties.remove(USER_DEFINED, "A"));
    assert!(
        properties
            .set(USER_DEFINED)
            .unwrap()
            .properties()
            .is_empty()
    );
}

#[test]
fn canonical_api_values_and_property_info_keep_types_and_owned_bytes() {
    let mut properties = DocumentProperties::default();
    for (index, value) in [
        VariantValue::Boolean(false),
        VariantValue::Integer(i64::MAX),
        VariantValue::Integer(i64::MIN),
        VariantValue::Double(-0.0),
        VariantValue::String("日本語".into()),
        VariantValue::Bytes(Bytes(Some(vec![0, 255]))),
        VariantValue::Bytes(Bytes(None)),
    ]
    .into_iter()
    .enumerate()
    {
        let name = format!("p{index}");
        properties.put(USER_DEFINED, &name, value.clone()).unwrap();
        let info = properties.info(USER_DEFINED, &name).unwrap();
        assert_eq!(info.value.value, value);
        assert_eq!(info.set, USER_DEFINED);
        assert_eq!(info.name, name);
        assert!(info.value.unit.is_empty() && info.from_parameter.is_empty());
        assert!(serde_json::to_string(&info).is_ok());
    }
    assert!(properties.info(USER_DEFINED, "missing").is_none());
    let snapshot = properties.clone();
    properties
        .put(
            USER_DEFINED,
            "p5",
            VariantValue::Bytes(Bytes(Some(vec![1]))),
        )
        .unwrap();
    assert_eq!(
        snapshot.get(USER_DEFINED, "p5").unwrap().value,
        VariantValue::Bytes(Bytes(Some(vec![0, 255])))
    );
    let VariantValue::Double(value) = snapshot.get(USER_DEFINED, "p3").unwrap().value else {
        panic!("expected a double");
    };
    assert_eq!(value.to_bits(), (-0.0_f64).to_bits());
}

#[test]
fn malformed_values_and_empty_names_do_not_create_or_modify_sets() {
    let mut properties = DocumentProperties::default();
    properties
        .put(USER_DEFINED, "Good", VariantValue::Integer(1))
        .unwrap();
    let before = properties.clone();
    for value in [
        VariantValue::Uninitialized,
        VariantValue::Double(f64::NAN),
        VariantValue::Double(f64::INFINITY),
        VariantValue::Double(f64::NEG_INFINITY),
    ] {
        let error = properties.put("new set", "bad", value).unwrap_err();
        assert!(error.to_string().contains("new set") && error.to_string().contains("expected"));
        assert!(std::error::Error::source(&error).is_none());
        assert_eq!(properties, before);
    }
    assert!(
        properties
            .put("", "name", VariantValue::Boolean(true))
            .is_err()
    );
    assert!(
        properties
            .put(USER_DEFINED, "", VariantValue::Boolean(true))
            .is_err()
    );
    assert_eq!(properties, before);
}

#[test]
fn parameter_exposure_uses_the_selected_value_and_retains_provenance_on_replace() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "25 mm")
        .unwrap();
    parameters.set_deviation(1, 0.1, -0.2).unwrap();
    parameters
        .set_model_value_type(1, MODEL_VALUE_UPPER)
        .unwrap();
    let mut properties = DocumentProperties::default();
    properties.expose_parameter(&parameters, 1).unwrap();
    assert_eq!(
        properties.get(USER_DEFINED, "width").unwrap().value,
        VariantValue::Double(2.6)
    );
    assert_eq!(
        properties
            .info(USER_DEFINED, "width")
            .unwrap()
            .from_parameter,
        "width"
    );
    parameters.set_expression(1, "30 mm").unwrap();
    properties.expose_parameter(&parameters, 1).unwrap();
    assert_eq!(properties.set(USER_DEFINED).unwrap().properties().len(), 1);
    assert_eq!(
        properties.get(USER_DEFINED, "width").unwrap().value,
        VariantValue::Double(3.1)
    );
    properties
        .put(USER_DEFINED, "width", VariantValue::Double(4.0))
        .unwrap();
    assert_eq!(
        properties
            .get(USER_DEFINED, "width")
            .unwrap()
            .from_parameter,
        "width"
    );
}

#[test]
fn unavailable_parameters_do_not_change_exposed_properties() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "2")
        .unwrap();
    parameters.insert_text(2, "label", "steel").unwrap();
    let mut properties = DocumentProperties::default();
    properties.expose_parameter(&parameters, 1).unwrap();
    let before = properties.clone();
    parameters.set_expression(1, "missing").unwrap();
    for id in [1, 99] {
        assert!(properties.expose_parameter(&parameters, id).is_err());
        assert_eq!(properties, before);
    }
}

#[test]
fn empty_byte_payloads_have_the_go_owned_slice_representation() {
    let mut properties = DocumentProperties::default();
    properties
        .put(
            USER_DEFINED,
            "empty",
            VariantValue::Bytes(Bytes(Some(Vec::new()))),
        )
        .unwrap();
    assert_eq!(
        properties.get(USER_DEFINED, "empty").unwrap().value,
        VariantValue::Bytes(Bytes(None))
    );
}

#[test]
fn typed_exposure_and_deleted_parameter_cleanup_preserve_authored_properties() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters.insert_text(1, "label", "steel").unwrap();
    parameters.insert_boolean(2, "approved", true).unwrap();
    for id in [1, 2] {
        let mut annotations = parameters.get(id).unwrap().annotations.clone();
        annotations.exposed_as_property = true;
        parameters.set_annotations(id, annotations).unwrap();
    }
    let mut properties = DocumentProperties::default();
    properties
        .put(USER_DEFINED, "manual", VariantValue::Integer(1))
        .unwrap();
    properties.sync_exposed_parameters(&parameters).unwrap();
    assert_eq!(
        properties.get(USER_DEFINED, "label").unwrap().value,
        VariantValue::String("steel".into())
    );
    assert_eq!(
        properties.get(USER_DEFINED, "approved").unwrap().value,
        VariantValue::Boolean(true)
    );
    parameters.remove(1).unwrap();
    properties.sync_exposed_parameters(&parameters).unwrap();
    assert!(properties.get(USER_DEFINED, "label").is_none());
    assert!(properties.get(USER_DEFINED, "manual").is_some());
}

#[test]
fn unhealthy_projections_are_removed_and_rebuilt_on_recovery() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "2 cm")
        .unwrap();
    let mut annotations = parameters.get(1).unwrap().annotations.clone();
    annotations.exposed_as_property = true;
    parameters.set_annotations(1, annotations).unwrap();
    let mut properties = DocumentProperties::default();
    properties
        .put(USER_DEFINED, "manual", VariantValue::Integer(7))
        .unwrap();
    properties.sync_exposed_parameters(&parameters).unwrap();
    parameters.set_expression(1, "missing").unwrap();
    properties.sync_exposed_parameters(&parameters).unwrap();
    assert!(properties.get(USER_DEFINED, "width").is_none());
    assert!(properties.get(USER_DEFINED, "manual").is_some());
    parameters.set_expression(1, "3 cm").unwrap();
    properties.sync_exposed_parameters(&parameters).unwrap();
    assert_eq!(
        properties.get(USER_DEFINED, "width").unwrap().value,
        VariantValue::Double(3.0)
    );
    assert_eq!(
        properties
            .get(USER_DEFINED, "width")
            .unwrap()
            .from_parameter,
        "width"
    );
}
