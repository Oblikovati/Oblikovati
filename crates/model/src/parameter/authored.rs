// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::quantity::Dimension;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Attach display units to unitless input; for example `parameters.qualify_authored("7", Dimension::Length)`.
    /// Failed or already dimensioned expressions retain their source and downstream diagnostics.
    pub fn qualify_authored(&self, source: &str, dimension: Dimension) -> String {
        let unit = self.units.preferred_name(dimension);
        if unit.is_empty() || !self.authored_is_unitless(source) {
            return source.into();
        }
        let trimmed = source.trim();
        if trimmed.parse::<f64>().is_ok() {
            return format!("{trimmed} {unit}");
        }
        format!("({trimmed}) * 1 {unit}")
    }

    /// Add user-authored numeric input; for example `parameters.insert_authored_expression(id, "width", USER_PARAM, "7", Dimension::Length)`.
    pub fn insert_authored_expression(
        &mut self,
        id: Id,
        name: &str,
        kind: ParameterKind,
        source: &str,
        dimension: Dimension,
    ) -> Result<(), ParameterError<Id>> {
        let qualified = self.qualify_authored(source, dimension);
        self.insert_expression(id, name, kind, &qualified)
    }

    /// Edit user-authored numeric input; for example `parameters.set_authored_expression(id, "7", Dimension::Length)`.
    /// The requested category supplies a display unit, not a permanent dimension constraint.
    pub fn set_authored_expression(
        &mut self,
        id: Id,
        source: &str,
        dimension: Dimension,
    ) -> Result<(), ParameterError<Id>> {
        let qualified = self.qualify_authored(source, dimension);
        self.set_expression(id, &qualified)
    }

    fn authored_is_unitless(&self, source: &str) -> bool {
        let Ok(expression) = Expression::parse(source, &self.units) else {
            return false;
        };
        expression
            .bind(|name| self.find(name))
            .evaluate(|id| self.records[&id].state.current_value())
            .is_ok_and(|quantity| quantity.dimension == Dimension::Unitless)
    }
}
