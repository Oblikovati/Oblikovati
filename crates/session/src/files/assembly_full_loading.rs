// SPDX-License-Identifier: GPL-2.0-only
//! Finish express loading only after the full reference graph is present.
use super::*;
use std::collections::{BTreeSet, VecDeque};

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Switch an express assembly to full mode after every component is attached.
    /// Example: `files.finish_express_assembly(id)?` after resolving its references.
    pub fn finish_express_assembly(&mut self, id: u64) -> Result<bool, FileWorkspaceError> {
        if self.retained_file(id)?.express_view.is_none() {
            return Ok(false);
        }
        self.require_loaded_assembly_graph(id)?;
        self.retained
            .get_mut(&id)
            .expect("validated file")
            .express_view = None;
        Ok(true)
    }

    fn require_loaded_assembly_graph(&self, root: u64) -> Result<(), FileWorkspaceError> {
        let mut seen = BTreeSet::new();
        let mut pending = VecDeque::from([root]);
        while let Some(owner) = pending.pop_front() {
            if !seen.insert(owner) {
                continue;
            }
            self.collect_loaded_components(owner, &mut pending)?;
        }
        Ok(())
    }

    fn collect_loaded_components(
        &self,
        owner: u64,
        pending: &mut VecDeque<u64>,
    ) -> Result<(), FileWorkspaceError> {
        let session = self
            .workspace
            .session(owner)
            .map_err(FileWorkspaceError::Workspace)?;
        let Some(occurrences) = &session.snapshot().assembly else {
            return Ok(());
        };
        for name in oblikovati_model::assembly::component_references(occurrences) {
            let child = self.document_reference(owner, name).ok_or_else(|| {
                FileWorkspaceError::ReferenceGraph(format!(
                    "express assembly {owner} reference {name:?}: expected a fully loaded component"
                ))
            })?;
            self.retained_file(child)?.require_full_content(child)?;
            pending.push_back(child);
        }
        Ok(())
    }
}
