// SPDX-License-Identifier: GPL-2.0-only
use crate::sketch::{Sketch2D, SketchEntityId, SketchError, SketchOffsetSpline2D};
impl Sketch2D {
    pub(in crate::sketch) fn offset_spline_column(
        &self,
        id: SketchEntityId,
    ) -> Result<usize, SketchError> {
        self.offset_splines
            .iter()
            .position(|offset| offset.id() == id)
            .map(|index| self.offset_spline_start() + index)
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "offset spline variable {id:?}: expected an owned signed distance"
                ))
            })
    }
    pub(in crate::sketch) fn offset_spline_start(&self) -> usize {
        self.points.len() * 2 + self.circles.len() + self.ellipses.len() * 3
    }
    pub(in crate::sketch) fn apply_offset_values(
        &mut self,
        values: &[f64],
    ) -> Result<(), SketchError> {
        for (offset, distance) in self.offset_splines.iter_mut().zip(values) {
            *offset = SketchOffsetSpline2D::new(offset.id(), offset.parent(), *distance)?;
        }
        Ok(())
    }
    pub(in crate::sketch) fn same_spline_recipes(&self, other: &Self) -> bool {
        self.splines == other.splines
            && self.offset_splines.len() == other.offset_splines.len()
            && self
                .offset_splines
                .iter()
                .zip(&other.offset_splines)
                .all(|(first, second)| {
                    first.id() == second.id()
                        && first.parent() == second.parent()
                        && first.distance().to_bits() == second.distance().to_bits()
                })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::sketch::{SketchSpline2D, SketchSplineKind};
    fn offset_owner() -> Sketch2D {
        use SketchSplineKind::Control;
        let [a, b, parent, offset] = [1, 2, 3, 4].map(SketchEntityId);
        let mut sketch = Sketch2D::default();
        sketch.add_point(a, [0.0, 0.0]).unwrap();
        sketch.add_point(b, [1.0, 0.0]).unwrap();
        let spline = SketchSpline2D::new(parent, vec![a, b], false, Control).unwrap();
        sketch.add_spline(spline).unwrap();
        sketch
            .add_offset_spline(SketchOffsetSpline2D::new(offset, parent, -1.0).unwrap())
            .unwrap();
        sketch
    }
    #[test]
    fn offset_candidate_writeback_validates_finite_signed_distance_atomically() {
        let sketch = offset_owner();
        let mut values = sketch.variable_values();
        assert_eq!(sketch.offset_spline_column(SketchEntityId(4)).unwrap(), 4);
        values[4] = -3.0;
        let candidate = sketch.candidate_values(&values).unwrap();
        assert_eq!(
            candidate
                .offset_spline(SketchEntityId(4))
                .unwrap()
                .distance(),
            -3.0
        );
        assert_bad_offset_candidate(&sketch, values);
    }
    fn assert_bad_offset_candidate(sketch: &Sketch2D, mut values: Vec<f64>) {
        values[4] = f64::INFINITY;
        assert!(sketch.candidate_values(&values).is_err());
        assert_eq!(
            sketch.offset_spline(SketchEntityId(4)).unwrap().distance(),
            -1.0
        );
        let error = sketch
            .offset_spline_column(SketchEntityId(99))
            .unwrap_err()
            .to_string();
        assert!(error.contains("99"));
        assert!(error.contains("expected"));
    }
}
