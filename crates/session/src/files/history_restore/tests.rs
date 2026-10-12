// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    feature::{AuthoredFeature, FeatureId, work_geometry::WorkGeometryDefinition},
    parameter::ParameterCollection,
    sketch::SketchPlane,
    units::UnitsOfMeasure,
};

fn snapshot() -> DocumentSnapshot {
    DocumentSnapshot {
        work_geometry: Default::default(),
        parameters: ParameterCollection::new(UnitsOfMeasure::default()),
        sketches: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
        assets: Default::default(),
        sketch_settings: None,
        properties: Default::default(),
    }
}
fn feature_snapshot() -> DocumentSnapshot {
    let mut snapshot = snapshot();
    let definition = WorkGeometryDefinition::Fixed(SketchPlane::xy());
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(FeatureId(3), "Plane", definition))
        .unwrap();
    snapshot
}
#[test]
fn restoring_a_mark_below_a_live_feature_refuses_instead_of_repairing_it() {
    let error = restore_history(
        feature_snapshot(),
        RecipeIdAllocator::default(),
        2,
        Default::default(),
        0,
    )
    .unwrap_err();
    assert!(matches!(
        error,
        FileWorkspaceError::History(HistoryError::Identity(RecipeIdError::BeyondHighWater {
            id: 3,
            high_water: 2
        }))
    ));
    let cause = std::error::Error::source(&error).unwrap();
    assert!(cause.source().is_some());
}
#[test]
fn restoring_marks_preserves_deleted_identity_reservations() {
    let history = restore_history(
        feature_snapshot(),
        RecipeIdAllocator::default(),
        7,
        Default::default(),
        9,
    )
    .unwrap();
    assert_eq!(history.feature_high_water(), 7);
    assert_eq!(history.derived_high_water(), 9);
    assert_eq!(
        history.sketch_reservations(),
        SketchReservationMarks::default()
    );
}
