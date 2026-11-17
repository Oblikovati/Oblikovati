// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, PrismResolution};
use oblikovati_model::{
    history::DocumentEditHistory,
    identity::{DocumentGuid, RecipeIdAllocator},
};

fn resolution() -> PrismResolution {
    PrismResolution {
        version: 1,
        weld: 1e-6,
        sew: 1e-7,
        angular: 1e-9,
        display_chordal: 1e-3,
        display_angular: 0.1,
        maximum_tolerance: 1e-5,
    }
}
fn history() -> DocumentEditHistory {
    let document = crate::scene_fixture::selected_prism_document();
    let ids = RecipeIdAllocator::restore(None, document.parameters.ids()).unwrap();
    DocumentEditHistory::new(document, ids).unwrap()
}
fn rendered(history: &DocumentEditHistory) -> HeadSceneUpdate {
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let mut evaluator = CertifiedNamedSolidEvaluator::new(history, document, resolution());
    let evaluated = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    assert!(evaluated.ordered_bodies().all(|body| {
        body.as_prism()
            .is_some_and(|prism| prism.certification().is_some())
    }));
    render_part_scene(&evaluated, (1, 1), |_| Ok(Some([0., 1., 0., 1.]))).unwrap()
}
fn maximum_z(scene: &ViewportScene) -> f64 {
    scene
        .triangles
        .iter()
        .flat_map(|triangle| triangle.positions)
        .map(|point| f64::from(point[2]))
        .fold(f64::NEG_INFINITY, f64::max)
}
#[test]
fn certified_part_replay_produces_a_scene_and_rebuilds_after_parameter_edit() {
    let mut history = history();
    let initial = rendered(&history);
    assert_eq!(initial.status.visible_bodies, 1);
    assert!(!initial.scene.triangles.is_empty());
    assert!((maximum_z(&initial.scene) - 2.).abs() < 1e-9);
    let token = history.begin("Depth").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let changed = rendered(&history);
    assert_eq!(changed.status.visible_bodies, 1);
    assert!((maximum_z(&changed.scene) - 3.).abs() < 1e-9);
    history.undo().unwrap();
    assert!((maximum_z(&rendered(&history).scene) - 2.).abs() < 1e-9);
}

#[test]
#[ignore = "manual release-mode face presentation timing"]
fn measure_named_face_selection_scene_cost() {
    use oblikovati_martensite::BodyPresentationSource;
    use std::time::Instant;
    let history = history();
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let mut evaluator = CertifiedNamedSolidEvaluator::new(&history, document, resolution());
    let evaluated = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let feature = evaluated
        .health()
        .iter()
        .find(|record| evaluated.body(record.feature).is_some())
        .unwrap()
        .feature;
    let body = evaluated.body(feature).unwrap();
    let face = body
        .face_reference_keys(&history, document)
        .unwrap()
        .remove(0);
    let selection = (1, 1, vec![(feature, face)], [0, 200, 160]);
    let mut plain = Vec::new();
    let mut named = Vec::new();
    let mut recolor = Vec::new();
    for iteration in 0..120 {
        let start = Instant::now();
        std::hint::black_box(body.body_triangles(DISPLAY_MESH).unwrap());
        let baseline = start.elapsed();
        let start = Instant::now();
        let cache = face_display::FaceMeshCache::build((1, 1), &evaluated, &history, document);
        let build = start.elapsed();
        let start = Instant::now();
        let scene = face_display::selected_face_scene(&evaluated, &cache, &selection, |_| {
            Ok(Some([0.5, 0.5, 0.5, 1.]))
        })
        .unwrap();
        let color = start.elapsed();
        assert_eq!(scene.status.visible_bodies, 1);
        assert!(scene.status.presentation_declines.is_empty());
        assert!(
            scene
                .scene
                .triangles
                .iter()
                .any(|item| item.color[1] > item.color[0])
        );
        if iteration >= 20 {
            plain.push(baseline);
            named.push(build);
            recolor.push(color);
        }
    }
    for (label, mut samples) in [
        ("body mesh", plain),
        ("named body mesh", named),
        ("cached face scene", recolor),
    ] {
        samples.sort();
        println!(
            "{label}: n={} p50={:?} p95={:?} max={:?}",
            samples.len(),
            samples[50],
            samples[95],
            samples[99]
        );
    }
}

#[test]
fn named_display_mesh_preserves_unselected_part_triangles() {
    let history = history();
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let mut evaluator = CertifiedNamedSolidEvaluator::new(&history, document, resolution());
    let evaluated = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let color = [0.2, 0.3, 0.4, 1.];
    let ordinary = render_part_scene(&evaluated, (1, 1), |_| Ok(Some(color))).unwrap();
    let cache = face_display::FaceMeshCache::build((1, 1), &evaluated, &history, document);
    let selection = (1, 1, Vec::new(), [0, 200, 160]);
    let named =
        face_display::selected_face_scene(&evaluated, &cache, &selection, |_| Ok(Some(color)))
            .unwrap();
    assert!(named.status.presentation_declines.is_empty());
    assert_eq!(named.status.visible_bodies, ordinary.status.visible_bodies);
    let canonical = |scene: &ViewportScene| {
        let mut triangles: Vec<_> = scene
            .triangles
            .iter()
            .map(|triangle| triangle.positions.map(|point| point.map(f32::to_bits)))
            .collect();
        triangles.sort();
        triangles
    };
    assert_eq!(canonical(&named.scene), canonical(&ordinary.scene));
    assert!(
        named
            .scene
            .triangles
            .iter()
            .all(|triangle| triangle.color == color)
    );
}
