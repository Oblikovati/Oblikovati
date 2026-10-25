// SPDX-License-Identifier: GPL-2.0-only
use super::super::{PlanarCurveFailure, circumcircle::classify_turn};
use martensite_tkmath::predicates::OrientationSign;
pub(super) fn classify(points: [[f64; 2]; 3]) -> Result<bool, PlanarCurveFailure> {
    let sign = classify_turn(points)?;
    if sign == OrientationSign::Zero && !opposite_rays(points) {
        return Err(PlanarCurveFailure::input(format!(
            "arc center/start/end {points:?}: expected two nonzero distinct radial directions"
        )));
    }
    // Go aa8297d1 leftTurn: a semicircle uses clockwise winding. No rounded cross product.
    Ok(sign == OrientationSign::Positive)
}
fn opposite_rays([center, start, end]: [[f64; 2]; 3]) -> bool {
    // Exact collinearity is already established. An ordered coordinate distinguishes the rays
    // without subtracting values that can overflow or round an exact predicate to zero.
    let axis = usize::from(start[0] == center[0]);
    (start[axis] > center[axis] && end[axis] < center[axis])
        || (start[axis] < center[axis] && end[axis] > center[axis])
}
