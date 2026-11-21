// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::quantity::Quantity;
mod validate;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Create a table with identities allocated by its document.
    /// Example: `parameters.add_derived_table(7, "gear.obk", &names, &source, &new_ids)?;`.
    pub fn add_derived_table(
        &mut self,
        id: i64,
        document: &str,
        linked: &[String],
        source: &BTreeMap<String, Quantity>,
        new_ids: &BTreeMap<String, Id>,
    ) -> Result<(), ParameterError<Id>> {
        let mut candidate = self.clone();
        candidate.restore_derived_table(id, document, &[], false)?;
        candidate.replace_derived_subset(id, linked, source, new_ids)?;
        *self = candidate;
        Ok(())
    }

    /// Replace a subset while retaining kept identities, names, and metadata.
    /// Example: `parameters.set_derived_table_linked(7, &names, &source, &new_ids)?;`.
    pub fn set_derived_table_linked(
        &mut self,
        id: i64,
        linked: &[String],
        source: &BTreeMap<String, Quantity>,
        new_ids: &BTreeMap<String, Id>,
    ) -> Result<(), ParameterError<Id>> {
        let mut candidate = self.clone();
        candidate.replace_derived_subset(id, linked, source, new_ids)?;
        *self = candidate;
        Ok(())
    }

    /// Inspect a link's local identity even after a local rename.
    /// Example: `let local = parameters.derived_parameter_id(7, "module")?;`.
    pub fn derived_parameter_id(
        &self,
        table: i64,
        name: &str,
    ) -> Result<Option<Id>, ParameterError<Id>> {
        Ok(self
            .require_derived_table(table)?
            .produced
            .get(name)
            .copied())
    }

    fn replace_derived_subset(
        &mut self,
        id: i64,
        linked: &[String],
        source: &BTreeMap<String, Quantity>,
        new_ids: &BTreeMap<String, Id>,
    ) -> Result<(), ParameterError<Id>> {
        let mut table = self.require_derived_table(id)?.clone();
        self.validate_derived_subset(&table, linked, source, new_ids)?;
        self.remove_unlinked_parameters(&mut table, linked)?;
        self.produce_linked_parameters(&mut table, linked, source, new_ids)?;
        table.linked = linked.to_vec();
        table.health = DerivedTableHealth::Healthy;
        *self
            .derived_tables
            .iter_mut()
            .find(|table| table.id == id)
            .expect("validated table") = table;
        Ok(())
    }

    fn remove_unlinked_parameters(
        &mut self,
        table: &mut DerivedTable<Id>,
        linked: &[String],
    ) -> Result<(), ParameterError<Id>> {
        // Removal follows source order, never local IDs or map order.
        for name in table.linked.clone() {
            if !linked.contains(&name)
                && let Some(id) = table.produced.remove(&name)
            {
                self.remove_for_owner(id)?;
            }
        }
        Ok(())
    }

    fn produce_linked_parameters(
        &mut self,
        table: &mut DerivedTable<Id>,
        linked: &[String],
        source: &BTreeMap<String, Quantity>,
        new_ids: &BTreeMap<String, Id>,
    ) -> Result<(), ParameterError<Id>> {
        for name in linked {
            if let Some(id) = table.produced.get(name) {
                self.update_measured(*id, source[name])?;
                continue;
            }
            let id = new_ids[name];
            self.insert_value(id, name, DERIVED_PARAM, source[name])?;
            table.produced.insert(name.clone(), id);
        }
        Ok(())
    }
}
