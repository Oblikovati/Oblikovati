// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::{
    expression::Expression,
    quantity::{Dimension, Quantity},
    units::{category_name, parse_category},
};

#[test]
fn category_names_are_stable_and_bijective_including_non_numeric_categories() {
    for (name, dimension) in [
        ("unitless", Dimension::Unitless),
        ("length", Dimension::Length),
        ("angle", Dimension::Angle),
        ("area", Dimension::Area),
        ("volume", Dimension::Volume),
        ("mass", Dimension::Mass),
        ("time", Dimension::Time),
        ("boolean", Dimension::Boolean),
        ("text", Dimension::Text),
    ] {
        assert_eq!(category_name(dimension), name);
        assert_eq!(parse_category(name), Some(dimension));
        let expression = Expression::constant(Quantity::new(1.25, dimension));
        assert_eq!(expression.source(), format!("1.25 {name}"));
        assert_eq!(
            expression.evaluate().unwrap(),
            Quantity::new(1.25, dimension)
        );
    }
    assert_eq!(parse_category("Length"), None);
    assert_eq!(parse_category("unknown"), None);
}
