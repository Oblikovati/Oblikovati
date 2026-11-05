// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::SketchPolygonSeed;
use serde::Deserialize;
#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
struct GoPolygonRecord {
    center: [f64; 2],
    through: [f64; 2],
    sides: usize,
    inscribed: bool,
    vertices: Vec<[f64; 2]>,
}
#[test]
fn polygon_seed_coordinates_follow_the_independent_go_constructors() {
    let records: Vec<GoPolygonRecord> =
        serde_json::from_str(include_str!("polygon_seed/go-seeds.json")).unwrap();
    assert_eq!(records.len(), 20);
    for record in records {
        let seed = SketchPolygonSeed {
            center: record.center,
            through: record.through,
            sides: record.sides,
            inscribed: record.inscribed,
        };
        let actual = seed.vertices().unwrap();
        assert_eq!(actual.len(), record.vertices.len());
        for (actual, expected) in actual
            .iter()
            .flatten()
            .zip(record.vertices.iter().flatten())
        {
            // Go and Rust use different trigonometric implementations; this bounds seed rounding only.
            let bound = 64.0 * f64::EPSILON * expected.abs().max(1.0);
            assert!(
                (actual - expected).abs() <= bound,
                "{seed:?}: actual {actual}, expected {expected}, bound {bound}"
            );
        }
    }
}
#[test]
fn polygon_seed_refuses_nonfinite_inputs_and_unrepresentable_output() {
    let seed = SketchPolygonSeed {
        center: [0.0; 2],
        through: [3.0, 4.0],
        sides: 5,
        inscribed: false,
    };
    for coordinate in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        for axis in 0..2 {
            let mut bad = seed;
            bad.center[axis] = coordinate;
            assert!(bad.vertices().is_err());
            let mut bad = seed;
            bad.through[axis] = coordinate;
            assert!(bad.vertices().is_err());
        }
    }
    assert!(
        SketchPolygonSeed {
            center: [f64::MAX, 0.0],
            through: [-f64::MAX, 1.0],
            ..seed
        }
        .vertices()
        .is_err()
    );
    assert!(
        SketchPolygonSeed {
            through: [1e-200, 0.0],
            ..seed
        }
        .vertices()
        .is_err()
    );
}
