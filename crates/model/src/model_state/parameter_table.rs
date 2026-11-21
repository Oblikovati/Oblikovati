// SPDX-License-Identifier: GPL-2.0-only
//! Inherited parameter expressions and per-member overrides for model-state tables.
use serde::{Deserialize, Serialize};

/// An inherited expression retained while any model-state row may override it.
/// Example: `ModelStateParameterBase { parameter: 4, expression: "1 in".into() }`.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ModelStateParameterBase {
    pub parameter: u64,
    pub expression: String,
}

/// One member's authored parameter expression.
/// Example: `ModelStateParameterValue { member: 1, parameter: 4, expression: "2 in".into() }`.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ModelStateParameterValue {
    pub member: u64,
    pub parameter: u64,
    pub expression: String,
}

/// Ordered displayed parameters and retained inherited and authored expressions.
/// Example: `ModelStateParameterTable::default()` creates an empty table.
#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ModelStateParameterTable {
    bases: Vec<ModelStateParameterBase>,
    values: Vec<ModelStateParameterValue>,
}
impl ModelStateParameterTable {
    /// Make a candidate that its members will validate; example: `ModelStateParameterTable::new(bases, values)`.
    pub fn new(bases: Vec<ModelStateParameterBase>, values: Vec<ModelStateParameterValue>) -> Self {
        Self { bases, values }
    }
    /// Read inherited expressions, including hidden columns; example: `table.bases()`.
    pub fn bases(&self) -> &[ModelStateParameterBase] {
        &self.bases
    }
    /// Read authored member expressions; example: `table.values()`.
    pub fn values(&self) -> &[ModelStateParameterValue] {
        &self.values
    }
    /// Resolve a member's expression; example: `table.expression(1, 4)`.
    pub fn expression(&self, member: u64, parameter: u64) -> Option<&str> {
        self.values
            .iter()
            .find(|value| value.member == member && value.parameter == parameter)
            .map(|value| value.expression.as_str())
            .or_else(|| {
                self.bases
                    .iter()
                    .find(|base| base.parameter == parameter)
                    .map(|base| base.expression.as_str())
            })
    }
    pub(super) fn is_empty(&self) -> bool {
        self.bases.is_empty() && self.values.is_empty()
    }
    pub(super) fn validate(&self, members: &[super::ModelStateMember]) -> Result<(), String> {
        self.validate_bases()?;
        self.validate_values(members)
    }
    fn validate_bases(&self) -> Result<(), String> {
        for (index, base) in self.bases.iter().enumerate() {
            validate_expression(base.parameter, &base.expression)?;
            if self.bases[..index]
                .iter()
                .any(|previous| previous.parameter == base.parameter)
            {
                return Err(format!(
                    "parameter base {base:?}: expected a unique parameter"
                ));
            }
        }
        Ok(())
    }
    fn validate_values(&self, members: &[super::ModelStateMember]) -> Result<(), String> {
        for (index, value) in self.values.iter().enumerate() {
            validate_expression(value.parameter, &value.expression)?;
            let valid_owner = members.iter().any(|member| member.id == value.member)
                && self
                    .bases
                    .iter()
                    .any(|base| base.parameter == value.parameter);
            let repeated = self.values[..index].iter().any(|previous| {
                previous.member == value.member && previous.parameter == value.parameter
            });
            if !valid_owner || repeated {
                return Err(format!(
                    "parameter value {value:?}: expected a unique existing member and inherited parameter"
                ));
            }
        }
        Ok(())
    }
    pub(super) fn clone_member(&mut self, source: u64, target: u64) {
        self.forget_member(target);
        let copied: Vec<_> = self
            .values
            .iter()
            .filter(|value| value.member == source)
            .cloned()
            .map(|mut value| {
                value.member = target;
                value
            })
            .collect();
        self.values.extend(copied);
    }
    pub(super) fn forget_member(&mut self, member: u64) {
        self.values.retain(|value| value.member != member);
    }
}
fn validate_expression(parameter: u64, expression: &str) -> Result<(), String> {
    if parameter == 0 || expression.trim().is_empty() || expression.contains('\0') {
        return Err(format!(
            "parameter {parameter} expression {expression:?}: expected a nonzero ID and nonempty expression without NUL"
        ));
    }
    Ok(())
}
