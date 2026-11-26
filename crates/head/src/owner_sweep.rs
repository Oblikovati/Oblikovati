// SPDX-License-Identifier: GPL-2.0-only
//! Preview and accept the exact Sweep draft for the current document generation.
use super::*;
use oblikovati_api::{Error, client::Client, wire::featureargs::Sweep};
use oblikovati_automation::{
    PartUpdateAccess, automatic_sweep, prepare_sweep, prepare_sweep_taper, prepare_sweep_twist,
};
use oblikovati_gui::{
    GuiSweepAction, GuiSweepEditSeed, GuiSweepOrientation, GuiSweepPreview, GuiSweepPreviewState,
    GuiSweepRequest,
};
use oblikovati_model::{
    feature::{
        FeatureId,
        part::PartFeatureDefinition,
        sketch_sweep::{SketchSweepDefinition, SketchSweepOrientation},
        work_geometry::DatumAngle,
    },
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum SweepCommit {
    Create(Box<Sweep>),
    Edit(FeatureId, SketchSweepDefinition),
}

pub(in crate::owner) struct SweepPreview {
    request: GuiSweepRequest,
    result: Result<Option<SweepCommit>, String>,
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
                },
            ))
        })
        .collect()
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
                    Ok(Some(_)) => GuiSweepPreview::Ready,
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

    fn preview_sweep(&mut self, request: &GuiSweepRequest) -> Result<Option<SweepCommit>, String> {
        if request.path_cleared {
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
        let prepared =
            prepare_sweep(snapshot, &datums, policy, &input).map_err(|cause| cause.to_string())?;
        self.preview_selected_blend(document, |candidate, ids| prepared.create(candidate, ids))?;
        Ok(Some(SweepCommit::Create(Box::new(input))))
    }

    fn preview_sweep_edit(
        &mut self,
        request: &GuiSweepRequest,
        id: FeatureId,
    ) -> Result<SweepCommit, String> {
        let snapshot = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let authored = snapshot
            .work_geometry
            .get(id)
            .map_err(|cause| cause.to_string())?;
        let PartFeatureDefinition::SketchSweep(definition) = authored.definition() else {
            return Err(format!("feature {id:?}: expected a Sweep"));
        };
        let definition = (**definition).clone();
        let edited = if request.orientation == GuiSweepOrientation::Fixed {
            definition.with_fixed_orientation()
        } else {
            let tapered = prepare_sweep_taper(snapshot, definition, &request.taper)
                .map_err(|cause| cause.to_string())?;
            prepare_sweep_twist(snapshot, tapered, &request.twist)
                .map_err(|cause| cause.to_string())?
        };
        self.preview_selected_blend(request.document, |candidate, _| {
            replace_sweep(candidate, id, edited.clone())?;
            Ok(id)
        })?;
        if let Some((_, _, _, kind)) = &mut self.extrusion_preview {
            *kind = super::super::owner_extrusion_preview::PreviewKind::Replacement;
        }
        Ok(SweepCommit::Edit(id, edited))
    }

    fn commit_sweep(&mut self, document: u64, commit: SweepCommit) -> Result<(), String> {
        match commit {
            SweepCommit::Create(input) => {
                Client::new(self.document_host())
                    .features()
                    .add_feature(input.as_ref())
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
