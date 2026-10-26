// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{PrismProbeSnapshot, retain_planar_prism, retain_prism_on_plane};
use oblikovati_model::sketch::SketchPlane;
fn oracle_plane(index: usize) -> SketchPlane {
    let (x, y) = match index {
        1 => ([1., 0., 0.], [0., 1., 0.]),
        2 => ([0., 1., 0.], [0., 0., 1.]),
        3 => ([0.6, 0.8, 0.], [0., 0., 1.]),
        4 => ([1., 0., 0.], [0., -1., 0.]),
        5 => ([1. + 5e-10, 0., 0.], [5e-10, 1., 0.]),
        _ => panic!("frame {index}: expected oracle frame 1..5"),
    };
    SketchPlane::new([10., -20., 30.], x, y, Default::default()).unwrap()
}
fn parse_placed_row(line: &str) -> (usize, [f64; 16]) {
    let (placement, row) = line.split_once(',').unwrap();
    (placement.parse().unwrap(), oracle::parse_oracle_row(row))
}
#[test]
fn all_placed_prism_faces_match_independent_occt_results() {
    let corpus = include_str!("occt-8.0.0-placed.csv");
    assert_eq!(corpus.lines().count(), 1080);
    let mut snapshot: Option<PrismProbeSnapshot> = None;
    let mut cases = 0;
    for line in corpus.lines() {
        let (placement, row) = parse_placed_row(line);
        if row[5] == 0. {
            snapshot = Some(
                retain_prism_on_plane(&oracle::oracle_definition(&row), oracle_plane(placement))
                    .unwrap(),
            );
            verify_placed_mesh(snapshot.as_ref().unwrap(), row[1]);
            cases += 1;
        }
        oracle::verify_face(&row, snapshot.as_ref().unwrap().report());
    }
    assert_eq!(cases, 135);
}
#[test]
fn default_xy_and_explicit_model_plane_use_the_same_construction() {
    let definition = with_hole();
    let default = retain_planar_prism(&definition).unwrap();
    let explicit = retain_prism_on_plane(&definition, SketchPlane::xy()).unwrap();
    assert_eq!(default.report(), explicit.report());
    assert_eq!(
        default.presentation(mesh::options()).unwrap(),
        explicit.presentation(mesh::options()).unwrap()
    );
}
#[test]
fn placed_mesh_preserves_the_authored_axes_and_analytic_reports() {
    let plane = oracle_plane(5);
    let snapshot = retain_prism_on_plane(&rectangle(), plane).unwrap();
    let expected = plane.to_model([4., 0.]).unwrap();
    let mesh = snapshot.presentation(mesh::options()).unwrap();
    assert!(
        mesh.faces
            .iter()
            .flat_map(|face| &face.triangles)
            .flatten()
            .any(|point| *point == expected)
    );
    assert_eq!(snapshot.report(), snapshot.detached().unwrap().report());
}
#[test]
fn placement_overflow_and_kernel_resolution_loss_are_named_declines() {
    let huge = huge_origin_plane();
    let mut definition = rectangle();
    polygon_mut(&mut definition.outer).1[1][0] = f64::MAX;
    assert!(
        retain_prism_on_plane(&definition, huge)
            .err()
            .unwrap()
            .contains("finite")
    );
    assert!(
        retain_prism_on_plane(&rectangle(), huge)
            .err()
            .unwrap()
            .contains("kernel collapsed")
    );
}

fn huge_origin_plane() -> SketchPlane {
    SketchPlane::new(
        [f64::MAX, 0., 0.],
        [1., 0., 0.],
        [0., 1., 0.],
        Default::default(),
    )
    .unwrap()
}

fn verify_placed_mesh(snapshot: &PrismProbeSnapshot, scale: f64) {
    let limits = oblikovati_martensite::PrismMeshOptions {
        chord_error: scale * 0.05,
        ..mesh::options()
    };
    let presentation = snapshot.presentation(limits).unwrap();
    mesh::verify_mesh_areas(&presentation, snapshot.report());
}
