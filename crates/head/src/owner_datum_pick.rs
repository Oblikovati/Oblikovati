// SPDX-License-Identifier: GPL-2.0-only
//! Pick analytic work-plane display bounds, not rasterized overlay triangles.
use super::*;
use crate::part_scene::pick_candidates::{PickCandidates, PickTarget};
use oblikovati_model::sketch::SketchPlane;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn append_visible_plane_hits(
        &self,
        document: u64,
        line: [[f64; 3]; 2],
        candidates: &mut PickCandidates,
    ) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        let snapshot = session.snapshot();
        let evaluated = self.part_scenes.current(session)?.datum_evaluation();
        candidates.include_visible_planes();
        for (id, frame) in evaluated.planes() {
            if let Some(reach) = pick_datum(snapshot, id, frame, line)? {
                candidates.push(PickTarget::WorkPlane(id), reach)?;
            }
        }
        Ok(())
    }
}

fn pick_datum(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    id: oblikovati_model::feature::FeatureId,
    frame: SketchPlane,
    line: [[f64; 3]; 2],
) -> Result<Option<f64>, String> {
    let feature = snapshot
        .work_geometry
        .get(id)
        .map_err(|error| error.to_string())?;
    if !feature.is_visible() {
        return Ok(None);
    }
    let Ok(corners) = super::datum_overlay::feature_corners(snapshot, feature, frame) else {
        return Ok(None); // The display path reports this same local decline.
    };
    if oblikovati_viewport::ViewportDatumPlane::validate_corners(&corners).is_err() {
        return Ok(None);
    }
    hit_visible_plane(frame, corners, line)
}

fn hit_visible_plane(
    frame: SketchPlane,
    corners: [[f64; 3]; 4],
    [origin, direction]: [[f64; 3]; 2],
) -> Result<Option<f64>, String> {
    let Some(local) = oblikovati_martensite::sketch_projection(
        frame,
        origin,
        direction,
        oblikovati_martensite::SKETCH_PROJECTION_MIN_NORMAL_COSINE,
    )?
    else {
        return Ok(None);
    };
    if !inside_display(frame, corners, local)? {
        return Ok(None);
    }
    let point = frame.to_model(local).map_err(|error| error.to_string())?;
    ray_reach(point, origin, direction)
}

fn ray_reach(
    point: [f64; 3],
    origin: [f64; 3],
    direction: [f64; 3],
) -> Result<Option<f64>, String> {
    let length = direction[0].hypot(direction[1]).hypot(direction[2]);
    if !length.is_finite() || length == 0. {
        return Err(format!(
            "pick direction {direction:?}: expected finite nonzero length"
        ));
    }
    let reach: f64 = (0..3)
        .map(|axis| (point[axis] - origin[axis]) * (direction[axis] / length))
        .sum();
    if !reach.is_finite() {
        return Err(format!(
            "plane hit {point:?}, ray {origin:?}/{direction:?}: expected finite distance"
        ));
    }
    Ok((reach >= 0.).then_some(reach))
}

fn inside_display(
    frame: SketchPlane,
    corners: [[f64; 3]; 4],
    point: [f64; 2],
) -> Result<bool, String> {
    let first = frame
        .to_sketch(corners[0])
        .map_err(|error| error.to_string())?;
    let last = frame
        .to_sketch(corners[2])
        .map_err(|error| error.to_string())?;
    Ok((0..2).all(|axis| {
        (first[axis].min(last[axis])..=first[axis].max(last[axis])).contains(&point[axis])
    }))
}

#[cfg(test)]
mod tests {
    use super::*;
    const CORNERS: [[f64; 3]; 4] = [[-5., -5., 0.], [5., -5., 0.], [5., 5., 0.], [-5., 5., 0.]];

    #[test]
    fn ray_hit_checks_bounds_direction_and_parallelism() {
        let plane = SketchPlane::xy();
        assert_eq!(
            hit_visible_plane(plane, CORNERS, [[1., 2., 10.], [0., 0., -2.]]).unwrap(),
            Some(10.)
        );
        assert_eq!(
            hit_visible_plane(plane, CORNERS, [[6., 2., 10.], [0., 0., -1.]]).unwrap(),
            None
        );
        assert_eq!(
            hit_visible_plane(plane, CORNERS, [[1., 2., 10.], [0., 0., 1.]]).unwrap(),
            None
        );
        assert_eq!(
            hit_visible_plane(plane, CORNERS, [[1., 2., 10.], [1., 0., 0.]]).unwrap(),
            None
        );
        assert!(hit_visible_plane(plane, CORNERS, [[1., 2., 10.], [0.; 3]]).is_err());
    }
}
