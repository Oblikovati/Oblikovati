// SPDX-License-Identifier: GPL-2.0-only
use crate::sketch::{Sketch2D, SketchEntityId, SketchError, variables::SketchVariable};
#[derive(Clone, Copy)]
pub(in crate::sketch) enum EllipseComponent {
    MajorRadius,
    MinorRadius,
    Orientation,
}
impl EllipseComponent {
    fn offset(self) -> usize {
        match self {
            Self::MajorRadius => 0,
            Self::MinorRadius => 1,
            Self::Orientation => 2,
        }
    }
}
pub(in crate::sketch) fn ellipse_variables(ellipse: SketchEntityId) -> [SketchVariable; 3] {
    [
        EllipseComponent::MajorRadius,
        EllipseComponent::MinorRadius,
        EllipseComponent::Orientation,
    ]
    .map(|component| SketchVariable::Ellipse { ellipse, component })
}
impl Sketch2D {
    pub(in crate::sketch) fn ellipse_column(
        &self,
        id: SketchEntityId,
        component: EllipseComponent,
    ) -> Result<usize, SketchError> {
        self.ellipses
            .iter()
            .position(|ellipse| ellipse.id() == id)
            .map(|index| {
                self.points.len() * 2 + self.circles.len() + index * 3 + component.offset()
            })
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "sketch ellipse {id:?}: expected an owned ellipse solver variable"
                ))
            })
    }
    pub(in crate::sketch) fn ellipse_values(&self) -> impl Iterator<Item = f64> + '_ {
        self.ellipses.iter().flat_map(|ellipse| {
            let [major, minor] = ellipse.radii();
            [major, minor, ellipse.orientation()]
        })
    }
    pub(in crate::sketch) fn apply_ellipse_values(
        &mut self,
        values: &[f64],
    ) -> Result<(), SketchError> {
        for (ellipse, values) in self.ellipses.iter_mut().zip(values.as_chunks::<3>().0) {
            *ellipse = ellipse
                .with_radii([values[0], values[1]])?
                .with_orientation(values[2])?;
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::sketch::SketchEllipse2D;
    fn owned() -> Sketch2D {
        let mut sketch = Sketch2D::default();
        sketch.add_point(SketchEntityId(1), [2.0, -3.0]).unwrap();
        sketch
            .add_circle(SketchEntityId(2), SketchEntityId(1), 1.0)
            .unwrap();
        sketch
            .add_ellipse(
                SketchEllipse2D::new(SketchEntityId(3), SketchEntityId(1), [3.0, 4.0], [3.0, 1.0])
                    .unwrap(),
            )
            .unwrap();
        sketch
    }
    #[test]
    fn mixed_variables_keep_circle_columns_and_add_exactly_three_ellipse_slots() {
        let sketch = owned();
        let columns = ellipse_variables(SketchEntityId(3))
            .map(|variable| sketch.variable_column(variable).unwrap());
        assert_eq!(columns, [3, 4, 5]);
        let values = sketch.variable_values();
        assert_eq!(&values[..5], &[2.0, -3.0, 1.0, 3.0, 1.0]);
        assert_eq!(
            values[5],
            sketch.ellipse(SketchEntityId(3)).unwrap().orientation()
        );
        assert!(sketch.same_recipe(&sketch.candidate_values(&values).unwrap()));
        assert!(
            sketch
                .ellipse_column(SketchEntityId(99), EllipseComponent::Orientation)
                .is_err()
        );
    }
    #[test]
    fn ellipse_candidate_edits_are_atomic_and_validate_radius_and_rotation_domains() {
        let sketch = owned();
        let original = sketch.variable_values();
        let mut values = original.clone();
        values[3..].copy_from_slice(&[4.0, 2.0, std::f64::consts::FRAC_PI_2]);
        let candidate = sketch.candidate_values(&values).unwrap();
        assert_eq!(
            candidate.ellipse(SketchEntityId(3)).unwrap().radii(),
            [4.0, 2.0]
        );
        assert!(candidate.ellipse(SketchEntityId(3)).unwrap().major_axis()[0].abs() < f64::EPSILON);
        assert_bad_candidates(&sketch, &original);
        assert_eq!(sketch.variable_values(), original);
    }
    fn assert_bad_candidates(sketch: &Sketch2D, original: &[f64]) {
        for tuple in [
            [0.0, 1.0, 0.0],
            [1.0, 2.0, 0.0],
            [3.0, 1.0, f64::NAN],
            [3.0, f64::INFINITY, 0.0],
        ] {
            let mut values = original.to_vec();
            values[3..].copy_from_slice(&tuple);
            assert!(sketch.candidate_values(&values).is_err());
        }
        assert!(sketch.candidate_values(&original[..5]).is_err());
    }
}
