// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    feature::{AuthoredFeature, FeatureId, work_plane::WorkPlaneDefinition},
    history::{DocumentEditHistory, DocumentSnapshot, EditError, HistoryError, RecipeReservations},
    identity::{RecipeIdAllocator, RecipeIdError},
    parameter::ParameterCollection,
    sketch::SketchPlane,
    units::UnitsOfMeasure,
};
fn snapshot() -> DocumentSnapshot {
    DocumentSnapshot {
        work_geometry: Default::default(),
        sketches: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
        parameters: ParameterCollection::new(UnitsOfMeasure::default()),
        properties: Default::default(),
    }
}
fn history() -> DocumentEditHistory {
    DocumentEditHistory::new(snapshot(), RecipeIdAllocator::default()).unwrap()
}
fn insert(snapshot: &mut DocumentSnapshot, id: FeatureId) -> Result<(), String> {
    snapshot
        .work_geometry
        .append(AuthoredFeature::work_plane(
            id,
            "Reference",
            WorkPlaneDefinition::Fixed(SketchPlane::xy()),
        ))
        .map(|_| ())
        .map_err(|error| error.to_string())
}
fn create(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<FeatureId, String> {
    let id = ids.allocate_feature().map_err(|error| error.to_string())?;
    insert(snapshot, id)?;
    Ok(id)
}
fn add(history: &mut DocumentEditHistory) -> FeatureId {
    let token = history.begin("Plane").unwrap();
    let id = history.edit(&token, create).unwrap();
    history.commit(&token, false).unwrap();
    id
}
#[test]
fn undo_redo_and_abort_keep_planes_owned_and_marks_permanent() {
    let mut owner = history();
    add(&mut owner);
    let original = owner.snapshot().work_geometry.clone();
    owner.undo().unwrap();
    assert!(owner.snapshot().work_geometry.features().is_empty());
    assert_eq!(owner.feature_high_water(), 1);
    owner.redo().unwrap();
    assert!(
        owner
            .snapshot()
            .work_geometry
            .same_work_plane_recipe(&original)
    );
    let token = owner.begin("Aborted plane").unwrap();
    assert_eq!(owner.edit(&token, create).unwrap(), FeatureId(2));
    owner.abort(&token).unwrap();
    assert_eq!(owner.feature_high_water(), 2);
    assert_eq!(add(&mut owner), FeatureId(3));
}
#[test]
fn rejected_commands_reserve_ids_without_adopting_recipe_or_generation() {
    let mut owner = history();
    let token = owner.begin("Rejected").unwrap();
    let generation = owner.generation();
    let result = owner.edit(&token, |snapshot, ids| {
        create(snapshot, ids)?;
        Err::<(), _>("Rejected".to_string())
    });
    assert!(matches!(result, Err(EditError::Mutation(_))));
    assert!(owner.snapshot().work_geometry.features().is_empty());
    assert_eq!(owner.generation(), generation);
    assert_eq!(owner.feature_high_water(), 1);
    owner.abort(&token).unwrap();
    assert_eq!(add(&mut owner), FeatureId(2));
}
#[test]
fn old_and_unreserved_ids_are_refused_atomically() {
    let mut owner = history().with_feature_reservations(5).unwrap();
    let token = owner.begin("Unreserved").unwrap();
    for id in [0, 1, 5, 6] {
        let result = owner.edit(&token, |snapshot, _| insert(snapshot, FeatureId(id)));
        let expected = HistoryError::UnreservedFeatureId(id);
        assert!(expected.to_string().contains(&format!("feature ID {id}")));
        assert!(matches!(result, Err(EditError::History(error)) if error == expected));
        assert!(owner.snapshot().work_geometry.features().is_empty());
    }
    owner.abort(&token).unwrap();
    assert_eq!(add(&mut owner), FeatureId(6));
}
#[test]
fn restore_validates_live_ids_and_cannot_rewind_reservations() {
    let mut initial = snapshot();
    insert(&mut initial, FeatureId(7)).unwrap();
    let owner = DocumentEditHistory::new(initial, RecipeIdAllocator::default()).unwrap();
    assert_eq!(owner.feature_high_water(), 7);
    assert!(owner.with_feature_reservations(6).is_err());
    let mut owner = history().with_feature_reservations(9).unwrap();
    assert_eq!(add(&mut owner), FeatureId(10));
    assert!(owner.with_feature_reservations(9).is_err());
    let mut invalid = snapshot();
    insert(&mut invalid, FeatureId(0)).unwrap();
    assert!(matches!(
        DocumentEditHistory::new(invalid, RecipeIdAllocator::default()),
        Err(RecipeIdError::ZeroId)
    ));
}
#[test]
fn exhausted_feature_namespace_does_not_publish_a_candidate() {
    let mut owner = history().with_feature_reservations(u64::MAX).unwrap();
    let token = owner.begin("Overflow").unwrap();
    let result = owner.edit(&token, create);
    assert!(matches!(result, Err(EditError::Mutation(message)) if message.contains("u64::MAX")));
    assert!(owner.snapshot().work_geometry.features().is_empty());
    assert_eq!(owner.feature_high_water(), u64::MAX);
    owner.abort(&token).unwrap();
}
#[test]
fn plane_edits_prevent_discard_and_refresh_observes_recipe_changes() {
    let mut owner = history();
    let id = add(&mut owner);
    let token = owner.begin("Name").unwrap();
    owner
        .edit(&token, |snapshot, _| {
            snapshot.work_geometry.rename(id, "Changed")
        })
        .unwrap();
    assert!(!owner.discard_unchanged(&token).unwrap());
    owner.abort(&token).unwrap();
    check_refresh(&mut owner, id);
}
fn check_refresh(owner: &mut DocumentEditHistory, id: FeatureId) {
    assert!(
        owner
            .refresh(|snapshot| snapshot.work_geometry.rename(id, "Changed"))
            .unwrap()
    );
    let generation = owner.generation();
    assert!(
        !owner
            .refresh(|snapshot| snapshot.work_geometry.rename(id, "Changed"))
            .unwrap()
    );
    assert_eq!(owner.generation(), generation);
    assert!(matches!(
        owner.refresh(|snapshot| insert(snapshot, FeatureId(2))),
        Err(EditError::History(HistoryError::UnreservedFeatureId(2)))
    ));
}
