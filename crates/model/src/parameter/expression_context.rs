// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{
    expression::ExpressionError,
    quantity::{Dimension, Quantity, QuantityError},
};

/// Authored numeric category and bare-input unit; independent of later display preferences.
#[derive(Clone, Debug, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct DimensionExpressionContext {
    version: u32,
    category: String,
    bare_unit: String,
}
impl DimensionExpressionContext {
    /// Capture the current input unit; example: `DimensionExpressionContext::new(&units, Dimension::Length)?`.
    pub fn new(units: &UnitsOfMeasure, dimension: Dimension) -> Result<Self, ExpressionError> {
        let context = Self {
            version: 1,
            category: crate::units::category_name(dimension).into(),
            bare_unit: units.preferred_name(dimension).into(),
        };
        context.validate(units)?;
        Ok(context)
    }
    /// Validate stored context before compiling; example: `context.validate(&units)?` on restore.
    pub fn validate(&self, units: &UnitsOfMeasure) -> Result<(), ExpressionError> {
        let expected = self.dimension()?;
        if self.bare_unit.is_empty() && expected == Dimension::Unitless {
            return Ok(());
        }
        let found = units
            .parse(&format!("1 {}", self.bare_unit), Dimension::Unitless)
            .map_err(|_| self.invalid_context())?;
        if self.bare_unit.is_empty() || found.dimension != expected {
            return Err(self.invalid_context());
        }
        Ok(())
    }
    fn dimension(&self) -> Result<Dimension, ExpressionError> {
        if self.version != 1 {
            return Err(self.invalid_context());
        }
        crate::units::parse_category(&self.category)
            .filter(|dimension| dimension.is_arithmetic())
            .ok_or_else(|| self.invalid_context())
    }
    pub(super) fn validate_quantity(&self, quantity: Quantity) -> Result<(), ExpressionError> {
        let expected = self.dimension()?;
        if quantity.dimension == expected {
            return Ok(());
        }
        Err(QuantityError::Incompatible {
            operator: "scalar input",
            left: quantity.dimension,
            right: expected,
        }
        .into())
    }
    fn invalid_context(&self) -> ExpressionError {
        ExpressionError::InputContext {
            found: format!("{self:?}"),
            expected: "version 1, a numeric category, and a matching captured bare-input unit",
        }
    }
}
pub(super) fn compile_context_source(
    source: &str,
    units: &UnitsOfMeasure,
    context: Option<&DimensionExpressionContext>,
) -> Result<Expression, ExpressionError> {
    let Some(context) = context else {
        return Expression::parse(source, units);
    };
    context.validate(units)?;
    Expression::parse_with_bare_unit(source, units, &context.bare_unit)
}
impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Store a dimensioned Model expression with its original text and captured unit.
    /// Example: `parameters.insert_scalar_expression(id, "depth", "width / 2", Dimension::Length)?`.
    pub fn insert_scalar_expression(
        &mut self,
        id: Id,
        name: &str,
        source: &str,
        dimension: Dimension,
    ) -> Result<(), ParameterError<Id>> {
        let context = DimensionExpressionContext::new(&self.units, dimension)?;
        self.restore_context_expression(id, name, MODEL_PARAM, source, context)
    }
    /// Restore one stored formula in its original input context, including User copies.
    /// Example: `parameters.restore_context_expression(id, "depth", MODEL_PARAM, "10", context)?`.
    pub fn restore_context_expression(
        &mut self,
        id: Id,
        name: &str,
        kind: ParameterKind,
        source: &str,
        context: DimensionExpressionContext,
    ) -> Result<(), ParameterError<Id>> {
        if !editable(kind) {
            return Err(ParameterError::Kind {
                kind,
                expected: "an editable Model, User, or Table kind",
            });
        }
        let expression = compile_context_source(source, &self.units, Some(&context))?;
        self.insert_compiled(id, name, kind, expression, Some(context))
    }
    /// Edit an owned scalar with a current display-unit context; example: `parameters.set_scalar_expression(id, "20", Dimension::Length)?`.
    pub fn set_scalar_expression(
        &mut self,
        id: Id,
        source: &str,
        dimension: Dimension,
    ) -> Result<(), ParameterError<Id>> {
        self.validate_owned_scalar(id)?;
        let context = DimensionExpressionContext::new(&self.units, dimension)?;
        let expression = compile_context_source(source, &self.units, Some(&context))?;
        self.records
            .get_mut(&id)
            .expect("validated scalar")
            .state
            .expression_context = Some(context);
        self.replace_compiled(id, expression)
    }
    /// Check an input without inserting a parameter; example: `parameters.evaluate_scalar_expression("width / 2", Dimension::Length)?`.
    pub fn evaluate_scalar_expression(
        &self,
        source: &str,
        dimension: Dimension,
    ) -> Result<Quantity, ExpressionError> {
        let context = DimensionExpressionContext::new(&self.units, dimension)?;
        let expression = compile_context_source(source, &self.units, Some(&context))?;
        let value = self.evaluate_compiled_expression(expression)?;
        context.validate_quantity(value)?;
        Ok(value)
    }
    /// Restore a typed constant without parsing its diagnostic category as a named unit.
    /// Example: `parameters.restore_context_value(id, "depth", MODEL_PARAM, quantity, context)?`.
    pub fn restore_context_value(
        &mut self,
        id: Id,
        name: &str,
        kind: ParameterKind,
        value: Quantity,
        context: DimensionExpressionContext,
    ) -> Result<(), ParameterError<Id>> {
        if !editable(kind) {
            return Err(ParameterError::Kind {
                kind,
                expected: "an editable Model, User, or Table kind",
            });
        }
        context.validate(&self.units)?;
        self.insert_compiled(id, name, kind, Expression::constant(value), Some(context))
    }
    pub(super) fn current_expression_context(
        &self,
        id: Id,
    ) -> Result<Option<DimensionExpressionContext>, ParameterError<Id>> {
        self.require_record(id)?
            .state
            .expression_context
            .as_ref()
            .map(|context| DimensionExpressionContext::new(&self.units, context.dimension()?))
            .transpose()
            .map_err(Into::into)
    }
}
