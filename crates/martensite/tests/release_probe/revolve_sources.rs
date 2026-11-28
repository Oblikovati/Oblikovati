// SPDX-License-Identifier: GPL-2.0-only
use super::document_snapshot;

use oblikovati_martensite::{
    CertifiedNamedSolidEvaluator, NamedSolid, PrismResolution, PrismResolutionPolicy,
    certified_sketch_regions_with_policy,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        AuthoredFeature, FeatureId, FeatureReadKey, FeatureSequence,
        part::PartFeatureDefinition,
        sketch_prism::{PrismBoolean, SketchPrismDefinition},
        sketch_revolve::{SketchRevolveDefinition, SketchRevolveDirection},
        work_geometry::{DatumAngle, DatumLength},
    },
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::{DocumentGuid, RecipeIdAllocator},
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};
use oblikovati_persistence::package::DocumentPackage;
use std::f64::consts::{FRAC_PI_2, FRAC_PI_4, PI, TAU};

const DOC: &str = "11111111-2222-3333-4444-555555555555";
const PROFILE_SKETCH: DocumentSketchId = DocumentSketchId(1);
const AXIS_EDGE: SketchEntityId = SketchEntityId(6);

fn resolution() -> PrismResolutionPolicy {
    PrismResolutionPolicy::Fixed(PrismResolution {
        version: 1,
        weld: 1e-7,
        sew: 1e-7,
        angular: 1e-9,
        display_chordal: 0.01,
        display_angular: 0.1,
        maximum_tolerance: 1e-6,
    })
}

fn solve(snapshot: &mut DocumentSnapshot) {
    let solved = snapshot.solve_document_sketches(Default::default(), || false);
    snapshot.sketches = solved.candidate;
}

fn rectangle_fixture() -> (DocumentSnapshot, SketchRevolveDefinition) {
    let mut snapshot = document_snapshot::empty_snapshot();
    let mut sketch = Sketch2D::default();
    for (id, point) in [
        (2, [-0.6, 0.0]),
        (3, [0.6, 0.0]),
        (4, [0.6, 0.5]),
        (5, [-0.6, 0.5]),
    ] {
        sketch.add_point(SketchEntityId(id), point).unwrap();
    }
    for (id, ends) in [(6, [2, 3]), (7, [3, 4]), (8, [4, 5]), (9, [5, 2])] {
        sketch
            .add_line(SketchEntityId(id), ends.map(SketchEntityId))
            .unwrap();
    }
    snapshot
        .sketches
        .add(PROFILE_SKETCH, "Sketch1", SketchPlane::xy(), sketch)
        .unwrap();
    solve(&mut snapshot);
    let regions =
        certified_sketch_regions_with_policy(&snapshot, PROFILE_SKETCH, resolution()).unwrap();
    assert_eq!(regions.len(), 1);
    let base = SketchRevolveDefinition::new(
        PROFILE_SKETCH,
        regions[0].selection(),
        DatumAngle::constant(TAU).unwrap(),
    )
    .unwrap()
    .with_axis(AXIS_EDGE)
    .unwrap()
    .with_full_turn();
    (snapshot, base)
}

fn owner(snapshot: DocumentSnapshot) -> DocumentEditHistory {
    DocumentEditHistory::new(snapshot, RecipeIdAllocator::default()).unwrap()
}

fn eval_single(history: &DocumentEditHistory, feature: FeatureId) -> NamedSolid {
    let document: DocumentGuid = DOC.parse().unwrap();
    let mut backend = CertifiedNamedSolidEvaluator::with_policy(history, document, resolution());
    let part = backend
        .evaluate_private_candidate(history.snapshot())
        .unwrap();
    assert_eq!(part.ordered_bodies().count(), 1, "{:?}", part.health());
    part.body(feature)
        .unwrap_or_else(|| panic!("missing body for {feature:?}: {:?}", part.health()))
        .clone()
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
fn full_revolve_replays_through_history_undo_redo_and_reopen() {
    let (snapshot, revolve) = rectangle_fixture();
    let document: DocumentGuid = DOC.parse().unwrap();
    let mut history = owner(snapshot);
    let token = history.begin("Revolution1").unwrap();
    let feature = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_revolve("Revolution1", revolve.clone(), ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    assert!(
        !history
            .snapshot()
            .sketches
            .get(PROFILE_SKETCH)
            .unwrap()
            .visible()
    );
    let mut program = history.snapshot().work_geometry.part_recipe().unwrap();
    assert_eq!(program.version, 44);
    program.version = 43;
    assert!(FeatureSequence::from_part_recipe(&program).is_err());

    let solid = eval_single(&history, feature);
    assert!((solid.volume(&history, document).unwrap() - 0.3 * PI).abs() < 1e-8);
    let keys = solid.reference_keys(&history, document).unwrap();
    assert_eq!(keys.len(), 4);
    let subentities = solid.subentity_reference_keys(&history, document).unwrap();

    assert_eq!(history.undo().unwrap(), "Revolution1");
    assert!(
        history
            .snapshot()
            .sketches
            .get(PROFILE_SKETCH)
            .unwrap()
            .visible()
    );
    assert_eq!(history.redo().unwrap(), "Revolution1");

    let reopened = reopen(&history);
    let restored = eval_single(&reopened, feature);
    assert_eq!(restored.reference_keys(&reopened, document).unwrap(), keys);
    assert_eq!(
        restored
            .subentity_reference_keys(&reopened, document)
            .unwrap(),
        subentities
    );
    assert!((restored.volume(&reopened, document).unwrap() - 0.3 * PI).abs() < 1e-8);
}

#[test]
fn partial_revolve_directions_and_asymmetric_match_analytic_volumes() {
    let (snapshot, base) = rectangle_fixture();
    let document: DocumentGuid = DOC.parse().unwrap();

    let mut centroids = Vec::new();
    for (direction, expected_vol) in [
        (SketchRevolveDirection::Positive, 0.075 * PI),
        (SketchRevolveDirection::Negative, 0.075 * PI),
        (SketchRevolveDirection::Symmetric, 0.075 * PI),
    ] {
        let def = SketchRevolveDefinition::new(
            PROFILE_SKETCH,
            base.selection().clone(),
            DatumAngle::constant(FRAC_PI_2).unwrap(),
        )
        .unwrap()
        .with_axis(AXIS_EDGE)
        .unwrap()
        .with_direction(direction);
        let mut history = owner(snapshot.clone());
        let token = history.begin("Revolution1").unwrap();
        let feature = history
            .edit(&token, |s, ids| {
                s.create_sketch_revolve("Revolution1", def, ids)
            })
            .unwrap();
        history.commit(&token, false).unwrap();
        let solid = eval_single(&history, feature);
        let key = solid
            .body_reference_key(&history, document)
            .unwrap()
            .to_vec();
        let props = solid.physical_properties(&history, document, &key).unwrap();
        assert!(
            (props.geometry.volume - expected_vol).abs() < 1e-8,
            "{direction:?}: got {}, expected {expected_vol}",
            props.geometry.volume
        );
        assert_eq!(solid.reference_keys(&history, document).unwrap().len(), 6);
        centroids.push(props.geometry.centroid);
    }
    // Positive, Negative, and Symmetric sweep distinct quadrants in Z/Y.
    assert!((centroids[0][2] - centroids[1][2]).abs() > 0.1);
    assert!((centroids[0][2] - centroids[2][2]).abs() > 0.05);

    let asym = SketchRevolveDefinition::new(
        PROFILE_SKETCH,
        base.selection().clone(),
        DatumAngle::constant(FRAC_PI_2).unwrap(),
    )
    .unwrap()
    .with_axis(AXIS_EDGE)
    .unwrap()
    .with_asymmetric_angle(DatumAngle::constant(FRAC_PI_4).unwrap())
    .unwrap();
    let mut history = owner(snapshot);
    let token = history.begin("Revolution1").unwrap();
    let feature = history
        .edit(&token, |s, ids| {
            s.create_sketch_revolve("Revolution1", asym, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let reopened = reopen(&history);
    let solid = eval_single(&reopened, feature);
    assert!((solid.volume(&reopened, document).unwrap() - 0.1125 * PI).abs() < 1e-8);
}

#[test]
fn surface_revolve_partial_and_asymmetric_retain_named_topology_through_reopen() {
    let (snapshot, base) = rectangle_fixture();
    let document: DocumentGuid = DOC.parse().unwrap();

    let surface = SketchRevolveDefinition::new(
        PROFILE_SKETCH,
        base.selection().clone(),
        DatumAngle::constant(FRAC_PI_2).unwrap(),
    )
    .unwrap()
    .with_axis(AXIS_EDGE)
    .unwrap()
    .with_asymmetric_angle(DatumAngle::constant(FRAC_PI_4).unwrap())
    .unwrap()
    .with_surface_output(true);

    let mut history = owner(snapshot);
    let token = history.begin("RevolutionSrf1").unwrap();
    let feature = history
        .edit(&token, |s, ids| {
            s.create_sketch_revolve("RevolutionSrf1", surface, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let body = eval_single(&history, feature);
    assert!(body.is_surface());
    let keys = body.reference_keys(&history, document).unwrap();
    assert_eq!(keys.len(), 4); // body + 3 non-axis faces
    let subentities = body.subentity_reference_keys(&history, document).unwrap();

    let reopened = reopen(&history);
    let restored = eval_single(&reopened, feature);
    assert!(restored.is_surface());
    assert_eq!(restored.reference_keys(&reopened, document).unwrap(), keys);
    assert_eq!(
        restored
            .subentity_reference_keys(&reopened, document)
            .unwrap(),
        subentities
    );
}

#[test]
fn revolve_booleans_join_cut_and_intersect_match_analytic_volumes() {
    let mut snapshot = document_snapshot::empty_snapshot();
    // Matches fixtures/cad/revolve/revolve-boolean-base.opd:
    // Block [0, 4] x [0, 3] x [0, 1] (volume = 12 cm³).
    let block_sketch = DocumentSketchId(1);
    let mut block = Sketch2D::default();
    for (id, point) in [(1, [0., 0.]), (2, [4., 0.]), (3, [4., 3.]), (4, [0., 3.])] {
        block.add_point(SketchEntityId(id), point).unwrap();
    }
    for (id, ends) in [(10, [1, 2]), (11, [2, 3]), (12, [3, 4]), (13, [4, 1])] {
        block
            .add_line(SketchEntityId(id), ends.map(SketchEntityId))
            .unwrap();
    }
    snapshot
        .sketches
        .add(block_sketch, "Profile", SketchPlane::xy(), block)
        .unwrap();
    solve(&mut snapshot);
    let block_regions =
        certified_sketch_regions_with_policy(&snapshot, block_sketch, resolution()).unwrap();
    let prism = SketchPrismDefinition::new(
        block_sketch,
        block_regions[0].selection(),
        [0., 0., 1.].map(DatumLength::Constant),
    )
    .unwrap();
    let prism_reads = prism.reads(&snapshot).unwrap();
    let base_feature = FeatureId(1);
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            base_feature,
            "Extrusion1",
            PartFeatureDefinition::SketchPrism(Box::new(prism)),
            prism_reads,
        ))
        .unwrap();

    // XZ sketch on front face Y = 0: x in [1, 2], z in [1.0, 1.4], axis along Z = 1.0 (top-front edge).
    // Full 360° revolve creates cylinder r=0.4, L=1.0 -> 1/4 inside block (0.04π), 3/4 outside (0.12π).
    let rev_sketch = DocumentSketchId(2);
    let mut rev = Sketch2D::default();
    for (id, point) in [
        (20, [1.0, 1.0]),
        (21, [2.0, 1.0]),
        (22, [2.0, 1.4]),
        (23, [1.0, 1.4]),
    ] {
        rev.add_point(SketchEntityId(id), point).unwrap();
    }
    for (id, ends) in [
        (30, [20, 21]),
        (31, [21, 22]),
        (32, [22, 23]),
        (33, [23, 20]),
    ] {
        rev.add_line(SketchEntityId(id), ends.map(SketchEntityId))
            .unwrap();
    }
    let xz =
        SketchPlane::new([0., 0., 0.], [1., 0., 0.], [0., 0., 1.], Default::default()).unwrap();
    snapshot
        .sketches
        .add(rev_sketch, "Sketch1", xz, rev)
        .unwrap();
    solve(&mut snapshot);
    let rev_regions =
        certified_sketch_regions_with_policy(&snapshot, rev_sketch, resolution()).unwrap();

    let document: DocumentGuid = DOC.parse().unwrap();
    let cyl = PI * 0.4_f64.powi(2) * 1.0;
    let quarter = 0.25 * cyl;
    for (operation, expected) in [
        (PrismBoolean::Join, 12. + 3. * quarter),
        (PrismBoolean::Cut, 12. - quarter),
        (PrismBoolean::Intersect, quarter),
    ] {
        let mut history = owner(snapshot.clone());
        let body_key = eval_single(&history, base_feature)
            .body_reference_key(&history, document)
            .unwrap()
            .to_vec();
        let def = SketchRevolveDefinition::new(
            rev_sketch,
            rev_regions[0].selection(),
            DatumAngle::constant(TAU).unwrap(),
        )
        .unwrap()
        .with_axis(SketchEntityId(30))
        .unwrap()
        .with_full_turn()
        .with_boolean_target(operation, body_key.clone())
        .unwrap();
        let token = history.begin("Revolution1").unwrap();
        let feature = history
            .edit(&token, |s, ids| {
                s.create_sketch_revolve("Revolution1", def, ids)
            })
            .unwrap();
        history.commit(&token, false).unwrap();
        assert!(
            history
                .snapshot()
                .work_geometry
                .get(feature)
                .unwrap()
                .reads()
                .keys()
                .contains(&FeatureReadKey::Feature(base_feature))
        );
        let reopened = reopen(&history);
        let solid = eval_single(&reopened, feature);
        assert_eq!(
            solid.body_reference_key(&reopened, document).unwrap(),
            body_key
        );
        let actual = solid.volume(&reopened, document).unwrap();
        assert!(
            (actual - expected).abs() < 1e-6,
            "{operation}: got {actual}, expected {expected}"
        );
    }
}

#[test]
fn revolved_cylinder_display_mesh_produces_smooth_outward_radial_normals() {
    use oblikovati_martensite::{BodyPresentationSource, PrismMeshOptions};
    let (snapshot, def) = rectangle_fixture();
    let mut history = owner(snapshot);
    let token = history.begin("Revolution1").unwrap();
    let feature = history
        .edit(&token, |s, ids| {
            s.create_sketch_revolve("Revolution1", def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let solid = eval_single(&history, feature);
    let faces = solid
        .body_face_display_triangles(PrismMeshOptions {
            chord_error: 0.002,
            angular_error: 0.1,
        })
        .unwrap();
    let mut curved_vertices = 0usize;
    let mut distinct_normals = std::collections::BTreeSet::new();
    for face in &faces {
        for tri in face {
            let normals = tri.normals.expect("expected analytical normals");
            for (pos, normal) in tri.positions.iter().zip(normals.iter()) {
                let len =
                    (normal[0] * normal[0] + normal[1] * normal[1] + normal[2] * normal[2]).sqrt();
                assert!((len - 1.0).abs() < 1e-5, "non-unit normal {normal:?}");
                let radius = (pos[1] * pos[1] + pos[2] * pos[2]).sqrt();
                if radius > 0.45 && normal[0].abs() < 0.05 {
                    let expected = [0., pos[1] / radius, pos[2] / radius];
                    let dot =
                        normal[0] * expected[0] + normal[1] * expected[1] + normal[2] * expected[2];
                    assert!(
                        dot > 0.999,
                        "expected outward radial normal {expected:?} at {pos:?}, got {normal:?}"
                    );
                    distinct_normals.insert((
                        (normal[1] * 100.).round() as i32,
                        (normal[2] * 100.).round() as i32,
                    ));
                    curved_vertices += 1;
                }
            }
        }
    }
    assert!(
        curved_vertices >= 60 && distinct_normals.len() >= 30,
        "expected smoothly tessellated radial cylinder normals: vertices={curved_vertices}, distinct={}",
        distinct_normals.len()
    );
}
