// SPDX-License-Identifier: GPL-2.0-only
use super::{PlanarCurveDefinition, sample_planar_curvature, station};
use crate::planar_curve::{PlanarConicFrame, PlanarCurveFailureKind, PlanarGeometry};

fn ellipse(counterclockwise: bool) -> PlanarCurveDefinition {
    PlanarCurveDefinition {
        geometry: PlanarGeometry::Ellipse {
            frame: PlanarConicFrame {
                center: [3., -4.],
                axis: [0.6, 0.8],
                counterclockwise,
            },
            radii: [2., 1.],
        },
        bounds: None,
        left_offset: 0.,
    }
}

fn expect_pair(actual: [f64; 2], expected: [f64; 2]) {
    for (actual, expected) in actual.into_iter().zip(expected) {
        assert!(
            (actual - expected).abs() < 1e-12,
            "coordinate {actual}: expected {expected}"
        );
    }
}

#[test]
fn ellipse_curvature_at_axes_matches_analytic_inward_vectors() {
    let stations = sample_planar_curvature(&ellipse(true), 16).unwrap();
    assert_eq!(stations.len(), 17);
    expect_pair(stations[0].position, [4.2, -2.4]);
    expect_pair(stations[0].curvature, [-1.2, -1.6]);
    expect_pair(stations[4].position, [2.2, -3.4]);
    expect_pair(stations[4].curvature, [0.2, -0.15]);
    expect_pair(stations[16].position, stations[0].position);
    let reverse = sample_planar_curvature(&ellipse(false), 16).unwrap();
    for (forward, backward) in stations.iter().zip(reverse.iter().rev()) {
        expect_pair(forward.position, backward.position);
        expect_pair(forward.curvature, backward.curvature);
    }
}

#[test]
fn curvature_requires_a_bounded_domain_density_and_regular_derivatives() {
    let mut line = PlanarCurveDefinition {
        geometry: PlanarGeometry::Line {
            origin: [1., 2.],
            direction: [1., 0.],
        },
        bounds: None,
        left_offset: 0.,
    };
    assert_eq!(
        sample_planar_curvature(&line, 4).unwrap_err().kind(),
        PlanarCurveFailureKind::InvalidInput
    );
    line.bounds = Some([0., 1.]);
    let stations = sample_planar_curvature(&line, 4).unwrap();
    assert_eq!(stations[4].position, [2., 2.]);
    assert!(stations.iter().all(|station| station.curvature == [0.; 2]));
    for count in [0, 257] {
        assert_eq!(
            sample_planar_curvature(&line, count).unwrap_err().kind(),
            PlanarCurveFailureKind::InvalidInput
        );
    }
    assert_eq!(
        station(0., [0.; 2], [0.; 2], [0.; 2]).unwrap_err().kind(),
        PlanarCurveFailureKind::InvalidResult
    );
}
