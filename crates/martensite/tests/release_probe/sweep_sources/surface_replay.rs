// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn surface_sweep_reopens_beside_the_solid_with_exact_references() {
    let (snapshot, sweep) = fixture();
    let mut history = owner(with_block_base(snapshot));
    let document = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let recipe = sweep.with_surface_output(true);
    let token = history.begin("SweepSrf1").unwrap();
    let feature = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_sweep("SweepSrf1", recipe.clone(), ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let program = history.snapshot().work_geometry.part_recipe().unwrap();
    assert_eq!(program.version, 34);

    let original = surface_keys(&history, document, feature);
    let reopened = reopen(&history);
    let saved = reopened.snapshot().work_geometry.get(feature).unwrap();
    assert_eq!(
        saved.definition(),
        &PartFeatureDefinition::SketchSweep(Box::new(recipe))
    );
    assert_eq!(surface_keys(&reopened, document, feature), original);
}

fn surface_keys(
    history: &DocumentEditHistory,
    document: DocumentGuid,
    feature: FeatureId,
) -> Vec<Vec<u8>> {
    let mut backend = CertifiedNamedSolidEvaluator::with_policy(history, document, resolution());
    let part = backend
        .evaluate_private_candidate(history.snapshot())
        .unwrap();
    assert_eq!(part.ordered_bodies().count(), 2, "{:?}", part.health());
    assert!(!part.body(FeatureId(2)).unwrap().is_surface());
    let surface = part.body(feature).unwrap();
    assert!(surface.is_surface());
    let body_key = surface.body_reference_key(history, document).unwrap();
    let topology = surface
        .topology_summary(history, document, body_key)
        .unwrap();
    assert_eq!(topology.faces, 1);
    assert_eq!(topology.edges, 3);
    assert_eq!(topology.vertices, 2);
    let NamedSolid::Derived(shell) = surface else {
        panic!("expected a named derived shell");
    };
    assert!(
        !shell
            .display_triangles(oblikovati_martensite::PrismMeshOptions {
                angular_error: 0.25,
                chord_error: 0.01,
            })
            .unwrap()
            .is_empty()
    );
    surface.subentity_reference_keys(history, document).unwrap()
}
