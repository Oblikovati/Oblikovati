// SPDX-License-Identifier: GPL-2.0-only
//! Preview and commit the first section's condition as a private Loft edit.
use super::*;
use oblikovati_automation::replace_loft_takeoff;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn change_loft_takeoff(
        &mut self,
        document: u64,
        condition: Option<LoftTakeoffCondition>,
    ) -> Result<bool, String> {
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        let Some(draft) = self.loft_selection.as_ref().filter(|draft| {
            draft.document == document
                && draft.generation == generation
                && draft.can_change_takeoff()
                && draft.takeoff != condition
        }) else {
            return Ok(false);
        };
        let feature = draft.feature.expect("takeoff belongs to an existing Loft");
        let preview = self.preview_loft_takeoff(document, feature, condition);
        if let Some(draft) = &mut self.loft_selection {
            draft.takeoff = condition;
            draft.preview = Some(preview);
        }
        Ok(true)
    }

    pub(super) fn preview_loft_takeoff(
        &mut self,
        document: u64,
        feature: FeatureId,
        condition: Option<LoftTakeoffCondition>,
    ) -> Result<(), String> {
        self.preview_selected_blend(document, |candidate, _| {
            replace_loft_takeoff(candidate, feature, condition)
        })?;
        Ok(())
    }

    pub(in crate::owner) fn gui_loft_takeoff_dirty(&self, document: u64, generation: u64) -> bool {
        self.loft_selection.as_ref().is_some_and(|draft| {
            draft.document == document && draft.generation == generation && draft.takeoff_dirty()
        })
    }

    pub(super) fn commit_loft_takeoff(
        &mut self,
        document: u64,
        generation: u64,
        feature: FeatureId,
        condition: Option<LoftTakeoffCondition>,
    ) -> Result<bool, String> {
        self.commit_loft_edit(document, generation, |candidate, _| {
            replace_loft_takeoff(candidate, feature, condition)
        })
    }
}

impl LoftSelection {
    fn can_change_takeoff(&self) -> bool {
        self.feature.is_some()
            && self.output == SketchLoftOutput::Surface
            && self.path_mode == GuiLoftPathMode::Rails
    }

    pub(super) fn takeoff_dirty(&self) -> bool {
        self.can_change_takeoff() && self.takeoff != self.saved_takeoff
    }
}
