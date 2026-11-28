// SPDX-License-Identifier: GPL-2.0-only
//! Viewport planar-face and visible-sketch-center picking for Part Modify Hole (#4093).
use super::*;
use oblikovati_automation::PartUpdateAccess;
use oblikovati_gui::GuiHolePick;
use oblikovati_model::sketch::OwnedSketchGeometry;

type SketchHoleHit = (Vec<u8>, Vec<u8>, [f64; 3]);

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Resolve a viewport click while the Hole Properties panel is open.
    /// Prefers visible unconsumed sketch circle centers near the click ray (e.g. hole-11),
    /// and otherwise falls back to planar face selection on the current solid.
    pub(in crate::owner) fn pick_hole_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.hole.active {
            return Ok(None);
        }
        let editing = self
            .create_features
            .hole
            .preview
            .as_ref()
            .and_then(|preview| preview.request.feature)
            .is_some();
        if editing {
            return Ok(None);
        }
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        if let Some((body, face, center)) = self.hit_sketch_hole_center(document, ray, tolerance)? {
            self.create_features.hole.pick = Some(GuiHolePick {
                document,
                generation,
                body,
                face,
                position: Some(center),
            });
            return Ok(Some(true));
        }
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
        let body = self.resolve_hole_body_for_face(document, &key)?;
        self.create_features.hole.pick = Some(GuiHolePick {
            document,
            generation,
            body,
            face: key,
            position: None,
        });
        Ok(Some(true))
    }

    fn hit_sketch_hole_center(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<SketchHoleHit>, String> {
        let datums = self
            .part_scenes
            .datum_evaluation(&mut self.files, document)?;
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let snapshot = session.snapshot();
        let mut best: Option<(f64, [f64; 3])> = None;
        for sketch in snapshot.sketches.sketches() {
            if !sketch.visible() {
                continue;
            }
            let Ok(resolved) =
                snapshot.require_converged_resolved_sketch_with_datums(sketch.id(), &datums)
            else {
                continue;
            };
            let geometry = resolved.sketch.geometry();
            let index = geometry.read_index();
            for id in index.entity_ids().iter().copied() {
                let Ok(OwnedSketchGeometry::Circle(circle)) = index.entity(id) else {
                    continue;
                };
                let Ok(center_pt) = geometry.point(circle.center) else {
                    continue;
                };
                let Ok(center_3d) = resolved.plane.to_model(center_pt.position) else {
                    continue;
                };
                let dist = point_to_ray_distance(center_3d, ray);
                let limit = (circle.radius * 1.5).max(tolerance.unwrap_or(0.5)).max(0.5);
                if dist <= limit && best.as_ref().is_none_or(|(d, _)| dist < *d) {
                    best = Some((dist, center_3d));
                }
            }
        }
        let Some((_, center)) = best else {
            return Ok(None);
        };
        let evaluation = self.part_scenes.current(session)?;
        for body in evaluation.ordered_bodies() {
            let Ok(body_key) = body.body_reference_key(session.history(), guid) else {
                continue;
            };
            let Ok(face_keys) = body.face_reference_keys(session.history(), guid) else {
                continue;
            };
            let Some(resolution) = <crate::part_scene::DesktopPartScenes as PartUpdateAccess<
                O,
                W,
                F,
            >>::resolution_policy(&self.part_scenes) else {
                continue;
            };
            let res = match resolution {
                oblikovati_martensite::PrismResolutionPolicy::Fixed(r)
                | oblikovati_martensite::PrismResolutionPolicy::ModelRelative {
                    reference: r,
                    ..
                } => r,
            };
            for face_key in face_keys {
                if body
                    .hole_axis(session.history(), guid, &face_key, Some(center), res)
                    .is_ok()
                {
                    return Ok(Some((body_key.to_vec(), face_key, center)));
                }
            }
        }
        Ok(None)
    }
}

fn point_to_ray_distance(point: [f64; 3], ray: [[f64; 3]; 2]) -> f64 {
    let [origin, dir] = ray;
    let dlen = (dir[0] * dir[0] + dir[1] * dir[1] + dir[2] * dir[2]).sqrt();
    if dlen <= 1e-12 {
        return f64::INFINITY;
    }
    let u = [dir[0] / dlen, dir[1] / dlen, dir[2] / dlen];
    let v = [
        point[0] - origin[0],
        point[1] - origin[1],
        point[2] - origin[2],
    ];
    let t = v[0] * u[0] + v[1] * u[1] + v[2] * u[2];
    let proj = [
        origin[0] + u[0] * t,
        origin[1] + u[1] * t,
        origin[2] + u[2] * t,
    ];
    let dx = point[0] - proj[0];
    let dy = point[1] - proj[1];
    let dz = point[2] - proj[2];
    (dx * dx + dy * dy + dz * dz).sqrt()
}
