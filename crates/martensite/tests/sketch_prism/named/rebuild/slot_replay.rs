// SPDX-License-Identifier: GPL-2.0-only
//! Real-kernel verification of Slot rounded, rectangular rotated, ToSurface, and seated cuts.
use super::*;
use oblikovati_model::feature::{
    hole::{HoleOriginPlane, HoleTerminationSurface},
    slot::{SlotCenterMarks, SlotDefinition, SlotEndMill, SlotRecipe, SlotSeat, SlotShape},
};

fn evaluate_slot(
    shape: SlotShape,
    length: f64,
    width: f64,
    rotation_deg: f64,
    extent: HoleExtent,
    seat: SlotSeat,
) -> f64 {
    let mut history = owner(hole_document());
    let recipe = SlotRecipe {
        version: 1,
        body: key("body", &[1]),
        face: key("end", &[1]),
        position: HolePosition::FaceCentroid,
        shape,
        length: DatumLength::Constant(length),
        width: DatumLength::Constant(width),
        rotation_deg,
        extent,
        kind: Default::default(),
        seat,
        end_mill: SlotEndMill::Flat,
        center_marks: SlotCenterMarks::None,
        direction_flipped: false,
        extend_start: false,
        imate: false,
        fastener: None,
    };
    let definition = SlotDefinition::from_recipe(recipe).unwrap();
    let token = history.begin("Slot").unwrap();
    let id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_slot("Slot1", definition, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let result = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        result
            .health()
            .iter()
            .all(|record| record.status == FeatureEvaluationStatus::Healthy),
        "health: {:?}",
        result.health()
    );
    let body = result.body(id).expect("expected Slot solid body");
    let summary = body
        .topology_summary(&history, document(), &key("body", &[1]))
        .unwrap();
    assert!(summary.solid, "expected valid solid: {summary:?}");
    body.volume(&history, document()).unwrap()
}

#[test]
fn rounded_rectangular_rotated_to_surface_and_seated_slots_match_analytic_volumes() {
    let pi = std::f64::consts::PI;

    // 1. Rounded blind slot 0.4 x 0.2 x 0.5 cm
    let v_round = evaluate_slot(
        SlotShape::Rounded,
        0.4,
        0.2,
        0.0,
        HoleExtent::Distance(DatumLength::Constant(0.5)),
        SlotSeat::None,
    );
    let area_round = (0.4 - 0.2) * 0.2 + pi * 0.1 * 0.1;
    let expected_round = 1.0 - area_round * 0.5;
    assert!(
        (v_round - expected_round).abs() < 1e-5,
        "Rounded: got {v_round}, expected {expected_round}"
    );

    // 2. Rectangular 45-degree rotated blind slot 0.4 x 0.2 x 0.5 cm
    let v_rect = evaluate_slot(
        SlotShape::Rectangular,
        0.4,
        0.2,
        45.0,
        HoleExtent::Distance(DatumLength::Constant(0.5)),
        SlotSeat::None,
    );
    let expected_rect = 1.0 - 0.4 * 0.2 * 0.5;
    assert!(
        (v_rect - expected_rect).abs() < 1e-5,
        "Rectangular 45°: got {v_rect}, expected {expected_rect}"
    );

    // 3. Rounded ToSurface (XY plane at z=0 -> 1.0 cm through)
    let v_xy = evaluate_slot(
        SlotShape::Rounded,
        0.4,
        0.2,
        0.0,
        HoleExtent::ToSurface(HoleTerminationSurface::OriginPlane(HoleOriginPlane::Xy)),
        SlotSeat::None,
    );
    let expected_xy = 1.0 - area_round * 1.0;
    assert!(
        (v_xy - expected_xy).abs() < 1e-5,
        "ToSurface(XY): got {v_xy}, expected {expected_xy}"
    );

    // 4. Rounded slot with Counterbore seat (width 0.3 cm, depth 0.15 cm)
    let v_cbore = evaluate_slot(
        SlotShape::Rounded,
        0.4,
        0.2,
        0.0,
        HoleExtent::Distance(DatumLength::Constant(0.5)),
        SlotSeat::Counterbore {
            width: DatumLength::Constant(0.3),
            depth: DatumLength::Constant(0.15),
        },
    );
    let area_seat = (0.5 - 0.3) * 0.3 + pi * 0.15 * 0.15;
    let expected_cbore = 1.0 - area_round * 0.5 - (area_seat - area_round) * 0.15;
    assert!(
        (v_cbore - expected_cbore).abs() < 1e-5,
        "Counterbore slot: got {v_cbore}, expected {expected_cbore}"
    );

    // 5. Rounded slot with Countersink seat (width 0.3 cm, 90°)
    let v_csink = evaluate_slot(
        SlotShape::Rounded,
        0.4,
        0.2,
        0.0,
        HoleExtent::Distance(DatumLength::Constant(0.5)),
        SlotSeat::Countersink {
            width: DatumLength::Constant(0.3),
            angle_deg: 90.0,
        },
    );
    assert!(
        v_csink < v_round && v_csink > v_cbore,
        "Countersink slot volume {v_csink} should lie between plain {v_round} and counterbore {v_cbore}"
    );
}
