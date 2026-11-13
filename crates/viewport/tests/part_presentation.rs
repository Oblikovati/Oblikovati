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
    let body = evaluated
        .body(oblikovati_model::feature::FeatureId(1))
        .unwrap();
    let original = body.physical_properties().unwrap();
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
    assert_eq!(body.physical_properties().unwrap(), original);
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
    let evaluated = document
        .evaluate_part_features(&mut oblikovati_martensite::MartensitePrismEvaluator)
        .unwrap();
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
