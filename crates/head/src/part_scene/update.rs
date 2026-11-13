// SPDX-License-Identifier: GPL-2.0-only
//! Integrate shared part replay with the desktop presentation owner.
use super::*;
use oblikovati_api::wire::FeatureError;

impl DesktopPartScenes {
    pub(super) fn requires_update_for_api<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<bool, String> {
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        Ok(oblikovati_automation::part_replay_is_stale(
            session,
            Some(&self.slot),
        ))
    }
    pub(super) fn recompute_for_api<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Vec<FeatureError>, String> {
        if self
            .slot
            .latest()
            .is_some_and(|view| view.document != document)
        {
            self.slot.clear();
        }
        self.adopt_current(files, document)?;
        self.invalidate_presentation();
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        oblikovati_automation::part_feature_errors(
            session.snapshot(),
            self.slot
                .current(session)
                .map_err(|error| error.to_string())?
                .health(),
        )
    }
}
