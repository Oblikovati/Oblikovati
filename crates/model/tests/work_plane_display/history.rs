// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::{DocumentEditHistory, DocumentSnapshot, EditError, HistoryError},
    identity::RecipeIdAllocator,
};

fn snapshot() -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot.work_geometry = program();
    snapshot
        .work_geometry
        .set_plane_display(FeatureId(1), settings())
        .unwrap();
    snapshot
}
fn replace_with_point(snapshot: &mut DocumentSnapshot) -> Result<(), String> {
    let point = WorkGeometryDefinition::Point([DatumLength::Constant(0.0); 3]);
    snapshot
        .work_geometry
        .replace(FeatureId(1), point, point.reads())
        .map(|_| ())
        .map_err(|error| error.to_string())
}
fn history() -> DocumentEditHistory {
    let ids = RecipeIdAllocator::restore(Some(1), &[1]).unwrap();
    DocumentEditHistory::new(snapshot(), ids).unwrap()
}
#[test]
fn history_and_recipe_capture_reject_display_metadata_on_a_point() {
    let mut invalid = snapshot();
    replace_with_point(&mut invalid).unwrap();
    assert!(invalid.work_geometry.work_plane_recipe().is_err());
    let error = DocumentEditHistory::new(invalid, RecipeIdAllocator::default()).unwrap_err();
    assert!(matches!(error, HistoryError::DatumMetadata(_)));
    assert!(error.to_string().contains("expected a plane frame"));
}
#[test]
fn candidate_rejects_kind_changes_without_publishing_metadata() {
    let mut history = history();
    let before = history.snapshot().work_geometry.clone();
    let token = history.begin("Invalid display kind").unwrap();
    let error = history
        .edit(&token, |snapshot, _| replace_with_point(snapshot))
        .unwrap_err();
    assert!(matches!(
        error,
        EditError::History(HistoryError::DatumMetadata(_))
    ));
    assert_retained(&history, &before);
    history.abort(&token).unwrap();
}
#[test]
fn refresh_rejects_kind_changes_without_publishing_metadata() {
    let mut history = history();
    let before = history.snapshot().work_geometry.clone();
    let error = history.refresh(replace_with_point).unwrap_err();
    assert!(matches!(
        error,
        EditError::History(HistoryError::DatumMetadata(_))
    ));
    assert_retained(&history, &before);
}
fn assert_retained(
    history: &DocumentEditHistory,
    before: &FeatureSequence<WorkGeometryDefinition>,
) {
    assert!(
        history
            .snapshot()
            .work_geometry
            .same_work_plane_recipe(before)
    );
}
