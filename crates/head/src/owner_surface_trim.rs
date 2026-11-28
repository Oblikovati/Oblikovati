// SPDX-License-Identifier: GPL-2.0-only
//! Edit seeds, highlighted two-region preview, and commit for Surface Trim (#3918).
#[path = "owner_surface_trim_pick.rs"]
mod pick;

use super::*;
use oblikovati_automation::{
    PartUpdateAccess, PreparedSurfaceTrim, SurfaceTrimFeatureInputs, build_surface_trim_definition,
    prepare_surface_trim_definition, replace_surface_trim,
};
use oblikovati_gui::{
    GuiSurfaceTrimAction, GuiSurfaceTrimEditSeed, GuiSurfaceTrimPick, GuiSurfaceTrimPickMode,
    GuiSurfaceTrimPreview, GuiSurfaceTrimPreviewState, GuiSurfaceTrimRequest,
};
use oblikovati_martensite::SurfaceTrimRegionPreview;
use oblikovati_model::{
    feature::{
        FeatureId,
        part::PartFeatureDefinition,
        surface_trim::{SurfaceTrimDefinition, SurfaceTrimSide, SurfaceTrimTool},
    },
    history::DocumentSnapshot,
};
use oblikovati_viewport::ViewportScene;
use pick::{
    append_display_triangles, find_body_by_key, resolve_preview_inputs, resolve_tool_in_eval,
};
use std::collections::BTreeMap;

/// Translucent red highlight color rendered over the candidate removal region during preview.
pub(crate) const SURFACE_TRIM_REMOVE_HIGHLIGHT: [f32; 4] = [0.95, 0.26, 0.18, 0.55];
const SURFACE_TRIM_KEPT_COLOR: [f32; 4] = [0.78, 0.66, 0.34, 0.90];

#[derive(Clone)]
enum SurfaceTrimCommit {
    Create(Box<PreparedSurfaceTrim>),
    Edit(FeatureId, Box<SurfaceTrimDefinition>),
}

enum SurfaceTrimOutcome {
    Ready(SurfaceTrimCommit),
    Incomplete,
}

pub(in crate::owner) struct SurfaceTrimPreview {
    request: GuiSurfaceTrimRequest,
    result: Result<SurfaceTrimOutcome, String>,
}

#[derive(Default)]
pub(in crate::owner) struct SurfaceTrimOwnerState {
    pub(in crate::owner) active: bool,
    pub(in crate::owner) editing: Option<FeatureId>,
    pub(in crate::owner) pick_mode: GuiSurfaceTrimPickMode,
    pub(in crate::owner) active_tool: Option<SurfaceTrimTool>,
    pub(in crate::owner) pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiSurfaceTrimPick>,
    pub(in crate::owner) preview: Option<SurfaceTrimPreview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiSurfaceTrimEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::SurfaceTrim(definition) = feature.definition() else {
                return None;
            };
            let recipe = definition.recipe();
            Some((
                feature.id(),
                GuiSurfaceTrimEditSeed {
                    tool: recipe.tool,
                    target_body: recipe.target_body,
                    remove_side: recipe.remove_side,
                },
            ))
        })
        .collect()
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
    pub(in crate::owner) fn gui_surface_trim_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiSurfaceTrimPreviewState> {
        let preview = self.create_features.surface_trim.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiSurfaceTrimPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(SurfaceTrimOutcome::Ready(_)) => GuiSurfaceTrimPreview::Ready,
                    Ok(SurfaceTrimOutcome::Incomplete) => GuiSurfaceTrimPreview::Incomplete,
                    Err(cause) => GuiSurfaceTrimPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_surface_trim_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiSurfaceTrimPick> {
        self.create_features
            .surface_trim
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn surface_trim_command(
        &mut self,
        action: GuiSurfaceTrimAction,
    ) -> Result<bool, String> {
        match action {
            GuiSurfaceTrimAction::Open => {
                let state = &mut self.create_features.surface_trim;
                state.active = true;
                state.editing = None;
                state.pick_mode = GuiSurfaceTrimPickMode::CuttingTool;
                state.active_tool = None;
                Ok(false)
            }
            GuiSurfaceTrimAction::OpenEdit(id) => {
                let state = &mut self.create_features.surface_trim;
                state.active = true;
                state.editing = Some(id);
                state.pick_mode = GuiSurfaceTrimPickMode::Remove;
                Ok(false)
            }
            GuiSurfaceTrimAction::SetPickMode(mode) => {
                self.create_features.surface_trim.pick_mode = mode;
                Ok(false)
            }
            GuiSurfaceTrimAction::Clear => {
                let state = &mut self.create_features.surface_trim;
                let had_state =
                    state.active | state.preview.take().is_some() | state.pick.take().is_some();
                state.active = false;
                state.editing = None;
                state.active_tool = None;
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiSurfaceTrimAction::Preview(request) => {
                let request = *request;
                self.require_surface_trim_generation(&request)?;
                let state = &mut self.create_features.surface_trim;
                state.active = true;
                state.editing = request.feature;
                state.active_tool = request.tool.clone();
                let result = self.preview_surface_trim(&request);
                self.create_features.surface_trim.preview =
                    Some(SurfaceTrimPreview { request, result });
                Ok(true)
            }
            GuiSurfaceTrimAction::Commit(request) => {
                let request = *request;
                self.require_surface_trim_generation(&request)?;
                let commit = self
                    .create_features
                    .surface_trim
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        SurfaceTrimOutcome::Ready(commit) => Some(commit),
                        SurfaceTrimOutcome::Incomplete => None,
                    })
                    .ok_or("surface trim: expected a current certified preview")?
                    .clone();
                self.commit_surface_trim(request.document, commit)?;
                let state = &mut self.create_features.surface_trim;
                state.preview = None;
                state.pick = None;
                state.active = false;
                state.editing = None;
                state.active_tool = None;
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_surface_trim_generation(
        &self,
        request: &GuiSurfaceTrimRequest,
    ) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("surface trim: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_surface_trim(
        &mut self,
        request: &GuiSurfaceTrimRequest,
    ) -> Result<SurfaceTrimOutcome, String> {
        let (Some(tool), Some(target_body), Some(remove_side)) = (
            request.tool.clone(),
            request.target_body.clone(),
            request.remove_side,
        ) else {
            self.clear_extrusion_preview();
            return Ok(SurfaceTrimOutcome::Incomplete);
        };
        let inputs = SurfaceTrimFeatureInputs {
            tool: tool.clone(),
            target_body: target_body.clone(),
            remove_side,
        };
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared = prepare_surface_trim_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                SurfaceTrimCommit::Create(Box::new(prepared))
            }
            Some(id) => {
                let edited =
                    build_surface_trim_definition(&inputs).map_err(|cause| cause.to_string())?;
                SurfaceTrimCommit::Edit(id, Box::new(edited))
            }
        };
        // Validate the real create/edit Martensite candidate first.
        self.validate_surface_trim_candidate(request.document, &commit)?;
        // Build the two-region highlighted preview scene (kept region + red/amber removal region).
        match self.build_surface_trim_highlight_scene(
            request.document,
            request.feature,
            &tool,
            &target_body,
            remove_side,
        ) {
            Ok(scene) => {
                self.extrusion_preview = Some((
                    request.document,
                    request.generation,
                    scene,
                    super::super::owner_extrusion_preview::PreviewKind::Replacement,
                ));
                self.part_scenes.invalidate_presentation();
                Ok(SurfaceTrimOutcome::Ready(commit))
            }
            Err(cause) => {
                self.clear_extrusion_preview();
                Err(cause)
            }
        }
    }

    fn validate_surface_trim_candidate(
        &self,
        document: u64,
        commit: &SurfaceTrimCommit,
    ) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            SurfaceTrimCommit::Create(prepared) => {
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
            SurfaceTrimCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_surface_trim(candidate, id, edited.clone()),
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

    fn build_surface_trim_highlight_scene(
        &self,
        document: u64,
        editing: Option<FeatureId>,
        tool: &SurfaceTrimTool,
        target_body: &[u8],
        remove_side: SurfaceTrimSide,
    ) -> Result<ViewportScene, String> {
        let (target_feature, regions) =
            self.compute_surface_trim_regions(document, editing, tool, target_body)?;
        let mut scene = self.part_scenes.part_scene_without_feature_body(
            &self.files,
            document,
            target_feature,
        )?;
        let (kept, removed) = match remove_side {
            SurfaceTrimSide::Positive => (&regions.negative_triangles, &regions.positive_triangles),
            SurfaceTrimSide::Negative => (&regions.positive_triangles, &regions.negative_triangles),
        };
        append_display_triangles(&mut scene, kept, SURFACE_TRIM_KEPT_COLOR);
        append_display_triangles(&mut scene, removed, SURFACE_TRIM_REMOVE_HIGHLIGHT);
        Ok(scene)
    }

    pub(super) fn compute_surface_trim_regions(
        &self,
        document: u64,
        editing: Option<FeatureId>,
        tool: &SurfaceTrimTool,
        target_body: &[u8],
    ) -> Result<(FeatureId, SurfaceTrimRegionPreview), String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        if let Some(edit_id) = editing {
            // Suppress the edited Trim feature in a private candidate so we obtain the full pre-trim surface.
            let mut candidate = session.snapshot().clone();
            candidate
                .work_geometry
                .set_suppressed(edit_id, true)
                .map_err(|cause| cause.to_string())?;
            let policy = <crate::part_scene::DesktopPartScenes as PartUpdateAccess<O, W, F>>::resolution_policy(&self.part_scenes);
            let preview = workspace
                .preview_owned_part(document, policy, &candidate)
                .map_err(|cause| cause.to_string())?;
            let eval = &preview.evaluation;
            let (surface, tool_body, resolved_tool) =
                resolve_preview_inputs(eval, session.history(), guid, tool, target_body)?;
            let regions = surface.preview_surface_trim_regions(
                session.history(),
                guid,
                tool_body,
                &resolved_tool,
                crate::part_scene::DISPLAY_MESH,
            )?;
            return Ok((edit_id, regions));
        }
        let eval = self.part_scenes.current(session)?;
        let (surface_feature, surface) =
            find_body_by_key(eval, session.history(), guid, target_body)?;
        let (tool_body, resolved_tool) = resolve_tool_in_eval(eval, session.history(), guid, tool)?;
        let regions = surface.preview_surface_trim_regions(
            session.history(),
            guid,
            tool_body,
            &resolved_tool,
            crate::part_scene::DISPLAY_MESH,
        )?;
        Ok((surface_feature, regions))
    }

    fn commit_surface_trim(
        &mut self,
        document: u64,
        commit: SurfaceTrimCommit,
    ) -> Result<(), String> {
        match commit {
            SurfaceTrimCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Trim Surface",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            SurfaceTrimCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Trim Surface",
                    |snapshot, _| replace_surface_trim(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
