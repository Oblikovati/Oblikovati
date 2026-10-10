// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Transfer table ownership to a derived component.
    /// Example: `parameters.mark_derived_table_owned(7)?;`.
    pub fn mark_derived_table_owned(&mut self, id: i64) -> Result<(), ParameterError<Id>> {
        self.require_derived_table(id)?;
        self.derived_tables
            .iter_mut()
            .find(|table| table.id == id)
            .expect("validated table")
            .owned_by_feature = true;
        Ok(())
    }

    /// Delete a user-owned table and invalidate readers of its removed parameters.
    /// Example: `parameters.delete_derived_table(7)?;`.
    pub fn delete_derived_table(&mut self, id: i64) -> Result<(), ParameterError<Id>> {
        if self.require_derived_table(id)?.owned_by_feature {
            return Err(table_error(
                id,
                "feature-owned table",
                "deletion through the owning component",
            ));
        }
        self.delete_derived_table_for_owner(id)
    }

    /// Delete a table as part of removing its owning component.
    /// Example: `parameters.delete_derived_table_for_owner(7)?;`.
    pub fn delete_derived_table_for_owner(&mut self, id: i64) -> Result<(), ParameterError<Id>> {
        let table = self.require_derived_table(id)?.clone();
        let mut candidate = self.clone();
        for name in &table.linked {
            if let Some(parameter) = table.produced.get(name) {
                candidate.remove_for_owner(*parameter)?;
            }
        }
        candidate.derived_tables.retain(|table| table.id != id);
        *self = candidate;
        Ok(())
    }
}
