// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{PrismFaceRole, retain_certified_named_sketch_prism};
use oblikovati_model::{
    feature::{AuthoredFeature, FeatureId},
    history::DocumentEditHistory,
    identity::{DocumentGuid, RecipeIdAllocator, SemanticReferenceName},
};
use serde::Deserialize;
use std::collections::BTreeSet;

const REORDERED: &[u8] = include_bytes!(
    "../../../../architecture/rust-port/receipts/2026-11-14-go-face-lineage/reordered.obk"
);
const ORIGINAL_WITNESS: &str = include_str!(
    "../../../../architecture/rust-port/receipts/2026-11-14-go-face-lineage/original.json"
);
const REORDERED_WITNESS: &str = include_str!(
    "../../../../architecture/rust-port/receipts/2026-11-14-go-face-lineage/reordered.json"
);
#[derive(Debug, Deserialize, PartialEq)]
struct GoSide {
    side_index: usize,
    source_line_id: u64,
    face_key_hex: String,
}
#[derive(Deserialize)]
struct GoLineage {
    before: Vec<GoSide>,
    after: Vec<GoSide>,
}
fn document_guid() -> DocumentGuid {
    "904a0ae7-db65-4a7d-b9d0-1dcf1bbcfd02".parse().unwrap()
}
fn imported_history(bytes: &[u8]) -> DocumentEditHistory {
    let package = DocumentPackage::decode(bytes).unwrap();
    let candidate = package
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    let mut snapshot = candidate_document(&candidate.sketches);
    let feature = AuthoredFeature::sketch_prism(
        FeatureId(1),
        &candidate.extrusions[0].name,
        candidate.extrusions[0].definition.clone(),
        &snapshot,
    )
    .unwrap();
    snapshot.work_geometry.append(feature).unwrap();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    DocumentEditHistory::new(snapshot, ids).unwrap()
}
fn native_side_name(line: u64) -> Vec<u8> {
    SemanticReferenceName::new(document_guid(), 1, 1, "sketch-prism", "side", &[1, line])
        .unwrap()
        .encode()
        .unwrap()
}
fn check_witness(history: &DocumentEditHistory, rows: &[GoSide], expected_area: f64) {
    let named = retain_certified_named_sketch_prism(
        history,
        document_guid(),
        FeatureId(1),
        app_resolution(),
    )
    .unwrap();
    let mut legacy = BTreeSet::new();
    let mut native = BTreeSet::new();
    for (expected_index, row) in rows.iter().enumerate() {
        assert_eq!(row.side_index, expected_index);
        let expected = format!("\x03Extrusion1:side#{expected_index}");
        let expected_hex: String = expected.bytes().map(|byte| format!("{byte:02x}")).collect();
        assert_eq!(row.face_key_hex, expected_hex);
        assert!(legacy.insert(row.face_key_hex.clone()));
        let key = native_side_name(row.source_line_id);
        assert!(native.insert(key.clone()));
        let face = named.face_report(history, document_guid(), &key).unwrap();
        assert_eq!(face.role, PrismFaceRole::ProfileEdge(row.source_line_id));
        assert_eq!(face.loops, 1);
        assert_analytic_scalar(face.area, expected_area);
    }
    assert_eq!(legacy.len(), 4);
    assert_eq!(native.len(), 4);
}
fn reopened_history(history: &DocumentEditHistory) -> DocumentEditHistory {
    let snapshot = history.snapshot();
    let package = DocumentPackage::decode(b"schemaVersion: 3\ndocumentType: 1\n")
        .unwrap()
        .with_parameters(&snapshot.parameters, snapshot.parameters.units().clone())
        .unwrap()
        .with_sketches(&snapshot.sketches, 1, 9)
        .unwrap()
        .with_work_geometry(
            &snapshot.work_geometry,
            snapshot
                .work_geometry
                .features()
                .iter()
                .map(|feature| feature.id().0)
                .max()
                .unwrap_or(0),
        )
        .unwrap();
    let reopened = DocumentPackage::decode(&package.encode().unwrap()).unwrap();
    let mut restored = snapshot.clone();
    restored.parameters = reopened.restore_document_parameters().unwrap().parameters;
    restored.sketches = reopened.restore_sketches().unwrap().sketches;
    restored.work_geometry = reopened.restore_work_geometry().unwrap().program;
    solve_selected_sketches(&mut restored);
    let ids = RecipeIdAllocator::restore(None, restored.parameters.ids()).unwrap();
    DocumentEditHistory::new(restored, ids).unwrap()
}
#[test]
fn go_side_witnesses_resolve_to_their_source_named_rust_faces() {
    for (fixture, witness) in [(GO_CUBE, ORIGINAL_WITNESS), (REORDERED, REORDERED_WITNESS)] {
        let data: GoLineage = serde_json::from_str(witness).unwrap();
        assert_eq!(data.before, data.after);
        let mut history = imported_history(fixture);
        check_witness(&history, &data.before, 1.0);
        let token = history.begin("size").unwrap();
        history
            .edit(&token, |snapshot, _| {
                let id = snapshot.parameters.find("size").unwrap();
                snapshot.parameters.set_expression(id, "20 mm")?;
                solve_selected_sketches(snapshot);
                Ok::<(), oblikovati_model::parameter::ParameterError<u64>>(())
            })
            .unwrap();
        history.commit(&token, false).unwrap();
        check_witness(&history, &data.after, 2.0);
        history.undo().unwrap();
        check_witness(&history, &data.before, 1.0);
        history.redo().unwrap();
        check_witness(&history, &data.after, 2.0);
        check_witness(&reopened_history(&history), &data.after, 2.0);
    }
}

#[test]
fn go_side_index_depends_on_source_traversal_order() {
    let original: GoLineage = serde_json::from_str(ORIGINAL_WITNESS).unwrap();
    let reordered: GoLineage = serde_json::from_str(REORDERED_WITNESS).unwrap();
    assert_eq!(
        original.before[0].face_key_hex,
        reordered.before[0].face_key_hex
    );
    assert_ne!(
        original.before[0].source_line_id,
        reordered.before[0].source_line_id
    );
}

#[test]
fn named_planar_face_frame_tracks_the_go_dependent_plane_witness() {
    let mut history = imported_history(GO_CUBE);
    let face_key = native_side_name(7);
    check_cube_frames(&history, 1.0);
    let prior = retain_certified_named_sketch_prism(
        &history,
        document_guid(),
        FeatureId(1),
        app_resolution(),
    )
    .unwrap();
    let token = history.begin("size").unwrap();
    history
        .edit(&token, |snapshot, _| {
            let id = snapshot.parameters.find("size").unwrap();
            snapshot.parameters.set_expression(id, "20 mm")?;
            solve_selected_sketches(snapshot);
            Ok::<(), oblikovati_model::parameter::ParameterError<u64>>(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert!(
        prior
            .face_plane(&history, document_guid(), &face_key)
            .is_err()
    );
    check_cube_frames(&history, 2.0);
    history.undo().unwrap();
    check_cube_frames(&history, 1.0);
    history.redo().unwrap();
    check_cube_frames(&history, 2.0);
    check_cube_frames(&reopened_history(&history), 2.0);
}
fn cap_name(tag: &str) -> Vec<u8> {
    SemanticReferenceName::new(document_guid(), 1, 1, "sketch-prism", tag, &[1])
        .unwrap()
        .encode()
        .unwrap()
}
fn check_cube_frames(history: &DocumentEditHistory, size: f64) {
    let named = retain_certified_named_sketch_prism(
        history,
        document_guid(),
        FeatureId(1),
        app_resolution(),
    )
    .unwrap();
    let cases = [
        (cap_name("start"), [0., 0., 0.], [0., 0., -1.]),
        (cap_name("end"), [0., 0., 1.], [0., 0., 1.]),
        (native_side_name(6), [0., 0., 0.], [0., -1., 0.]),
        (native_side_name(7), [size, 0., 0.], [1., 0., 0.]),
        (native_side_name(8), [0., size, 0.], [0., 1., 0.]),
        (native_side_name(9), [0., 0., 0.], [-1., 0., 0.]),
    ];
    for (key, origin, normal) in cases {
        let plane = named.face_plane(history, document_guid(), &key).unwrap();
        assert_frame_values(plane.recipe().origin, origin);
        assert_frame_values(plane.normal(), normal);
    }
    assert!(
        named
            .face_plane(history, document_guid(), b"invalid")
            .is_err()
    );
    let body_key = named.body_reference_key(history, document_guid()).unwrap();
    assert!(
        named
            .face_plane(history, document_guid(), body_key)
            .is_err()
    );
}
fn assert_frame_values(actual: [f64; 3], expected: [f64; 3]) {
    for (actual, expected) in actual.into_iter().zip(expected) {
        assert!(
            (actual - expected).abs() < 1e-8,
            "value {actual}, expected {expected}"
        );
    }
}

fn face_offset_history(key: Vec<u8>) -> DocumentEditHistory {
    use oblikovati_model::feature::work_plane::{
        WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneFaceReference, WorkPlaneReference,
    };
    let base = imported_history(GO_CUBE);
    let mut snapshot = base.snapshot().clone();
    let reference = WorkPlaneFaceReference::new(key).unwrap();
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(
            FeatureId(2),
            "Face offset",
            WorkPlaneDefinition::Offset {
                base: WorkPlaneReference::Face(reference),
                distance: WorkPlaneDistance::Constant(0.5),
            },
        ))
        .unwrap();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    DocumentEditHistory::new(snapshot, ids).unwrap()
}
fn assert_face_offset(history: &DocumentEditHistory, expected_x: f64) {
    use oblikovati_martensite::CertifiedNamedPrismEvaluator;
    use oblikovati_model::feature::work_plane::WorkPlaneResolver;
    let mut evaluator =
        CertifiedNamedPrismEvaluator::new(history, document_guid(), app_resolution());
    let replay = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let plane = replay.plane(FeatureId(2)).unwrap();
    assert_frame_values(plane.recipe().origin, [expected_x, 0., 0.]);
    assert_frame_values(plane.normal(), [1., 0., 0.]);
}
#[test]
fn face_offset_replays_from_named_body_through_edit_history_and_recipe() {
    let mut history = face_offset_history(native_side_name(7));
    assert_face_offset(&history, 1.5);
    let token = history.begin("size").unwrap();
    history
        .edit(&token, |snapshot, _| {
            let id = snapshot.parameters.find("size").unwrap();
            snapshot.parameters.set_expression(id, "20 mm")?;
            solve_selected_sketches(snapshot);
            Ok::<(), oblikovati_model::parameter::ParameterError<u64>>(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert_face_offset(&history, 2.5);
    history.undo().unwrap();
    assert_face_offset(&history, 1.5);
    history.redo().unwrap();
    assert_face_offset(&history, 2.5);
    assert_face_offset(&reopened_history(&history), 2.5);
}

#[test]
fn face_offset_rejects_a_foreign_document_name_without_dropping_its_body() {
    use oblikovati_martensite::CertifiedNamedPrismEvaluator;
    use oblikovati_model::feature::work_plane::WorkPlaneResolver;
    let foreign = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let key = SemanticReferenceName::new(foreign, 1, 1, "sketch-prism", "side", &[1, 7])
        .unwrap()
        .encode()
        .unwrap();
    let history = face_offset_history(key);
    let mut evaluator =
        CertifiedNamedPrismEvaluator::new(&history, document_guid(), app_resolution());
    let replay = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    assert!(replay.body(FeatureId(1)).is_some());
    assert!(replay.plane(FeatureId(2)).is_err());
}

#[test]
fn ordered_recompute_moves_face_sketch_after_size_edit() {
    use oblikovati_martensite::CertifiedNamedPrismEvaluator;
    use oblikovati_model::{
        document_sketch::{PlacedSketchHealth, SketchEvaluation, SketchPlacement},
        history::DocumentSketchCreation,
    };
    let history = face_offset_history(native_side_name(7));
    let mut evaluator =
        CertifiedNamedPrismEvaluator::new(&history, document_guid(), app_resolution());
    let datums = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap()
        .datum_evaluation();
    let request = DocumentSketchCreation {
        name: "Face sketch".into(),
        placement: SketchPlacement::WorkPlane(FeatureId(2)),
        orientation: None,
        project_origin: true,
    };
    history
        .preview(|candidate, ids| {
            let sketch = candidate.create_document_sketch_with_datums(&request, ids, &datums)?;
            let size = candidate.parameters.find("size").unwrap();
            candidate.parameters.set_expression(size, "20 mm").unwrap();
            candidate.sketches = candidate
                .solve_pending_document_sketches(Default::default(), || false)
                .candidate;
            let mut evaluator =
                CertifiedNamedPrismEvaluator::new(&history, document_guid(), app_resolution());
            let (result, sketches) = candidate
                .recompute_part_with_sketches(&mut evaluator, Default::default())
                .unwrap();
            let mut updated = candidate.clone();
            updated.sketches = sketches;
            let datums = result.datum_evaluation();
            let plane = updated
                .resolve_sketch_plane_with_datums(sketch, &datums)
                .unwrap();
            assert_frame_values(plane.recipe().origin, [2.5, 0., 0.]);
            let health = updated
                .document_sketch_health_with_datums(sketch, &datums)
                .unwrap();
            assert!(
                matches!(
                    health,
                    PlacedSketchHealth::Current(SketchEvaluation::Numerical(_))
                ),
                "{health:?}"
            );
            Ok::<_, oblikovati_model::history::DocumentSketchCreationError>(())
        })
        .unwrap();
}
