// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{CertifiedNamedPrismEvaluator, PrismResolution};
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
    let mut evaluator = CertifiedNamedPrismEvaluator::new(history, document, resolution());
    let evaluated = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    assert!(
        evaluated
            .ordered_bodies()
            .all(|body| body.certification().is_some())
    );
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
