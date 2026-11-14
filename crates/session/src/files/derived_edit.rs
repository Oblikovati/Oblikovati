// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_model::{
    history::{DocumentSnapshot, RecipeReservations},
    quantity::Quantity,
};

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Link an open source and export its selected numeric user parameters.
    /// Example: `let table = files.add_derived_table(target, "gear.opd", &names)?;`.
    pub fn add_derived_table(
        &mut self,
        target: u64,
        source: &str,
        linked: &[String],
    ) -> Result<i64, FileWorkspaceError> {
        self.edit_derived_links(target, None, source, linked)
    }

    /// Replace selected links, retaining identities and metadata of kept parameters.
    /// Example: `files.set_derived_links(target, table, &names)?;`.
    pub fn set_derived_links(
        &mut self,
        target: u64,
        table: i64,
        linked: &[String],
    ) -> Result<(), FileWorkspaceError> {
        let source = self.derived_table_source(target, table)?;
        self.edit_derived_links(target, Some(table), &source, linked)
            .map(|_| ())
    }

    fn edit_derived_links(
        &mut self,
        target: u64,
        table: Option<i64>,
        source: &str,
        linked: &[String],
    ) -> Result<i64, FileWorkspaceError> {
        let source_id = self.validate_derived_link(target, source)?;
        let values = self
            .derived_source_values(source, target)?
            .expect("validated source");
        self.workspace
            .run_exporting_source(target, source_id, linked, |snapshot, ids| {
                apply_links(snapshot, ids, table, source, linked, &values)
            })
            .map_err(FileWorkspaceError::DerivedRefresh)
    }

    fn validate_derived_link(&self, target: u64, source: &str) -> Result<u64, FileWorkspaceError> {
        let kind = self.document_info(target)?.type_;
        if !matches!(kind.as_str(), "part" | "assembly") {
            return Err(FileWorkspaceError::DerivedRefresh(format!(
                "target document type {kind:?}: expected part or assembly"
            )));
        }
        let source_id = self.resolve_parameter_source(source)?.ok_or_else(|| {
            FileWorkspaceError::DerivedRefresh(format!(
                "source {source:?}: expected an open part or assembly"
            ))
        })?;
        let mut graph = self.derived_graph()?;
        let mut drivers = graph.drivers(target).map_err(link_error)?;
        drivers.push(source_id);
        graph
            .replace_drivers(target, &drivers)
            .map_err(link_error)?;
        Ok(source_id)
    }
}

fn apply_links(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    table: Option<i64>,
    source: &str,
    linked: &[String],
    values: &BTreeMap<String, Quantity>,
) -> Result<i64, String> {
    let id = match table {
        Some(id) => id,
        None => ids
            .allocate_derived_table()
            .map_err(|error| error.to_string())?,
    };
    let mut new_ids = BTreeMap::new();
    for name in linked {
        let kept = table.is_some()
            && snapshot
                .parameters
                .derived_parameter_id(id, name)
                .map_err(|error| error.to_string())?
                .is_some();
        if !kept && !new_ids.contains_key(name) {
            new_ids.insert(
                name.clone(),
                ids.allocate().map_err(|error| error.to_string())?,
            );
        }
    }
    match table {
        Some(_) => snapshot
            .parameters
            .set_derived_table_linked(id, linked, values, &new_ids),
        None => snapshot
            .parameters
            .add_derived_table(id, source, linked, values, &new_ids),
    }
    .map_err(|error| error.to_string())?;
    Ok(id)
}

fn link_error(error: impl std::fmt::Display) -> FileWorkspaceError {
    FileWorkspaceError::DerivedRefresh(error.to_string())
}
