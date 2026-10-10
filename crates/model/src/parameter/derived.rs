// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::wire::DerivedParameterReference;
mod availability;
mod delete;
mod edit;
mod sync;

/// Cross-document resolution is separate from the stored numeric snapshot.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum DerivedTableHealth {
    Unresolved,
    Healthy,
    Unavailable,
    Missing(Vec<String>),
}

/// A link retains source order and stable table identity; the host resolves documents.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct DerivedTable<Id> {
    pub id: i64,
    pub source_document: String,
    pub linked: Vec<String>,
    pub owned_by_feature: bool,
    pub health: DerivedTableHealth,
    produced: BTreeMap<String, Id>,
}

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Resolve local names and source names through the canonical API.
    /// Example: `parameters.derived_references(7)?;`.
    pub fn derived_references(
        &self,
        id: i64,
    ) -> Result<Vec<DerivedParameterReference>, ParameterError<Id>> {
        let table = self.require_derived_table(id)?;
        Ok(table
            .linked
            .iter()
            .filter_map(|name| {
                let parameter = self.get(*table.produced.get(name)?)?;
                Some(DerivedParameterReference {
                    parameter: parameter.name.clone(),
                    source_document: table.source_document.clone(),
                    source_parameter: name.clone(),
                })
            })
            .collect())
    }

    /// Inspect tables in creation order; for example `parameters.derived_tables()`.
    pub fn derived_tables(&self) -> &[DerivedTable<Id>] {
        &self.derived_tables
    }

    /// Find a persisted table identity; for example `parameters.derived_table(7)`.
    pub fn derived_table(&self, id: i64) -> Option<&DerivedTable<Id>> {
        self.derived_tables.iter().find(|table| table.id == id)
    }

    /// Reconnect a persisted table without reading its source document.
    /// Example: `parameters.restore_derived_table(7, "gear.obk", &["module".into()], false)?;`.
    pub fn restore_derived_table(
        &mut self,
        id: i64,
        source: &str,
        linked: &[String],
        owned: bool,
    ) -> Result<(), ParameterError<Id>> {
        self.restore_derived_table_names(id, source, linked, owned, &BTreeMap::new())
    }

    /// Restore explicit local names while retaining the source link names.
    /// Example: `parameters.restore_derived_table_names(7, "gear.opd", &names, false, &aliases)?;`.
    pub fn restore_derived_table_names(
        &mut self,
        id: i64,
        source: &str,
        linked: &[String],
        owned: bool,
        local_names: &BTreeMap<String, String>,
    ) -> Result<(), ParameterError<Id>> {
        if let Some(name) = local_names.keys().find(|name| !linked.contains(name)) {
            return Err(table_error(
                id,
                name,
                "a local-name mapping for a selected source link",
            ));
        }
        if id <= 0 || source.is_empty() || self.derived_table(id).is_some() {
            return Err(table_error(
                id,
                source,
                "a unique positive ID and nonempty source document",
            ));
        }
        let produced = self.reconnect_derived_links(id, linked, local_names)?;
        self.derived_tables.push(DerivedTable {
            id,
            source_document: source.into(),
            linked: linked.to_vec(),
            owned_by_feature: owned,
            health: DerivedTableHealth::Unresolved,
            produced,
        });
        Ok(())
    }

    fn reconnect_derived_links(
        &self,
        table: i64,
        linked: &[String],
        local_names: &BTreeMap<String, String>,
    ) -> Result<BTreeMap<String, Id>, ParameterError<Id>> {
        let mut seen = BTreeSet::new();
        let mut produced = BTreeMap::new();
        for name in linked {
            if name.is_empty() || !seen.insert(name) {
                return Err(table_error(
                    table,
                    name,
                    "unique nonempty linked parameter names",
                ));
            }
            let local = local_names.get(name).unwrap_or(name);
            let resolved = self.find(local);
            if local_names.contains_key(name) && resolved.is_none() {
                return Err(table_error(
                    table,
                    local,
                    "an existing derived parameter for an explicit local name",
                ));
            }
            if let Some(id) = resolved {
                if produced.values().any(|other| *other == id) {
                    return Err(table_error(
                        table,
                        local,
                        "one source link per local derived parameter",
                    ));
                }
                self.require_unowned_derived(table, name, id)?;
                produced.insert(name.clone(), id);
            }
        }
        Ok(produced)
    }

    fn require_unowned_derived(
        &self,
        table: i64,
        name: &str,
        id: Id,
    ) -> Result<(), ParameterError<Id>> {
        let owned = self
            .derived_tables
            .iter()
            .any(|table| table.produced.values().any(|other| *other == id));
        if self.records[&id].state.kind != DERIVED_PARAM || owned {
            return Err(table_error(
                table,
                name,
                "an unowned derived parameter or an absent local name",
            ));
        }
        Ok(())
    }

    fn require_derived_table(&self, id: i64) -> Result<&DerivedTable<Id>, ParameterError<Id>> {
        self.derived_table(id)
            .ok_or_else(|| table_error(id, "missing", "an existing table ID"))
    }

    pub(super) fn detach_derived_parameter(&mut self, id: Id) {
        for table in &mut self.derived_tables {
            if table.produced.values().any(|produced| *produced == id) {
                table.produced.retain(|_, produced| *produced != id);
                table.health = DerivedTableHealth::Unresolved;
            }
        }
    }
}

fn table_error<Id>(id: i64, found: impl ToString, expected: &'static str) -> ParameterError<Id> {
    ParameterError::DerivedTable {
        id,
        found: found.to_string(),
        expected,
    }
}

fn validate_source_quantity<Id>(
    table: i64,
    name: &str,
    quantity: crate::quantity::Quantity,
) -> Result<(), ParameterError<Id>> {
    if !quantity.value.is_finite() || !quantity.dimension.is_arithmetic() {
        return Err(table_error(
            table,
            format!("{name}={quantity:?}"),
            "a finite numeric source quantity",
        ));
    }
    Ok(())
}
