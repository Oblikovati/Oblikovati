// SPDX-License-Identifier: GPL-2.0-only
//! Resolve Through All against conservative analytic bounds of the current target.
use super::{hole_evaluation::analytic_bounds, *};
use oblikovati_model::{feature::sketch_prism::ThroughAllDirection, sketch::SketchPlane};

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn named_prism_through_all_span(
        &self,
        body: &NamedSolid,
        plane: SketchPlane,
        direction: ThroughAllDirection,
    ) -> Result<(f64, f64), String> {
        let resolution = match body {
            NamedSolid::Prism(body) => body
                .certification
                .as_ref()
                .map(|evidence| evidence.resolution),
            NamedSolid::Derived(body) => body.resolution(),
        }
        .ok_or("extrude Through All: expected certified target bounds")?;
        let shape = body.native_shape(self.history, self.document)?;
        let bounds = analytic_bounds(shape, "extrude Through All")?;
        normal_span(bounds, plane, direction, resolution.maximum_tolerance)
    }
}

fn normal_span(
    bounds: [f64; 6],
    plane: SketchPlane,
    direction: ThroughAllDirection,
    tolerance: f64,
) -> Result<(f64, f64), String> {
    let origin = plane.recipe().origin;
    let normal = plane.normal();
    let [lower, upper] = [false, true].map(|upper| {
        (0..3)
            .map(|i| {
                let extreme = if upper == (normal[i] >= 0.) {
                    bounds[i + 3]
                } else {
                    bounds[i]
                };
                (extreme - origin[i]) * normal[i]
            })
            .sum::<f64>()
    });
    if !lower.is_finite() || !upper.is_finite() || lower >= upper {
        return Err(format!(
            "extrude Through All projected bounds [{lower}, {upper}]: expected finite increasing offsets"
        ));
    }
    // An outward margin covers the admitted target tolerance and rounding in
    // the three coordinate differences, products, sums and frame translation.
    let coordinate_scale = bounds
        .into_iter()
        .chain(origin)
        .map(f64::abs)
        .fold(0., f64::max);
    let margin = tolerance + 16. * f64::EPSILON * coordinate_scale;
    let before = (lower - margin).next_down();
    let after = (upper + margin).next_up();
    let (start, depth) = match direction {
        ThroughAllDirection::Positive if upper > 0. => (0., after),
        ThroughAllDirection::Negative if lower < 0. => (0., before),
        ThroughAllDirection::Symmetric => {
            let start = before.min(0.);
            (start, (after.max(0.) - start).next_up())
        }
        _ => {
            return Err(format!(
                "extrude Through All {direction:?}: target offsets [{lower}, {upper}] do not reach the selected side"
            ));
        }
    };
    if !start.is_finite() || !depth.is_finite() || depth == 0. {
        return Err(format!(
            "extrude Through All start {start}, sweep {depth}: expected a finite nonzero span"
        ));
    }
    Ok((start, depth))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn directional_spans_cover_only_the_selected_side_of_the_sketch() {
        let bounds = [-1., -1., -3., 1., 1., 5.];
        let plane = SketchPlane::xy();
        let positive = normal_span(bounds, plane, ThroughAllDirection::Positive, 1e-6).unwrap();
        assert_eq!(positive.0, 0.);
        assert!(positive.1 > 5. && positive.1 < 5.01);
        let negative = normal_span(bounds, plane, ThroughAllDirection::Negative, 1e-6).unwrap();
        assert_eq!(negative.0, 0.);
        assert!(negative.1 < -3. && negative.1 > -3.01);
        let both = normal_span(bounds, plane, ThroughAllDirection::Symmetric, 1e-6).unwrap();
        assert!(both.0 < -3.);
        assert!(both.0 + both.1 > 5.);
        assert!(
            normal_span(
                [-1., -1., -3., 1., 1., -1.],
                plane,
                ThroughAllDirection::Positive,
                1e-6,
            )
            .is_err()
        );
    }
}
