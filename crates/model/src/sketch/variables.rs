// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, circle::require_radius};

#[derive(Clone, Copy)]
pub(super) enum SketchVariable {
    Coordinate { point: SketchEntityId, axis: usize },
    Radius(SketchEntityId),
}
impl Sketch2D {
    pub(super) fn variable_values(&self) -> Vec<f64> {
        self.points
            .iter()
            .flat_map(|point| point.position)
            .chain(self.circles.iter().map(|circle| circle.radius))
            .collect()
    }
    pub(super) fn point_variables(&self, points: &[SketchEntityId]) -> Vec<SketchVariable> {
        points
            .iter()
            .flat_map(|point| {
                [
                    SketchVariable::Coordinate {
                        point: *point,
                        axis: 0,
                    },
                    SketchVariable::Coordinate {
                        point: *point,
                        axis: 1,
                    },
                ]
            })
            .collect()
    }
    pub(super) fn variable_column(&self, variable: SketchVariable) -> Result<usize, SketchError> {
        match variable {
            SketchVariable::Coordinate { point, axis } => self
                .points
                .iter()
                .position(|entry| entry.id == point)
                .map(|index| index * 2 + axis)
                .ok_or_else(|| {
                    SketchError::Definition(format!(
                        "sketch point {point:?}: expected an owned solver point"
                    ))
                }),
            SketchVariable::Radius(circle) => self
                .circles
                .iter()
                .position(|entry| entry.id == circle)
                .map(|index| self.points.len() * 2 + index)
                .ok_or_else(|| {
                    SketchError::Definition(format!(
                        "sketch circle {circle:?}: expected an owned solver radius"
                    ))
                }),
        }
    }
    pub(super) fn candidate_values(&self, values: &[f64]) -> Result<Self, SketchError> {
        let point_count = self.points.len() * 2;
        let expected = point_count + self.circles.len();
        if values.len() != expected {
            return Err(SketchError::Definition(format!(
                "sketch candidate has {} variables: expected {expected} coordinate/radius values",
                values.len()
            )));
        }
        let mut candidate = self.clone();
        for (point, coordinates) in candidate
            .points
            .iter_mut()
            .zip(values[..point_count].as_chunks::<2>().0)
        {
            point.position = *coordinates;
        }
        for (circle, radius) in candidate.circles.iter_mut().zip(&values[point_count..]) {
            require_radius(circle.id, *radius)?;
            circle.radius = *radius;
        }
        Ok(candidate)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn candidate_shape_domain_and_missing_variable_references_are_named_errors() {
        let mut sketch = Sketch2D::default();
        let center = SketchEntityId(1);
        let circle = SketchEntityId(2);
        sketch.add_point(center, [0.0, 0.0]).unwrap();
        sketch.add_circle(circle, center, 1.0).unwrap();
        assert!(
            sketch
                .candidate_values(&[])
                .unwrap_err()
                .to_string()
                .contains("expected 3")
        );
        assert!(
            sketch
                .candidate_values(&[0.0, 0.0, -1.0])
                .unwrap_err()
                .to_string()
                .contains("radius -1.0")
        );
        assert!(
            sketch
                .variable_column(SketchVariable::Radius(SketchEntityId(999)))
                .unwrap_err()
                .to_string()
                .contains("999")
        );
        assert!(
            sketch
                .variable_column(SketchVariable::Coordinate {
                    point: SketchEntityId(999),
                    axis: 0
                })
                .is_err()
        );
        assert_eq!(sketch.circles[0].radius, 1.0);
    }
}
