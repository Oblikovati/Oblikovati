// SPDX-License-Identifier: GPL-2.0-only
use super::{
    Sketch2D, SketchEndpointInference, SketchInferencePolicy, SketchLinePlacementInference,
};
use crate::sketch::SketchPoint2D;
pub(super) fn propose(
    sketch: &Sketch2D,
    report: &mut SketchLinePlacementInference,
    constrain: bool,
    policy: SketchInferencePolicy,
) {
    for endpoint in 0..2 {
        let Some(target) = nearest_point(
            sketch.points(),
            report.candidate_endpoints[endpoint],
            policy.snap_distance_working,
        ) else {
            continue;
        };
        report.points.push(SketchEndpointInference {
            endpoint,
            target: target.id,
        });
        if constrain {
            report.candidate_endpoints[endpoint] = target.position;
        }
    }
}
fn nearest_point(
    points: &[SketchPoint2D],
    position: [f64; 2],
    reach: f64,
) -> Option<SketchPoint2D> {
    let mut nearest = None;
    let mut distance = reach;
    for point in points {
        let separation = (position[0] - point.position[0]).hypot(position[1] - point.position[1]);
        // Go chooses the last authored point at equal distance; identity order does not replace it.
        if separation <= distance {
            nearest = Some(*point);
            distance = separation;
        }
    }
    nearest
}
