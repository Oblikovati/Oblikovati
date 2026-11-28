// SPDX-License-Identifier: GPL-2.0-only
//! Real-kernel verification of Hole v2 seats, drill points, To Surface, and Extend Start.
use super::*;
use oblikovati_model::feature::hole::{
    HoleDrillPoint, HoleOptions, HoleOriginPlane, HoleSeat, HoleTerminationSurface,
};

fn evaluate_v2_hole(extent: HoleExtent, options: HoleOptions) -> f64 {
    let mut history = owner(hole_document());
    let mut recipe = HoleRecipe {
        version: 2,
        body: key("body", &[1]),
        face: key("end", &[1]),
        position: HolePosition::FaceCentroid,
        diameter: DatumLength::Constant(0.2),
        extent,
        options,
    };
    recipe.version = recipe.canonical_version();
    let definition = HoleDefinition::from_recipe(recipe).unwrap();
    let token = history.begin("Hole v2").unwrap();
    let id = history.create_hole(&token, "Hole1", definition).unwrap();
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
    let body = result.body(id).expect("expected Hole v2 solid body");
    let summary = body
        .topology_summary(&history, document(), &key("body", &[1]))
        .unwrap();
    assert!(summary.solid, "expected valid solid: {summary:?}");
    body.volume(&history, document()).unwrap()
}

#[test]
fn counterbore_countersink_spotface_drill_point_and_to_surface_match_analytic_volumes() {
    let pi = std::f64::consts::PI;
    let r = 0.1_f64;

    // 1. To Surface (XY origin plane at z=0 through the 1 cm cube -> 1 cm through bore)
    let v_to_xy = evaluate_v2_hole(
        HoleExtent::ToSurface(HoleTerminationSurface::OriginPlane(HoleOriginPlane::Xy)),
        HoleOptions::default(),
    );
    let expected_through = 1.0 - pi * r * r * 1.0;
    assert!(
        (v_to_xy - expected_through).abs() < 1e-6,
        "ToSurface(XY): got {v_to_xy}, expected {expected_through}"
    );

    // 2. Counterbore (0.2 cm bore + 0.4 cm x 0.2 cm stepped seat)
    let rs = 0.2_f64;
    let hs = 0.2_f64;
    let v_cbore = evaluate_v2_hole(
        HoleExtent::ThroughAll,
        HoleOptions {
            seat: HoleSeat::Counterbore {
                diameter: DatumLength::Constant(0.4),
                depth: DatumLength::Constant(hs),
            },
            ..Default::default()
        },
    );
    let expected_cbore = 1.0 - pi * r * r * 1.0 - pi * (rs * rs - r * r) * hs;
    assert!(
        (v_cbore - expected_cbore).abs() < 1e-6,
        "Counterbore: got {v_cbore}, expected {expected_cbore}"
    );

    // 3. Countersink (0.2 cm bore + 0.4 cm, 90-degree conical seat)
    let h_csink = rs - r; // tan(45°) = 1
    let v_csink_frustum = pi * h_csink * (rs * rs + rs * r + r * r) / 3.0;
    let expected_csink = 1.0 - pi * r * r * (1.0 - h_csink) - v_csink_frustum;
    let v_csink = evaluate_v2_hole(
        HoleExtent::ThroughAll,
        HoleOptions {
            seat: HoleSeat::Countersink {
                diameter: DatumLength::Constant(0.4),
                angle_deg: 90.0,
            },
            ..Default::default()
        },
    );
    assert!(
        (v_csink - expected_csink).abs() < 1e-5,
        "Countersink: got {v_csink}, expected {expected_csink}"
    );

    // 4. Spotface + Extend Start on finite 0.5 cm hole
    let v_spot = evaluate_v2_hole(
        HoleExtent::Distance(DatumLength::Constant(0.5)),
        HoleOptions {
            seat: HoleSeat::Spotface {
                diameter: DatumLength::Constant(0.4),
                depth: DatumLength::Constant(0.1),
            },
            extend_start: true,
            ..Default::default()
        },
    );
    let expected_spot = 1.0 - pi * r * r * 0.5 - pi * (rs * rs - r * r) * 0.1;
    assert!(
        (v_spot - expected_spot).abs() < 1e-6,
        "Spotface+ExtendStart: got {v_spot}, expected {expected_spot}"
    );

    // 5. Angled drill point (118°) vs flat on finite 0.5 cm hole
    let v_angled = evaluate_v2_hole(
        HoleExtent::Distance(DatumLength::Constant(0.5)),
        HoleOptions {
            drill_point: HoleDrillPoint::Angle(118.0),
            ..Default::default()
        },
    );
    let tip_h = r / (59.0_f64.to_radians()).tan();
    let expected_angled = 1.0 - pi * r * r * 0.5 - pi * r * r * tip_h / 3.0;
    assert!(
        (v_angled - expected_angled).abs() < 1e-5,
        "AngledDrillPoint: got {v_angled}, expected {expected_angled}"
    );
}
