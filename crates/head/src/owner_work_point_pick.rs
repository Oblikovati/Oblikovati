// SPDX-License-Identifier: GPL-2.0-only
//! Analytic solid vertex and circular edge-loop viewport picking for Work Point (#3918).
use super::super::*;
use oblikovati_gui::GuiWorkPointMode;
use oblikovati_model::{
    document_sketch::SketchFrameTopologyOrigin,
    feature::{
        work_geometry::WorkGeometryDefinition,
        work_point::{WorkPointConstruction, WorkPointDefinition},
    },
    identity::ReferenceEnvelope,
};

struct PickedSolidVertex {
    key: Vec<u8>,
    position: [f64; 3],
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
    pub(in crate::owner) fn pick_work_point_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        depth_window: Option<f64>,
    ) -> Result<Option<bool>, String> {
        let Some(session) = self.create_features.work_point.session.as_ref() else {
            return Ok(None);
        };
        if session.document != document {
            return Ok(None);
        }
        if session.grounded.is_some() {
            return Ok(Some(false));
        }
        match session.mode {
            GuiWorkPointMode::OnVertex => self.commit_vertex_pick(document, ray, depth_window),
            GuiWorkPointMode::Grounded => {
                self.stage_grounded_vertex_pick(document, ray, depth_window)
            }
            GuiWorkPointMode::LoopCenter => self.commit_loop_pick(document, ray, depth_window),
            GuiWorkPointMode::General => {
                if self.commit_vertex_pick(document, ray, depth_window)? == Some(true) {
                    return Ok(Some(true));
                }
                self.commit_loop_pick(document, ray, depth_window)
            }
            GuiWorkPointMode::TorusCenter
            | GuiWorkPointMode::SphereCenter
            | GuiWorkPointMode::TwoLines
            | GuiWorkPointMode::PlaneAndLine
            | GuiWorkPointMode::ThreePlanes => Ok(Some(false)),
        }
    }

    fn commit_vertex_pick(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        depth_window: Option<f64>,
    ) -> Result<Option<bool>, String> {
        let Some(picked) = self.pick_solid_vertex_on_ray(document, ray, depth_window)? else {
            return Ok(Some(false));
        };
        let origin = SketchFrameTopologyOrigin::new(picked.key)?;
        let def = WorkPointDefinition::new(WorkPointConstruction::OnVertex(origin))
            .map_err(|cause| cause.to_string())?;
        self.commit_work_point(document, WorkGeometryDefinition::WorkPoint(Box::new(def)))?;
        self.finish_work_point_session();
        Ok(Some(true))
    }

    fn stage_grounded_vertex_pick(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        depth_window: Option<f64>,
    ) -> Result<Option<bool>, String> {
        let Some(picked) = self.pick_solid_vertex_on_ray(document, ray, depth_window)? else {
            return Ok(Some(false));
        };
        if let Some(active) = self.create_features.work_point.session.as_mut() {
            active.grounded = Some(picked.position);
            let prompt = active.prompt();
            self.command_notice = Some(DesktopCommandNotice::PlacementPrompt { message: prompt });
        }
        Ok(Some(true))
    }

    fn commit_loop_pick(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        depth_window: Option<f64>,
    ) -> Result<Option<bool>, String> {
        let Some(key) = self.pick_circular_edge_on_ray(document, ray, depth_window)? else {
            return Ok(Some(false));
        };
        let origin = SketchFrameTopologyOrigin::new(key)?;
        let def = WorkPointDefinition::new(WorkPointConstruction::LoopCenter(origin))
            .map_err(|cause| cause.to_string())?;
        self.commit_work_point(document, WorkGeometryDefinition::WorkPoint(Box::new(def)))?;
        self.finish_work_point_session();
        Ok(Some(true))
    }

    fn pick_solid_vertex_on_ray(
        &self,
        document: u64,
        ray: [[f64; 3]; 2],
        depth_window: Option<f64>,
    ) -> Result<Option<PickedSolidVertex>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        let tolerance = depth_window.unwrap_or(0.25).max(0.25);
        let mut best: Option<(f64, f64, PickedSolidVertex)> = None;
        for (_, body) in evaluation.ordered_body_outputs() {
            let Ok(keys) = body.subentity_reference_keys(session.history(), guid) else {
                continue;
            };
            for key in keys {
                let Ok(envelope) = ReferenceEnvelope::decode(&key) else {
                    continue;
                };
                if envelope.kind != 3 {
                    continue;
                }
                let Ok(position) = body.frame_origin(session.history(), guid, &key) else {
                    continue;
                };
                let Some((dist, reach)) = ray_point_distance(ray, position) else {
                    continue;
                };
                if dist <= tolerance
                    && best.as_ref().is_none_or(|(d, r, _)| {
                        dist < *d - 1.0e-6 || ((dist - *d).abs() <= 1.0e-6 && reach < *r)
                    })
                {
                    best = Some((dist, reach, PickedSolidVertex { key, position }));
                }
            }
        }
        Ok(best.map(|(_, _, picked)| picked))
    }

    fn pick_circular_edge_on_ray(
        &self,
        document: u64,
        ray: [[f64; 3]; 2],
        depth_window: Option<f64>,
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
        let tolerance = depth_window.unwrap_or(0.25).max(0.25);
        let mut best: Option<(f64, Vec<u8>)> = None;
        for (_, body) in evaluation.ordered_body_outputs() {
            let Ok(hits) = body.edge_hits_on_ray(session.history(), guid, ray, tolerance) else {
                continue;
            };
            for hit in hits {
                if hit.distance > tolerance {
                    continue;
                }
                if body.frame_origin(session.history(), guid, &hit.key).is_ok()
                    && best.as_ref().is_none_or(|(reach, _)| hit.reach < *reach)
                {
                    best = Some((hit.reach, hit.key));
                }
            }
        }
        Ok(best.map(|(_, key)| key))
    }
}

pub(super) fn ray_point_distance(ray: [[f64; 3]; 2], point: [f64; 3]) -> Option<(f64, f64)> {
    let [origin, direction] = ray;
    let length = direction[0].hypot(direction[1]).hypot(direction[2]);
    if !length.is_finite() || length <= 1.0e-12 {
        return None;
    }
    let dir = [
        direction[0] / length,
        direction[1] / length,
        direction[2] / length,
    ];
    let rel = [
        point[0] - origin[0],
        point[1] - origin[1],
        point[2] - origin[2],
    ];
    let reach = rel[0] * dir[0] + rel[1] * dir[1] + rel[2] * dir[2];
    if !reach.is_finite() || reach < 0.0 {
        return None;
    }
    let dx = rel[0] - reach * dir[0];
    let dy = rel[1] - reach * dir[1];
    let dz = rel[2] - reach * dir[2];
    let dist = dx.hypot(dy).hypot(dz);
    dist.is_finite().then_some((dist, reach))
}

pub(super) fn ready_construction(
    active: &super::WorkPointSession,
) -> Option<WorkPointConstruction> {
    match active.mode {
        GuiWorkPointMode::ThreePlanes if active.planes.len() == 3 => {
            Some(WorkPointConstruction::ThreePlanes([
                active.planes[0].clone(),
                active.planes[1].clone(),
                active.planes[2].clone(),
            ]))
        }
        GuiWorkPointMode::TwoLines if active.axes.len() == 2 => {
            Some(WorkPointConstruction::TwoLines {
                first: active.axes[0],
                second: active.axes[1],
            })
        }
        GuiWorkPointMode::PlaneAndLine | GuiWorkPointMode::General
            if !active.planes.is_empty() && !active.axes.is_empty() =>
        {
            Some(WorkPointConstruction::PlaneAndLine {
                plane: active.planes[0].clone(),
                line: active.axes[0],
            })
        }
        GuiWorkPointMode::General if active.planes.len() == 3 => {
            Some(WorkPointConstruction::ThreePlanes([
                active.planes[0].clone(),
                active.planes[1].clone(),
                active.planes[2].clone(),
            ]))
        }
        GuiWorkPointMode::General if active.axes.len() == 2 => {
            Some(WorkPointConstruction::TwoLines {
                first: active.axes[0],
                second: active.axes[1],
            })
        }
        _ => None,
    }
}

pub(super) fn origin_plane(
    kind: oblikovati_gui::GuiOriginPlaneKind,
) -> oblikovati_model::feature::work_plane::WorkPlaneOrigin {
    use oblikovati_model::feature::work_plane::WorkPlaneOrigin;
    match kind {
        oblikovati_gui::GuiOriginPlaneKind::Xy => WorkPlaneOrigin::XY,
        oblikovati_gui::GuiOriginPlaneKind::Xz => WorkPlaneOrigin::XZ,
        oblikovati_gui::GuiOriginPlaneKind::Yz => WorkPlaneOrigin::YZ,
    }
}

pub(super) fn next_work_point_name(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
) -> String {
    let taken = |name: &str| {
        snapshot
            .work_geometry
            .features()
            .iter()
            .any(|feature| feature.name() == name)
    };
    for number in 1_u32.. {
        let candidate = format!("Work Point{number}");
        if !taken(&candidate) {
            return candidate;
        }
    }
    "Work Point".into()
}
