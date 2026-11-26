// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::EvaluationPhase;
use oblikovati_model::feature::work_geometry::DatumAngle;

#[test]
fn signed_taper_and_zero_resolve_from_the_saved_angle() {
    let (snapshot, sweep) = fixture();
    let datums = snapshot
        .work_geometry
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap();
    for degrees in [-5.0_f64, 0., 5.] {
        let angle = degrees.to_radians();
        let authored = sweep
            .clone()
            .with_taper(DatumAngle::constant(angle).unwrap())
            .unwrap();
        let bytes = serde_json::to_vec(authored.recipe()).unwrap();
        let restored =
            SketchSweepDefinition::from_recipe(serde_json::from_slice(&bytes).unwrap()).unwrap();
        assert!(authored.same_recipe(&restored));
        let input = restored.resolve(&snapshot, &datums).unwrap();
        assert_eq!(input.taper, angle);
        let result = build_sweep(input, resolution()).unwrap();
        let end_radius = 0.5 + 3. * angle.tan();
        let expected = PI * (0.25 + 0.5 * end_radius + end_radius * end_radius);
        assert!((result.properties().unwrap().volume - expected).abs() < 1e-8);
        assert_eq!(
            result.face_roles().collect::<Vec<_>>(),
            [
                SweepFaceRole::Start,
                SweepFaceRole::End,
                SweepFaceRole::Side {
                    profile: SECTION,
                    path: SPINE
                }
            ]
        );
    }
}

#[test]
fn taper_refuses_a_collapsed_section() {
    let (snapshot, sweep) = fixture();
    let datums = snapshot
        .work_geometry
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap();
    let authored = sweep
        .with_taper(DatumAngle::constant(-0.2).unwrap())
        .unwrap();
    let result = build_sweep(authored.resolve(&snapshot, &datums).unwrap(), resolution());
    let error = result
        .err()
        .expect("expected the collapsed section to fail");
    assert_eq!(error.phase(), EvaluationPhase::Input);
    assert!(error.to_string().contains("end radius"));
}

#[test]
fn taper_parameter_changes_geometry_without_replacing_its_source() {
    use oblikovati_api::types::USER_PARAM;

    let (mut snapshot, sweep) = fixture();
    let taper_id = 42;
    snapshot
        .parameters
        .insert_expression(taper_id, "DraftAngle", USER_PARAM, "5 deg")
        .unwrap();
    let authored = sweep.with_taper(DatumAngle::parameter(taper_id)).unwrap();
    let reads = authored.reads(&snapshot).unwrap();
    assert!(reads.keys().contains(&FeatureReadKey::Parameter(taper_id)));
    let feature = FeatureId(2);
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            feature,
            "Sweep1",
            oblikovati_model::feature::part::PartFeatureDefinition::SketchSweep(Box::new(
                authored.clone(),
            )),
            reads,
        ))
        .unwrap();
    let before = snapshot.clone();
    snapshot
        .parameters
        .set_expression(taper_id, "-5 deg")
        .unwrap();
    assert_eq!(
        before.part_replay_change(&snapshot),
        Some(oblikovati_model::feature::FeatureReplayRange { start: 1, end: 2 })
    );
    for (source, sign) in [(&before, 1.), (&snapshot, -1.)] {
        let datums = source
            .work_geometry
            .evaluate_work_geometry(&source.parameters)
            .unwrap();
        let input = authored.resolve(source, &datums).unwrap();
        let volume = build_sweep(input, resolution())
            .unwrap()
            .properties()
            .unwrap()
            .volume;
        assert!((volume - 0.75 * PI) * sign > 0.);
    }
}
