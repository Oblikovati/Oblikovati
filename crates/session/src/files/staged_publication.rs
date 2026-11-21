// SPDX-License-Identifier: GPL-2.0-only
//! Coordinate source and linked-document publication (#4122).
use super::*;
use oblikovati_model::history::{DocumentSnapshot, RecipeReservations};

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Verify the source and all linked reader candidates without publishing them.
    /// Example: `files.verify_named_candidate(id, &candidate)?`.
    pub fn verify_named_candidate(
        &self,
        document: u64,
        candidate: &DocumentSnapshot,
    ) -> Result<(), String> {
        self.workspace
            .verify_unlinked_named_candidate(document, candidate)?;
        let plan = self
            .staged_derived_plan(document)
            .map_err(|error| error.to_string())?;
        for (id, snapshot) in plan
            .evaluate(candidate.clone())
            .map_err(|error| error.to_string())?
        {
            let session = self
                .workspace
                .session(id)
                .map_err(|error| error.to_string())?;
            if session.snapshot().drawing.is_some() {
                return Err(format!(
                    "linked reader {id}: expected a part or assembly document"
                ));
            }
            session.preview_refresh_snapshot(snapshot.clone())?;
            if session.snapshot().assembly.is_none() {
                self.workspace
                    .verify_unlinked_named_candidate(id, &snapshot)?;
            }
        }
        Ok(())
    }

    /// Navigate a source and its linked readers after preparing all named replays.
    /// Example: `files.navigate_named_prepared(id, policy, direction)?`.
    pub fn navigate_named_prepared(
        &mut self,
        document: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        direction: oblikovati_model::history::PreparedNavigation,
    ) -> Result<(), String> {
        let plan = self
            .staged_derived_plan(document)
            .map_err(|error| error.to_string())?;
        let readers = plan.reader_ids();
        self.workspace.run_named_navigation(
            document,
            policy,
            readers,
            direction,
            |_, (), _| Ok(()),
            |snapshot| {
                plan.evaluate(snapshot.clone())
                    .map_err(|error| error.to_string())
            },
        )
    }

    /// Commit a source, its named replay, and all linked readers as one checked edit.
    /// Example: `files.run_named_prepared(id, policy, edit, project)?`.
    pub fn run_named_prepared<R, T>(
        &mut self,
        document: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        edit: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, String>,
        project: impl FnOnce(&DocumentSnapshot, R, u64) -> Result<T, String>,
    ) -> Result<T, String> {
        let plan = self
            .staged_derived_plan(document)
            .map_err(|error| error.to_string())?;
        let readers = plan.reader_ids();
        self.workspace
            .run_named_prepared(document, policy, readers, edit, project, |snapshot| {
                plan.evaluate(snapshot.clone())
                    .map_err(|error| error.to_string())
            })
    }
}
