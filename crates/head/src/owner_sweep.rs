// SPDX-License-Identifier: GPL-2.0-only
//! Preview and accept the exact Sweep draft for the current document generation.
use super::*;
use oblikovati_api::{Error, client::Client, wire::featureargs::Sweep};
use oblikovati_automation::{
    PartUpdateAccess, PreparedSweep, automatic_sweep, prepare_surface_sweep,
    prepare_sweep_for_target, prepare_sweep_taper, prepare_sweep_twist, sweep_boolean_target,
};
use oblikovati_gui::{
    GuiSweepAction, GuiSweepEditSeed, GuiSweepOrientation, GuiSweepOutput, GuiSweepPreview,
    GuiSweepPreviewState, GuiSweepProfile, GuiSweepRequest,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId,
        part::PartFeatureDefinition,
        sketch_prism::PrismBoolean,
        sketch_sweep::{SketchSweepDefinition, SketchSweepOrientation},
        work_geometry::DatumAngle,
    },
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[path = "owner_sweep/picking.rs"]
mod picking;
pub(in crate::owner) use picking::{ray_hits_segment, ray_segment_fraction};

#[derive(Clone)]
enum SweepCommit {
    Create(Box<Sweep>),
    CreateSurface(Box<PreparedSweep>),
    Edit(FeatureId, SketchSweepDefinition),
}

enum SweepOutcome {
    Ready(SweepCommit),
    UnchangedCut(String),
}

pub(in crate::owner) struct SweepPreview {
    request: GuiSweepRequest,
    result: Result<Option<SweepOutcome>, String>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiSweepEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::SketchSweep(definition) = feature.definition() else {
                return None;
            };
            let recipe = definition.recipe();
            let taper = angle_field(snapshot, recipe.taper)?;
            let twist = angle_field(snapshot, recipe.twist)?;
            let orientation = match recipe.orientation {
                SketchSweepOrientation::FollowPath => GuiSweepOrientation::FollowPath,
                SketchSweepOrientation::Fixed => GuiSweepOrientation::Fixed,
            };
            Some((
                feature.id(),
                GuiSweepEditSeed {
                    taper,
                    twist,
                    orientation,
                    output: saved_sweep_output(definition),
                },
            ))
        })
        .collect()
}

fn saved_sweep_output(definition: &SketchSweepDefinition) -> GuiSweepOutput {
    match definition
        .boolean_target()
        .map(|(operation, _, _)| operation)
    {
        Some(PrismBoolean::Join) => GuiSweepOutput::Join,
        Some(PrismBoolean::Cut) => GuiSweepOutput::Cut,
        None if definition.recipe().surface => GuiSweepOutput::Surface,
        None => GuiSweepOutput::NewSolid,
    }
}

fn angle_field(snapshot: &DocumentSnapshot, angle: Option<DatumAngle>) -> Option<String> {
    let Some(angle) = angle else {
        return Some("0 deg".into());
    };
    match angle.parameter_id() {
        Some(id) => Some(snapshot.parameters.get(id)?.name.clone()),
        None => Some(format!(
            "{} deg",
            angle.resolve(&snapshot.parameters).ok()?.to_degrees()
        )),
    }
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
    ) -> Option<GuiSweepPreviewState> {
        let preview = self.sweep_preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiSweepPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(Some(SweepOutcome::Ready(_))) => GuiSweepPreview::Ready,
                    Ok(Some(SweepOutcome::UnchangedCut(cause))) => {
                        GuiSweepPreview::UnchangedCut(cause.clone())
                    }
                    Ok(None) if preview.request.profile == GuiSweepProfile::Empty => {
                        GuiSweepPreview::MissingProfile
                    }
                    Ok(None) if preview.request.path_cleared => GuiSweepPreview::MissingPath,
                    Ok(None) if preview.request.orientation == GuiSweepOrientation::Guide => {
                        GuiSweepPreview::MissingGuide
                    }
                    Ok(None) => GuiSweepPreview::MissingInputs,
                    Err(cause) => GuiSweepPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn sweep_command(
        &mut self,
        action: GuiSweepAction,
    ) -> Result<bool, String> {
        match action {
            GuiSweepAction::Open | GuiSweepAction::OpenEdit(_) => Ok(false),
            GuiSweepAction::Clear => {
                Ok(self.sweep_preview.is_some() && self.clear_extrusion_preview())
            }
            GuiSweepAction::Preview(request) => {
                self.require_sweep_generation(&request)?;
                self.clear_extrusion_preview();
                let result = self.preview_sweep(&request);
                self.sweep_preview = Some(SweepPreview { request, result });
                Ok(true)
            }
            GuiSweepAction::Commit(request) => {
                self.require_sweep_generation(&request)?;
                let commit = self
                    .sweep_preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(Option::as_ref)
                    .and_then(|outcome| match outcome {
                        SweepOutcome::Ready(commit) => Some(commit),
                        SweepOutcome::UnchangedCut(_) => None,
                    })
                    .ok_or("sweep: expected a current certified preview")?
                    .clone();
                self.commit_sweep(request.document, commit)?;
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_sweep_generation(&self, request: &GuiSweepRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("sweep: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_sweep(&mut self, request: &GuiSweepRequest) -> Result<Option<SweepOutcome>, String> {
        if request.path_cleared || request.profile == GuiSweepProfile::Empty {
            if let Some(feature) = request.feature {
                let scene = self.part_scenes.part_scene_without_feature_body(
                    &self.files,
                    request.document,
                    feature,
                )?;
                self.extrusion_preview = Some((
                    request.document,
                    request.generation,
                    scene,
                    super::super::owner_extrusion_preview::PreviewKind::Replacement,
                ));
                self.part_scenes.invalidate_presentation();
            }
            return Ok(None);
        }
        if request.orientation == GuiSweepOrientation::Guide {
            return Ok(None);
        }
        if let Some(feature) = request.feature {
            return self.preview_sweep_edit(request, feature).map(Some);
        }
        let document = request.document;
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
        let Some(mut input) = automatic_sweep(snapshot, &datums, policy) else {
            return Ok(None);
        };
        let target = if let Some(operation) = sweep_boolean(request.output) {
            input.operation = operation.to_string().to_ascii_lowercase();
            Some(
                sweep_boolean_target(&self.files, &self.part_scenes, document, None)
                    .map_err(|cause| cause.to_string())?,
            )
        } else {
            None
        };
        if request.orientation == GuiSweepOrientation::Fixed {
            input.orientation = "parallelToOriginalProfile".into();
        } else {
            if request.taper.trim().is_empty() {
                return Err("Sweep taper requires an angle.".into());
            }
            if request.taper != "0 deg" {
                input.taper.clone_from(&request.taper);
            }
            if request.twist.trim().is_empty() {
                return Err("Sweep twist requires an angle.".into());
            }
            if request.twist != "0 deg" {
                input.twist.clone_from(&request.twist);
            }
        }
        if request.output == GuiSweepOutput::Surface {
            let prepared = prepare_surface_sweep(snapshot, &datums, policy, &input)
                .map_err(|cause| cause.to_string())?;
            let preview = prepared.clone();
            self.preview_selected_blend(document, |candidate, ids| preview.create(candidate, ids))?;
            return Ok(Some(SweepOutcome::Ready(SweepCommit::CreateSurface(
                Box::new(prepared),
            ))));
        }
        let prepared = prepare_sweep_for_target(snapshot, &datums, policy, &input, target)
            .map_err(|cause| cause.to_string())?;
        self.preview_selected_blend(document, |candidate, ids| prepared.create(candidate, ids))?;
        Ok(Some(SweepOutcome::Ready(SweepCommit::Create(Box::new(
            input,
        )))))
    }

    fn preview_sweep_edit(
        &mut self,
        request: &GuiSweepRequest,
        id: FeatureId,
    ) -> Result<SweepOutcome, String> {
        let snapshot = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let definition = sweep_definition(snapshot, id)?.clone();
        let definition = match &request.profile {
            GuiSweepProfile::Picked(profile) => definition
                .with_profile(profile.clone())
                .map_err(|cause| cause.to_string())?,
            _ => definition,
        };
        let definition = if let Some(operation) = sweep_boolean(request.output) {
            let key = match definition.boolean_target() {
                Some((_, key, _)) => key.to_vec(),
                None => {
                    sweep_boolean_target(&self.files, &self.part_scenes, request.document, Some(id))
                        .map_err(|cause| cause.to_string())?
                }
            };
            definition
                .with_boolean_target(operation, key)
                .map_err(|cause| cause.to_string())?
        } else {
            definition.with_surface_output(request.output == GuiSweepOutput::Surface)
        };
        let edited = if request.orientation == GuiSweepOrientation::Fixed {
            definition.with_fixed_orientation()
        } else {
            let tapered = prepare_sweep_taper(snapshot, definition, &request.taper)
                .map_err(|cause| cause.to_string())?;
            prepare_sweep_twist(snapshot, tapered, &request.twist)
                .map_err(|cause| cause.to_string())?
        };
        self.preview_sweep_edit_scene(request, id, edited)
    }

    fn preview_sweep_edit_scene(
        &mut self,
        request: &GuiSweepRequest,
        id: FeatureId,
        edited: SketchSweepDefinition,
    ) -> Result<SweepOutcome, String> {
        use crate::part_scene::FeatureEditPreviewError;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let projection = session
            .preview_projected(
                |candidate, _| replace_sweep(candidate, id, edited.clone()),
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
                self.extrusion_preview = Some((
                    request.document,
                    request.generation,
                    scene,
                    super::super::owner_extrusion_preview::PreviewKind::Replacement,
                ));
                self.part_scenes.invalidate_presentation();
                Ok(SweepOutcome::Ready(SweepCommit::Edit(id, edited)))
            }
            Err(FeatureEditPreviewError::UnchangedCut(cause)) => {
                Ok(SweepOutcome::UnchangedCut(cause))
            }
            Err(cause) => Err(cause.to_string()),
        }
    }

    fn commit_sweep(&mut self, document: u64, commit: SweepCommit) -> Result<(), String> {
        match commit {
            SweepCommit::Create(input) => {
                Client::new(self.document_host())
                    .features()
                    .add_feature(input.as_ref())
                    .map_err(|cause| cause.to_string())?;
            }
            SweepCommit::CreateSurface(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Surface Sweep",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            SweepCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Sweep",
                    |snapshot, _| replace_sweep(snapshot, id, definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}

fn sweep_boolean(output: GuiSweepOutput) -> Option<PrismBoolean> {
    match output {
        GuiSweepOutput::NewSolid | GuiSweepOutput::Surface => None,
        GuiSweepOutput::Join => Some(PrismBoolean::Join),
        GuiSweepOutput::Cut => Some(PrismBoolean::Cut),
    }
}

fn sweep_definition(
    snapshot: &DocumentSnapshot,
    id: FeatureId,
) -> Result<&SketchSweepDefinition, String> {
    let authored = snapshot
        .work_geometry
        .get(id)
        .map_err(|cause| cause.to_string())?;
    let PartFeatureDefinition::SketchSweep(definition) = authored.definition() else {
        return Err(format!("feature {id:?}: expected a Sweep"));
    };
    Ok(definition)
}

fn replace_sweep(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: SketchSweepDefinition,
) -> Result<(), Error> {
    let reads = definition
        .reads(snapshot)
        .map_err(|cause| Error(cause.to_string()))?;
    snapshot
        .work_geometry
        .replace(
            id,
            PartFeatureDefinition::SketchSweep(Box::new(definition)),
            reads,
        )
        .map_err(|cause| Error(cause.to_string()))
        .map(|_| ())
}
