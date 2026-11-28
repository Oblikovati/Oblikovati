// SPDX-License-Identifier: GPL-2.0-only
//! Face picking, analytic area preview, face tinting, and commits for Part Modify Finish (#3918).
use super::*;
#[path = "owner_finish_seed.rs"]
mod seed;
use oblikovati_automation::{
    FinishFeatureInputs, PreparedFinish, build_finish_definition, prepare_finish_definition,
    replace_finish,
};
use oblikovati_gui::{
    GuiFinishAction, GuiFinishPick, GuiFinishPreview, GuiFinishPreviewState, GuiFinishRequest,
};
use oblikovati_martensite::finish_appearance_rgb_u8;
use oblikovati_model::feature::{FeatureId, finish::FinishDefinition};
pub(in crate::owner) use seed::edit_seeds;

#[derive(Clone)]
enum FinishCommit {
    Create(Box<PreparedFinish>),
    Edit(FeatureId, Box<FinishDefinition>),
}

enum FinishOutcome {
    Ready { commit: FinishCommit, area_mm2: f64 },
    Incomplete,
}

pub(in crate::owner) struct FinishPreview {
    request: GuiFinishRequest,
    result: Result<FinishOutcome, String>,
}

#[derive(Default)]
pub(in crate::owner) struct FinishOwnerState {
    pub(in crate::owner) active: bool,
    pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiFinishPick>,
    pub(in crate::owner) preview: Option<FinishPreview>,
    pub(in crate::owner) preview_faces: Vec<(FeatureId, Vec<u8>)>,
    pub(in crate::owner) preview_color_rgb: [u8; 3],
    pub(in crate::owner) preview_hidden: bool,
}

pub(in crate::owner) type ActiveFaceSelection = (u64, u64, Vec<(FeatureId, Vec<u8>)>, [u8; 3]);

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in crate::owner) fn active_face_selection(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<ActiveFaceSelection> {
        if self.active_sketch_edit().is_some() {
            return None;
        }
        let finish = &self.create_features.finish;
        if finish.active && !finish.preview_hidden && !finish.preview_faces.is_empty() {
            return Some((
                document,
                generation,
                finish.preview_faces.clone(),
                finish.preview_color_rgb,
            ));
        }
        let faces: Vec<_> = if self.create_features.delete_face.active
            && !self
                .create_features
                .delete_face
                .highlighted_faces
                .is_empty()
        {
            self.create_features.delete_face.highlighted_faces.clone()
        } else if self.create_features.replace_face.active
            && !self
                .create_features
                .replace_face
                .highlighted_faces
                .is_empty()
        {
            self.create_features.replace_face.highlighted_faces.clone()
        } else {
            self.current_face_appearance_picks(document)
                .iter()
                .map(|pick| (pick.face.feature, pick.face.key.clone()))
                .collect()
        };
        if faces.is_empty() {
            return None;
        }
        let color = &self.color_schemes.active().secondary_select;
        Some((document, generation, faces, [color.r, color.g, color.b]))
    }

    pub(in crate::owner) fn gui_finish_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiFinishPreviewState> {
        let preview = self.create_features.finish.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiFinishPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(FinishOutcome::Ready { area_mm2, .. }) => GuiFinishPreview::Ready {
                        area_mm2: *area_mm2,
                    },
                    Ok(FinishOutcome::Incomplete) => GuiFinishPreview::Incomplete,
                    Err(cause) => GuiFinishPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_finish_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiFinishPick> {
        self.create_features
            .finish
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn pick_finish_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.finish.active {
            return Ok(None);
        }
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        let allowed = oblikovati_session::selection::SelectionFilter::only([
            oblikovati_session::selection::SelectionKind::Face,
        ]);
        let candidates = self
            .part_scenes
            .part_candidates(&self.files, document, ray, &allowed)?;
        let Ok(Some(picked)) =
            candidates.resolve_hit(&allowed, &self.ambient_selection, tolerance.unwrap_or(0.0))
        else {
            return Ok(None);
        };
        let crate::part_scene::pick_candidates::PickTarget::Face(_, key) = picked.target else {
            return Ok(None);
        };
        let (_, body) = self.resolve_finish_face_owner(document, &key)?;
        let state = &mut self.create_features.finish;
        state.pick_seq = state.pick_seq.saturating_add(1);
        state.pick = Some(GuiFinishPick {
            document,
            generation,
            sequence: state.pick_seq,
            body,
            face: key,
        });
        self.part_scenes.invalidate_presentation();
        Ok(Some(true))
    }

    pub(in crate::owner) fn finish_command(
        &mut self,
        action: GuiFinishAction,
    ) -> Result<bool, String> {
        match action {
            GuiFinishAction::Open | GuiFinishAction::OpenEdit(_) => {
                self.create_features.finish.active = true;
                self.create_features.finish.preview_faces.clear();
                self.part_scenes.invalidate_presentation();
                Ok(false)
            }
            GuiFinishAction::Clear => {
                let state = &mut self.create_features.finish;
                let had_state = state.active
                    | state.preview.take().is_some()
                    | state.pick.take().is_some()
                    | !state.preview_faces.is_empty();
                state.active = false;
                state.preview_faces.clear();
                if had_state {
                    self.part_scenes.invalidate_presentation();
                }
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiFinishAction::Preview(request) => {
                let request = *request;
                self.require_finish_generation(&request)?;
                self.create_features.finish.active = true;
                self.sync_finish_preview_tint(&request)?;
                let result = self.preview_finish(&request);
                self.create_features.finish.preview = Some(FinishPreview { request, result });
                Ok(true)
            }
            GuiFinishAction::Commit(request) => {
                let request = *request;
                self.require_finish_generation(&request)?;
                let commit = self
                    .create_features
                    .finish
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        FinishOutcome::Ready { commit, .. } => Some(commit),
                        FinishOutcome::Incomplete => None,
                    })
                    .ok_or("finish: expected a current certified preview")?
                    .clone();
                self.commit_finish(request.document, commit)?;
                let state = &mut self.create_features.finish;
                state.preview = None;
                state.pick = None;
                state.preview_faces.clear();
                state.active = false;
                self.model_face = None;
                self.face_appearance_picks.clear();
                self.clear_extrusion_preview();
                self.part_scenes.invalidate_presentation();
                Ok(true)
            }
        }
    }

    fn require_finish_generation(&self, request: &GuiFinishRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("finish: expected the current document generation".into());
        }
        Ok(())
    }

    fn sync_finish_preview_tint(&mut self, request: &GuiFinishRequest) -> Result<(), String> {
        let mut next_faces = Vec::new();
        for face in &request.included_faces {
            if request.excluded_faces.iter().any(|ex| ex == face) {
                continue;
            }
            if let Ok((feature, _)) = self.resolve_finish_face_owner(request.document, face) {
                next_faces.push((feature, face.clone()));
            }
        }
        let next_rgb = finish_appearance_rgb_u8(&request.appearance);
        let state = &mut self.create_features.finish;
        if state.preview_faces != next_faces
            || state.preview_color_rgb != next_rgb
            || state.preview_hidden != request.preview_hidden
        {
            state.preview_faces = next_faces;
            state.preview_color_rgb = next_rgb;
            state.preview_hidden = request.preview_hidden;
            self.part_scenes.invalidate_presentation();
        }
        Ok(())
    }

    fn preview_finish(&mut self, request: &GuiFinishRequest) -> Result<FinishOutcome, String> {
        self.clear_extrusion_preview();
        let Some(first_face) = request.included_faces.first() else {
            return Ok(FinishOutcome::Incomplete);
        };
        let (owner_feature, resolved_body) =
            self.resolve_finish_face_owner(request.document, first_face)?;
        let body = request.body.clone().unwrap_or(resolved_body);
        let area_mm2 = self.compute_finish_face_area_mm2(
            request.document,
            owner_feature,
            &request.included_faces,
            &request.excluded_faces,
        )?;
        let inputs = FinishFeatureInputs {
            body,
            included_faces: request.included_faces.clone(),
            excluded_faces: request.excluded_faces.clone(),
            finish_type: request.finish_type,
            process: request.process.clone(),
            appearance: request.appearance.clone(),
            thickness: request.thickness.clone(),
            hardness: request.hardness.clone(),
            depth: request.depth.clone(),
            description: request.description.clone(),
            short_description: request.short_description.clone(),
            comment: request.comment.clone(),
            export_all_parameters: request.export_all_parameters,
        };
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared = prepare_finish_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                FinishCommit::Create(Box::new(prepared))
            }
            Some(id) => {
                let edited = build_finish_definition(&inputs).map_err(|cause| cause.to_string())?;
                FinishCommit::Edit(id, Box::new(edited))
            }
        };
        self.validate_finish_candidate(request.document, &commit)?;
        Ok(FinishOutcome::Ready { commit, area_mm2 })
    }

    fn compute_finish_face_area_mm2(
        &self,
        document: u64,
        owner_feature: FeatureId,
        included: &[Vec<u8>],
        excluded: &[Vec<u8>],
    ) -> Result<f64, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        let solid = evaluation
            .body(owner_feature)
            .ok_or_else(|| format!("finish body {owner_feature:?}: expected a current body"))?;
        solid.finish_area_mm2(session.history(), guid, included, excluded)
    }

    fn validate_finish_candidate(
        &self,
        document: u64,
        commit: &FinishCommit,
    ) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            FinishCommit::Create(prepared) => {
                let cloned = (**prepared).clone();
                let projection = session
                    .preview_projected(
                        |candidate, ids| cloned.create(candidate, ids),
                        |candidate, id| {
                            Ok(self.part_scenes.project_feature_edit(
                                &self.files,
                                document,
                                candidate,
                                id,
                            ))
                        },
                    )
                    .map_err(|cause| cause.to_string())?;
                projection.map(|_| ()).map_err(|cause| cause.to_string())
            }
            FinishCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_finish(candidate, id, edited.clone()),
                        |candidate, ()| {
                            Ok(self.part_scenes.project_feature_edit(
                                &self.files,
                                document,
                                candidate,
                                id,
                            ))
                        },
                    )
                    .map_err(|cause| cause.to_string())?;
                projection.map(|_| ()).map_err(|cause| cause.to_string())
            }
        }
    }

    fn resolve_finish_face_owner(
        &self,
        document: u64,
        face: &[u8],
    ) -> Result<(FeatureId, Vec<u8>), String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        for (feature, body) in evaluation.ordered_body_outputs() {
            let Ok(keys) = body.face_reference_keys(session.history(), guid) else {
                continue;
            };
            if keys.iter().any(|k| k.as_slice() == face) {
                return Ok((
                    feature,
                    body.body_reference_key(session.history(), guid)?.to_vec(),
                ));
            }
        }
        Err("finish: expected a current body face".into())
    }

    fn commit_finish(&mut self, document: u64, commit: FinishCommit) -> Result<(), String> {
        match commit {
            FinishCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Finish",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            FinishCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Finish",
                    |snapshot, _| replace_finish(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
