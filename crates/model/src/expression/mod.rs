// SPDX-License-Identifier: GPL-2.0-only

//! Compile authored formulas and bind references to caller-owned stable identities.
mod binding;
mod error;
mod evaluate;
mod functions;
mod node;
mod parser;
mod rename;
mod token;

use crate::{quantity::Quantity, units::UnitsOfMeasure};
pub use binding::BoundExpression;
pub use error::ExpressionError;
use node::Node;
pub(crate) use token::reference_name;

#[derive(Clone, Debug)]
struct ReferenceSpan {
    start: usize,
    end: usize,
    reference: usize,
}

/// An owned expression snapshot. Syntax nodes are flat to keep evaluation and drop stack-safe.
#[derive(Clone, Debug)]
pub struct Expression {
    source: String,
    nodes: Vec<Node>,
    references: Vec<String>,
    spans: Vec<ReferenceSpan>,
    root: usize,
}

impl Expression {
    /// Store an already measured quantity without parsing it; for example `Expression::constant(area)`.
    pub fn constant(quantity: Quantity) -> Self {
        Self {
            source: format!(
                "{} {}",
                quantity.value,
                crate::units::category_name(quantity.dimension)
            ),
            nodes: vec![Node::Literal(quantity)],
            references: Vec::new(),
            spans: Vec::new(),
            root: 0,
        }
    }

    /// Compile in document working units; for example `Expression::parse("2 * width", &units)`.
    /// Bare expression numbers are unitless, independent of display preferences.
    pub fn parse(source: &str, units: &UnitsOfMeasure) -> Result<Self, ExpressionError> {
        parser::compile(source, units)
    }

    /// Compile a bare numeric input in its captured unit; formula operands remain unitless.
    /// Example: `Expression::parse_with_bare_unit("10", &units, "mm")?` retains source `10`.
    pub fn parse_with_bare_unit(
        source: &str,
        units: &UnitsOfMeasure,
        unit: &str,
    ) -> Result<Self, ExpressionError> {
        if !source.trim().parse::<f64>().is_ok_and(f64::is_finite) || unit.is_empty() {
            return Self::parse(source, units);
        }
        let mut expression = Self::parse(&format!("{source} {unit}"), units)?;
        expression.source = source.into();
        Ok(expression)
    }

    /// Return the authored source unchanged; for example `assert_eq!(formula.source(), "width");`.
    pub fn source(&self) -> &str {
        &self.source
    }

    /// Read distinct names in source order; for example `formula.references()`.
    pub fn references(&self) -> &[String] {
        &self.references
    }

    /// Evaluate a reference-free formula; for example `formula.evaluate()`.
    pub fn evaluate(&self) -> Result<Quantity, ExpressionError> {
        self.evaluate_with(|index| {
            Err(ExpressionError::Reference {
                name: self.references[index].clone(),
                expected: "a binding to a stable parameter ID",
            })
        })
    }

    /// Bind an independent snapshot; for example `formula.bind(|name| parameter_ids.get(name).copied())`.
    pub fn bind<Id: Copy + Ord>(
        &self,
        mut resolve: impl FnMut(&str) -> Option<Id>,
    ) -> BoundExpression<Id> {
        let bindings = self.references.iter().map(|name| resolve(name)).collect();
        BoundExpression {
            expression: self.clone(),
            bindings,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn constant_folding_collapses_only_successful_subtrees() {
        let units = UnitsOfMeasure::default();
        let constant = Expression::parse("2 * (3 + 4)", &units).unwrap();
        assert!(matches!(
            constant.nodes[constant.root],
            Node::Literal(Quantity { value: 14.0, .. })
        ));
        let invalid = Expression::parse("1 / 0", &units).unwrap();
        assert!(matches!(invalid.nodes[invalid.root], Node::Binary(..)));
        let dynamic = Expression::parse("width * (3 + 4)", &units).unwrap();
        let Node::Binary(_, _, right) = dynamic.nodes[dynamic.root] else {
            panic!("expected a dynamic product")
        };
        assert!(matches!(
            dynamic.nodes[right],
            Node::Literal(Quantity { value: 7.0, .. })
        ));
    }
}
