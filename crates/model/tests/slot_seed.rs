// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::{SketchPathMember, SketchSlotPlacement};
use serde::Deserialize;
#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
struct GoSlotRecord {
    variant: String,
    input: [[f64; 2]; 3],
    width: f64,
    sweep: f64,
    #[serde(rename = "CCW")]
    ccw: bool,
    points: Vec<[f64; 2]>,
    members: Vec<GoSlotMember>,
}
#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
struct GoSlotMember {
    kind: String,
    center: usize,
    ends: [usize; 2],
    #[serde(rename = "CCW")]
    ccw: bool,
}
fn placement(r: &GoSlotRecord) -> SketchSlotPlacement {
    let [center, start, end] = r.input;
    match r.variant.as_str() {
        "centers" => SketchSlotPlacement::Centers([center, start]),
        "overall" => SketchSlotPlacement::Overall([center, start]),
        "slotCenter" => SketchSlotPlacement::SlotCenter([center, start]),
        "arc" => SketchSlotPlacement::Arc {
            center,
            endpoints: [start, end],
            counter_clockwise: r.ccw,
        },
        "arcCenterPoint" => SketchSlotPlacement::ArcCenterPoint {
            center,
            start,
            sweep: r.sweep,
        },
        _ => panic!("unknown oracle variant {}", r.variant),
    }
}
#[test]
fn every_slot_seed_and_shared_reference_matches_the_independent_go_capture() {
    let records: Vec<GoSlotRecord> =
        serde_json::from_str(include_str!("slot_seed/go-seeds.json")).unwrap();
    assert_eq!(records.len(), 20);
    for record in records {
        let layout = placement(&record).layout(record.width).unwrap();
        assert_eq!(layout.points.len(), record.points.len());
        for (actual, expected) in layout
            .points
            .iter()
            .flatten()
            .zip(record.points.iter().flatten())
        {
            // Only seed rounding is compared; trigonometric libraries may differ.
            assert!(
                (actual - expected).abs() <= 64.0 * f64::EPSILON * expected.abs().max(1.0),
                "{}: {actual} != {expected}",
                record.variant
            );
        }
        for (actual, expected) in layout.members.iter().zip(record.members) {
            let member = if expected.kind == "line" {
                SketchPathMember::Line(expected.ends)
            } else {
                SketchPathMember::Arc {
                    center: expected.center,
                    endpoints: expected.ends,
                    counter_clockwise: expected.ccw,
                }
            };
            assert_eq!(*actual, member);
        }
    }
}
#[test]
fn malformed_and_unrepresentable_slot_placements_fail_before_ownership() {
    let valid = SketchSlotPlacement::Centers([[0.0, 0.0], [4.0, 0.0]]);
    for width in [0.0, -1.0, f64::NAN, f64::INFINITY, f64::from_bits(1)] {
        assert!(valid.layout(width).is_err());
    }
    for placement in [
        SketchSlotPlacement::Centers([[0.0; 2]; 2]),
        SketchSlotPlacement::Centers([[0.0; 2], [1e-200, 0.0]]),
        SketchSlotPlacement::Centers([[f64::MAX, 0.0], [-f64::MAX, 0.0]]),
        SketchSlotPlacement::Overall([[0.0; 2], [1.0, 0.0]]),
        SketchSlotPlacement::SlotCenter([[f64::MAX, 0.0], [1.0, 0.0]]),
        SketchSlotPlacement::Arc {
            center: [0.0; 2],
            endpoints: [[1.0, 0.0], [0.0, 1.0]],
            counter_clockwise: true,
        },
        SketchSlotPlacement::ArcCenterPoint {
            center: [0.0; 2],
            start: [2.0, 0.0],
            sweep: f64::NAN,
        },
    ] {
        assert!(placement.layout(2.0).is_err(), "{placement:?}");
    }
}

#[test]
fn rounded_away_slot_width_cannot_publish_collapsed_caps() {
    let placement = SketchSlotPlacement::Centers([[1e16, 1e16], [1e16 + 4.0, 1e16]]);
    assert!(placement.layout(0.1).is_err());
}
