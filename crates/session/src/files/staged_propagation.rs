// SPDX-License-Identifier: GPL-2.0-only
//! Prepare linked parameter snapshots without changing live documents (#4122).

use super::*;
use oblikovati_model::history::DocumentSnapshot;

struct StagedLink {
    table: i64,
    name: String,
    source: Option<u64>,
}
struct StagedReader {
    document: u64,
    snapshot: DocumentSnapshot,
    options: oblikovati_solve::nonlinear::SolveOptions,
    links: Vec<StagedLink>,
}
pub(super) struct StagedDerivedPlan {
    source: u64,
    inputs: BTreeMap<u64, DocumentSnapshot>,
    readers: Vec<StagedReader>,
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    pub(super) fn staged_derived_plan(
        &self,
        source: u64,
    ) -> Result<StagedDerivedPlan, FileWorkspaceError> {
        let graph = self.derived_graph()?;
        let order = graph
            .recompute_order(source)
            .map_err(|error| FileWorkspaceError::DerivedRefresh(error.to_string()))?;
        let mut inputs = BTreeMap::new();
        for &id in self.workspace.documents() {
            let session = self
                .workspace
                .session(id)
                .map_err(FileWorkspaceError::Workspace)?;
            inputs.insert(id, session.snapshot().clone());
        }
        let readers = order
            .into_iter()
            .filter(|id| *id != source)
            .map(|id| self.staged_reader(id))
            .collect::<Result<_, _>>()?;
        Ok(StagedDerivedPlan {
            source,
            inputs,
            readers,
        })
    }

    fn staged_reader(&self, id: u64) -> Result<StagedReader, FileWorkspaceError> {
        let session = self
            .workspace
            .session(id)
            .map_err(FileWorkspaceError::Workspace)?;
        if session.in_transaction() {
            return Err(FileWorkspaceError::DerivedRefresh(format!(
                "derived document {id}: expected a closed transaction before staged refresh"
            )));
        }
        let links = session
            .snapshot()
            .parameters
            .derived_tables()
            .iter()
            .map(|table| self.staged_link(id, table.id, &table.source_document))
            .collect::<Result<_, _>>()?;
        Ok(StagedReader {
            document: id,
            snapshot: session.snapshot().clone(),
            options: session.sketch_options,
            links,
        })
    }

    fn staged_link(
        &self,
        target: u64,
        table: i64,
        name: &str,
    ) -> Result<StagedLink, FileWorkspaceError> {
        let source = self.resolve_parameter_source(name)?;
        if source == Some(target) {
            return Err(FileWorkspaceError::DerivedRefresh(format!(
                "source {name:?} resolves to target {target}: expected a different document"
            )));
        }
        if let Some(id) = source {
            let session = self
                .workspace
                .session(id)
                .map_err(FileWorkspaceError::Workspace)?;
            if session.in_transaction() {
                return Err(FileWorkspaceError::DerivedRefresh(format!(
                    "source {name:?} in document {id}: expected a closed transaction before staged refresh"
                )));
            }
        }
        Ok(StagedLink {
            table,
            name: name.into(),
            source,
        })
    }

    #[cfg(test)]
    fn staged_derived_snapshots(
        &self,
        source: u64,
        candidate: DocumentSnapshot,
    ) -> Result<BTreeMap<u64, DocumentSnapshot>, FileWorkspaceError> {
        self.staged_derived_plan(source)?.evaluate(candidate)
    }
}

impl StagedDerivedPlan {
    pub(super) fn reader_ids(&self) -> Vec<u64> {
        self.readers.iter().map(|reader| reader.document).collect()
    }
    pub(super) fn evaluate(
        mut self,
        candidate: DocumentSnapshot,
    ) -> Result<BTreeMap<u64, DocumentSnapshot>, FileWorkspaceError> {
        self.inputs.insert(self.source, candidate);
        let mut staged = BTreeMap::new();
        for reader in self.readers {
            let candidate = reader.evaluate(&self.inputs)?;
            self.inputs.insert(reader.document, candidate.clone());
            staged.insert(reader.document, candidate);
        }
        Ok(staged)
    }
}
impl StagedReader {
    fn evaluate(
        &self,
        inputs: &BTreeMap<u64, DocumentSnapshot>,
    ) -> Result<DocumentSnapshot, FileWorkspaceError> {
        let mut candidate = self.snapshot.clone();
        for link in &self.links {
            let values = link
                .source
                .map(|id| self.source_values(inputs, id, &link.name, &candidate))
                .transpose()?;
            crate::refresh::refresh_source_snapshot(
                &mut candidate,
                link.table,
                values.as_ref(),
                self.options,
            )
            .map_err(|error| FileWorkspaceError::DerivedRefresh(error.to_string()))?;
        }
        Ok(candidate)
    }
    fn source_values(
        &self,
        inputs: &BTreeMap<u64, DocumentSnapshot>,
        source: u64,
        name: &str,
        target: &DocumentSnapshot,
    ) -> Result<BTreeMap<String, oblikovati_model::quantity::Quantity>, FileWorkspaceError> {
        let snapshot = inputs.get(&source).ok_or_else(|| {
            FileWorkspaceError::DerivedRefresh(format!(
                "source {name:?} in document {source}: expected a retained source snapshot"
            ))
        })?;
        super::source::source_values(name, &snapshot.parameters, target.parameters.units())
    }
}

#[cfg(test)]
#[path = "staged_propagation_tests.rs"]
mod tests;
