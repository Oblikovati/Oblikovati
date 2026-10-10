// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{
    parameter::{ParameterAnnotations, ParameterCollection},
    property::{DocumentProperties, USER_DEFINED},
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

fn parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "1 cm")
        .unwrap();
    parameters.drain_changed();
    parameters
}

#[test]
fn unchanged_recipe_ignores_pending_change_queue() {
    let before = parameters();
    let mut after = before.clone();
    after.set_expression(1, "1 cm").unwrap();
    assert!(before.same_recipe(&after));
    assert!(!after.drain_changed().is_empty());
    assert!(before.same_recipe(&after));
}

#[test]
fn identity_source_metadata_groups_and_settings_are_recipe_changes() {
    let before = parameters();
    let mut variants = vec![before.clone(); 5];
    variants[0]
        .insert_expression(2, "other", USER_PARAM, "1 cm")
        .unwrap();
    variants[1].set_expression(1, "10 mm").unwrap();
    variants[2]
        .set_annotations(
            1,
            ParameterAnnotations {
                exposed_as_property: true,
                ..Default::default()
            },
        )
        .unwrap();
    variants[3].add_group("key", "display", "client").unwrap();
    let mut settings = before.settings().clone();
    settings.linear_dimension_precision += 1;
    variants[4].set_settings(settings).unwrap();
    for after in variants {
        assert!(!before.same_recipe(&after));
        assert!(!after.same_recipe(&before));
    }
}

#[test]
fn measured_values_tolerance_bits_and_units_are_compared_exactly() {
    let mut before = parameters();
    before
        .insert_value(
            2,
            "measured",
            REFERENCE_PARAM,
            Quantity::new(0.0, Dimension::Length),
        )
        .unwrap();
    let mut after = before.clone();
    after
        .update_measured(2, Quantity::new(-0.0, Dimension::Length))
        .unwrap();
    assert!(!before.same_recipe(&after));
    let mut positive = parameters();
    positive.set_deviation(1, 0.0, 0.0).unwrap();
    let mut negative = positive.clone();
    negative.set_deviation(1, 0.0, -0.0).unwrap();
    assert!(!positive.same_recipe(&negative));
    assert_eq!(
        negative.get(1).unwrap().tolerance.lower.to_bits(),
        (-0.0_f64).to_bits()
    );
    let mut units = UnitsOfMeasure::default();
    units.set_preferred(Dimension::Length, "in").unwrap();
    assert!(
        !ParameterCollection::<u64>::new(units)
            .same_recipe(&ParameterCollection::new(UnitsOfMeasure::default()))
    );
}

#[test]
fn derived_table_recipe_difference_is_visible_even_without_produced_parameters() {
    let before = parameters();
    let mut after = before.clone();
    after
        .restore_derived_table(1, "source.obk", &["missing".into()], false)
        .unwrap();
    assert!(!before.same_recipe(&after));
    assert!(after.same_recipe(&after.clone()));
}

#[test]
fn property_comparison_preserves_order_payload_and_signed_zero() {
    let mut before = DocumentProperties::default();
    before
        .put(USER_DEFINED, "Number", VariantValue::Double(0.0))
        .unwrap();
    before
        .put(USER_DEFINED, "Title", VariantValue::String("part".into()))
        .unwrap();
    assert!(before.same_recipe(&before.clone()));
    let mut after = before.clone();
    after
        .put(USER_DEFINED, "Number", VariantValue::Double(-0.0))
        .unwrap();
    assert!(!before.same_recipe(&after));
    after = before.clone();
    after.remove(USER_DEFINED, "Number");
    after
        .put(USER_DEFINED, "Number", VariantValue::Double(0.0))
        .unwrap();
    assert!(!before.same_recipe(&after));
}
