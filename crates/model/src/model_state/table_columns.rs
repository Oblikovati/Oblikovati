// SPDX-License-Identifier: GPL-2.0-only
//! One ordered namespace for property, suppression and parameter table columns.
use super::ModelStateMembers;
use oblikovati_api::types::ModelStatePropertyKey;
use serde::{Deserialize, Serialize};

/// A stable identity for one visible model-state table column.
/// Example: `ModelStateTableColumn::Suppression(7)` refers to feature 7.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum ModelStateTableColumn {
    Property(ModelStatePropertyKey),
    Suppression(u64),
    Parameter(u64),
}
impl Eq for ModelStateTableColumn {}
impl ModelStateMembers {
    /// Inspect visible columns in grid order; example: `members.table_columns()`.
    pub fn table_columns(&self) -> &[ModelStateTableColumn] {
        &self.table_columns
    }
    /// Replace the visible grid order after validating every referenced owner.
    /// Example: `members.set_table_columns(columns)?` retains hidden values.
    pub fn set_table_columns(&mut self, columns: Vec<ModelStateTableColumn>) -> Result<(), String> {
        self.validate_table_columns(&columns)?;
        self.table_columns = columns;
        Ok(())
    }
    pub(super) fn validate_table_columns(
        &self,
        columns: &[ModelStateTableColumn],
    ) -> Result<(), String> {
        for (index, column) in columns.iter().enumerate() {
            let known = match column {
                ModelStateTableColumn::Property(key) => self.property_table.base(key).is_some(),
                ModelStateTableColumn::Suppression(feature) => *feature != 0,
                ModelStateTableColumn::Parameter(parameter) => self
                    .parameter_table
                    .bases()
                    .iter()
                    .any(|base| base.parameter == *parameter),
            };
            if !known || columns[..index].contains(column) {
                return Err(format!(
                    "model-state column {column:?}: expected a unique existing owner and retained base"
                ));
            }
        }
        Ok(())
    }
}
