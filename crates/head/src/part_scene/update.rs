// SPDX-License-Identifier: GPL-2.0-only
//! Integrate shared part replay with the desktop presentation owner.
use super::*;
use oblikovati_api::wire::FeatureError;
use oblikovati_automation::PartUpdateAccess;

impl DesktopPartScenes {
    pub(super) fn requires_update_for_api<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &mut self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<bool, String> {
        self.replay.part_requires_update(files, document)
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
        let errors = self.replay.update_part(files, document)?;
        self.active_replay = Some(document);
        self.invalidate_presentation();
        Ok(errors)
    }
}
