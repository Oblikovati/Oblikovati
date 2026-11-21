// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[allow(dead_code)]
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
mod fixture;
fn evaluate_document(
    document: &oblikovati_model::history::DocumentSnapshot,
) -> (
    oblikovati_model::history::SnapshotReplay,
    oblikovati_model::feature::part::PartFeatureEvaluation<
        oblikovati_martensite::NamedSolid,
        oblikovati_martensite::EvaluationFailure,
    >,
) {
    let history = oblikovati_model::history::SnapshotReplay::new(document).unwrap();
    let guid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let mut evaluator = oblikovati_martensite::CertifiedNamedSolidEvaluator::with_optional_policy(
        history.history(),
        guid,
        None,
    );
    let result = history
        .history()
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    (history, result)
}
#[test]
fn mesh_declines_retain_feature_identity_without_mutating_analytic_outputs() {
    let document = fixture::selected_prism_document();
    let (history, evaluated) = evaluate_document(&document);
    let body = evaluated
        .body(oblikovati_model::feature::FeatureId(1))
        .unwrap();
    let guid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let key = body.body_reference_key(history.history(), guid).unwrap();
    let original = body
        .physical_properties(history.history(), guid, key)
        .unwrap();
    // tol:oracle independent analytic volume integration permits floating-point roundoff.
    assert!((original.geometry.volume - 24.).abs() < 24. * f64::EPSILON * 8.);
    let options = oblikovati_martensite::PrismMeshOptions {
        chord_error: 0.,
        angular_error: 0.5,
    };
    let presentation =
        ViewportScene::from_part_evaluation(&evaluated, options, [0.; 3], [0., 1., 0., 1.])
            .unwrap();
    verify_mesh_decline(&presentation);
    assert_eq!(
        body.physical_properties(history.history(), guid, key)
            .unwrap(),
        original
    );
}

fn verify_mesh_decline(presentation: &PartPresentation) {
    assert!(presentation.scene.triangles.is_empty());
    assert_eq!(presentation.visible_bodies, 0);
    assert_eq!(
        presentation.declines[0].feature,
        oblikovati_model::feature::FeatureId(1)
    );
    assert!(presentation.declines[0].cause.contains("expected"));
}

#[test]
fn hidden_bodies_are_not_meshed_and_visibility_errors_are_not_empty_success() {
    let document = fixture::selected_prism_document();
    let (_history, evaluated) = evaluate_document(&document);
    let options = oblikovati_martensite::PrismMeshOptions {
        chord_error: 0.,
        angular_error: 0.5,
    };
    let hidden = ViewportScene::from_part_evaluation_with_visibility(
        &evaluated,
        options,
        [0.; 3],
        [0., 1., 0., 1.],
        |_| Ok(false),
    )
    .unwrap();
    assert_eq!(hidden.visible_bodies, 0);
    assert!(hidden.scene.triangles.is_empty() && hidden.declines.is_empty());
    let rejected = ViewportScene::from_part_evaluation_with_visibility(
        &evaluated,
        options,
        [0.; 3],
        [0., 1., 0., 1.],
        |_| Err("stale visibility key".into()),
    )
    .unwrap();
    assert_eq!(rejected.declines.len(), 1);
    assert_eq!(
        rejected.declines[0].feature,
        oblikovati_model::feature::FeatureId(1)
    );
    assert_eq!(rejected.declines[0].cause, "stale visibility key");
}

#[test]
fn a_body_style_failure_does_not_hide_other_valid_bodies() {
    use oblikovati_model::feature::{
        AuthoredFeature, FeatureId, sketch_prism::SketchPrismDefinition, work_geometry::DatumLength,
    };
    let mut document = fixture::selected_prism_document();
    let mut recipe = fixture::selected_prism_definition().recipe();
    recipe.displacement[2] = DatumLength::Constant(2.);
    let definition = SketchPrismDefinition::from_recipe(&recipe).unwrap();
    let feature =
        AuthoredFeature::sketch_prism(FeatureId(2), "Independent", definition, &document).unwrap();
    document.work_geometry.append(feature).unwrap();
    let (_history, evaluated) = evaluate_document(&document);
    let options = oblikovati_martensite::PrismMeshOptions {
        chord_error: 0.05,
        angular_error: 0.5,
    };
    let mut count = 0;
    let result = ViewportScene::from_part_evaluation_styled(&evaluated, options, [0.; 3], |_| {
        count += 1;
        if count == 1 {
            Err("missing appearance".into())
        } else {
            Ok(Some([0., 1., 0., 1.]))
        }
    })
    .unwrap();
    assert_eq!(result.visible_bodies, 1);
    assert!(!result.scene.triangles.is_empty());
    assert_eq!(result.declines.len(), 1);
    assert_eq!(result.declines[0].feature, FeatureId(1));
    assert_eq!(result.declines[0].cause, "missing appearance");
}

#[test]
fn real_named_fillet_replay_produces_a_viewport_scene() {
    use oblikovati_martensite::{CertifiedNamedSolidEvaluator, PrismResolution};
    use oblikovati_model::{
        feature::FeatureId,
        history::DocumentEditHistory,
        identity::{DocumentGuid, RecipeIdAllocator},
    };
    let mut snapshot = fixture::selected_prism_document();
    fixture::solve_selected_sketches(&mut snapshot);
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    let mut history = DocumentEditHistory::new(snapshot, ids).unwrap();
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let resolution = PrismResolution {
        version: 1,
        weld: 1e-6,
        sew: 1e-7,
        angular: 1e-9,
        display_chordal: 1e-3,
        display_angular: 0.1,
        maximum_tolerance: 1e-5,
    };
    let prism = oblikovati_martensite::retain_certified_named_sketch_prism(
        &history,
        document,
        FeatureId(1),
        resolution,
    )
    .unwrap();
    let edge = prism
        .subentity_reference_keys(&history, document)
        .unwrap()
        .into_iter()
        .find(|key| {
            oblikovati_model::identity::ReferenceEnvelope::decode(key)
                .unwrap()
                .kind
                == 2
        })
        .unwrap();
    let body = prism
        .body_reference_key(&history, document)
        .unwrap()
        .to_vec();
    append_fillet_for_viewport(&mut history, body, edge);
    let mut evaluator = CertifiedNamedSolidEvaluator::new(&history, document, resolution);
    let replay = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let scene = ViewportScene::from_part_evaluation(
        &replay,
        oblikovati_martensite::PrismMeshOptions {
            chord_error: 0.02,
            angular_error: 0.2,
        },
        [0.; 3],
        [0.4, 0.5, 0.6, 1.],
    )
    .unwrap();
    assert_eq!(scene.visible_bodies, 1);
    assert!(scene.declines.is_empty());
    assert!(!scene.scene.triangles.is_empty());
}

fn append_fillet_for_viewport(
    history: &mut oblikovati_model::history::DocumentEditHistory,
    body: Vec<u8>,
    edge: Vec<u8>,
) {
    use oblikovati_model::feature::{
        AuthoredFeature, fillet::FilletDefinition, part::PartFeatureDefinition,
        work_geometry::DatumLength,
    };
    let definition = FilletDefinition::new(body, vec![edge], DatumLength::Constant(0.1)).unwrap();
    let reads = definition.reads();
    let token = history.begin("Fillet").unwrap();
    history
        .edit(&token, |snapshot, ids| {
            let id = ids.allocate_feature().map_err(|cause| cause.to_string())?;
            snapshot
                .work_geometry
                .append(AuthoredFeature::new(
                    id,
                    "Fillet",
                    PartFeatureDefinition::Fillet(Box::new(definition)),
                    reads,
                ))
                .map_err(|cause| cause.to_string())?;
            Ok::<(), String>(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
}
