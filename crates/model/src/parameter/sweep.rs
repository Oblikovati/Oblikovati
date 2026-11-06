// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::{ModelValueType, Tolerance};

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Select all nondefault tolerances atomically; for example `parameters.set_all_model_value_type(MODEL_VALUE_UPPER)`.
    /// Returns the eligible count, including parameters already at that selection, as in Go.
    pub fn set_all_model_value_type(
        &mut self,
        selection: ModelValueType,
    ) -> Result<usize, ParameterError<Id>> {
        tolerance::require_selection(selection)?;
        let eligible: Vec<Id> = self
            .order
            .iter()
            .copied()
            .filter(|id| self.records[id].state.tolerance != Tolerance::default())
            .collect();
        // #3581: validate the complete sweep before changing any selection or dirty flag.
        for &id in &eligible {
            self.require_numeric_value(id)?;
        }
        for &id in &eligible {
            self.store_model_selection(id, selection);
        }
        Ok(eligible.len())
    }
}
