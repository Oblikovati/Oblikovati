// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::{
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

#[test]
fn transfer_scales_length_powers_and_preserves_other_dimensions() {
    let source = UnitsOfMeasure::default().with_working_scale(4.0).unwrap();
    let target = UnitsOfMeasure::default().with_working_scale(0.5).unwrap();
    for (dimension, expected) in [
        (Dimension::Length, 16.0),
        (Dimension::Area, 128.0),
        (Dimension::Volume, 1024.0),
        (Dimension::Unitless, 2.0),
        (Dimension::Angle, 2.0),
        (Dimension::Mass, 2.0),
        (Dimension::Time, 2.0),
    ] {
        let input = Quantity::new(2.0, dimension);
        let converted = source.transfer_quantity(input, &target).unwrap();
        assert_eq!(converted, Quantity::new(expected, dimension));
        assert_eq!(target.transfer_quantity(converted, &source).unwrap(), input);
    }
}

#[test]
fn equal_scales_and_zero_keep_exact_input_bits() {
    let source = UnitsOfMeasure::default();
    let target = source.clone().with_working_scale(0.001).unwrap();
    for value in [-0.0, 0.0, f64::from_bits(1), f64::MAX] {
        let input = Quantity::new(value, Dimension::Volume);
        assert_eq!(
            source
                .transfer_quantity(input, &source)
                .unwrap()
                .value
                .to_bits(),
            value.to_bits()
        );
    }
    assert_eq!(
        source
            .transfer_quantity(Quantity::new(-0.0, Dimension::Length), &target)
            .unwrap()
            .value
            .to_bits(),
        (-0.0f64).to_bits()
    );
}

#[test]
fn extreme_scale_ratios_do_not_destroy_representable_results() {
    let large = UnitsOfMeasure::default()
        .with_working_scale(2f64.powi(300))
        .unwrap();
    let small = UnitsOfMeasure::default()
        .with_working_scale(2f64.powi(-300))
        .unwrap();
    for (source, target, input, expected) in [
        (&large, &small, 2f64.powi(-1000), 2f64.powi(800)),
        (&small, &large, 2f64.powi(1000), 2f64.powi(-800)),
    ] {
        assert_eq!(
            source
                .transfer_quantity(Quantity::new(input, Dimension::Volume), target)
                .unwrap()
                .value,
            expected
        );
    }
    let fractional = UnitsOfMeasure::default().with_working_scale(0.5).unwrap();
    let subnormal = UnitsOfMeasure::default()
        .with_working_scale(2f64.powi(-358))
        .unwrap();
    assert_eq!(
        fractional
            .transfer_quantity(
                Quantity::new(f64::from_bits(1), Dimension::Volume),
                &subnormal
            )
            .unwrap()
            .value,
        0.125
    );
    assert!(
        large
            .transfer_quantity(Quantity::new(1.0, Dimension::Volume), &small)
            .is_err()
    );
    assert!(
        small
            .transfer_quantity(Quantity::new(1.0, Dimension::Volume), &large)
            .is_err()
    );
}

#[test]
fn nonfinite_and_nonnumeric_inputs_are_refused_even_at_equal_scales() {
    let units = UnitsOfMeasure::default();
    for value in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        assert!(
            units
                .transfer_quantity(Quantity::new(value, Dimension::Length), &units)
                .is_err()
        );
    }
    for dimension in [Dimension::Boolean, Dimension::Text] {
        assert!(
            units
                .transfer_quantity(Quantity::new(1.0, dimension), &units)
                .is_err()
        );
    }
}
