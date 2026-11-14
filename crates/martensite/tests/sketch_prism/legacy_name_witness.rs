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
        .with_work_geometry(&snapshot.work_geometry, 1)
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
