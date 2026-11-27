// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_api::types::{MODEL_PARAM, USER_PARAM};
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureReadFootprint, FeatureReadKey,
        work_geometry::{DatumLength, WorkGeometryDefinition},
        work_plane::{WorkPlaneOrigin, WorkPlaneReference},
    },
    history::{DocumentEditHistory, DocumentSnapshot, EditError, HistoryError},
    identity::RecipeIdAllocator,
    sketch::SketchPlane,
};
#[path = "document_scalar_ownership/deletion.rs"]
mod deletion;

fn offset() -> WorkGeometryDefinition {
    WorkGeometryDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: DatumLength::Parameter(9),
    }
}
fn snapshot() -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    snapshot
        .parameters
        .insert_expression(9, "offset", MODEL_PARAM, "width / 2")
        .unwrap();
    let feature = AuthoredFeature::datum(FeatureId(3), "Offset", offset())
        .with_owned_scalar("offset", 9)
        .unwrap();
    snapshot.work_geometry.append(feature).unwrap();
    snapshot
}
fn history() -> DocumentEditHistory {
    let ids = RecipeIdAllocator::restore(Some(12), &[1, 9]).unwrap();
    DocumentEditHistory::new(snapshot(), ids)
        .unwrap()
        .with_feature_reservations(7)
        .unwrap()
}
fn remove_owner(snapshot: &mut DocumentSnapshot) -> Result<(), String> {
    snapshot
        .work_geometry
        .remove(FeatureId(3))
        .map(|_| ())
        .map_err(|error| error.to_string())
}
fn remove_parameter(snapshot: &mut DocumentSnapshot) -> Result<(), String> {
    snapshot
        .parameters
        .remove_for_owner(9)
        .map_err(|error| error.to_string())
}
fn assert_source_retained(snapshot: &DocumentSnapshot) {
    assert_eq!(snapshot.parameters.get(9).unwrap().source, "width / 2");
    assert_eq!(
        snapshot
            .work_geometry
            .get(FeatureId(3))
            .unwrap()
            .owned_scalars()[0]
            .parameter,
        9
    );
}

#[test]
fn owner_construction_refuses_missing_or_nonmodel_parameters() {
    let mut missing = snapshot();
    remove_parameter(&mut missing).unwrap();
    let error = DocumentEditHistory::new(missing, RecipeIdAllocator::default()).unwrap_err();
    assert!(matches!(error, HistoryError::ScalarOwnership(_)));
    assert!(error.to_string().contains("parameter: 9"));
    assert!(std::error::Error::source(&error).is_some());
    let mut wrong_kind = snapshot();
    remove_parameter(&mut wrong_kind).unwrap();
    wrong_kind
        .parameters
        .insert_expression(9, "offset", USER_PARAM, "2 cm")
        .unwrap();
    assert!(wrong_kind.validate_scalar_ownership().is_err());
}
#[test]
fn unavailable_formulas_retain_valid_ownership_and_local_health() {
    let mut snapshot = snapshot();
    snapshot
        .parameters
        .set_expression(9, "missing / 2")
        .unwrap();
    snapshot.validate_scalar_ownership().unwrap();
    let ids = RecipeIdAllocator::restore(Some(9), &[1, 9]).unwrap();
    let owner = DocumentEditHistory::new(snapshot, ids).unwrap();
    assert_eq!(
        owner.snapshot().parameters.get(9).unwrap().source,
        "missing / 2"
    );
    assert!(owner.snapshot().parameters.model_value(9).is_err());
}
#[test]
fn authored_edits_cannot_delete_only_one_half_of_ownership() {
    for mutation in [remove_owner, remove_parameter] {
        let mut history = history();
        let token = history.begin("Invalid deletion").unwrap();
        let error = history
            .edit(&token, |snapshot, _| mutation(snapshot))
            .unwrap_err();
        assert!(matches!(
            error,
            EditError::History(HistoryError::ScalarOwnership(_))
        ));
        assert_source_retained(history.snapshot());
        history.abort(&token).unwrap();
    }
}
#[test]
fn refresh_cannot_orphan_or_remove_an_owned_parameter() {
    for mutation in [remove_owner, remove_parameter] {
        let mut history = history();
        assert!(history.refresh(mutation).is_err());
        assert_source_retained(history.snapshot());
    }
}
#[test]
fn atomic_delete_undo_redo_preserve_sources_and_reservation_marks() {
    let mut history = history();
    let token = history.begin("Delete datum").unwrap();
    let report = history
        .edit(&token, |snapshot, _| snapshot.delete_datum(FeatureId(3)))
        .unwrap();
    assert_eq!(report.parameters, [9]);
    history.commit(&token, false).unwrap();
    assert!(history.snapshot().parameters.get(9).is_none());
    history.undo().unwrap();
    assert_source_retained(history.snapshot());
    history.redo().unwrap();
    assert!(history.snapshot().work_geometry.features().is_empty());
    assert_eq!(history.high_water(), 12);
    assert_eq!(history.feature_high_water(), 7);
}
#[test]
fn removing_an_owner_cannot_transfer_its_scalar_identity() {
    let mut history = history();
    let token = history.begin("Transfer owner").unwrap();
    let result = history.edit(&token, |snapshot, ids| {
        remove_owner(snapshot)?;
        let id = ids.allocate_feature().map_err(|error| error.to_string())?;
        let feature = AuthoredFeature::datum(id, "Replacement", offset())
            .with_owned_scalar("offset", 9)
            .unwrap();
        snapshot
            .work_geometry
            .append(feature)
            .map_err(|error| error.to_string())
    });
    assert!(result.is_err());
    assert_source_retained(history.snapshot());
    assert_eq!(history.feature_high_water(), 8);
}

#[test]
fn a_text_parameter_converted_to_model_kind_is_not_a_numeric_scalar() {
    let mut snapshot = snapshot();
    remove_parameter(&mut snapshot).unwrap();
    snapshot
        .parameters
        .insert_text(9, "offset", "two centimetres")
        .unwrap();
    snapshot.parameters.convert_kind(9, MODEL_PARAM).unwrap();
    assert!(snapshot.validate_scalar_ownership().is_err());
}
