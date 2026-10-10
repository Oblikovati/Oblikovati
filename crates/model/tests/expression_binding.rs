// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::{
    expression::{Expression, ExpressionError},
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};
use std::collections::BTreeMap;

struct ParameterScope {
    names: BTreeMap<String, u64>,
    values: BTreeMap<u64, Quantity>,
}

impl ParameterScope {
    fn new() -> Self {
        Self {
            names: [
                ("width".into(), 1),
                ("alias".into(), 1),
                ("height".into(), 2),
            ]
            .into(),
            values: [
                (1, Quantity::new(10.0, Dimension::Length)),
                (2, Quantity::new(3.0, Dimension::Length)),
            ]
            .into(),
        }
    }
    fn resolve(&self, name: &str) -> Option<u64> {
        self.names.get(name).copied()
    }
    fn value_of(&self, id: u64) -> Option<Quantity> {
        self.values.get(&id).copied()
    }
}

fn compile(source: &str) -> Expression {
    Expression::parse(source, &UnitsOfMeasure::default()).unwrap()
}

#[test]
fn ordered_references_deduplicate_names_and_ids_independently() {
    let scope = ParameterScope::new();
    let expression = compile(" width + height + width + alias ");
    assert_eq!(expression.source(), " width + height + width + alias ");
    assert_eq!(expression.references(), ["width", "height", "alias"]);
    let bound = expression.bind(|name| scope.resolve(name));
    assert!(bound.unresolved().is_empty());
    assert_eq!(bound.bound_ids(), [1, 2]);
    assert_eq!(bound.evaluate(|id| scope.value_of(id)).unwrap().value, 33.0);
}

#[test]
fn binding_snapshot_survives_rename_and_reads_current_values() {
    let mut scope = ParameterScope::new();
    let expression = compile("2 * width + 5 mm");
    let bound = expression.bind(|name| scope.resolve(name));
    scope.names.remove("width");
    scope.names.insert("renamed".into(), 1);
    scope
        .values
        .insert(1, Quantity::new(12.0, Dimension::Length));
    assert_eq!(bound.evaluate(|id| scope.value_of(id)).unwrap().value, 24.5);
    assert_eq!(
        expression.bind(|name| scope.resolve(name)).unresolved(),
        ["width"]
    );
    assert!(expression.evaluate().is_err());
    assert!(bound.clone().unresolved().is_empty());
}

#[test]
fn unresolved_and_missing_values_are_distinct_health_inputs() {
    let scope = ParameterScope::new();
    let expression = compile("unknown + unknown + height");
    let bound = expression.bind(|name| scope.resolve(name));
    assert_eq!(bound.unresolved(), ["unknown"]);
    let error = bound.evaluate(|id| scope.value_of(id)).unwrap_err();
    assert!(error.to_string().contains("binding to a stable"));
    let missing = compile("width").bind(|_| Some(999_u64));
    assert!(
        missing
            .evaluate(|id| scope.value_of(id))
            .unwrap_err()
            .to_string()
            .contains("current value")
    );
}

#[test]
fn errors_follow_source_order_and_unknown_calls_precede_arguments() {
    assert!(matches!(
        compile("missing + 1/0").evaluate(),
        Err(ExpressionError::Reference { .. })
    ));
    assert!(matches!(
        compile("1/0 + missing").evaluate(),
        Err(ExpressionError::Arithmetic(_))
    ));
    let error = compile("unknown(1/0)").evaluate().unwrap_err();
    assert!(matches!(error, ExpressionError::Function { .. }));
    assert!(error.to_string().contains("registered built-in"));
    assert!(matches!(
        compile("sin(missing, 1/0)").evaluate(),
        Err(ExpressionError::Reference { .. })
    ));
}
