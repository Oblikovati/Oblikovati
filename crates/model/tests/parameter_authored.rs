// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{REFERENCE_PARAM, USER_PARAM};
use oblikovati_model::{
    expression::Expression,
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

#[test]
fn qualification_matches_go_source_examples() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "10 mm")
        .unwrap();
    for (source, expected) in [
        ("7", "7 mm"),
        (" 7 ", "7 mm"),
        ("3*2", "(3*2) * 1 mm"),
        ("7 mm", "7 mm"),
        ("1 cm", "1 cm"),
        ("width/2", "width/2"),
        ("-2.5e2", "-2.5e2 mm"),
        ("(1/8)", "((1/8)) * 1 mm"),
    ] {
        assert_eq!(
            parameters.qualify_authored(source, Dimension::Length),
            expected
        );
    }
    assert_eq!(
        parameters.qualify_authored("45", Dimension::Angle),
        "45 deg"
    );
    assert_eq!(
        parameters.qualify_authored(" 5 ", Dimension::Unitless),
        " 5 "
    );
}

#[test]
fn bare_and_compound_edits_use_display_units_at_each_working_scale() {
    for scale in [0.001, 0.1, 1.0, 100.0] {
        let units = UnitsOfMeasure::default().with_working_scale(scale).unwrap();
        let mut parameters = ParameterCollection::new(units);
        parameters
            .insert_authored_expression(1, "width", USER_PARAM, "7", Dimension::Length)
            .unwrap();
        assert_eq!(
            parameters.model_value(1).unwrap(),
            Quantity::new(7.0 * 0.1 / scale, Dimension::Length)
        );
        parameters
            .set_authored_expression(1, "3*2", Dimension::Length)
            .unwrap();
        assert_eq!(
            parameters.model_value(1).unwrap().value,
            6.0 * (0.1 / scale)
        );
        parameters
            .set_authored_expression(1, "1 cm", Dimension::Length)
            .unwrap();
        assert_eq!(parameters.model_value(1).unwrap().value, 1.0 / scale);
    }
}

#[test]
fn qualification_uses_every_numeric_display_category() {
    let mut units = UnitsOfMeasure::default();
    units.set_preferred(Dimension::Length, "in").unwrap();
    let parameters = ParameterCollection::<u64>::new(units.clone());
    for (dimension, suffix) in [
        (Dimension::Length, "in"),
        (Dimension::Angle, "deg"),
        (Dimension::Area, "mm^2"),
        (Dimension::Volume, "mm^3"),
        (Dimension::Mass, "kg"),
        (Dimension::Time, "s"),
    ] {
        let source = parameters.qualify_authored("2", dimension);
        assert_eq!(source, format!("2 {suffix}"));
        let value = Expression::parse(&source, &units)
            .unwrap()
            .evaluate()
            .unwrap();
        assert_eq!(value, units.from_preferred(2.0, dimension));
    }
}

#[test]
fn unitless_reference_formulas_remain_bound_after_qualification() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "count", USER_PARAM, "3")
        .unwrap();
    parameters
        .insert_authored_expression(2, "width", USER_PARAM, "count * 2", Dimension::Length)
        .unwrap();
    assert_eq!(parameters.get(2).unwrap().source, "(count * 2) * 1 mm");
    assert_eq!(parameters.drivers(2).unwrap(), [1]);
    parameters.set_expression(1, "5").unwrap();
    assert_eq!(parameters.model_value(2).unwrap().value, 1.0);
    parameters.rename(1, "copies").unwrap();
    assert_eq!(parameters.get(2).unwrap().source, "(copies * 2) * 1 mm");
    assert_eq!(parameters.drivers(2).unwrap(), [1]);
}

#[test]
fn qualification_preserves_invalid_unresolved_and_nonnumeric_source() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters.insert_text(1, "material", "steel").unwrap();
    parameters
        .insert_expression(2, "sick", USER_PARAM, "missing")
        .unwrap();
    for source in [
        "",
        "7 +",
        "missing / 2",
        "material",
        "sick",
        "1 / 0",
        "1 cm + 1 rad",
    ] {
        assert_eq!(
            parameters.qualify_authored(source, Dimension::Length),
            source
        );
    }
    for dimension in [Dimension::Text, Dimension::Boolean] {
        assert_eq!(parameters.qualify_authored("7", dimension), "7");
    }
}

#[test]
fn authored_edit_entry_points_preserve_atomic_and_read_only_rules() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "1 cm")
        .unwrap();
    parameters
        .insert_value(
            2,
            "measured",
            REFERENCE_PARAM,
            Quantity::new(1.0, Dimension::Length),
        )
        .unwrap();
    parameters.drain_changed();
    let before = parameters.get(1).unwrap().clone();
    assert!(
        parameters
            .set_authored_expression(1, "3 +", Dimension::Length)
            .is_err()
    );
    assert!(
        parameters
            .set_authored_expression(2, "7", Dimension::Length)
            .is_err()
    );
    assert!(
        parameters
            .set_authored_expression(99, "7", Dimension::Length)
            .is_err()
    );
    assert!(
        parameters
            .insert_authored_expression(3, "new", REFERENCE_PARAM, "7", Dimension::Length)
            .is_err()
    );
    assert_eq!(parameters.get(1), Some(&before));
    assert_eq!(parameters.ids(), &[1, 2]);
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn registered_area_volume_literals_are_dimensions_not_general_exponentiation() {
    let units = UnitsOfMeasure::default().with_working_scale(0.1).unwrap();
    for (source, dimension, expected) in [
        ("2 mm^2", Dimension::Area, 2.0 * 0.01 / (0.1 * 0.1)),
        ("3 cm^3", Dimension::Volume, 3.0 / (0.1 * 0.1 * 0.1)),
        (
            "2mm^2 / 1 mm",
            Dimension::Length,
            (2.0 * 0.01 / (0.1 * 0.1)) / (0.1 / 0.1),
        ),
    ] {
        let expression = Expression::parse(source, &units).unwrap();
        assert!(expression.references().is_empty());
        assert_eq!(
            expression.evaluate().unwrap(),
            Quantity::new(expected, dimension)
        );
    }
    for source in [
        "2^3", "area^2", "1 mm^4", "1 kg^2", "1 cm^2^3", "1 mm ^ 2", "1 mm^", "1 cm^2.0",
    ] {
        assert!(Expression::parse(source, &units).is_err(), "{source}");
    }
}

#[test]
fn powered_units_do_not_become_reference_names_or_rename_targets() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "mm", USER_PARAM, "2 mm^2")
        .unwrap();
    parameters
        .insert_expression(2, "area", USER_PARAM, "mm + 1 mm^2")
        .unwrap();
    parameters.rename(1, "base").unwrap();
    assert_eq!(parameters.get(2).unwrap().source, "base + 1 mm^2");
    assert!(parameters.rename(1, "mm^2").is_err());
    assert_eq!(parameters.drivers(2).unwrap(), [1]);
}
