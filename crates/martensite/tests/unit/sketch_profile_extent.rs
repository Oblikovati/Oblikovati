// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::planar_curve::{PlanarConicFrame, PlanarGeometry};
fn circle(bounds: Option<[f64; 2]>) -> PlanarCurveDefinition {
    PlanarCurveDefinition {
        geometry: PlanarGeometry::Circle {
            frame: PlanarConicFrame {
                center: [0., 0.],
                axis: [1., 0.],
                counterclockwise: true,
            },
            radius: 1000.,
        },
        bounds,
        left_offset: 0.,
    }
}
#[test]
fn native_bounds_use_arc_span_instead_of_the_full_carrier() {
    let mut full = BoundingBox2::default();
    add_curve_bounds(&mut full, &circle(None)).unwrap();
    let mut short = BoundingBox2::default();
    add_curve_bounds(&mut short, &circle(Some([0., 0.001]))).unwrap();
    let full = finish_bounds(&full).unwrap();
    let short = finish_bounds(&short).unwrap();
    assert!(full.diagonal > 2000.);
    assert!(short.diagonal < 2.);
}
#[test]
fn empty_and_nonfinite_extents_are_refused() {
    assert!(finish_bounds(&BoundingBox2::default()).is_err());
    let mut invalid = circle(None);
    invalid.left_offset = f64::NAN;
    assert!(add_curve_bounds(&mut BoundingBox2::default(), &invalid).is_err());
}
