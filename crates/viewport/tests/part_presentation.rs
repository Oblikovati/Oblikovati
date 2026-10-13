// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[allow(dead_code)]
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
mod fixture;
#[test]
fn mesh_declines_retain_feature_identity_without_mutating_analytic_outputs() {
    let document = fixture::selected_prism_document();
    let evaluated = document
        .evaluate_part_features(&mut oblikovati_martensite::MartensitePrismEvaluator)
        .unwrap();
    let options = oblikovati_martensite::PrismMeshOptions {
        chord_error: 0.,
        angular_error: 0.5,
    };
    let presentation =
        ViewportScene::from_part_evaluation(&evaluated, options, [0.; 3], [0., 1., 0., 1.])
            .unwrap();
    verify_mesh_decline(&presentation);
    let body = evaluated
        .body(oblikovati_model::feature::FeatureId(1))
        .unwrap();
    assert_eq!(body.report().properties.volume, 24.);
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
    let evaluated = document
        .evaluate_part_features(&mut oblikovati_martensite::MartensitePrismEvaluator)
        .unwrap();
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
    );
    assert_eq!(rejected.err().unwrap(), "stale visibility key");
}
