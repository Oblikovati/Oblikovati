// SPDX-License-Identifier: GPL-2.0-only
//! Viewport geometry picking (cutting plane/surface and surface removal region) for Surface Trim (#3918).
use crate::owner::*;
use oblikovati_gui::{GuiSurfaceTrimPick, GuiSurfaceTrimPickKind, GuiSurfaceTrimPickMode};
use oblikovati_martensite::{DisplayTriangle, NamedSolid};
use oblikovati_model::{
    feature::{
        FeatureId,
        hole::HoleOriginPlane,
        surface_trim::{ResolvedSurfaceTrimTool, SurfaceTrimSide, SurfaceTrimTool},
        work_plane::WorkPlaneResolver,
    },
    sketch::SketchPlane,
};
use oblikovati_viewport::{ViewportScene, ViewportTriangle};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in crate::owner) fn pick_surface_trim_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.surface_trim.active {
            return Ok(None);
        }
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        let kind = match self.create_features.surface_trim.pick_mode {
            GuiSurfaceTrimPickMode::CuttingTool => {
                self.pick_surface_trim_tool_on_ray(document, ray, tolerance)?
            }
            GuiSurfaceTrimPickMode::Remove => {
                self.pick_surface_trim_region_on_ray(document, ray, tolerance)?
            }
        };
        let Some(kind) = kind else {
            return Ok(None);
        };
        let state = &mut self.create_features.surface_trim;
        state.pick_seq = state.pick_seq.saturating_add(1);
        state.pick = Some(GuiSurfaceTrimPick {
            document,
            generation,
            sequence: state.pick_seq,
            kind,
        });
        Ok(Some(true))
    }

    fn pick_surface_trim_tool_on_ray(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<GuiSurfaceTrimPickKind>, String> {
        let plane_filter = oblikovati_session::selection::SelectionFilter::only([
            oblikovati_session::selection::SelectionKind::WorkPlane,
        ]);
        let mut candidates = crate::part_scene::pick_candidates::PickCandidates::new(
            crate::part_scene::pick_candidates::PickCoverage::VisiblePlanesAndNearestFaces,
        );
        self.append_visible_plane_hits(document, ray, &mut candidates)?;
        if let Ok(Some(hit)) = candidates.resolve_hit(
            &plane_filter,
            &self.ambient_selection,
            tolerance.unwrap_or(0.0),
        ) && let crate::part_scene::pick_candidates::PickTarget::WorkPlane(feature) = hit.target
        {
            return Ok(Some(GuiSurfaceTrimPickKind::CuttingTool(
                SurfaceTrimTool::WorkPlane(feature),
            )));
        }
        let face_filter = oblikovati_session::selection::SelectionFilter::only([
            oblikovati_session::selection::SelectionKind::Face,
        ]);
        let face_candidates =
            self.part_scenes
                .part_candidates(&self.files, document, ray, &face_filter)?;
        let Ok(Some(hit)) = face_candidates.resolve_hit(
            &face_filter,
            &self.ambient_selection,
            tolerance.unwrap_or(0.0),
        ) else {
            return Ok(None);
        };
        let crate::part_scene::pick_candidates::PickTarget::Face(_, face) = hit.target else {
            return Ok(None);
        };
        let Some(body) = self.resolve_surface_owner_key(document, &face)? else {
            return Ok(None);
        };
        Ok(Some(GuiSurfaceTrimPickKind::CuttingTool(
            SurfaceTrimTool::SurfaceFace { body, face },
        )))
    }

    fn pick_surface_trim_region_on_ray(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<GuiSurfaceTrimPickKind>, String> {
        let Some(tool) = self.create_features.surface_trim.active_tool.clone() else {
            return Ok(None);
        };
        let (plane_origin, plane_normal) = self.resolve_trim_tool_plane(document, &tool)?;
        let face_filter = oblikovati_session::selection::SelectionFilter::only([
            oblikovati_session::selection::SelectionKind::Face,
        ]);
        let candidates =
            self.part_scenes
                .part_candidates(&self.files, document, ray, &face_filter)?;
        let Ok(Some(hit)) = candidates.resolve_hit(
            &face_filter,
            &self.ambient_selection,
            tolerance.unwrap_or(0.0),
        ) else {
            return Ok(None);
        };
        let crate::part_scene::pick_candidates::PickTarget::Face(_, face) = hit.target else {
            return Ok(None);
        };
        let Some(target_body) = self.resolve_surface_owner_key(document, &face)? else {
            return Ok(None);
        };
        let t = hit.line_parameter;
        let hit_point = [
            ray[0][0] + t * ray[1][0],
            ray[0][1] + t * ray[1][1],
            ray[0][2] + t * ray[1][2],
        ];
        let signed = (hit_point[0] - plane_origin[0]) * plane_normal[0]
            + (hit_point[1] - plane_origin[1]) * plane_normal[1]
            + (hit_point[2] - plane_origin[2]) * plane_normal[2];
        let remove_side = if signed >= 0.0 {
            SurfaceTrimSide::Positive
        } else {
            SurfaceTrimSide::Negative
        };
        Ok(Some(GuiSurfaceTrimPickKind::RemoveRegion {
            target_body,
            remove_side,
        }))
    }

    fn resolve_surface_owner_key(
        &self,
        document: u64,
        face: &[u8],
    ) -> Result<Option<Vec<u8>>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        for (_, body) in evaluation.ordered_body_outputs() {
            if !body.is_surface() {
                continue;
            }
            let Ok(keys) = body.face_reference_keys(session.history(), guid) else {
                continue;
            };
            if keys.iter().any(|k| k.as_slice() == face) {
                return Ok(Some(
                    body.body_reference_key(session.history(), guid)?.to_vec(),
                ));
            }
        }
        Ok(None)
    }

    fn resolve_trim_tool_plane(
        &self,
        document: u64,
        tool: &SurfaceTrimTool,
    ) -> Result<([f64; 3], [f64; 3]), String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        let plane = match tool {
            SurfaceTrimTool::OriginPlane(origin) => origin_plane(*origin),
            SurfaceTrimTool::WorkPlane(id) => evaluation
                .datum_evaluation()
                .plane(*id)
                .map_err(|cause| format!("surface trim work plane {id:?}: {cause}"))?,
            SurfaceTrimTool::SurfaceFace { body, face } => {
                let tool_body = evaluation
                    .ordered_bodies()
                    .find(|candidate| {
                        candidate
                            .body_reference_key(session.history(), guid)
                            .is_ok_and(|key| key == body.as_slice())
                    })
                    .ok_or_else(|| {
                        "surface trim tool face: expected active tool body".to_string()
                    })?;
                tool_body.face_plane(session.history(), guid, face)?
            }
        };
        Ok((plane.recipe().origin, plane.normal()))
    }
}

pub(super) fn find_body_by_key<'a>(
    eval: &'a oblikovati_model::feature::part::PartFeatureEvaluation<
        NamedSolid,
        oblikovati_martensite::EvaluationFailure,
    >,
    history: &oblikovati_model::history::DocumentEditHistory,
    guid: oblikovati_model::identity::DocumentGuid,
    target_key: &[u8],
) -> Result<(FeatureId, &'a NamedSolid), String> {
    eval.ordered_body_outputs()
        .find(|(_, body)| {
            body.body_reference_key(history, guid)
                .is_ok_and(|key| key == target_key)
        })
        .ok_or_else(|| "surface trim: expected target surface body in current evaluation".into())
}

pub(super) fn resolve_tool_in_eval<'a>(
    eval: &'a oblikovati_model::feature::part::PartFeatureEvaluation<
        NamedSolid,
        oblikovati_martensite::EvaluationFailure,
    >,
    history: &oblikovati_model::history::DocumentEditHistory,
    guid: oblikovati_model::identity::DocumentGuid,
    tool: &SurfaceTrimTool,
) -> Result<(Option<&'a NamedSolid>, ResolvedSurfaceTrimTool), String> {
    match tool {
        SurfaceTrimTool::OriginPlane(origin) => {
            Ok((None, ResolvedSurfaceTrimTool::Plane(origin_plane(*origin))))
        }
        SurfaceTrimTool::WorkPlane(id) => {
            let plane = eval
                .datum_evaluation()
                .plane(*id)
                .map_err(|cause| format!("surface trim work plane {id:?}: {cause}"))?;
            Ok((None, ResolvedSurfaceTrimTool::Plane(plane)))
        }
        SurfaceTrimTool::SurfaceFace { body, face } => {
            let (_, tool_body) = find_body_by_key(eval, history, guid, body)?;
            Ok((
                Some(tool_body),
                ResolvedSurfaceTrimTool::SurfaceFace {
                    face_key: face.clone(),
                },
            ))
        }
    }
}

pub(super) fn resolve_preview_inputs<'a>(
    eval: &'a oblikovati_model::feature::part::PartFeatureEvaluation<
        NamedSolid,
        oblikovati_martensite::EvaluationFailure,
    >,
    history: &oblikovati_model::history::DocumentEditHistory,
    guid: oblikovati_model::identity::DocumentGuid,
    tool: &SurfaceTrimTool,
    target_body: &[u8],
) -> Result<
    (
        &'a NamedSolid,
        Option<&'a NamedSolid>,
        ResolvedSurfaceTrimTool,
    ),
    String,
> {
    let (_, surface) = find_body_by_key(eval, history, guid, target_body)?;
    let (tool_body, resolved_tool) = resolve_tool_in_eval(eval, history, guid, tool)?;
    Ok((surface, tool_body, resolved_tool))
}

pub(super) fn origin_plane(plane: HoleOriginPlane) -> SketchPlane {
    match plane {
        HoleOriginPlane::Xy => SketchPlane::xy(),
        HoleOriginPlane::Xz => SketchPlane::xz(),
        HoleOriginPlane::Yz => SketchPlane::yz(),
    }
}

pub(super) fn append_display_triangles(
    scene: &mut ViewportScene,
    triangles: &[DisplayTriangle],
    color: [f32; 4],
) {
    for tri in triangles {
        scene.triangles.push(ViewportTriangle {
            positions: tri.positions.map(|p| p.map(|c| c as f32)),
            normals: tri.normals.map(|ns| ns.map(|n| n.map(|c| c as f32))),
            color,
            face_index: None,
        });
    }
}
