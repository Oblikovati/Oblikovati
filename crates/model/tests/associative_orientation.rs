// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_model::{
    document_sketch::{
        DocumentSketchId, DocumentSketches, SketchAxisReference, SketchAxisResolver,
        SketchOrientationDefinition,
    },
    feature::{AuthoredFeature, FeatureId, FeatureReadKey, work_plane::WorkPlaneDefinition},
    history::DocumentSnapshot,
    sketch::{Sketch2D, SketchEntityId, SketchError, SketchFrameAxis, SketchPlane},
};

struct CurrentAxisDirections {
    feature: FeatureId,
    direction: Option<[f64; 3]>,
}
impl SketchAxisResolver for CurrentAxisDirections {
    fn axis_direction(&self, feature: FeatureId) -> Result<[f64; 3], SketchError> {
        if feature == self.feature
            && let Some(direction) = self.direction
        {
            return Ok(direction);
        }
        Err(SketchError::Definition(format!(
            "axis {feature:?}: expected an available current direction"
        )))
    }
}
fn orientation(reference: SketchAxisReference) -> SketchOrientationDefinition {
    SketchOrientationDefinition {
        reference,
        sketch_axis: SketchFrameAxis::Y,
        reverse: false,
        origin: None,
    }
}
fn snapshot() -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [3.0, 5.0]).unwrap();
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Sketch", SketchPlane::xy(), geometry)
        .unwrap();
    snapshot
}
fn assign(snapshot: &mut DocumentSnapshot, reference: SketchAxisReference) {
    snapshot
        .sketches
        .set_orientation(DocumentSketchId(1), Some(orientation(reference)))
        .unwrap();
}
fn resolved_y(snapshot: &DocumentSnapshot, axes: &CurrentAxisDirections) -> [f64; 3] {
    snapshot
        .resolve_sketch_plane_with_axes(DocumentSketchId(1), axes)
        .unwrap()
        .recipe()
        .y_axis
}

#[test]
fn feature_axes_are_resolved_again_and_never_frozen_in_the_recipe() {
    let mut snapshot = snapshot();
    assign(&mut snapshot, SketchAxisReference::Feature(FeatureId(99)));
    let before = snapshot.sketches.recipe();
    let mut axes = current_axis(Some([0.0, 1.0, 0.0]));
    assert_eq!(resolved_y(&snapshot, &axes), [0.0, 1.0, 0.0]);
    axes.direction = Some([1.0, 0.0, 0.0]);
    assert_eq!(resolved_y(&snapshot, &axes), [1.0, 0.0, 0.0]);
    assert_eq!(before, snapshot.sketches.recipe());
    let placed = snapshot.sketches.get(DocumentSketchId(1)).unwrap();
    assert!(
        placed
            .read_footprint()
            .keys()
            .contains(&FeatureReadKey::Feature(FeatureId(99)))
    );
    let expected = orientation(SketchAxisReference::Feature(FeatureId(99)));
    assert_eq!(placed.orientation(), Some(expected));
}

#[test]
fn missing_axes_decline_without_returning_a_previous_frame() {
    let mut snapshot = snapshot();
    assign(&mut snapshot, SketchAxisReference::Feature(FeatureId(99)));
    let axes = current_axis(None);
    let error = snapshot
        .resolve_sketch_plane_with_axes(DocumentSketchId(1), &axes)
        .unwrap_err();
    assert!(error.to_string().contains("99"));
    let error = snapshot
        .resolve_sketch_plane(DocumentSketchId(1))
        .unwrap_err();
    assert!(error.to_string().contains("current axis"));
}
#[test]
fn parallel_axes_and_unknown_sketches_are_refused() {
    let mut snapshot = snapshot();
    assign(&mut snapshot, SketchAxisReference::Feature(FeatureId(99)));
    let axes = current_axis(Some([0.0, 0.0, 1.0]));
    for id in [1, 999] {
        assert!(
            snapshot
                .resolve_sketch_plane_with_axes(DocumentSketchId(id), &axes)
                .is_err()
        );
    }
}
fn current_axis(direction: Option<[f64; 3]>) -> CurrentAxisDirections {
    CurrentAxisDirections {
        feature: FeatureId(99),
        direction,
    }
}
fn origin_snapshot(reference: SketchAxisReference) -> DocumentSnapshot {
    let mut snapshot = snapshot();
    let base = if reference == SketchAxisReference::OriginZ {
        SketchPlane::xz()
    } else {
        SketchPlane::xy()
    };
    snapshot
        .sketches
        .replace_plane(DocumentSketchId(1), base)
        .unwrap();
    assign(&mut snapshot, reference);
    snapshot
}
#[test]
fn origin_axes_resolve_without_an_external_catalogue() {
    for (reference, expected) in [
        (SketchAxisReference::OriginX, [1.0, 0.0, 0.0]),
        (SketchAxisReference::OriginY, [0.0, 1.0, 0.0]),
        (SketchAxisReference::OriginZ, [0.0, 0.0, 1.0]),
    ] {
        let snapshot = origin_snapshot(reference);
        let result = snapshot.resolve_sketch_plane(DocumentSketchId(1)).unwrap();
        assert_eq!(result.recipe().y_axis, expected);
        let placed = snapshot.sketches.get(DocumentSketchId(1)).unwrap();
        assert!(placed.read_footprint().keys().is_empty());
    }
}

fn add_base_plane(snapshot: &mut DocumentSnapshot, plane: SketchPlane) {
    snapshot
        .work_geometry
        .append(AuthoredFeature::work_plane(
            FeatureId(1),
            "Base",
            WorkPlaneDefinition::Fixed(plane),
        ))
        .unwrap();
    snapshot
        .sketches
        .attach_work_plane(DocumentSketchId(1), FeatureId(1))
        .unwrap();
}
#[test]
fn oriented_work_plane_recipes_follow_current_base_facing_and_fail_on_degeneracy() {
    let mut snapshot = snapshot();
    add_base_plane(&mut snapshot, SketchPlane::xz());
    assign(&mut snapshot, SketchAxisReference::OriginZ);
    let recipe = snapshot.sketches.recipe();
    assert_eq!(recipe.version, 3);
    snapshot.sketches = DocumentSketches::from_recipe(&recipe).unwrap();
    let plane = snapshot.resolve_sketch_plane(DocumentSketchId(1)).unwrap();
    assert_eq!(plane.recipe().y_axis, [0.0, 0.0, 1.0]);
    let changed = WorkPlaneDefinition::Fixed(SketchPlane::xy());
    snapshot
        .work_geometry
        .replace(FeatureId(1), changed.clone(), changed.reads())
        .unwrap();
    assert!(snapshot.resolve_sketch_plane(DocumentSketchId(1)).is_err());
    assert_eq!(snapshot.sketches.recipe(), recipe);
}

#[path = "associative_orientation/recipes.rs"]
mod recipes;

#[test]
fn changing_world_orientation_keeps_local_coordinates_and_solver_verification() {
    let mut snapshot = verified_snapshot();
    let before = snapshot.sketches.get(DocumentSketchId(1)).unwrap().clone();
    assign(&mut snapshot, SketchAxisReference::OriginX);
    let after = snapshot.sketches.get(DocumentSketchId(1)).unwrap();
    assert!(before.geometry().same_recipe(after.geometry()));
    assert_eq!(
        before.health(&snapshot.parameters),
        after.health(&snapshot.parameters)
    );
    let frame = snapshot.resolve_sketch_plane(DocumentSketchId(1)).unwrap();
    assert_eq!(frame.to_model([3.0, 5.0]).unwrap(), [5.0, -3.0, 0.0]);
}

fn assign_definition(
    snapshot: &mut DocumentSnapshot,
    definition: Option<SketchOrientationDefinition>,
) -> Result<(), SketchError> {
    snapshot
        .sketches
        .set_orientation(DocumentSketchId(1), definition)
}

fn verified_snapshot() -> DocumentSnapshot {
    let mut snapshot = snapshot();
    let solved = snapshot
        .sketches
        .solve(&snapshot.parameters, Default::default(), || false);
    assert!(solved.report.reports[0].result.is_ok());
    snapshot.sketches = solved.candidate;
    let placed = snapshot.sketches.get(DocumentSketchId(1)).unwrap();
    assert!(matches!(
        placed.health(&snapshot.parameters),
        oblikovati_model::document_sketch::PlacedSketchHealth::Current(_)
    ));
    snapshot
}
