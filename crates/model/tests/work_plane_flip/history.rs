// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::{DocumentEditHistory, DocumentSnapshot, EditError, HistoryError},
    identity::RecipeIdAllocator,
};

fn snapshot() -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot.work_geometry = sequence();
    snapshot.parameters = parameters();
    snapshot
        .work_geometry
        .flip_plane_normal(FeatureId(2))
        .unwrap();
    snapshot
}
fn replace_with_point(snapshot: &mut DocumentSnapshot) -> Result<(), String> {
    let point = WorkGeometryDefinition::Point([DatumLength::Constant(0.0); 3]);
    snapshot
        .work_geometry
        .replace(FeatureId(2), point, point.reads())
        .map(|_| ())
        .map_err(|error| error.to_string())
}

#[test]
fn history_construction_rejects_plane_flags_on_another_datum_kind() {
    let mut invalid = snapshot();
    replace_with_point(&mut invalid).unwrap();
    let error = DocumentEditHistory::new(invalid, RecipeIdAllocator::default()).unwrap_err();
    assert!(matches!(error, HistoryError::DatumMetadata(_)));
    assert!(error.to_string().contains("expected a plane frame"));
    assert!(std::error::Error::source(&error).is_some());
}
#[test]
fn candidate_edit_cannot_turn_a_flipped_plane_into_a_non_plane() {
    let mut history = history();
    let before = history.snapshot().work_geometry.clone();
    let token = history.begin("Invalid kind replacement").unwrap();
    let error = history
        .edit(&token, |snapshot, _| replace_with_point(snapshot))
        .unwrap_err();
    assert!(matches!(
        error,
        EditError::History(HistoryError::DatumMetadata(_))
    ));
    let current = &history.snapshot().work_geometry;
    assert!(current.same_work_plane_recipe(&before));
    history.abort(&token).unwrap();
}
#[test]
fn refresh_and_save_cannot_bypass_orientation_validation() {
    let mut history = history();
    let before = history.snapshot().work_geometry.clone();
    let error = history.refresh(replace_with_point).unwrap_err();
    assert!(matches!(
        error,
        EditError::History(HistoryError::DatumMetadata(_))
    ));
    let current = &history.snapshot().work_geometry;
    assert!(current.same_work_plane_recipe(&before));
    let mut invalid = snapshot();
    replace_with_point(&mut invalid).unwrap();
    assert!(invalid.work_geometry.work_plane_recipe().is_err());
}

fn history() -> DocumentEditHistory {
    let ids = RecipeIdAllocator::restore(Some(1), &[1]).unwrap();
    DocumentEditHistory::new(snapshot(), ids).unwrap()
}
