// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{client::Caller, wire::MoveSketchDimensionArgs};
fn move_call(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    index: i64,
    position: Option<Vec<f64>>,
) -> Result<OKResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .dimension()
        .move_(&MoveSketchDimensionArgs {
            dimension_index: index,
            text_point: position,
            ..Default::default()
        })
}
#[test]
fn canonical_placement_round_trips_in_both_document_kinds_without_a_solve() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_owner_placement(kind);
    }
}
fn assert_owner_placement(kind: DocumentType) {
    let (mut files, document, mut source) = populated(kind);
    let mut expected = owned_geometry(&files, document).clone();
    let selected = selected_id(&files, document, 2);
    expected
        .set_dimension_text_point(DocumentSketchId(1), selected, [2.5, -7.25])
        .unwrap();
    let mut rows = dimensions(&files);
    let focus = files.workspace().active_document();
    assert!(
        move_call(&mut files, &mut source, 2, Some(vec![2.5, -7.25]))
            .unwrap()
            .ok
    );
    rows[2].text_point = Some(vec![2.5, -7.25]);
    assert_eq!(dimensions(&files), rows);
    assert!(expected.same_evaluated_state(owned_geometry(&files, document)));
    assert_eq!(files.workspace().active_document(), focus);
    assert!(SKETCH_COMMAND_METHODS.contains(&METHOD_SKETCH_MOVE_DIMENSION));
}
#[test]
fn canonical_centimetres_convert_to_working_units_at_each_scale() {
    for scale in [1e-6, 1.0, 1e6] {
        assert_scaled_placement(scale);
    }
}
fn assert_scaled_placement(scale: f64) {
    let mut files = super::lifecycle::populated_scaled(scale);
    let document = files.workspace().active_document().unwrap();
    let mut source = QueryGuids(1000);
    move_call(&mut files, &mut source, 0, Some(vec![2.5, -7.25])).unwrap();
    let rows = dimensions(&files);
    let position = rows[0].text_point.as_ref().unwrap();
    for (actual, expected) in position.iter().zip([2.5, -7.25]) {
        assert!(
            (actual - expected).abs()
                <= SolveOptions::default().relative_tolerance * expected.abs()
        );
    }
    assert_working_position(&files, document, scale);
}
#[test]
fn malformed_positions_and_indices_preserve_history_and_metadata() {
    let (mut files, document, mut source) = populated(DOCUMENT_PART);
    let before = owner_revision(&files, document);
    refuse_bad_positions(&mut files, &mut source);
    for index in [-1, 19, i64::MAX] {
        expect_api_decline(
            move_call(&mut files, &mut source, index, Some(vec![1.0, 2.0])),
            "dimension index",
        );
    }
    refuse_bad_placement_payload(&mut files, &mut source);
    assert_eq!(owner_revision(&files, document), before);
    assert!(
        dimensions(&files)
            .iter()
            .all(|row| row.text_point.is_none())
    );
}
#[test]
fn annotation_navigation_reopen_and_copy_keep_placements() {
    let (mut files, document, mut source) = populated(DOCUMENT_PART);
    move_call(&mut files, &mut source, 0, Some(vec![2.5, -7.25])).unwrap();
    let placed = dimensions(&files);
    files.commands(document).unwrap().undo().unwrap();
    assert!(
        dimensions(&files)
            .iter()
            .all(|row| row.text_point.is_none())
    );
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(dimensions(&files), placed);
    let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(dimensions(&files), placed);
    copy_placement(&mut files, reopened, &mut source);
    assert_eq!(dimensions(&files), placed);
}
#[test]
fn workspace_wrapper_moves_labels_and_wrong_owner_is_refused() {
    let (mut files, document, _) = populated(DOCUMENT_PART);
    let id = selected_id(&files, document, 0);
    files
        .commands(document)
        .unwrap()
        .move_sketch_dimension(DocumentSketchId(1), id, [2.5, -7.25])
        .unwrap();
    assert_eq!(dimensions(&files)[0].text_point, Some(vec![2.5, -7.25]));
    let (mut drawing, _, mut source) = configured_files(DOCUMENT_DRAWING, Default::default());
    expect_api_decline(
        move_call(&mut drawing, &mut source, 0, Some(vec![1.0, 2.0])),
        "part or assembly",
    );
}

fn refuse_bad_positions(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    for position in [
        None,
        Some(vec![]),
        Some(vec![1.0]),
        Some(vec![1.0, 2.0, 3.0]),
        Some(vec![f64::INFINITY, 0.0]),
    ] {
        assert!(move_call(files, source, 0, position).is_err());
    }
}

fn copy_placement(files: &mut DiscoveryWorkspace, reopened: u64, source: &mut QueryGuids) {
    crate::file_discovery::sketch_api::entities::activate_copy(
        files,
        reopened,
        source,
        Path::new("/fixtures/annotation-copy.obk"),
    );
}

fn selected_id(files: &DiscoveryWorkspace, document: u64, index: usize) -> SketchConstraintId {
    owned_geometry(files, document)
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .dimensional_constraints()
        .nth(index)
        .unwrap()
        .0
        .id
}

fn assert_working_position(files: &DiscoveryWorkspace, document: u64, scale: f64) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    let id = selected_id(files, document, 0);
    assert_eq!(
        geometry.dimension_text_point(id).unwrap(),
        Some([2.5 / scale, -7.25 / scale])
    );
}

fn refuse_bad_placement_payload(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    let host = FileHost::new(files, source, "test");
    expect_api_decline(
        host.call(METHOD_SKETCH_MOVE_DIMENSION, br#"{"textPoint":"bad"}"#),
        METHOD_SKETCH_MOVE_DIMENSION,
    );
}
