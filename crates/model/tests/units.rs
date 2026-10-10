// SPDX-License-Identifier: GPL-2.0-only

use Dimension::*;
use oblikovati_model::quantity::{Dimension, Quantity};
use oblikovati_model::units::{UnitError, UnitsOfMeasure};

// Expected physical values use metres, square metres and cubic metres, independent
// of the production registry's centimetre factors.
const PHYSICAL_UNITS: [(&str, Dimension, f64); 26] = [
    ("pm", Length, 1e-12),
    ("nm", Length, 1e-9),
    ("um", Length, 1e-6),
    ("µm", Length, 1e-6),
    ("mm", Length, 0.001),
    ("cm", Length, 0.01),
    ("m", Length, 1.0),
    ("km", Length, 1000.0),
    ("in", Length, 0.0254),
    ("ft", Length, 0.3048),
    ("rad", Angle, 1.0),
    ("deg", Angle, std::f64::consts::TAU / 360.0),
    ("mm^2", Area, 1e-6),
    ("cm^2", Area, 1e-4),
    ("m^2", Area, 1.0),
    ("mm^3", Volume, 1e-9),
    ("cm^3", Volume, 1e-6),
    ("m^3", Volume, 1.0),
    ("l", Volume, 0.001),
    ("kg", Mass, 1.0),
    ("g", Mass, 0.001),
    ("lb", Mass, 0.45359237),
    ("s", Time, 1.0),
    ("ms", Time, 0.001),
    ("min", Time, 60.0),
    ("hr", Time, 3600.0),
];

fn assert_numeric_close(actual: f64, expected: f64) {
    let relative_error = ((actual - expected) / expected).abs();
    assert!(
        relative_error <= 8.0 * f64::EPSILON,
        "{actual} != {expected}: {relative_error}"
    );
}

fn physical_value(quantity: Quantity, scale: f64) -> f64 {
    let exponent = match quantity.dimension {
        Length => 1,
        Area => 2,
        Volume => 3,
        _ => 0,
    };
    quantity.value * (scale / 100.0).powi(exponent)
}

#[test]
fn all_registered_units_retain_physical_meaning_at_all_working_scales() {
    for scale in [1.0, 1e-10, 1e-4, 1e5] {
        let units = UnitsOfMeasure::default().with_working_scale(scale).unwrap();
        for (name, dimension, expected) in PHYSICAL_UNITS {
            let quantity = units.parse(&format!("1 {name}"), Text).unwrap();
            assert_eq!(quantity.dimension, dimension);
            assert_numeric_close(physical_value(quantity, scale), expected);
        }
    }
}

#[test]
fn every_preference_round_trips_and_explicit_suffix_overrides_it() {
    for (name, dimension, _) in PHYSICAL_UNITS {
        let mut units = UnitsOfMeasure::default().centered_on_length("µm").unwrap();
        units.set_preferred(dimension, name).unwrap();
        assert_eq!(units.preferred_name(dimension), name);
        let quantity = units.from_preferred(3.75, dimension);
        assert_numeric_close(units.to_preferred(quantity), 3.75);
        assert_eq!(units.parse("3.75", dimension).unwrap(), quantity);
        assert_eq!(units.parse("2 cm", dimension).unwrap().dimension, Length);
    }
}

#[test]
fn default_storage_and_preference_edits_are_independent() {
    let original = UnitsOfMeasure::default();
    let mut copy = original.clone();
    let stored = original.parse("25", Length).unwrap();
    copy.set_preferred(Length, "cm").unwrap();
    assert_eq!(original.working_scale(), 1.0);
    assert_eq!(original.preferred_name(Length), "mm");
    assert_eq!(stored, Quantity::new(2.5, Length));
    assert_eq!(copy.to_preferred(stored), 2.5);
    assert_eq!(original.to_preferred(stored), 25.0);
    assert_eq!(copy.working_scale(), original.working_scale());
}

#[test]
fn categories_without_units_keep_bare_values() {
    let units = UnitsOfMeasure::default().centered_on_length("km").unwrap();
    for dimension in [Unitless, Boolean, Text] {
        assert_eq!(units.preferred_name(dimension), "");
        let quantity = units.parse("7.25", dimension).unwrap();
        assert_eq!(quantity, Quantity::new(7.25, dimension));
        assert_eq!(units.to_preferred(quantity), 7.25);
    }
}

#[test]
fn new_document_recommendation_preserves_conditioning_policy() {
    for name in ["mm", "cm", "m", "in", "ft"] {
        assert_eq!(UnitsOfMeasure::recommended_scale(name).unwrap(), 1.0);
    }
    for (name, expected) in [
        ("pm", 1e-10),
        ("nm", 1e-7),
        ("µm", 1e-4),
        ("um", 1e-4),
        ("km", 1e5),
    ] {
        assert_eq!(UnitsOfMeasure::recommended_scale(name).unwrap(), expected);
        let units = UnitsOfMeasure::default().centered_on_length(name).unwrap();
        assert_eq!(units.working_scale(), expected);
        assert_numeric_close(
            units.parse(&format!("5 {name}"), Length).unwrap().value,
            5.0,
        );
    }
}

#[test]
fn bad_scales_cannot_enter_a_conversion_context() {
    for scale in [
        0.0,
        -0.0,
        -1.0,
        f64::NAN,
        f64::INFINITY,
        f64::NEG_INFINITY,
        f64::MIN_POSITIVE,
        f64::MAX,
    ] {
        let error = UnitsOfMeasure::default()
            .with_working_scale(scale)
            .unwrap_err();
        assert!(matches!(error, UnitError::InvalidScale(_)));
        assert!(error.to_string().contains("expected positive finite"));
        assert!(error.to_string().contains(&scale.to_string()));
    }
}

#[test]
fn invalid_preferences_are_atomic_and_name_the_expected_category() {
    let mut units = UnitsOfMeasure::default();
    let before = units.clone();
    let error = units.set_preferred(Length, "deg").unwrap_err();
    assert_eq!(
        error,
        UnitError::WrongCategory {
            name: "deg".into(),
            expected: Length
        }
    );
    assert!(error.to_string().contains("expected a registered Length"));
    assert!(units.set_preferred(Length, "furlong").is_err());
    assert!(units.set_preferred(Text, "cm").is_err());
    assert!(UnitsOfMeasure::recommended_scale("deg").is_err());
    assert!(units.clone().centered_on_length("bad").is_err());
    assert_eq!(units, before);
}

#[test]
fn literal_scanner_accepts_whitespace_signs_and_exponents() {
    let units = UnitsOfMeasure::default();
    for (input, expected) in [
        ("25mm", 2.5),
        ("\u{2003}-5 mm\t", -0.5),
        ("1.5e2 mm", 15.0),
        ("+2E-1m", 20.0),
        (".5cm", 0.5),
        ("1.cm", 1.0),
    ] {
        assert_eq!(
            units.parse(input, Text).unwrap(),
            Quantity::new(expected, Length)
        );
    }
    let zero = units.parse("-0 cm", Length).unwrap();
    assert_eq!(zero.value.to_bits(), (-0.0_f64).to_bits());
}

const MALFORMED_LITERALS: [&str; 19] = [
    "",
    " ",
    "abc",
    "µm",
    ".",
    "1..2",
    "1e",
    "1e+",
    "1+2",
    "--2",
    "NaN",
    "inf",
    "1e999",
    "0x1p2",
    "1_mm",
    "12 furlong",
    "2 MM",
    "2μm",
    "💠",
];

#[test]
fn malformed_literals_are_errors_without_panics() {
    let units = UnitsOfMeasure::default();
    for input in MALFORMED_LITERALS {
        let error = units.parse(input, Length).unwrap_err();
        assert!(error.to_string().contains("expected"));
    }
    assert_eq!(
        units.parse("1e", Length).unwrap_err(),
        UnitError::InvalidNumber("1e".into())
    );
    assert_eq!(
        units.parse("12 furlong", Length).unwrap_err(),
        UnitError::UnknownUnit("furlong".into())
    );
}
