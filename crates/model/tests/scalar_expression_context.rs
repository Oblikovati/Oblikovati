// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::{MODEL_PARAM, REFERENCE_PARAM, USER_PARAM};
use oblikovati_model::{
    parameter::{DimensionExpressionContext, ParameterCollection},
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

fn scalars() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "2 cm")
        .unwrap();
    parameters
        .insert_scalar_expression(2, "depth", " 10 ", Dimension::Length)
        .unwrap();
    parameters
}

fn prefer_inches(parameters: &mut ParameterCollection<u64>) {
    let mut units = parameters.units().clone();
    units.set_preferred(Dimension::Length, "in").unwrap();
    parameters.set_unit_preferences(units).unwrap();
}

#[test]
fn authored_bare_unit_survives_preference_changes_and_copy() {
    let mut parameters = scalars();
    prefer_inches(&mut parameters);
    parameters.copy_to_user(2, 3).unwrap();
    assert_eq!(parameters.get(2).unwrap().source, " 10 ");
    assert_eq!(
        parameters.model_value(2).unwrap(),
        Quantity::new(1.0, Dimension::Length)
    );
    assert_eq!(
        parameters.model_value(3).unwrap(),
        parameters.model_value(2).unwrap()
    );
    assert_eq!(
        parameters.get(3).unwrap().expression_context,
        parameters.get(2).unwrap().expression_context
    );
}

#[test]
fn explicit_edits_capture_the_current_input_unit() {
    let mut parameters = scalars();
    prefer_inches(&mut parameters);
    parameters
        .set_scalar_expression(2, "2", Dimension::Length)
        .unwrap();
    assert_eq!(parameters.model_value(2).unwrap().value, 5.08);
    parameters.set_expression(2, "3").unwrap();
    assert_eq!(parameters.model_value(2).unwrap().value, 7.62);
}

#[test]
fn formula_operands_are_unitless_and_driver_rename_preserves_context() {
    let mut parameters = scalars();
    parameters
        .set_scalar_expression(2, "width / 2", Dimension::Length)
        .unwrap();
    let context = parameters.get(2).unwrap().expression_context.clone();
    prefer_inches(&mut parameters);
    parameters.rename(1, "span").unwrap();
    assert_eq!(parameters.get(2).unwrap().source, "span / 2");
    assert_eq!(parameters.get(2).unwrap().expression_context, context);
    assert_eq!(parameters.drivers(2).unwrap(), [1]);
    parameters.set_expression(1, "8 cm").unwrap();
    assert_eq!(parameters.model_value(2).unwrap().value, 4.0);
}

#[test]
fn incompatible_formula_is_local_health_and_can_recover() {
    let mut parameters = scalars();
    parameters
        .set_scalar_expression(2, "width * width", Dimension::Length)
        .unwrap();
    assert!(parameters.model_value(2).is_err());
    assert_eq!(parameters.get(2).unwrap().source, "width * width");
    parameters.set_expression(2, "width / 2").unwrap();
    assert_eq!(parameters.model_value(2).unwrap().value, 1.0);
}

#[test]
fn malformed_edits_leave_source_context_and_dependencies_unchanged() {
    let mut parameters = scalars();
    let before = parameters.get(2).unwrap().clone();
    for (id, source) in [(2, "width +"), (1, "2"), (99, "2")] {
        assert!(
            parameters
                .set_scalar_expression(id, source, Dimension::Length)
                .is_err()
        );
    }
    assert!(parameters.set_expression(2, "(").is_err());
    assert_eq!(parameters.get(2).unwrap(), &before);
}

#[test]
fn context_restore_checks_version_category_unit_and_kind() {
    let mut parameters = scalars();
    for fields in [
        (2, "length", "mm"),
        (1, "text", ""),
        (1, "length", "deg"),
        (1, "length", "bad"),
    ] {
        let context: DimensionExpressionContext = serde_json::from_value(serde_json::json!({
            "version": fields.0, "category": fields.1, "bare_unit": fields.2
        }))
        .unwrap();
        assert!(
            parameters
                .restore_context_expression(3, "bad", MODEL_PARAM, "1", context)
                .is_err()
        );
    }
    assert_eq!(parameters.ids(), &[1, 2]);
}

#[test]
fn context_restore_refuses_read_only_parameter_kind() {
    let mut parameters = scalars();
    let context = DimensionExpressionContext::new(parameters.units(), Dimension::Length).unwrap();
    assert!(
        parameters
            .restore_context_expression(3, "bad", REFERENCE_PARAM, "1", context)
            .is_err()
    );
    assert_eq!(parameters.ids(), &[1, 2]);
}

#[test]
fn context_scales_bare_inputs_and_keeps_signed_zero() {
    let units = UnitsOfMeasure::default().with_working_scale(0.1).unwrap();
    let mut parameters = ParameterCollection::<u64>::new(units);
    parameters
        .insert_scalar_expression(1, "depth", "10", Dimension::Length)
        .unwrap();
    assert_eq!(parameters.model_value(1).unwrap().value, 10.0);
    parameters
        .set_scalar_expression(1, "-0", Dimension::Length)
        .unwrap();
    assert_eq!(
        parameters.model_value(1).unwrap().value.to_bits(),
        (-0.0_f64).to_bits()
    );
    assert_eq!(parameters.get(1).unwrap().source, "-0");
}

#[test]
fn typed_constant_restore_does_not_parse_category_as_a_unit() {
    let mut parameters = scalars();
    let state = parameters.get(2).unwrap();
    let context = state.expression_context.clone().unwrap();
    parameters
        .restore_context_value(
            3,
            "constant",
            USER_PARAM,
            Quantity::new(3.0, Dimension::Length),
            context,
        )
        .unwrap();
    assert_eq!(parameters.model_value(3).unwrap().value, 3.0);
    assert_eq!(parameters.get(3).unwrap().source, "3 length");
    parameters
        .set_value(3, Quantity::new(4.0, Dimension::Length))
        .unwrap();
    assert_eq!(parameters.model_value(3).unwrap().value, 4.0);
}

#[test]
fn scalar_preflight_declines_missing_drivers_wrong_units_and_overflow() {
    let parameters = scalars();
    for source in ["missing", "1 deg", "1e309 mm", "width +", "width * width"] {
        assert!(
            parameters
                .evaluate_scalar_expression(source, Dimension::Length)
                .is_err(),
            "{source}"
        );
    }
    assert_eq!(
        parameters
            .evaluate_scalar_expression("width / 2", Dimension::Length)
            .unwrap()
            .value,
        1.0
    );
}

#[test]
fn the_same_context_supports_angle_and_unitless_inputs() {
    let parameters = scalars();
    let angle = parameters
        .evaluate_scalar_expression("90", Dimension::Angle)
        .unwrap();
    assert_eq!(angle.dimension, Dimension::Angle);
    assert_eq!(angle.value, std::f64::consts::FRAC_PI_2);
    let count = parameters
        .evaluate_scalar_expression("3", Dimension::Unitless)
        .unwrap();
    assert_eq!(count, Quantity::new(3.0, Dimension::Unitless));
}

#[test]
fn typed_context_refuses_literal_categories_and_read_only_constants() {
    let parameters = scalars();
    assert!(DimensionExpressionContext::new(parameters.units(), Dimension::Text).is_err());
    let context = DimensionExpressionContext::new(parameters.units(), Dimension::Length).unwrap();
    let mut parameters = parameters;
    let value = Quantity::new(3.0, Dimension::Length);
    assert!(
        parameters
            .restore_context_value(3, "bad", REFERENCE_PARAM, value, context)
            .is_err()
    );
}
