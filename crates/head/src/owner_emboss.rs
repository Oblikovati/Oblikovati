// SPDX-License-Identifier: GPL-2.0-only
//! Preview, edit seeds, and checkpointed commit for the desktop Emboss Properties panel.
#[path = "owner_emboss_helpers.rs"]
mod helpers;
pub(in crate::owner) use helpers::edit_seeds;

use super::*;
use helpers::{
    emboss_definition, emboss_inputs, has_open_visible_sketch, is_range_error, planes_coplanar,
    replace_emboss,
};
use oblikovati_automation::{
    PartUpdateAccess, PreparedEmboss, build_emboss_definition, prepare_emboss_definition,
    sweep_boolean_target,
};
use oblikovati_gui::{GuiEmbossAction, GuiEmbossPreview, GuiEmbossPreviewState, GuiEmbossRequest};
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, certified_sketch_regions_with_datums};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchPlacement},
    feature::{
        FeatureId, sketch_emboss::SketchEmbossDefinition, sketch_prism::SketchPrismSelection,
        work_geometry::WorkGeometryEvaluation,
    },
};

#[derive(Clone)]
enum EmbossCommit {
    Create(Box<PreparedEmboss>),
    Edit(FeatureId, Box<SketchEmbossDefinition>),
}

enum EmbossOutcome {
    Ready(EmbossCommit),
    OpenLoop(String),
    MissingProfile,
    MissingWrapFace,
    RangeError(String),
}

pub(in crate::owner) struct EmbossPreview {
    request: GuiEmbossRequest,
    result: Result<EmbossOutcome, String>,
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
    pub(in crate::owner) fn gui_emboss_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiEmbossPreviewState> {
        let preview = self.emboss_preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiEmbossPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(EmbossOutcome::Ready(_)) => GuiEmbossPreview::Ready,
                    Ok(EmbossOutcome::OpenLoop(msg)) => GuiEmbossPreview::OpenLoop(msg.clone()),
                    Ok(EmbossOutcome::MissingProfile) => GuiEmbossPreview::MissingProfile,
                    Ok(EmbossOutcome::MissingWrapFace) => GuiEmbossPreview::MissingWrapFace,
                    Ok(EmbossOutcome::RangeError(msg)) => GuiEmbossPreview::RangeError(msg.clone()),
                    Err(cause) => GuiEmbossPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn emboss_command(
        &mut self,
        action: GuiEmbossAction,
    ) -> Result<bool, String> {
        match action {
            GuiEmbossAction::Open | GuiEmbossAction::OpenEdit(_) => Ok(false),
            GuiEmbossAction::ClearSelectedFace => {
                let cleared = self.model_face.take().is_some();
                if cleared {
                    self.part_scenes.invalidate_presentation();
                }
                Ok(cleared)
            }
            GuiEmbossAction::Clear => {
                Ok(self.emboss_preview.take().is_some() | self.clear_extrusion_preview())
            }
            GuiEmbossAction::Preview(request) => {
                let request = *request;
                self.require_emboss_generation(&request)?;
                let result = self.preview_emboss(&request);
                self.emboss_preview = Some(EmbossPreview { request, result });
                Ok(true)
            }
            GuiEmbossAction::Commit(request) => {
                let request = *request;
                self.require_emboss_generation(&request)?;
                let commit = self
                    .emboss_preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        EmbossOutcome::Ready(commit) => Some(commit),
                        _ => None,
                    })
                    .ok_or("emboss: expected a current certified preview")?
                    .clone();
                self.commit_emboss(request.document, commit)?;
                self.emboss_preview = None;
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_emboss_generation(&self, request: &GuiEmbossRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("emboss: expected the current document generation".into());
        }
        Ok(())
    }

    fn clear_emboss_scene(&mut self) {
        if self.extrusion_preview.take().is_some() {
            self.part_scenes.invalidate_presentation();
        }
    }

    fn preview_emboss(&mut self, request: &GuiEmbossRequest) -> Result<EmbossOutcome, String> {
        if request.wrap_to_face && request.wrap_face.is_none() {
            self.clear_emboss_scene();
            return Ok(EmbossOutcome::MissingWrapFace);
        }
        let outcome = match request.feature {
            Some(feature) => self.preview_emboss_edit(request, feature),
            None => self.preview_emboss_create(request),
        };
        if !matches!(outcome, Ok(EmbossOutcome::Ready(_))) {
            self.clear_emboss_scene();
        }
        outcome
    }

    fn datums_for_sketch(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
    ) -> Result<WorkGeometryEvaluation, String> {
        let placement = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot()
            .sketches
            .get(sketch)
            .map_err(|cause| cause.to_string())?
            .placement();
        if let SketchPlacement::Face(face) = placement {
            let current =
                self.part_scenes
                    .current_part_datums_for_face(&mut self.files, document, &face)?;
            let session = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?;
            return current
                .current(session)
                .cloned()
                .map_err(|cause| cause.to_string());
        }
        self.part_scenes.datum_evaluation(&mut self.files, document)
    }

    fn resolve_emboss_profile(
        &mut self,
        document: u64,
        policy: oblikovati_martensite::PrismResolutionPolicy,
    ) -> Result<
        Option<(
            DocumentSketchId,
            SketchPrismSelection,
            WorkGeometryEvaluation,
        )>,
        String,
    > {
        let visible_sketches: Vec<DocumentSketchId> = {
            let snapshot = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?
                .snapshot();
            let mut unconsumed: Vec<_> = snapshot
                .sketches
                .sketches()
                .iter()
                .filter(|s| s.visible())
                .filter(|s| {
                    snapshot
                        .work_geometry
                        .sketch_consumers(s.id())
                        .next()
                        .is_none()
                })
                .map(|s| s.id())
                .collect();
            if unconsumed.is_empty() {
                unconsumed = snapshot
                    .sketches
                    .sketches()
                    .iter()
                    .filter(|s| s.visible())
                    .map(|s| s.id())
                    .collect();
            }
            unconsumed
        };
        for sketch in visible_sketches {
            let datums = self.datums_for_sketch(document, sketch)?;
            let snapshot = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?
                .snapshot();
            match certified_sketch_regions_with_datums(snapshot, sketch, policy, &datums) {
                Ok(regions) => {
                    if let Some(first) = regions.into_iter().next() {
                        return Ok(Some((sketch, first.selection(), datums)));
                    }
                }
                Err(cause) => {
                    if has_open_visible_sketch(snapshot)
                        && snapshot
                            .sketches
                            .get(sketch)
                            .is_ok_and(|s| s.geometry().circles().is_empty())
                    {
                        continue;
                    }
                    return Err(cause.to_string());
                }
            }
        }
        Ok(None)
    }

    fn preview_emboss_create(
        &mut self,
        request: &GuiEmbossRequest,
    ) -> Result<EmbossOutcome, String> {
        let document = request.document;
        let policy =
            <crate::part_scene::DesktopPartScenes as PartUpdateAccess<O, W, F>>::resolution_policy(
                &self.part_scenes,
            )
            .ok_or("emboss: expected a certified resolution policy")?;
        let Some((sketch, selection, datums)) = self.resolve_emboss_profile(document, policy)?
        else {
            let snapshot = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?
                .snapshot();
            if has_open_visible_sketch(snapshot) {
                return Ok(EmbossOutcome::OpenLoop(
                    "Open loop found in sketch profile. Close the profile before embossing.".into(),
                ));
            }
            return Ok(EmbossOutcome::MissingProfile);
        };
        let target_body = sweep_boolean_target(&self.files, &self.part_scenes, document, None)
            .map_err(|cause| cause.to_string())?;
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let prepared = match prepare_emboss_definition(
            snapshot,
            &datums,
            emboss_inputs(
                request,
                sketch,
                selection,
                request.wrap_face.clone(),
                target_body,
            ),
        ) {
            Ok(prepared) => prepared,
            Err(cause) => {
                let message = cause.to_string();
                if is_range_error(&message) {
                    return Ok(EmbossOutcome::RangeError(message));
                }
                return Err(message);
            }
        };
        if request.preview_hidden {
            self.clear_emboss_scene();
        } else {
            let preview = prepared.clone();
            self.preview_selected_blend(document, |candidate, ids| preview.create(candidate, ids))?;
        }
        Ok(EmbossOutcome::Ready(EmbossCommit::Create(Box::new(
            prepared,
        ))))
    }

    fn preview_emboss_edit(
        &mut self,
        request: &GuiEmbossRequest,
        id: FeatureId,
    ) -> Result<EmbossOutcome, String> {
        let existing = {
            let snapshot = self
                .files
                .workspace()
                .session(request.document)
                .map_err(|cause| cause.to_string())?
                .snapshot();
            emboss_definition(snapshot, id)?.clone()
        };
        let datums = self.datums_for_sketch(request.document, existing.sketch())?;
        let wrap_face = if request.wrap_to_face {
            let Some(picked) = request.wrap_face.as_deref() else {
                return Ok(EmbossOutcome::MissingWrapFace);
            };
            Some(self.resolve_wrap_face_for_edit(request.document, id, picked)?)
        } else {
            None
        };
        let snapshot = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let edited = match build_emboss_definition(
            snapshot,
            &datums,
            emboss_inputs(
                request,
                existing.sketch(),
                existing.selection().clone(),
                wrap_face,
                existing.target_body().0.to_vec(),
            ),
        ) {
            Ok(edited) => edited,
            Err(cause) => {
                let message = cause.to_string();
                if is_range_error(&message) {
                    return Ok(EmbossOutcome::RangeError(message));
                }
                return Err(message);
            }
        };
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let projection = session
            .preview_projected(
                |candidate, _| replace_emboss(candidate, id, edited.clone()),
                |candidate, ()| {
                    Ok(self.part_scenes.project_feature_edit(
                        &self.files,
                        request.document,
                        candidate,
                        id,
                    ))
                },
            )
            .map_err(|cause| cause.to_string())?;
        match projection {
            Ok(scene) => {
                if request.preview_hidden {
                    self.clear_emboss_scene();
                } else {
                    self.extrusion_preview = Some((
                        request.document,
                        request.generation,
                        scene,
                        super::super::owner_extrusion_preview::PreviewKind::Replacement,
                    ));
                    self.part_scenes.invalidate_presentation();
                }
                Ok(EmbossOutcome::Ready(EmbossCommit::Edit(
                    id,
                    Box::new(edited),
                )))
            }
            Err(cause) => Err(cause.to_string()),
        }
    }

    fn resolve_wrap_face_for_edit(
        &self,
        document: u64,
        id: FeatureId,
        picked: &[u8],
    ) -> Result<Vec<u8>, String> {
        let policy =
            <crate::part_scene::DesktopPartScenes as PartUpdateAccess<O, W, F>>::resolution_policy(
                &self.part_scenes,
            )
            .ok_or("emboss: expected a certified resolution policy")?;
        let workspace = self.files.workspace();
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let history = session.history();
        let mut pre_snapshot = session.snapshot().clone();
        pre_snapshot
            .work_geometry
            .set_suppressed(id, true)
            .map_err(|cause| cause.to_string())?;
        let mut eval = CertifiedNamedSolidEvaluator::with_policy(history, guid, policy);
        let pre_part = eval
            .evaluate_private_candidate(&pre_snapshot)
            .map_err(|cause| cause.to_string())?;
        for body in pre_part.ordered_bodies() {
            if body.face_plane(history, guid, picked).is_ok() {
                return Ok(picked.to_vec());
            }
        }
        let current = self.part_scenes.current(session)?;
        let picked_plane = current
            .ordered_bodies()
            .find_map(|body| body.face_plane(history, guid, picked).ok())
            .ok_or("emboss wrap face: expected a planar face on the target solid")?;
        for body in pre_part.ordered_bodies() {
            let keys = body.face_reference_keys(history, guid)?;
            for key in keys {
                if let Ok(plane) = body.face_plane(history, guid, &key)
                    && planes_coplanar(plane, picked_plane)
                {
                    return Ok(key);
                }
            }
        }
        Err("emboss wrap face: could not map selected face onto target solid".into())
    }

    fn commit_emboss(&mut self, document: u64, commit: EmbossCommit) -> Result<(), String> {
        match commit {
            EmbossCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Emboss",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            EmbossCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Emboss",
                    |snapshot, _| replace_emboss(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
