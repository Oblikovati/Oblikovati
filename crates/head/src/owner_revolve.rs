// SPDX-License-Identifier: GPL-2.0-only
//! Preview, viewport axis/profile picking, and commit for the desktop Revolution panel.
#[path = "owner_revolve_eval.rs"]
mod eval;
use eval::revolve_definition;

use super::*;
use crate::owner::owner_blend::owner_sweep::ray_hits_segment;
use oblikovati_automation::{PartUpdateAccess, PreparedRevolve, automatic_revolve_profile};
use oblikovati_gui::{
    FeatureOutputMode, GuiRevolveAction, GuiRevolveEditSeed, GuiRevolveExtent, GuiRevolvePreview,
    GuiRevolvePreviewState, GuiRevolveRequest,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId, part::PartFeatureDefinition, sketch_prism::PrismBoolean,
        sketch_revolve::SketchRevolveDefinition, work_geometry::DatumAngle,
    },
    history::DocumentSnapshot,
    sketch::{OwnedSketchGeometry, SketchEntityId},
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum RevolveCommit {
    Create(Box<PreparedRevolve>),
    Edit(FeatureId, SketchRevolveDefinition, Vec<(u64, String)>),
}

enum RevolveOutcome {
    Ready(RevolveCommit),
    UnchangedCut(String),
}

pub(in crate::owner) struct RevolvePreview {
    request: GuiRevolveRequest,
    result: Result<Option<RevolveOutcome>, String>,
    profile_curves: usize,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiRevolveEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::SketchRevolve(definition) = feature.definition() else {
                return None;
            };
            let extent = if definition.is_full() {
                GuiRevolveExtent::Full
            } else {
                GuiRevolveExtent::Angle
            };
            let owned = feature.owned_scalars();
            let angle = if definition.is_full() {
                "360 deg".into()
            } else {
                angle_field(snapshot, owned, Some(definition.angle()))?
            };
            let second_angle = definition
                .second_angle()
                .and_then(|value| angle_field(snapshot, owned, Some(value)))
                .unwrap_or_else(|| "45,00 deg".into());
            Some((
                feature.id(),
                GuiRevolveEditSeed {
                    angle,
                    second_angle,
                    extent,
                    direction: definition.direction(),
                    output: saved_output(definition),
                    axis: definition.axis(),
                },
            ))
        })
        .collect()
}

fn saved_output(definition: &SketchRevolveDefinition) -> FeatureOutputMode {
    if definition.is_surface() {
        return FeatureOutputMode::Surface;
    }
    match definition.boolean_target().map(|(op, _, _)| op) {
        Some(PrismBoolean::Join) => FeatureOutputMode::Join,
        Some(PrismBoolean::Cut) => FeatureOutputMode::Cut,
        Some(PrismBoolean::Intersect) => FeatureOutputMode::Intersect,
        None => FeatureOutputMode::NewSolid,
    }
}

fn angle_field(
    snapshot: &DocumentSnapshot,
    owned: &[oblikovati_model::feature::FeatureScalarBinding],
    angle: Option<DatumAngle>,
) -> Option<String> {
    let angle = angle?;
    match angle.parameter_id() {
        Some(id) => {
            let parameter = snapshot.parameters.get(id)?;
            if owned.iter().any(|binding| binding.parameter == id) {
                Some(parameter.source.clone())
            } else {
                Some(parameter.name.clone())
            }
        }
        None => {
            let degrees = angle.resolve(&snapshot.parameters).ok()?.to_degrees();
            Some(format!("{degrees:.2} deg").replace(".00 deg", " deg"))
        }
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
    pub(in crate::owner) fn gui_revolve_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiRevolvePreviewState> {
        let preview = self.revolve_preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiRevolvePreviewState {
                request: preview.request.clone(),
                profile_curves: preview.profile_curves,
                result: match &preview.result {
                    Ok(Some(RevolveOutcome::Ready(_))) => GuiRevolvePreview::Ready,
                    Ok(Some(RevolveOutcome::UnchangedCut(cause))) => {
                        GuiRevolvePreview::UnchangedCut(cause.clone())
                    }
                    Ok(None) if preview.request.profile_cleared => {
                        GuiRevolvePreview::MissingProfile
                    }
                    Ok(None) if preview.request.extent == GuiRevolveExtent::ToReference => {
                        GuiRevolvePreview::MissingToFace
                    }
                    Ok(None) if preview.request.axis.is_none() => GuiRevolvePreview::MissingAxis,
                    Ok(None) => GuiRevolvePreview::MissingInputs,
                    Err(cause) => GuiRevolvePreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn revolve_command(
        &mut self,
        action: GuiRevolveAction,
    ) -> Result<bool, String> {
        match action {
            GuiRevolveAction::Open | GuiRevolveAction::OpenEdit(_) => Ok(false),
            GuiRevolveAction::Clear => {
                Ok(self.revolve_preview.take().is_some() | self.clear_extrusion_preview())
            }
            GuiRevolveAction::Preview(request) => {
                self.require_revolve_generation(&request)?;
                let (profile_curves, result) = self.preview_revolve(&request);
                self.revolve_preview = Some(RevolvePreview {
                    request,
                    result,
                    profile_curves,
                });
                Ok(true)
            }
            GuiRevolveAction::Commit(request) => {
                self.require_revolve_generation(&request)?;
                let commit = self
                    .revolve_preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(Option::as_ref)
                    .and_then(|outcome| match outcome {
                        RevolveOutcome::Ready(commit) => Some(commit),
                        RevolveOutcome::UnchangedCut(_) => None,
                    })
                    .ok_or("revolve: expected a current certified preview")?
                    .clone();
                self.commit_revolve(request.document, commit)?;
                self.revolve_preview = None;
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    pub(in crate::owner) fn pick_revolve_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        let Some(mut request) = self
            .revolve_preview
            .as_ref()
            .map(|preview| preview.request.clone())
            .filter(|req| req.document == document)
        else {
            return Ok(None);
        };
        let Some(tolerance) = tolerance else {
            return Ok(None);
        };
        if !request.profile_cleared && request.axis.is_some() {
            return Ok(None);
        }
        let Some(sketch_id) = self.revolve_source_sketch(&request)? else {
            return Ok(Some(false));
        };
        let Some(line_id) = self.hit_sketch_line(document, sketch_id, ray, tolerance)? else {
            return Ok(Some(false));
        };
        if request.profile_cleared {
            request.profile_cleared = false;
        } else {
            request.axis = Some(line_id);
        }
        self.revolve_command(GuiRevolveAction::Preview(request))
            .map(Some)
    }

    fn hit_sketch_line(
        &mut self,
        document: u64,
        sketch_id: DocumentSketchId,
        ray: [[f64; 3]; 2],
        tolerance: f64,
    ) -> Result<Option<SketchEntityId>, String> {
        let datums = self
            .part_scenes
            .datum_evaluation(&mut self.files, document)?;
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let resolved = snapshot
            .require_converged_resolved_sketch_with_datums(sketch_id, &datums)
            .map_err(|cause| cause.to_string())?;
        let geometry = resolved.sketch.geometry();
        let index = geometry.read_index();
        for id in index.entity_ids().iter().copied() {
            let Ok(OwnedSketchGeometry::Line(line)) = index.entity(id) else {
                continue;
            };
            let Ok(p0) = geometry.point(line.endpoints[0]) else {
                continue;
            };
            let Ok(p1) = geometry.point(line.endpoints[1]) else {
                continue;
            };
            let (Ok(start), Ok(end)) = (
                resolved.plane.to_model(p0.position),
                resolved.plane.to_model(p1.position),
            ) else {
                continue;
            };
            if ray_hits_segment(ray, [start, end], tolerance) {
                return Ok(Some(id));
            }
        }
        Ok(None)
    }

    fn revolve_source_sketch(
        &mut self,
        request: &GuiRevolveRequest,
    ) -> Result<Option<DocumentSketchId>, String> {
        let document = request.document;
        if let Some(feature) = request.feature {
            let snapshot = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?
                .snapshot();
            return Ok(Some(revolve_definition(snapshot, feature)?.sketch()));
        }
        let policy =
            <crate::part_scene::DesktopPartScenes as PartUpdateAccess<O, W, F>>::resolution_policy(
                &self.part_scenes,
            )
            .ok_or("revolve: expected a certified resolution policy")?;
        let datums = self
            .part_scenes
            .datum_evaluation(&mut self.files, document)?;
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        Ok(automatic_revolve_profile(snapshot, &datums, policy).map(|(sketch, _)| sketch))
    }

    fn require_revolve_generation(&self, request: &GuiRevolveRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("revolve: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_revolve(
        &mut self,
        request: &GuiRevolveRequest,
    ) -> (usize, Result<Option<RevolveOutcome>, String>) {
        if request.profile_cleared
            || request.extent == GuiRevolveExtent::ToReference
            || request.axis.is_none()
        {
            if let Some(feature) = request.feature
                && request.profile_cleared
                && let Ok(scene) = self.part_scenes.part_scene_without_feature_body(
                    &self.files,
                    request.document,
                    feature,
                )
            {
                self.extrusion_preview = Some((
                    request.document,
                    request.generation,
                    scene,
                    super::super::owner_extrusion_preview::PreviewKind::Replacement,
                ));
                self.part_scenes.invalidate_presentation();
            } else if request.feature.is_none() {
                self.clear_preview_mesh();
            }
            let curves = self.revolve_profile_curve_count(request).unwrap_or(4);
            return (curves, Ok(None));
        }
        if let Some(feature) = request.feature {
            let curves = self.revolve_profile_curve_count(request).unwrap_or(4);
            return (
                curves,
                self.preview_revolve_edit(request, feature).map(Some),
            );
        }
        match self.preview_revolve_create(request) {
            Ok((curves, outcome)) => (curves, Ok(outcome)),
            Err(cause) => {
                // Keep preceding valid preview visible while the invalid angle indicator is shown.
                let curves = self.revolve_profile_curve_count(request).unwrap_or(4);
                (curves, Err(cause))
            }
        }
    }

    fn clear_preview_mesh(&mut self) {
        if self.extrusion_preview.take().is_some() {
            self.part_scenes.invalidate_presentation();
        }
    }

    fn revolve_profile_curve_count(&mut self, request: &GuiRevolveRequest) -> Option<usize> {
        let document = request.document;
        if let Some(feature) = request.feature {
            let snapshot = self.files.workspace().session(document).ok()?.snapshot();
            let def = revolve_definition(snapshot, feature).ok()?;
            return Some(def.selection().outer.len());
        }
        let policy =
            <crate::part_scene::DesktopPartScenes as PartUpdateAccess<O, W, F>>::resolution_policy(
                &self.part_scenes,
            )?;
        let datums = self
            .part_scenes
            .datum_evaluation(&mut self.files, document)
            .ok()?;
        let snapshot = self.files.workspace().session(document).ok()?.snapshot();
        automatic_revolve_profile(snapshot, &datums, policy).map(|(_, sel)| sel.outer.len())
    }
}
