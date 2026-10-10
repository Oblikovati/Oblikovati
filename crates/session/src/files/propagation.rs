// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_model::dependency::DependencyGraph;
use std::collections::BTreeSet;

/// Partial refresh results; changed and failed may both contain a document with several tables.
#[derive(Debug, Default, PartialEq, Eq)]
pub struct DerivedRefreshReport {
    pub changed: Vec<u64>,
    pub failures: Vec<DerivedRefreshFailure>,
    pub blocked: Vec<BlockedDerivedDocument>,
}

/// A failed table leaves its previous snapshot intact and blocks downstream documents.
#[derive(Debug, PartialEq, Eq)]
pub struct DerivedRefreshFailure {
    pub document: u64,
    pub table: i64,
    pub reason: String,
}

/// A reader skipped because at least one direct source failed or was blocked.
#[derive(Debug, PartialEq, Eq)]
pub struct BlockedDerivedDocument {
    pub document: u64,
    pub sources: Vec<u64>,
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Refresh a changed document and its readers in dependency order, without target undo steps.
    /// Example: `let report = host.refresh_derived_from(source_id)?;` after a committed source edit.
    pub fn refresh_derived_from(
        &mut self,
        source: u64,
    ) -> Result<DerivedRefreshReport, FileWorkspaceError> {
        let session = self
            .workspace
            .session(source)
            .map_err(FileWorkspaceError::Workspace)?;
        if session.in_transaction() {
            return Err(FileWorkspaceError::DerivedRefresh(format!(
                "source document {source}: expected a closed transaction before propagation"
            )));
        }
        let graph = self.derived_graph()?;
        let order = graph.recompute_order(source).map_err(graph_error)?;
        self.refresh_order(&graph, order)
    }

    /// Refresh all open links after sources are opened, closed, or rebound.
    /// Example: `let report = host.refresh_all_derived()?;` after closing a source.
    pub fn refresh_all_derived(&mut self) -> Result<DerivedRefreshReport, FileWorkspaceError> {
        let graph = self.derived_graph()?;
        let order = graph.recompute_all();
        self.refresh_order(&graph, order)
    }

    fn derived_graph(&self) -> Result<DependencyGraph<u64>, FileWorkspaceError> {
        let mut graph = DependencyGraph::default();
        for &id in self.workspace.documents() {
            graph.insert(id).map_err(graph_error)?;
        }
        for &id in self.workspace.documents() {
            graph
                .replace_drivers(id, &self.derived_drivers(id)?)
                .map_err(graph_error)?;
        }
        Ok(graph)
    }

    fn derived_drivers(&self, id: u64) -> Result<Vec<u64>, FileWorkspaceError> {
        let session = self
            .workspace
            .session(id)
            .map_err(FileWorkspaceError::Workspace)?;
        session
            .snapshot()
            .parameters
            .derived_tables()
            .iter()
            .map(|table| self.resolve_parameter_source(&table.source_document))
            .collect::<Result<Vec<_>, _>>()
            .map(|sources| sources.into_iter().flatten().collect())
    }

    fn refresh_order(
        &mut self,
        graph: &DependencyGraph<u64>,
        order: Vec<u64>,
    ) -> Result<DerivedRefreshReport, FileWorkspaceError> {
        let mut report = DerivedRefreshReport::default();
        let mut failed = BTreeSet::new();
        for id in order {
            let sources: Vec<_> = graph
                .drivers(id)
                .map_err(graph_error)?
                .into_iter()
                .filter(|source| failed.contains(source))
                .collect();
            if !sources.is_empty() {
                report.blocked.push(BlockedDerivedDocument {
                    document: id,
                    sources,
                });
                failed.insert(id);
                continue;
            }
            if !self.refresh_document(id, &mut report) {
                failed.insert(id);
            }
        }
        Ok(report)
    }

    fn refresh_document(&mut self, id: u64, report: &mut DerivedRefreshReport) -> bool {
        let tables: Vec<_> = self
            .workspace
            .session(id)
            .expect("planned owner")
            .snapshot()
            .parameters
            .derived_tables()
            .iter()
            .map(|table| table.id)
            .collect();
        let before = report.failures.len();
        let mut changed = false;
        for table in tables {
            match self.refresh_derived_table(id, table) {
                Ok(updated) => changed |= updated,
                Err(error) => report.failures.push(DerivedRefreshFailure {
                    document: id,
                    table,
                    reason: error.to_string(),
                }),
            }
        }
        if changed {
            report.changed.push(id);
        }
        before == report.failures.len()
    }
}

fn graph_error(error: oblikovati_model::dependency::DependencyError<u64>) -> FileWorkspaceError {
    FileWorkspaceError::DerivedRefresh(error.to_string())
}
