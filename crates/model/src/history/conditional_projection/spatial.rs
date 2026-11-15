// SPDX-License-Identifier: GPL-2.0-only
//! Conditional point displacement bounds, not operation tolerance qualification.
use super::{ConditionalFrameSource, ConditionalProjectedSketchRoot};
use crate::sketch::{SketchEntityId, interval_equations::enclosure::Enclosure};

/// Upper bound for one accepted point's displacement, in document length units.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct ConditionalPointDisplacement {
    pub point: SketchEntityId,
    pub upper: f64,
}
impl ConditionalProjectedSketchRoot<'_, '_> {
    /// Bound each point's distance from the retained center to the enclosed root.
    /// Example: `evidence.canonical_point_displacements()?`.
    /// The bound is conditional on the root evidence and is not a weld or sew tolerance.
    pub fn canonical_point_displacements(
        &self,
    ) -> Result<Vec<ConditionalPointDisplacement>, String> {
        if !matches!(
            self.projection.frame_source,
            ConditionalFrameSource::CanonicalOrigin(_)
        ) {
            return Err(
                "point displacement frame: expected a source-proven canonical origin plane".into(),
            );
        }
        let points = self.root.sketch().points();
        let center = self.root.center();
        let bounds = self.root.enclosure();
        if center.len() != points.len() * 2 || bounds.len() != center.len() {
            return Err(format!(
                "point displacement coordinates: expected two coordinates for each of {} points, got {} centers and {} intervals",
                points.len(),
                center.len(),
                bounds.len()
            ));
        }
        points
            .iter()
            .enumerate()
            .map(|(index, point)| {
                let column = index * 2;
                if point.position != [center[column], center[column + 1]] {
                    return Err(format!(
                        "point {:?} center {:?}: expected retained center {:?}",
                        point.id,
                        point.position,
                        &center[column..column + 2]
                    ));
                }
                Ok(ConditionalPointDisplacement {
                    point: point.id,
                    upper: displacement(
                        center[column..column + 2].try_into().unwrap(),
                        bounds[column..column + 2].try_into().unwrap(),
                    )?,
                })
            })
            .collect()
    }
}
fn displacement(center: [f64; 2], bounds: [[f64; 2]; 2]) -> Result<f64, String> {
    let mut sum = Enclosure::new([0., 0.])?;
    for axis in 0..2 {
        let [lo, hi] = Enclosure::new(bounds[axis])?
            .difference(Enclosure::new([center[axis], center[axis]])?)?
            .bounds();
        let maximum = lo.abs().max(hi.abs());
        sum = sum.sum(Enclosure::new([maximum, maximum])?)?;
    }
    Ok(sum.bounds()[1])
}
#[cfg(test)]
mod tests {
    use super::*;
    use num_rational::BigRational;
    fn q(x: f64) -> BigRational {
        BigRational::from_float(x).unwrap()
    }
    #[test]
    fn bound_covers_exact_squared_euclidean_corner_distances() {
        for (center, bounds) in [
            ([0.1, -0.3], [[-1., 2.], [-2., 1.]]),
            ([0., 0.], [[0., 0.], [0., 0.]]),
            ([1e100, -1e-100], [[1e100, 1e100], [-2e-100, 0.]]),
        ] {
            let upper = q(displacement(center, bounds).unwrap());
            for x in bounds[0] {
                for y in bounds[1] {
                    let dx = q(x) - q(center[0]);
                    let dy = q(y) - q(center[1]);
                    assert!(&dx * &dx + &dy * &dy <= &upper * &upper);
                }
            }
        }
    }
    #[test]
    fn unbounded_and_overflowing_inputs_decline() {
        assert!(displacement([0., 0.], [[f64::NEG_INFINITY, 1.], [0., 0.]]).is_err());
        assert!(displacement([-f64::MAX, 0.], [[f64::MAX, f64::MAX], [0., 0.]]).is_err());
    }
}
