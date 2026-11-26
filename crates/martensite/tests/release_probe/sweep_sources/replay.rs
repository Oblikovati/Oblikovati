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

#[test]
fn sweep_join_replaces_the_named_base_and_retains_its_reference() {
    let (snapshot, sweep) = fixture();
    let mut history = owner(with_block_base(snapshot));
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let base = FeatureId(2);
    let key = named(&history, document, base)
        .body_reference_key(&history, document)
        .unwrap()
        .to_vec();
    let token = history.begin("Sweep1").unwrap();
    let feature = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_sweep("Sweep1", sweep.clone(), ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let mut backend = CertifiedNamedSolidEvaluator::with_policy(&history, document, resolution());
    assert_eq!(
        backend
            .evaluate_private_candidate(history.snapshot())
            .unwrap()
            .ordered_bodies()
            .count(),
        2
    );

    let joined = sweep.with_join_target(key.clone()).unwrap();
    let token = history.begin("Join Sweep1").unwrap();
    history
        .edit(&token, |snapshot, _| {
            let reads = joined.reads(snapshot).unwrap();
            assert!(reads.keys().contains(&FeatureReadKey::Feature(base)));
            snapshot.work_geometry.replace(
                feature,
                PartFeatureDefinition::SketchSweep(Box::new(joined)),
                reads,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let mut recipe = history.snapshot().work_geometry.part_recipe().unwrap();
    assert_eq!(recipe.version, 32);
    let restored = oblikovati_model::feature::FeatureSequence::from_part_recipe(&recipe).unwrap();
    assert!(matches!(restored.get(feature).unwrap().definition(),
        PartFeatureDefinition::SketchSweep(sweep) if sweep.join_target() == Some((key.as_slice(), base))));
    recipe.version = 31;
    assert!(oblikovati_model::feature::FeatureSequence::from_part_recipe(&recipe).is_err());

    let solid = named(&history, document, feature);
    assert_eq!(
        solid.body_reference_key(&history, document).unwrap(),
        key.as_slice()
    );
    assert!((solid.volume(&history, document).unwrap() - (8. + 0.75 * PI)).abs() < 1e-8);
}

#[test]
fn inward_sweep_cut_preserves_the_body_key_through_reopen() {
    use oblikovati_model::feature::sketch_prism::PrismBoolean;

    let (mut snapshot, sweep) = fixture();
    snapshot
        .sketches
        .replace_geometry(PATH, path(-1., SPINE))
        .unwrap();
    let mut history = owner(with_block_base(snapshot));
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let base = FeatureId(2);
    let body_key = named(&history, document, base)
        .body_reference_key(&history, document)
        .unwrap()
        .to_vec();
    let cut = sweep
        .with_boolean_target(PrismBoolean::Cut, body_key.clone())
        .unwrap();
    let token = history.begin("Sweep Cut").unwrap();
    let feature = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_sweep("Sweep1", cut, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert_eq!(
        history
            .snapshot()
            .work_geometry
            .part_recipe()
            .unwrap()
            .version,
        33
    );

    let reopened = reopen(&history);
    let restored = reopened.snapshot().work_geometry.get(feature).unwrap();
    assert!(
        restored
            .reads()
            .keys()
            .contains(&FeatureReadKey::Feature(base))
    );
    assert!(matches!(restored.definition(),
        PartFeatureDefinition::SketchSweep(sweep)
            if sweep.boolean_target() == Some((PrismBoolean::Cut, body_key.as_slice(), base))));
    let solid = named(&reopened, document, feature);
    assert_eq!(
        solid.body_reference_key(&reopened, document).unwrap(),
        body_key
    );
    assert!((solid.volume(&reopened, document).unwrap() - (8. - 0.25 * PI)).abs() < 1e-8);
}

fn with_block_base(mut snapshot: DocumentSnapshot) -> DocumentSnapshot {
    use oblikovati_model::feature::{
        sketch_prism::SketchPrismDefinition, work_geometry::DatumLength,
    };
    let sketch = DocumentSketchId(3);
    let mut outline = Sketch2D::default();
    let points = [20, 21, 22, 23].map(SketchEntityId);
    for (id, point) in points
        .into_iter()
        .zip([[-1., -1.], [1., -1.], [1., 1.], [-1., 1.]])
    {
        outline.add_point(id, point).unwrap();
    }
    for (i, id) in [24, 25, 26, 27].map(SketchEntityId).into_iter().enumerate() {
        outline
            .add_line(id, [points[i], points[(i + 1) % 4]])
            .unwrap();
    }
    snapshot
        .sketches
        .add(sketch, "Block", SketchPlane::xy(), outline)
        .unwrap();
    solve(&mut snapshot);
    let region = certified_sketch_regions_with_policy(&snapshot, sketch, resolution()).unwrap();
    let base = SketchPrismDefinition::new(
        sketch,
        region[0].selection(),
        [0., 0., -2.].map(DatumLength::Constant),
    )
    .unwrap();
    let reads = base.reads(&snapshot).unwrap();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(2),
            "Extrusion1",
            PartFeatureDefinition::SketchPrism(Box::new(base)),
            reads,
        ))
        .unwrap();
    snapshot
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
        .with_sketch_reservations(&current.sketches, history.sketch_reservations())
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
fn coaxial_twist_reopens_with_its_angle_and_analytic_tapered_body() {
    use oblikovati_model::feature::{FeatureSequence, work_geometry::DatumAngle};

    let (snapshot, sweep) = fixture();
    let taper = 5_f64.to_radians();
    let twist = DatumAngle::constant(PI / 2.).unwrap();
    let authored = sweep
        .with_taper(DatumAngle::constant(taper).unwrap())
        .unwrap()
        .with_twist(twist)
        .unwrap();
    let mut history = owner(snapshot);
    let token = history.begin("Sweep1").unwrap();
    let feature = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_sweep("Sweep1", authored, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let mut program = history.snapshot().work_geometry.part_recipe().unwrap();
    assert_eq!(program.version, 31);
    program.version = 30;
    assert!(FeatureSequence::from_part_recipe(&program).is_err());

    let reopened = reopen(&history);
    let entry = reopened.snapshot().work_geometry.get(feature).unwrap();
    let PartFeatureDefinition::SketchSweep(restored) = entry.definition() else {
        panic!("expected a persisted Sweep")
    };
    assert_eq!(restored.recipe().twist, Some(twist));
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let original = named(&history, document, feature);
    let solid = named(&reopened, document, feature);
    assert_eq!(
        solid.reference_keys(&reopened, document).unwrap(),
        original.reference_keys(&history, document).unwrap()
    );
    let end_radius = 0.5 + 3. * taper.tan();
    let expected = PI * (0.25 + 0.5 * end_radius + end_radius * end_radius);
    assert!((solid.volume(&reopened, document).unwrap() - expected).abs() < 1e-8);
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
