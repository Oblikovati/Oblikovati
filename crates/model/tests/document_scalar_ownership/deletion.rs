// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId, history::DocumentDatumDeleteError, sketch::Sketch2D,
};

fn assert_refused(snapshot: &mut DocumentSnapshot) -> DocumentDatumDeleteError {
    let before = snapshot.clone();
    let error = snapshot.delete_datum(FeatureId(3)).unwrap_err();
    assert!(
        before
            .work_geometry
            .same_work_geometry_recipe(&snapshot.work_geometry)
    );
    assert!(before.parameters.same_recipe(&snapshot.parameters));
    error
}
#[test]
fn authored_parameter_readers_block_delete_even_when_cycle_edges_are_detached() {
    let mut snapshot = cycle_reader_snapshot();
    assert!(snapshot.parameters.readers(9).unwrap().is_empty());
    let error = assert_refused(&mut snapshot);
    assert!(error.to_string().contains("Parameter(10)"));
    assert!(matches!(
        error,
        DocumentDatumDeleteError::Consumed {
            target: FeatureReadKey::Parameter(9),
            ..
        }
    ));
    assert!(snapshot.parameters.authored_readers(99).is_err());
}
#[test]
fn independent_feature_scalar_consumers_block_delete() {
    let mut snapshot = snapshot();
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(FeatureId(4), "Reader", offset()))
        .unwrap();
    let error = assert_refused(&mut snapshot);
    assert!(error.to_string().contains("Feature(FeatureId(4))"));
}
#[test]
fn hosted_sketches_block_delete_and_last_sketch_prunes_owned_construction_scalars() {
    let mut snapshot = hosted_construction_snapshot();
    assert_refused(&mut snapshot);
    let report = snapshot.delete_sketch(DocumentSketchId(2)).unwrap();
    assert_eq!(report.construction_datums, [FeatureId(3)]);
    assert!(snapshot.parameters.get(9).is_none());
    assert!(snapshot.parameters.get(1).is_some());
}
#[test]
fn unknown_reads_refuse_delete_without_changing_the_document() {
    let mut snapshot = snapshot();
    let definition = WorkGeometryDefinition::Fixed(SketchPlane::xy());
    let feature = AuthoredFeature::new(
        FeatureId(4),
        "Unknown",
        definition,
        FeatureReadFootprint::new([], true),
    );
    snapshot.work_geometry.append(feature).unwrap();
    let error = assert_refused(&mut snapshot);
    assert!(error.to_string().contains("complete read attribution"));
}
#[test]
fn invalid_owner_and_unknown_feature_are_named_delete_failures() {
    let mut snapshot = snapshot();
    let error = snapshot.delete_datum(FeatureId(99)).unwrap_err();
    assert!(matches!(error, DocumentDatumDeleteError::Feature(_)));
    assert!(error.to_string().contains("99"));
    assert!(std::error::Error::source(&error).is_some());
    remove_parameter(&mut snapshot).unwrap();
    let error = snapshot.delete_datum(FeatureId(3)).unwrap_err();
    assert!(matches!(error, DocumentDatumDeleteError::Ownership(_)));
    assert!(error.to_string().contains("parameter: 9"));
    assert!(std::error::Error::source(&error).is_some());
}

#[test]
fn direct_candidate_edits_cannot_bypass_the_consumer_guard() {
    let mut history = history_with_reader();
    let token = history.begin("Bypass").unwrap();
    let result = history.edit(&token, |snapshot, _| {
        remove_owner(snapshot)?;
        remove_parameter(snapshot)
    });
    assert!(matches!(
        result,
        Err(EditError::History(HistoryError::DatumDeletion(_)))
    ));
    assert_source_retained(history.snapshot());
}
#[test]
fn deleting_a_hosted_sketch_and_its_owned_construction_datum_is_one_history_edit() {
    let snapshot = hosted_construction_snapshot();
    let ids = RecipeIdAllocator::restore(Some(9), &[1, 9]).unwrap();
    let mut history = DocumentEditHistory::new(snapshot, ids).unwrap();
    let token = history.begin("Delete sketch").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.delete_sketch(DocumentSketchId(2))
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert!(history.snapshot().parameters.get(9).is_none());
    history.undo().unwrap();
    assert_source_retained(history.snapshot());
    assert!(history.snapshot().sketches.get(DocumentSketchId(2)).is_ok());
}

fn cycle_reader_snapshot() -> DocumentSnapshot {
    let mut snapshot = snapshot();
    snapshot
        .parameters
        .insert_expression(10, "reader", USER_PARAM, "offset * 2")
        .unwrap();
    assert_eq!(snapshot.parameters.authored_readers(9).unwrap(), [10]);
    assert!(
        snapshot
            .parameters
            .set_expression(10, "reader + offset")
            .is_err()
    );
    snapshot
}

fn hosted_construction_snapshot() -> DocumentSnapshot {
    let mut snapshot = snapshot();
    snapshot
        .work_geometry
        .set_work_plane_construction(FeatureId(3), true)
        .unwrap();
    snapshot
        .sketches
        .add_on_work_plane(
            DocumentSketchId(2),
            "Sketch",
            FeatureId(3),
            Sketch2D::default(),
        )
        .unwrap();
    snapshot
}

fn history_with_reader() -> DocumentEditHistory {
    let mut history = history();
    let token = history.begin("Reader").unwrap();
    history
        .edit(&token, |snapshot, ids| {
            let id = ids.allocate().unwrap();
            snapshot
                .parameters
                .insert_expression(id, "reader", USER_PARAM, "offset * 2")
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    history
}

#[test]
fn parameter_deletion_declines_preserve_the_typed_parameter_cause() {
    let cause = snapshot().parameters.authored_readers(99).unwrap_err();
    let error = DocumentDatumDeleteError::Parameter(cause);
    assert!(error.to_string().contains("99"));
    assert!(std::error::Error::source(&error).is_some());
}
#[test]
fn consumer_declines_remain_the_terminal_cause_of_history_errors() {
    let cause = DocumentDatumDeleteError::Consumed {
        target: FeatureReadKey::Parameter(9),
        readers: vec![FeatureReadKey::Feature(FeatureId(4))],
    };
    let error = HistoryError::DatumDeletion(cause);
    assert!(error.to_string().contains("FeatureId(4)"));
    let source = std::error::Error::source(&error).unwrap();
    assert!(source.source().is_none());
}
