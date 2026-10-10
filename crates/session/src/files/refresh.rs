// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Resolve and refresh one retained table without changing focus or target history.
    /// Example: `host.refresh_derived_table(target_id, table_id)?` after a source change.
    pub fn refresh_derived_table(
        &mut self,
        target: u64,
        table: i64,
    ) -> Result<bool, FileWorkspaceError> {
        let source = self.derived_table_source(target, table)?;
        if self.resolve_parameter_source(&source)? == Some(target) {
            return Err(FileWorkspaceError::DerivedRefresh(format!(
                "source {source:?} resolves to target {target}: expected a different document"
            )));
        }
        self.require_committed_source(&source)?;
        let values = self.derived_source_values(&source, target)?;
        self.workspace
            .commands(target)
            .map_err(FileWorkspaceError::Workspace)?
            .refresh_derived_table(table, values.as_ref())
            .map_err(|error| FileWorkspaceError::DerivedRefresh(error.to_string()))
    }

    fn require_committed_source(&self, name: &str) -> Result<(), FileWorkspaceError> {
        let Some(id) = self.resolve_parameter_source(name)? else {
            return Ok(());
        };
        let session = self
            .workspace
            .session(id)
            .map_err(FileWorkspaceError::Workspace)?;
        if session.in_transaction() {
            return Err(FileWorkspaceError::DerivedRefresh(format!(
                "source {name:?} in document {id}: expected a closed transaction before refresh"
            )));
        }
        Ok(())
    }

    pub(super) fn derived_table_source(
        &self,
        target: u64,
        table: i64,
    ) -> Result<String, FileWorkspaceError> {
        let session = self
            .workspace
            .session(target)
            .map_err(FileWorkspaceError::Workspace)?;
        session
            .snapshot()
            .parameters
            .derived_table(table)
            .map(|entry| entry.source_document.clone())
            .ok_or_else(|| {
                FileWorkspaceError::DerivedRefresh(format!(
                    "table {table} in document {target}: expected an existing derived table"
                ))
            })
    }
}
