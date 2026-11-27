// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    pub(super) fn validate_derived_subset(
        &self,
        table: &DerivedTable<Id>,
        linked: &[String],
        source: &BTreeMap<String, Quantity>,
        new_ids: &BTreeMap<String, Id>,
    ) -> Result<(), ParameterError<Id>> {
        let mut selected = BTreeSet::new();
        for name in linked {
            if name.is_empty() || !selected.insert(name) {
                return Err(table_error(table.id, name, "unique nonempty linked names"));
            }
            let value = source.get(name).ok_or_else(|| {
                table_error(table.id, name, "an available numeric source parameter")
            })?;
            validate_source_quantity(table.id, name, *value)?;
        }
        self.validate_new_link_ids(table, &selected, new_ids)
    }

    fn validate_new_link_ids(
        &self,
        table: &DerivedTable<Id>,
        selected: &BTreeSet<&String>,
        new_ids: &BTreeMap<String, Id>,
    ) -> Result<(), ParameterError<Id>> {
        let needed: BTreeSet<&String> = selected
            .iter()
            .copied()
            .filter(|name| !table.produced.contains_key(*name))
            .collect();
        if needed != new_ids.keys().collect() {
            return Err(table_error(
                table.id,
                format!("{:?}", new_ids.keys()),
                "exactly one allocated ID for each new or unproduced selected link",
            ));
        }
        let mut unique_ids = BTreeSet::new();
        for (name, id) in new_ids {
            if self.records.contains_key(id) || !unique_ids.insert(id) {
                return Err(table_error(
                    table.id,
                    format!("{name}={id:?}"),
                    "unique unused parameter IDs",
                ));
            }
            let removed_owner = self.find(name).filter(|owner| {
                table
                    .produced
                    .iter()
                    .any(|(source, id)| id == owner && !selected.contains(source))
            });
            self.require_available_name(name, removed_owner)?;
        }
        Ok(())
    }
}
