// SPDX-License-Identifier: GPL-2.0-only
//! Own the Sweep preview and accept only the certified inputs for the same document generation.
use super::*;
use oblikovati_api::{client::Client, wire::featureargs::Sweep};
use oblikovati_automation::{PartUpdateAccess, PreparedSweep, automatic_sweep, prepare_sweep};
use oblikovati_gui::{GuiSweepAction, GuiSweepPreview, GuiSweepRequest};

pub(in crate::owner) struct SweepPreview {
    scope: GuiSweepRequest,
    result: Result<Option<Sweep>, String>,
}

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in crate::owner) fn gui_sweep_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiSweepPreview> {
        let preview = self.sweep_preview.as_ref()?;
        (preview.scope
            == GuiSweepRequest {
                document,
                generation,
            })
        .then(|| match &preview.result {
            Ok(Some(_)) => GuiSweepPreview::Ready,
            Ok(None) => GuiSweepPreview::MissingInputs,
            Err(cause) => GuiSweepPreview::Failed(cause.clone()),
        })
    }

    pub(in crate::owner) fn sweep_command(
        &mut self,
        action: GuiSweepAction,
    ) -> Result<bool, String> {
        match action {
            GuiSweepAction::Open => Ok(false),
            GuiSweepAction::Clear => {
                Ok(self.sweep_preview.is_some() && self.clear_extrusion_preview())
            }
            GuiSweepAction::Preview(scope) => {
                self.require_sweep_generation(scope)?;
                self.clear_extrusion_preview();
                let result = self.preview_sweep(scope.document);
                self.sweep_preview = Some(SweepPreview { scope, result });
                Ok(true)
            }
            GuiSweepAction::Commit(scope) => {
                self.require_sweep_generation(scope)?;
                let input = self
                    .sweep_preview
                    .as_ref()
                    .filter(|preview| preview.scope == scope)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(Option::as_ref)
                    .ok_or("sweep: expected a current certified preview")?
                    .clone();
                Client::new(self.document_host())
                    .features()
                    .add_feature(&input)
                    .map_err(|cause| cause.to_string())?;
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_sweep_generation(&self, scope: GuiSweepRequest) -> Result<(), String> {
        self.require_browser_document(scope.document)?;
        let session = self
            .files
            .workspace()
            .session(scope.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != scope.generation {
            return Err("sweep: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_sweep(&mut self, document: u64) -> Result<Option<Sweep>, String> {
        let Some((input, prepared)) = self.prepare_automatic_sweep(document)? else {
            return Ok(None);
        };
        self.preview_selected_blend(document, |candidate, ids| prepared.create(candidate, ids))?;
        Ok(Some(input))
    }

    fn prepare_automatic_sweep(
        &mut self,
        document: u64,
    ) -> Result<Option<(Sweep, PreparedSweep)>, String> {
        let policy =
            <crate::part_scene::DesktopPartScenes as PartUpdateAccess<O, W, F>>::resolution_policy(
                &self.part_scenes,
            )
            .ok_or("sweep: expected a certified resolution policy")?;
        let datums = self
            .part_scenes
            .datum_evaluation(&mut self.files, document)?;
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        automatic_sweep(snapshot, &datums, policy)
            .map(|input| {
                prepare_sweep(snapshot, &datums, policy, &input)
                    .map(|prepared| (input, prepared))
                    .map_err(|cause| cause.to_string())
            })
            .transpose()
    }
}
