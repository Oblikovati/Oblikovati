// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{PrismMeshOptions, retain_planar_prism};
pub(super) fn options() -> PrismMeshOptions {
    PrismMeshOptions {
        chord_error: 0.05,
        angular_error: 0.5,
    }
}
fn triangle_area(points: [[f64; 3]; 3]) -> f64 {
    let u: [f64; 3] = std::array::from_fn(|i| points[1][i] - points[0][i]);
    let v: [f64; 3] = std::array::from_fn(|i| points[2][i] - points[0][i]);
    let cross = [
        u[1] * v[2] - u[2] * v[1],
        u[2] * v[0] - u[0] * v[2],
        u[0] * v[1] - u[1] * v[0],
    ];
    cross
        .into_iter()
        .map(|value| value * value)
        .sum::<f64>()
        .sqrt()
        / 2.
}
#[test]
fn derived_mesh_keeps_all_face_roles_and_preserves_analytic_snapshot_queries() {
    let snapshot = retain_planar_prism(&with_hole()).unwrap();
    let before = snapshot.report().clone();
    let presentation = snapshot.presentation(options()).unwrap();
    assert_eq!(presentation.faces.len(), before.faces.len());
    verify_mesh_areas(&presentation, &before);
    assert_eq!(snapshot.report(), &before);
    assert_eq!(snapshot.detached().unwrap().report(), &before);
    assert_eq!(snapshot.presentation(options()).unwrap(), presentation);
}
#[test]
fn malformed_mesh_approximation_limits_are_named_refusals() {
    let snapshot = retain_planar_prism(&rectangle()).unwrap();
    for limits in malformed_options() {
        assert!(
            snapshot
                .presentation(limits)
                .unwrap_err()
                .contains("finite positive")
        );
    }
}

pub(super) fn verify_mesh_areas(
    presentation: &oblikovati_martensite::PrismPresentation,
    before: &oblikovati_martensite::PrismProbeReport,
) {
    for mesh in &presentation.faces {
        let analytic = before
            .faces
            .iter()
            .find(|face| face.role == mesh.role)
            .unwrap();
        let area = mesh
            .triangles
            .iter()
            .copied()
            .map(triangle_area)
            .sum::<f64>();
        assert!((area - analytic.area).abs() < analytic.area * 1e-11);
    }
}

fn malformed_options() -> [PrismMeshOptions; 3] {
    [
        PrismMeshOptions {
            chord_error: 0.,
            ..options()
        },
        PrismMeshOptions {
            angular_error: f64::NAN,
            ..options()
        },
        PrismMeshOptions {
            chord_error: f64::INFINITY,
            ..options()
        },
    ]
}

#[test]
fn every_prism_corpus_case_has_matching_derived_face_areas() {
    for line in include_str!("occt-8.0.0.csv").lines() {
        let row = super::oracle::parse_oracle_row(line);
        if row[5] != 0. {
            continue;
        }
        let definition = super::oracle::oracle_definition(&row);
        let snapshot = retain_planar_prism(&definition).unwrap();
        let limits = PrismMeshOptions {
            chord_error: row[1] * 0.05,
            ..options()
        };
        let mesh = snapshot.presentation(limits).unwrap();
        verify_mesh_areas(&mesh, snapshot.report());
    }
}
