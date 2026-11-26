// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, NamedSolid};
use oblikovati_model::{
    feature::part::PartFeatureDefinition,
    history::DocumentEditHistory,
    identity::{DocumentGuid, RecipeIdAllocator, ReferenceEnvelope},
};
use oblikovati_persistence::package::DocumentPackage;

#[test]
fn named_sweep_replays_through_creation_history_path_edit_and_reopen() {
    let (snapshot, sweep) = fixture();
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let mut history = owner(snapshot);
    let before = history.snapshot().clone();
    let prefix = before
        .evaluate_part_features(&mut CertifiedNamedSolidEvaluator::with_policy(
            &history,
            document,
            resolution(),
        ))
        .unwrap();
    let token = history.begin("Sweep1").unwrap();
    let feature = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_sweep("Sweep1", sweep.clone(), ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let candidate = history.snapshot();
    let mut backend = CertifiedNamedSolidEvaluator::with_policy(&history, document, resolution());
    let mut replay_candidate = candidate.clone();
    for sketch in [PROFILE, PATH] {
        assert!(!candidate.sketches.get(sketch).unwrap().visible());
        replay_candidate.sketches.set_visible(sketch, true).unwrap();
    }
    let tail = prefix
        .replay_candidate_tail(&replay_candidate, &mut backend)
        .unwrap();
    assert_eq!(tail.body_count(), 1);
    let original = named(&history, document, feature);
    let original_keys = keys(&original, &history, document, 3.);
    assert_eq!(
        original_keys,
        keys(tail.body(feature).unwrap(), &history, document, 3.)
    );
    assert_eq!(candidate.work_geometry.part_recipe().unwrap().version, 28);
    for sketch in [PROFILE, PATH] {
        let consumers: Vec<_> = candidate
            .work_geometry
            .sketch_consumers(sketch)
            .map(|entry| entry.id())
            .collect();
        assert_eq!(consumers, [feature]);
    }

    assert_eq!(history.undo().unwrap(), "Sweep1");
    assert!(history.snapshot().work_geometry.get(feature).is_err());
    for sketch in [PROFILE, PATH] {
        assert!(history.snapshot().sketches.get(sketch).unwrap().visible());
    }
    assert!(original.reference_keys(&history, document).is_err());
    assert_eq!(history.redo().unwrap(), "Sweep1");
    assert_eq!(
        original_keys,
        keys(&named(&history, document, feature), &history, document, 3.)
    );

    let token = history.begin("Edit sweep path").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.sketches.replace_geometry(PATH, path(4., SPINE))?;
            solve(snapshot);
            Ok::<_, oblikovati_model::sketch::SketchError>(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let edited = named(&history, document, feature);
    assert_eq!(original_keys, keys(&edited, &history, document, 4.));

    let reopened = reopen(&history);
    let authored = reopened.snapshot().work_geometry.get(feature).unwrap();
    assert_eq!(
        authored.definition(),
        &PartFeatureDefinition::SketchSweep(Box::new(sweep))
    );
    assert_eq!(
        original_keys,
        keys(
            &named(&reopened, document, feature),
            &reopened,
            document,
            4.
        )
    );
    assert!(edited.reference_keys(&reopened, document).is_err());
}

fn owner(snapshot: DocumentSnapshot) -> DocumentEditHistory {
    DocumentEditHistory::new(snapshot, RecipeIdAllocator::default()).unwrap()
}

fn named(history: &DocumentEditHistory, document: DocumentGuid, feature: FeatureId) -> NamedSolid {
    let mut backend = CertifiedNamedSolidEvaluator::with_policy(history, document, resolution());
    let result = backend
        .evaluate_private_candidate(history.snapshot())
        .unwrap();
    assert_eq!(result.ordered_bodies().count(), 1, "{:?}", result.health());
    result.body_from_source(feature).unwrap().clone()
}

fn keys(
    body: &NamedSolid,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    height: f64,
) -> Vec<Vec<u8>> {
    assert!((body.volume(history, document).unwrap() - PI * height / 4.).abs() < 1e-8);
    let mut keys = body.reference_keys(history, document).unwrap();
    assert_eq!(keys.len(), 4);
    let subentities = body.subentity_reference_keys(history, document).unwrap();
    let edge_lengths = subentities
        .iter()
        .filter(|key| ReferenceEnvelope::decode(key).unwrap().kind == 2)
        .map(|key| body.edge_length(history, document, key).unwrap())
        .collect::<Vec<_>>();
    assert_eq!(subentities.len(), 5);
    assert_eq!(edge_lengths.len(), 3);
    assert_eq!(
        edge_lengths
            .iter()
            .filter(|length| (*length - PI).abs() < 1e-8)
            .count(),
        2
    );
    assert!(
        edge_lengths
            .iter()
            .any(|length| (*length - height).abs() < 1e-8)
    );
    keys.extend(subentities);
    keys
}

fn reopen(history: &DocumentEditHistory) -> DocumentEditHistory {
    let current = history.snapshot();
    let package = DocumentPackage::decode(b"schemaVersion: 3\ndocumentType: 1\n")
        .unwrap()
        .with_sketches(&current.sketches, PATH.0, SPINE.0)
        .unwrap()
        .with_work_geometry(&current.work_geometry, history.feature_high_water())
        .unwrap();
    let reopened = DocumentPackage::decode(&package.encode().unwrap()).unwrap();
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot.work_geometry = reopened.restore_work_geometry().unwrap().program;
    snapshot.sketches = reopened.restore_sketches().unwrap().sketches;
    solve(&mut snapshot);
    owner(snapshot)
}

#[test]
fn tapered_sweep_keeps_named_topology_when_restored_or_set_to_zero() {
    use oblikovati_model::feature::work_geometry::DatumAngle;

    let (snapshot, sweep) = fixture();
    let original = sweep
        .clone()
        .with_taper(DatumAngle::constant(5_f64.to_radians()).unwrap())
        .unwrap();
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let mut history = owner(snapshot);
    let token = history.begin("Sweep1").unwrap();
    let feature = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_sweep("Sweep1", original.clone(), ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let body = named(&history, document, feature);
    let faces = body.reference_keys(&history, document).unwrap();
    let subentities = body.subentity_reference_keys(&history, document).unwrap();
    assert_eq!(faces.len(), 4);
    assert_eq!(subentities.len(), 5);
    let recipe = history.snapshot().work_geometry.part_recipe().unwrap();
    assert_eq!(recipe.version, 29);
    let mut old_recipe = recipe.clone();
    old_recipe.version = 28;
    assert!(oblikovati_model::feature::FeatureSequence::from_part_recipe(&old_recipe).is_err());

    let reopened = reopen(&history);
    let reopened_body = named(&reopened, document, feature);
    assert_eq!(
        reopened_body.reference_keys(&reopened, document).unwrap(),
        faces
    );
    assert_eq!(
        reopened_body
            .subentity_reference_keys(&reopened, document)
            .unwrap(),
        subentities
    );
    let zero = sweep.with_taper(DatumAngle::constant(0.).unwrap()).unwrap();
    let token = history.begin("Edit Sweep1").unwrap();
    history
        .edit(&token, |snapshot, _| {
            let reads = zero.reads(snapshot).unwrap();
            snapshot.work_geometry.replace(
                feature,
                PartFeatureDefinition::SketchSweep(Box::new(zero)),
                reads,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let straight = named(&history, document, feature);
    assert_eq!(straight.reference_keys(&history, document).unwrap(), faces);
    assert_eq!(
        straight
            .subentity_reference_keys(&history, document)
            .unwrap(),
        subentities
    );
    assert!((straight.volume(&history, document).unwrap() - 0.75 * PI).abs() < 1e-8);
}

#[test]
fn fixed_orientation_preserves_section_and_topology_after_reopen() {
    use oblikovati_model::feature::{
        sketch_sweep::SketchSweepOrientation, work_geometry::DatumAngle,
    };
    let (snapshot, sweep) = fixture();
    let fixed = sweep
        .with_taper(DatumAngle::constant(5_f64.to_radians()).unwrap())
        .unwrap()
        .with_fixed_orientation();
    assert!(fixed.recipe().taper.is_none());
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let mut history = owner(snapshot);
    let token = history.begin("Sweep1").unwrap();
    let feature = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_sweep("Sweep1", fixed, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let body = named(&history, document, feature);
    let initial_keys = keys(&body, &history, document, 3.);
    assert_eq!(
        history
            .snapshot()
            .work_geometry
            .part_recipe()
            .unwrap()
            .version,
        30
    );
    let reopened = reopen(&history);
    let PartFeatureDefinition::SketchSweep(sweep) = reopened
        .snapshot()
        .work_geometry
        .get(feature)
        .unwrap()
        .definition()
    else {
        panic!("expected the saved Sweep");
    };
    assert_eq!(sweep.recipe().orientation, SketchSweepOrientation::Fixed);
    assert_eq!(
        keys(
            &named(&reopened, document, feature),
            &reopened,
            document,
            3.
        ),
        initial_keys
    );
}
