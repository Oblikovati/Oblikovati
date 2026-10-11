// SPDX-License-Identifier: GPL-2.0-only
use super::{
    PlanarConstraint, Sketch2D, SketchEntityId, SketchError,
    circular_frame::CircularFrame,
    circular_projection::{CircularProjection, FrameOutput},
    residual::{PlanarEquation, PreparedConstraint},
};
impl Sketch2D {
    pub(super) fn relation_frames(
        &self,
        relation: &PlanarConstraint,
    ) -> Result<Vec<CircularFrame>, SketchError> {
        let curves = match relation {
            PlanarConstraint::Radius { curve, .. }
            | PlanarConstraint::Diameter { curve, .. }
            | PlanarConstraint::PointOnCircularCurve { curve, .. }
            | PlanarConstraint::LineTangent { curve, .. }
            | PlanarConstraint::TangentDistance { curve, .. } => vec![*curve],
            PlanarConstraint::Concentric { curves }
            | PlanarConstraint::EqualRadius { curves }
            | PlanarConstraint::CircularTangent { curves, .. } => curves.to_vec(),
            _ => Vec::new(),
        };
        curves
            .into_iter()
            .map(|curve| self.circular_frame(curve))
            .collect()
    }
}
pub(super) fn prepare_relation(
    sketch: &Sketch2D,
    relation: &PlanarConstraint,
    equation: PlanarEquation,
) -> Result<PreparedConstraint, SketchError> {
    let (equation, touch) = contact_equation(sketch, relation, equation)?;
    let (projection, variables) = relation_projection(sketch, relation, touch)?;
    Ok(PreparedConstraint {
        equation,
        projection,
        variables,
    })
}
pub(super) fn relation_projection(
    sketch: &Sketch2D,
    relation: &PlanarConstraint,
    touch: Option<SketchEntityId>,
) -> Result<(Option<CircularProjection>, Vec<usize>), SketchError> {
    let points = sketch.constraint_points(relation)?;
    let frames = sketch.relation_frames(relation)?;
    let variables = frame_columns(sketch, &points, &frames, touch)?;
    let projection = (!frames.is_empty()).then(|| CircularProjection {
        prefix: points.len() * 2,
        frames: frames.iter().map(|frame| frame.radius).collect(),
        output: frame_output(relation),
        touch: touch.is_some(),
    });
    Ok((projection, variables))
}

fn frame_columns(
    sketch: &Sketch2D,
    points: &[SketchEntityId],
    frames: &[CircularFrame],
    touch: Option<SketchEntityId>,
) -> Result<Vec<usize>, SketchError> {
    let mut variables = sketch.point_variables(points);
    for frame in frames {
        variables.extend(sketch.frame_variables(*frame));
    }
    if let Some(touch) = touch {
        variables.extend(sketch.point_variables(&[touch]));
    }
    variables
        .into_iter()
        .map(|variable| sketch.variable_column(variable))
        .collect()
}
fn frame_output(relation: &PlanarConstraint) -> FrameOutput {
    match relation {
        PlanarConstraint::Radius { .. }
        | PlanarConstraint::Diameter { .. }
        | PlanarConstraint::EqualRadius { .. } => FrameOutput::Radii,
        PlanarConstraint::Concentric { .. } => FrameOutput::Centers,
        _ => FrameOutput::Frames,
    }
}
fn contact_equation(
    sketch: &Sketch2D,
    relation: &PlanarConstraint,
    equation: PlanarEquation,
) -> Result<(PlanarEquation, Option<SketchEntityId>), SketchError> {
    match relation {
        PlanarConstraint::LineTangent { line, curve, side } => {
            let Some(at_b) = sketch.shared_line_contact(*line, *curve)? else {
                return Ok((equation, None));
            };
            Ok((
                PlanarEquation::PinnedLineTangent { at_b, side: *side },
                None,
            ))
        }
        PlanarConstraint::CircularTangent { curves, contact } => {
            let Some(touch) = sketch.shared_circular_contact(*curves)? else {
                return Ok((equation, None));
            };
            Ok((PlanarEquation::PinnedCircularTangent(*contact), Some(touch)))
        }
        _ => Ok((equation, None)),
    }
}
