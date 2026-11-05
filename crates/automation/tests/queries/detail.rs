// SPDX-License-Identifier: GPL-2.0-only

use super::parameters::parameter_workspace;
use super::*;
use oblikovati_model::parameter::{CustomPropertyFormat, ParameterAnnotations};

#[test]
fn detail_defaults_include_numeric_tolerance_and_property_format() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    let (workspace, id) = parameter_workspace(parameters);
    let client = Client::new(SessionQueryHost::new(&workspace));
    let detail = client
        .parameters()
        .get_detail(&ParameterNameArgs {
            name: "width".into(),
        })
        .unwrap();
    assert_eq!(
        detail.parameter_info,
        client
            .parameters()
            .get(&ParameterNameArgs {
                name: "width".into()
            })
            .unwrap()
    );
    assert_eq!(detail.units, "mm");
    assert_eq!(detail.model_value, 4.0);
    assert_eq!(detail.model_value_type, "nominal");
    assert_eq!(detail.display_format, "decimal");
    assert!(detail.visible);
    assert!(!detail.in_use);
    assert_eq!(detail.tolerance.unwrap().type_, "default");
    assert_eq!(
        detail.custom_property_format.unwrap().precision,
        "threeDecimalPlaces"
    );
    assert_eq!(detail.expression_list, None);
    assert_eq!(detail.driven_by, None);
    assert_eq!(detail.dependents, None);
    assert_eq!(workspace.session(id).unwrap().generation(), 0);
    assert!(!workspace.needs_save(id).unwrap());
}

#[test]
fn detail_reads_tolerance_presentation_choices_and_graph_without_edits() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", MODEL_PARAM, "4 cm")
        .unwrap();
    parameters
        .insert_expression(2, "area", USER_PARAM, "width * width")
        .unwrap();
    parameters.set_deviation(1, 0.2, -0.1).unwrap();
    parameters
        .set_model_value_type(1, MODEL_VALUE_UPPER)
        .unwrap();
    parameters.rename(1, "span").unwrap();
    parameters
        .set_annotations(
            1,
            ParameterAnnotations {
                comment: "design input".into(),
                is_key: true,
                visible: false,
                precision: 4,
                display_format: DISPLAY_FORMAT_FRACTIONAL,
                ..Default::default()
            },
        )
        .unwrap();
    parameters
        .set_custom_property_format(
            1,
            CustomPropertyFormat {
                property_type: CUSTOM_PROPERTY_NUMBER,
                units: "in".into(),
                precision: PRECISION_HALF_FRACTIONAL,
                show_leading_zeros: false,
                show_trailing_zeros: true,
                show_units_string: false,
            },
        )
        .unwrap();
    parameters
        .set_expression_list(1, vec!["4 cm".into(), "8 cm".into()], true)
        .unwrap();
    parameters
        .set_disabled_action_types(1, ActionType(ACTION_EDIT.0 | ACTION_DELETE.0))
        .unwrap();
    let (workspace, _) = parameter_workspace(parameters);
    let notices = workspace.observer().notifications;
    let client = Client::new(SessionQueryHost::new(&workspace));
    let detail = client
        .parameters()
        .get_detail(&ParameterNameArgs {
            name: "span".into(),
        })
        .unwrap();
    assert_eq!(detail.model_value, 4.2);
    assert_eq!(detail.model_value_type, "upper");
    assert_eq!(detail.parameter_info.value, "40 mm");
    assert_eq!(detail.comment, "design input");
    assert!(detail.is_key && detail.in_use && detail.renamed);
    assert!(!detail.visible);
    assert_eq!(detail.precision, 4);
    assert_eq!(detail.display_format, "fractional");
    assert_eq!(
        detail.disabled_action_types,
        Some(vec!["edit".into(), "delete".into()])
    );
    assert_eq!(detail.dependents, Some(vec!["area".into()]));
    let tolerance = detail.tolerance.unwrap();
    assert_eq!(
        (tolerance.type_.as_str(), tolerance.upper, tolerance.lower),
        ("deviation", 0.2, -0.1)
    );
    let property = detail.custom_property_format.unwrap();
    assert_eq!(
        (
            property.property_type.as_str(),
            property.units.as_str(),
            property.precision.as_str()
        ),
        ("number", "in", "halfFractional")
    );
    assert!(
        !property.show_leading_zeros && property.show_trailing_zeros && !property.show_units_string
    );
    let choices = detail.expression_list.unwrap();
    assert_eq!(
        choices.expressions,
        Some(vec!["4 cm".into(), "8 cm".into()])
    );
    assert!(choices.allow_custom_values && choices.custom_order);
    assert_eq!(
        client
            .parameters()
            .get_detail(&ParameterNameArgs {
                name: "area".into()
            })
            .unwrap()
            .driven_by,
        Some(vec!["span".into()])
    );
    assert_eq!(workspace.observer().notifications, notices);
}

#[test]
fn detail_keeps_failed_numeric_values_and_omits_non_numeric_tolerances() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters.set_deviation(1, 0.2, -0.1).unwrap();
    parameters
        .set_model_value_type(1, MODEL_VALUE_LOWER)
        .unwrap();
    parameters.set_expression(1, "missing").unwrap();
    parameters.insert_text(2, "material", "steel").unwrap();
    parameters.insert_boolean(3, "vented", true).unwrap();
    parameters
        .insert_expression(4, "sick", USER_PARAM, "missing")
        .unwrap();
    assert!(parameters.model_value(1).is_err());
    let (workspace, _) = parameter_workspace(parameters);
    let client = Client::new(SessionQueryHost::new(&workspace));
    let failed = client
        .parameters()
        .get_detail(&ParameterNameArgs {
            name: "width".into(),
        })
        .unwrap();
    assert_eq!(failed.model_value, 3.9);
    assert!(failed.parameter_info.health.contains("missing"));
    for (name, units) in [("material", "text"), ("vented", "boolean")] {
        let detail = client
            .parameters()
            .get_detail(&ParameterNameArgs { name: name.into() })
            .unwrap();
        assert_eq!(detail.units, units);
        assert_eq!(detail.tolerance, None);
        assert_eq!(detail.model_value, 0.0);
    }
    let sick = client
        .parameters()
        .get_detail(&ParameterNameArgs {
            name: "sick".into(),
        })
        .unwrap();
    assert_eq!(sick.units, "unitless");
    assert_eq!(sick.model_value, 0.0);
    assert!(
        client
            .parameters()
            .get_detail(&ParameterNameArgs {
                name: "absent".into()
            })
            .is_err()
    );
    let empty = Workspace::new(QueryObserver::default());
    assert!(
        Client::new(SessionQueryHost::new(&empty))
            .parameters()
            .get_detail(&ParameterNameArgs::default())
            .is_err()
    );
}
