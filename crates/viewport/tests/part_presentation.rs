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
