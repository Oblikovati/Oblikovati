// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::SketchRectangleSeed;
use serde::Deserialize;
#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
struct GoRectangleRecord {
    variant: String,
    inputs: Vec<[f64; 2]>,
    bits: [[u64; 2]; 4],
}
#[test]
fn rectangle_seeds_match_independent_go_coordinates_bit_for_bit() {
    let records: Vec<GoRectangleRecord> =
        serde_json::from_str(include_str!("rectangle_seed/go-seeds.json")).unwrap();
    assert_eq!(records.len(), 21);
    for record in records {
        let points = record.inputs;
        let seed = match record.variant.as_str() {
            "corners" => SketchRectangleSeed::Corners(points.try_into().unwrap()),
            "center" => SketchRectangleSeed::Center(points.try_into().unwrap()),
            "threePoint" => SketchRectangleSeed::ThreePoints(points.try_into().unwrap()),
            other => panic!("unknown Go variant {other:?}; expected rectangle constructor"),
        };
        let actual = seed
            .vertices()
            .unwrap()
            .map(|point| point.map(f64::to_bits));
        assert_eq!(actual, record.bits, "{seed:?}");
    }
}
#[test]
fn rectangle_corner_selection_preserves_signed_zero_and_coincident_draft_points() {
    let first = [-0.0, 0.0];
    let opposite = [0.0, -0.0];
    let actual = SketchRectangleSeed::Corners([first, opposite])
        .vertices()
        .unwrap();
    let expected = [first, [0.0, 0.0], opposite, [-0.0, -0.0]];
    assert_eq!(
        actual.map(|p| p.map(f64::to_bits)),
        expected.map(|p| p.map(f64::to_bits))
    );
    assert_eq!(
        SketchRectangleSeed::Center([[2.0, 3.0]; 2])
            .vertices()
            .unwrap(),
        [[2.0, 3.0]; 4]
    );
}
#[test]
fn every_rectangle_constructor_refuses_nonfinite_authored_positions() {
    for bad in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        for index in 0..3 {
            for axis in 0..2 {
                let mut points = [[0.0, 0.0], [3.0, 4.0], [7.0, 2.0]];
                points[index][axis] = bad;
                assert!(SketchRectangleSeed::ThreePoints(points).vertices().is_err());
                if index < 2 {
                    let pair = [points[0], points[1]];
                    assert!(SketchRectangleSeed::Corners(pair).vertices().is_err());
                    assert!(SketchRectangleSeed::Center(pair).vertices().is_err());
                    assert!(
                        SketchRectangleSeed::DirectedCenter(pair)
                            .vertices()
                            .is_err()
                    );
                }
            }
        }
    }
}
#[test]
fn rectangle_declines_unrepresentable_intermediate_or_output_coordinates() {
    let large = f64::MAX;
    for seed in [
        SketchRectangleSeed::Center([[-large, 0.0], [large, 1.0]]),
        SketchRectangleSeed::Center([[large, 0.0], [0.0, 1.0]]),
        SketchRectangleSeed::ThreePoints([[0.0; 2]; 3]),
        SketchRectangleSeed::ThreePoints([[0.0; 2], [1e-200, 0.0], [0.0, 1.0]]),
        SketchRectangleSeed::ThreePoints([[0.0; 2], [1e200, 0.0], [0.0, 1.0]]),
        SketchRectangleSeed::ThreePoints([[0.0, -large], [1.0, -large], [0.0, large]]),
        SketchRectangleSeed::ThreePoints([[0.0; 2], [1.0, 1.0], [-large, large]]),
        SketchRectangleSeed::ThreePoints([[large, 0.0], [large, 1.0], [-large, 0.0]]),
    ] {
        assert!(seed.vertices().is_err(), "accepted {seed:?}");
    }
}

#[test]
fn directed_centers_keep_corner_order_without_changing_raw_center_order() {
    let inputs = [[3., 4.], [-3., -4.]];
    assert_eq!(
        SketchRectangleSeed::DirectedCenter(inputs)
            .vertices()
            .unwrap(),
        [[9., 12.], [-3., 12.], [-3., -4.], [9., -4.]]
    );
    assert_eq!(
        SketchRectangleSeed::Center(inputs).vertices().unwrap(),
        [[-3., -4.], [9., -4.], [9., 12.], [-3., 12.]]
    );
}
