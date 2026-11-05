// SPDX-License-Identifier: GPL-2.0-only
use super::{
    Sketch2D, SketchDirectionInference, SketchEntityId, SketchError, SketchInferencePolicy,
};
use crate::sketch::{normalize_direction, point_difference};
use oblikovati_api::types::*;
use std::f64::consts::FRAC_PI_2;
pub(super) fn winner(
    sketch: &Sketch2D,
    endpoints: [[f64; 2]; 2],
    priority: ConstraintInferencePriority,
    policy: SketchInferencePolicy,
) -> Result<Option<SketchDirectionInference>, SketchError> {
    let Some(direction) = segment_direction(endpoints) else {
        return Ok(None);
    };
    let axial = axial_suggestion(direction, policy);
    let relative = relative_suggestion(sketch, direction, policy)?;
    Ok(choose_family(axial, relative, priority))
}
fn choose_family(
    axial: Option<SketchDirectionInference>,
    relative: Option<SketchDirectionInference>,
    priority: ConstraintInferencePriority,
) -> Option<SketchDirectionInference> {
    let (Some(axial), Some(relative)) = (axial, relative) else {
        return axial.or(relative);
    };
    if priority == PRIORITY_PARALLEL_PERPENDICULAR
        || (priority == PRIORITY_NONE && relative.score > axial.score)
    {
        return Some(relative);
    }
    Some(axial)
}
fn segment_direction(endpoints: [[f64; 2]; 2]) -> Option<[f64; 2]> {
    let first = [endpoints[0][0], endpoints[0][1], 0.0];
    let second = [endpoints[1][0], endpoints[1][1], 0.0];
    let displacement = point_difference(first, second);
    if displacement == [0.0; 3] {
        return None;
    }
    Some([displacement[0], displacement[1]])
}
fn axial_suggestion(
    direction: [f64; 2],
    policy: SketchInferencePolicy,
) -> Option<SketchDirectionInference> {
    let angle = direction[1].abs().atan2(direction[0].abs());
    classified_suggestion(angle, INFER_HORIZONTAL, INFER_VERTICAL, None, policy)
}
fn relative_suggestion(
    sketch: &Sketch2D,
    direction: [f64; 2],
    policy: SketchInferencePolicy,
) -> Result<Option<SketchDirectionInference>, SketchError> {
    let mut strongest: Option<SketchDirectionInference> = None;
    for line in sketch.lines() {
        let candidate = relative_for_line(sketch, line, direction, policy)?;
        if let Some(candidate) =
            candidate.filter(|candidate| strongest.is_none_or(|best| candidate.score > best.score))
        {
            strongest = Some(candidate);
        }
    }
    Ok(strongest)
}
fn relative_for_line(
    sketch: &Sketch2D,
    line: &crate::sketch::SketchLine2D,
    direction: [f64; 2],
    policy: SketchInferencePolicy,
) -> Result<Option<SketchDirectionInference>, SketchError> {
    let points = [
        sketch.point(line.endpoints[0])?.position,
        sketch.point(line.endpoints[1])?.position,
    ];
    let Some(reference) = segment_direction(points) else {
        return Ok(None);
    };
    Ok(classified_suggestion(
        relative_angle(direction, reference)?,
        INFER_PARALLEL,
        INFER_PERPENDICULAR,
        Some(line.id),
        policy,
    ))
}
fn relative_angle(first: [f64; 2], second: [f64; 2]) -> Result<f64, SketchError> {
    let first_squared = squared_length(first);
    let second_squared = squared_length(second);
    let denominator = first_squared.sqrt() * second_squared.sqrt();
    let dot = first[0] * second[0] + first[1] * second[1];
    // Retain Go aa8297d1's score quantization on conditioned inputs; normalize only on range loss (#3774).
    let cosine = if first_squared.is_normal()
        && second_squared.is_normal()
        && denominator.is_normal()
        && dot.is_finite()
    {
        dot / denominator
    } else {
        normalized_cosine(first, second)?
    };
    let angle = cosine.clamp(-1.0, 1.0).acos();
    Ok(angle.min(std::f64::consts::PI - angle))
}
fn squared_length(direction: [f64; 2]) -> f64 {
    direction[0] * direction[0] + direction[1] * direction[1]
}
fn normalized_cosine(first: [f64; 2], second: [f64; 2]) -> Result<f64, SketchError> {
    let first = normalize_direction([first[0], first[1], 0.0])?;
    let second = normalize_direction([second[0], second[1], 0.0])?;
    Ok(first[0] * second[0] + first[1] * second[1])
}
fn classified_suggestion(
    angle: f64,
    aligned: ConstraintInferenceKind,
    perpendicular: ConstraintInferenceKind,
    target: Option<SketchEntityId>,
    policy: SketchInferencePolicy,
) -> Option<SketchDirectionInference> {
    let (kind, miss) = if angle <= policy.angle_tolerance_radians {
        (aligned, angle)
    } else if FRAC_PI_2 - angle <= policy.angle_tolerance_radians {
        (perpendicular, FRAC_PI_2 - angle)
    } else {
        return None;
    };
    Some(SketchDirectionInference {
        kind,
        target,
        score: quality_score(miss, policy.angle_tolerance_radians),
    })
}
fn quality_score(miss: f64, tolerance: f64) -> u8 {
    // Go truncates this quality to an integer; family and target ties depend on that quantization.
    (100.0 * (1.0 - miss / tolerance)).clamp(0.0, 100.0) as u8
}
