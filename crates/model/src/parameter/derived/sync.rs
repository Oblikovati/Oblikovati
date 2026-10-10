// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::quantity::Quantity;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Refresh caller-resolved values atomically; `None` means the source is unavailable.
    /// Example: `parameters.sync_derived_table(7, Some(&source_values))?;`.
    pub fn sync_derived_table(
        &mut self,
        id: i64,
        source: Option<&BTreeMap<String, Quantity>>,
    ) -> Result<(), ParameterError<Id>> {
        let mut candidate = self.clone();
        candidate.refresh_derived_table(id, source)?;
        *self = candidate;
        Ok(())
    }

    fn refresh_derived_table(
        &mut self,
        id: i64,
        source: Option<&BTreeMap<String, Quantity>>,
    ) -> Result<(), ParameterError<Id>> {
        let table = self.require_derived_table(id)?.clone();
        self.fail_missing_links(&table, source)?;
        let missing = self.refresh_available_links(&table, source)?;
        self.derived_tables
            .iter_mut()
            .find(|table| table.id == id)
            .expect("validated table")
            .health = match source {
            None => DerivedTableHealth::Unavailable,
            Some(_) if missing.is_empty() => DerivedTableHealth::Healthy,
            Some(_) => DerivedTableHealth::Missing(missing),
        };
        Ok(())
    }

    fn fail_missing_links(
        &mut self,
        table: &DerivedTable<Id>,
        source: Option<&BTreeMap<String, Quantity>>,
    ) -> Result<(), ParameterError<Id>> {
        // Fail all absent drivers before any updated driver can overwrite a reader's last-good value.
        for name in &table.linked {
            if source.and_then(|source| source.get(name)).is_none() {
                self.refresh_derived_link(table, name, None)?;
            }
        }
        Ok(())
    }

    fn refresh_available_links(
        &mut self,
        table: &DerivedTable<Id>,
        source: Option<&BTreeMap<String, Quantity>>,
    ) -> Result<Vec<String>, ParameterError<Id>> {
        let mut missing = Vec::new();
        for name in &table.linked {
            let value = source.and_then(|source| source.get(name));
            if value.is_none() || !self.refresh_derived_link(table, name, value)? {
                missing.push(name.clone());
            }
        }
        Ok(missing)
    }

    pub(super) fn refresh_derived_link(
        &mut self,
        table: &DerivedTable<Id>,
        name: &str,
        value: Option<&Quantity>,
    ) -> Result<bool, ParameterError<Id>> {
        let Some(id) = table.produced.get(name).copied() else {
            return Ok(false);
        };
        if let Some(value) = value {
            validate_source_quantity(table.id, name, *value)?;
            self.update_measured(id, *value)?;
            return Ok(true);
        }
        self.records.get_mut(&id).expect("owned link").state.health =
            ParameterHealth::Failed(ParameterFailure::ExternalLink {
                source_document: table.source_document.clone(),
                source_parameter: name.into(),
            });
        self.recompute(id)?;
        Ok(false)
    }
}
