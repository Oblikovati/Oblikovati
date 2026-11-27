// SPDX-License-Identifier: GPL-2.0-only

use super::expression_context::compile_context_source;
use super::*;
use crate::quantity::Quantity;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Add an editable formula; for example `parameters.insert_expression(id, "width", USER_PARAM, "10 cm")`.
    pub fn insert_expression(
        &mut self,
        id: Id,
        name: &str,
        kind: ParameterKind,
        source: &str,
    ) -> Result<(), ParameterError<Id>> {
        if !editable(kind) {
            return Err(ParameterError::Kind {
                kind,
                expected: "an editable Model, User, or Table kind",
            });
        }
        let expression = Expression::parse(source, &self.units)?;
        self.insert_compiled(id, name, kind, expression, None)
    }

    /// Add a constant, including read-only measurements; for example `parameters.insert_value(id, "span", REFERENCE_PARAM, measured)`.
    pub fn insert_value(
        &mut self,
        id: Id,
        name: &str,
        kind: ParameterKind,
        quantity: Quantity,
    ) -> Result<(), ParameterError<Id>> {
        require_kind(kind)?;
        require_numeric_quantity(id, quantity)?;
        self.insert_compiled(id, name, kind, Expression::constant(quantity), None)
    }

    /// Edit a formula and recompute its readers; for example `parameters.set_expression(id, "2 * width")`.
    /// Malformed syntax changes nothing. Cycles retain failed authored source and detach its drivers.
    pub fn set_expression(&mut self, id: Id, source: &str) -> Result<(), ParameterError<Id>> {
        self.require_editable(id)?;
        self.require_formula(id)?;
        let context = self.current_expression_context(id)?;
        let expression = compile_context_source(source, &self.units, context.as_ref())?;
        self.records
            .get_mut(&id)
            .expect("validated parameter")
            .state
            .expression_context = context;
        self.replace_compiled(id, expression)
    }

    /// Replace an editable formula with a constant; for example `parameters.set_value(id, length)`.
    pub fn set_value(&mut self, id: Id, quantity: Quantity) -> Result<(), ParameterError<Id>> {
        self.require_editable(id)?;
        self.require_formula(id)?;
        require_numeric_quantity(id, quantity)?;
        self.replace_compiled(id, Expression::constant(quantity))
    }

    /// Update a geometry/link-owned value; for example `parameters.update_measured(id, length)`.
    pub fn update_measured(
        &mut self,
        id: Id,
        quantity: Quantity,
    ) -> Result<(), ParameterError<Id>> {
        self.require_formula(id)?;
        let kind = self.require_record(id)?.state.kind;
        if !matches!(kind, REFERENCE_PARAM | DERIVED_PARAM) {
            return Err(ParameterError::Kind {
                kind,
                expected: "a Reference or Derived measurement",
            });
        }
        require_numeric_quantity(id, quantity)?;
        self.replace_compiled(id, Expression::constant(quantity))
    }

    pub(super) fn insert_compiled(
        &mut self,
        id: Id,
        name: &str,
        kind: ParameterKind,
        expression: Expression,
        context: Option<DimensionExpressionContext>,
    ) -> Result<(), ParameterError<Id>> {
        if expression.references().contains(&name.to_owned()) {
            return Err(DependencyError::Cycle { id, driver: id }.into());
        }
        let binding = expression.bind(|name| self.find(name));
        let mut record = ParameterRecord::new(name, kind, expression.source().into());
        record.state.expression_context = context;
        record.formula = Some(ParameterFormula {
            expression,
            binding,
        });
        self.insert_record(id, record)?;
        self.rebind(id)?;
        self.resolve_forward_references(name, id);
        Ok(())
    }

    pub(super) fn insert_record(
        &mut self,
        id: Id,
        record: ParameterRecord<Id>,
    ) -> Result<(), ParameterError<Id>> {
        if self.records.contains_key(&id) {
            return Err(DependencyError::Duplicate(id).into());
        }
        self.require_available_name(&record.state.name, None)?;
        self.graph.insert(id)?;
        self.names.insert(record.state.name.clone(), id);
        self.order.push(id);
        self.records.insert(id, record);
        Ok(())
    }

    pub(super) fn replace_compiled(
        &mut self,
        id: Id,
        expression: Expression,
    ) -> Result<(), ParameterError<Id>> {
        let record = self.records.get_mut(&id).expect("validated parameter");
        record.state.source = expression.source().into();
        record.state.health = ParameterHealth::Healthy;
        record
            .formula
            .as_mut()
            .expect("numeric parameter")
            .expression = expression;
        self.rebind(id)
    }

    pub(super) fn resolve_forward_references(&mut self, name: &str, added: Id) {
        let waiting: Vec<Id> = self
            .order
            .iter()
            .copied()
            .filter(|id| *id != added)
            .filter(|id| {
                self.records[id]
                    .references()
                    .iter()
                    .any(|reference| reference == name)
            })
            .collect();
        for id in waiting {
            let _ = self.rebind(id);
        }
    }
}

fn require_numeric_quantity<Id>(id: Id, quantity: Quantity) -> Result<(), ParameterError<Id>> {
    if !quantity.dimension.is_arithmetic() {
        return Err(ParameterError::Value {
            id,
            found: format!("{quantity:?}"),
            expected: "a numeric quantity; use typed literal insertion or editing for text and Boolean values",
        });
    }
    Ok(())
}
