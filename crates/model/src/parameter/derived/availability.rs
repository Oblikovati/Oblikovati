// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Restore observed link failure without treating saved values as current source values.
    /// Example: `parameters.restore_derived_failure(table, None)?` for an unavailable source.
    pub fn restore_derived_failure(
        &mut self,
        id: i64,
        missing: Option<&[String]>,
    ) -> Result<(), ParameterError<Id>> {
        let table = self.require_derived_table(id)?.clone();
        let selected = missing.unwrap_or(&table.linked);
        let unique: BTreeSet<_> = selected.iter().collect();
        if (missing.is_some() && selected.is_empty())
            || unique.len() != selected.len()
            || selected.iter().any(|name| !table.linked.contains(name))
        {
            return Err(table_error(
                id,
                format!("{selected:?}"),
                "unique selected source names for a nonempty missing subset",
            ));
        }
        let mut candidate = self.clone();
        for name in selected {
            candidate.refresh_derived_link(&table, name, None)?;
        }
        let health = match missing {
            None => DerivedTableHealth::Unavailable,
            Some(_) => DerivedTableHealth::Missing(
                table
                    .linked
                    .iter()
                    .filter(|name| unique.contains(name))
                    .cloned()
                    .collect(),
            ),
        };
        candidate
            .derived_tables
            .iter_mut()
            .find(|table| table.id == id)
            .expect("validated table")
            .health = health;
        *self = candidate;
        Ok(())
    }
}
