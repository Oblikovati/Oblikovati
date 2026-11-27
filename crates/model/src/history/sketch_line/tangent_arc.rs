// SPDX-License-Identifier: GPL-2.0-only
//! An arc that leaves a line's end along the line's own direction.
use super::*;
use crate::sketch::{SketchEntityId, SketchError, SketchGeometricConstraintCreation};

/// The line and which of its ends the arc leaves, and where it arrives.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchTangentArcCreation {
    pub line: SketchEntityId,
    pub vertex: SketchEntityId,
    pub end: [f64; 2],
}
/// The arc, its new end vertex and the tangency relation that binds it to the line.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchTangentArcReport {
    pub sketch: DocumentSketchId,
    pub arc: SketchEntityId,
    pub end: SketchEntityId,
    pub constraint: SketchConstraintId,
}
/// The circle through the vertex and the end, tangent to the line at the vertex.
#[derive(Clone, Copy, Debug, PartialEq)]
pub(crate) struct TangentCircle {
    pub center: [f64; 2],
    pub counter_clockwise: bool,
}
/// Solve for the circle; an end straight along the line has no finite radius.
pub(crate) fn tangent_circle(
    from: [f64; 2],
    vertex: [f64; 2],
    end: [f64; 2],
) -> Result<TangentCircle, SketchError> {
    let along = [vertex[0] - from[0], vertex[1] - from[1]];
    let length = along[0].hypot(along[1]);
    let chord = [end[0] - vertex[0], end[1] - vertex[1]];
    let cross = (along[0] * chord[1] - along[1] * chord[0]) / length.max(f64::MIN_POSITIVE);
    if length == 0. || cross == 0. {
        return Err(SketchError::Definition(format!(
            "tangent arc from {vertex:?} along {along:?} to {end:?}: expected an end off the line"
        )));
    }
    let signed_radius = (chord[0] * chord[0] + chord[1] * chord[1]) / (2. * cross);
    let normal = [-along[1] / length, along[0] / length];
    Ok(TangentCircle {
        center: [
            vertex[0] + normal[0] * signed_radius,
            vertex[1] + normal[1] * signed_radius,
        ],
        counter_clockwise: signed_radius > 0.,
    })
}
impl DocumentSnapshot {
    /// Author one tangent arc against current solved geometry.
    /// Example: `snapshot.create_tangent_arc(sketch, &input, options, ids)?;`.
    pub fn create_tangent_arc(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchTangentArcCreation,
        solve_options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchTangentArcReport, SketchLineCreationError> {
        let mut geometry = self
            .current_sketch_authoring_geometry(sketch, solve_options)
            .map_err(SketchLineCreationError::Sketch)?;
        let circle = input
            .circle(&geometry)
            .map_err(SketchLineCreationError::Sketch)?;
        let primitive = SketchPrimitiveCreation::Arc {
            center: SketchPointInput::New(circle.center),
            endpoints: [
                SketchPointInput::Owned(input.vertex),
                SketchPointInput::New(input.end),
            ],
            counter_clockwise: circle.counter_clockwise,
            construction: false,
        };
        let report = append_primitive(&mut geometry, sketch, &primitive, ids)
            .map_err(SketchLineCreationError::Primitive)?;
        let constraint = ids
            .allocate_sketch_constraint(sketch)
            .map_err(SketchLineCreationError::Constraint)?;
        SketchGeometricConstraintCreation::LineTangent {
            line: input.line,
            curve: report.entity,
        }
        .append(&mut geometry, constraint)
        .map_err(SketchLineCreationError::Sketch)?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchLineCreationError::Sketch)?;
        Ok(SketchTangentArcReport {
            sketch,
            arc: report.entity,
            end: report.points[2],
            constraint,
        })
    }
}
impl SketchTangentArcCreation {
    fn circle(&self, geometry: &Sketch2D) -> Result<TangentCircle, SketchError> {
        let [first, second] = geometry.line(self.line)?.endpoints;
        let other = if first == self.vertex {
            second
        } else if second == self.vertex {
            first
        } else {
            return Err(SketchError::Definition(format!(
                "tangent arc vertex {:?} on line {:?} with ends {first:?} and {second:?}: expected one of its ends",
                self.vertex, self.line
            )));
        };
        tangent_circle(
            geometry.point(other)?.position,
            geometry.point(self.vertex)?.position,
            self.end,
        )
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn a_left_turn_is_counter_clockwise_and_its_centre_sits_on_the_normal() {
        let circle = tangent_circle([-1., 0.], [0., 0.], [1., 1.]).unwrap();
        assert!(circle.counter_clockwise);
        assert_eq!(circle.center, [0., 1.]);
    }
    #[test]
    fn a_right_turn_is_clockwise() {
        let circle = tangent_circle([-1., 0.], [0., 0.], [1., -1.]).unwrap();
        assert!(!circle.counter_clockwise);
        assert_eq!(circle.center, [0., -1.]);
    }
    #[test]
    fn an_end_straight_along_the_line_is_refused_with_both_points() {
        let cause = tangent_circle([-1., 0.], [0., 0.], [5., 0.])
            .unwrap_err()
            .to_string();
        assert!(cause.contains("expected an end off the line"), "{cause}");
    }
}
