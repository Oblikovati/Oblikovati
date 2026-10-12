// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::USER_PARAM;
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
#[path = "work_plane_flip/history.rs"]
mod history;
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureSequence,
        work_geometry::{DatumLength, WorkGeometryDefinition},
        work_plane::{WorkPlaneOrigin, WorkPlaneReference, WorkPlaneResolver},
    },
    parameter::ParameterCollection,
    sketch::SketchPlane,
    units::UnitsOfMeasure,
};

fn fixed_frame() -> SketchPlane {
    SketchPlane::new(
        [1.0, 2.0, 3.0],
        [1.0, 0.0, -0.0],
        [0.0, 1.0, 0.0],
        Default::default(),
    )
    .unwrap()
}
fn sequence() -> FeatureSequence<WorkGeometryDefinition> {
    let mut sequence = FeatureSequence::default();
    sequence
        .append(AuthoredFeature::datum(
            FeatureId(1),
            "Base",
            WorkGeometryDefinition::Fixed(fixed_frame()),
        ))
        .unwrap();
    let offset = WorkGeometryDefinition::Offset {
        base: WorkPlaneReference::Feature(FeatureId(1)),
        distance: DatumLength::Parameter(1),
    };
    sequence
        .append(AuthoredFeature::datum(FeatureId(2), "Offset", offset))
        .unwrap();
    sequence
}
fn parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "depth", USER_PARAM, "2 cm")
        .unwrap();
    parameters
}
fn frame(
    sequence: &FeatureSequence<WorkGeometryDefinition>,
    parameters: &ParameterCollection<u64>,
    id: u64,
) -> SketchPlane {
    sequence
        .evaluate_work_geometry(parameters)
        .unwrap()
        .plane(FeatureId(id))
        .unwrap()
}

#[test]
fn frame_flip_preserves_origin_and_x_axis_and_is_a_bit_exact_involution() {
    let original = fixed_frame();
    let flipped = original.flipped_normal().unwrap();
    assert_eq!(flipped.recipe().origin, original.recipe().origin);
    assert_eq!(flipped.recipe().x_axis, original.recipe().x_axis);
    assert_eq!(flipped.normal(), [0.0, 0.0, -1.0]);
    assert!(flipped.flipped_normal().unwrap().same_recipe(original));
}
#[test]
fn upstream_flip_changes_dependent_offset_direction_without_changing_distance() {
    let mut sequence = sequence();
    let parameters = parameters();
    let reads = sequence.get(FeatureId(2)).unwrap().reads().clone();
    assert_eq!(
        frame(&sequence, &parameters, 2).recipe().origin,
        [1.0, 2.0, 5.0]
    );
    let range = sequence.flip_plane_normal(FeatureId(1)).unwrap();
    assert_eq!((range.start, range.end), (0, 2));
    assert_eq!(
        frame(&sequence, &parameters, 2).recipe().origin,
        [1.0, 2.0, 1.0]
    );
    assert_eq!(sequence.get(FeatureId(2)).unwrap().reads(), &reads);
    assert_eq!(parameters.get(1).unwrap().source, "2 cm");
}
#[test]
fn offset_flip_leaves_current_origin_and_signed_distance_unchanged() {
    let mut sequence = sequence();
    let mut parameters = parameters();
    sequence.flip_plane_normal(FeatureId(2)).unwrap();
    assert_eq!(
        frame(&sequence, &parameters, 2).recipe().origin,
        [1.0, 2.0, 5.0]
    );
    assert_eq!(frame(&sequence, &parameters, 2).normal(), [0.0, 0.0, -1.0]);
    parameters.set_expression(1, "4 cm").unwrap();
    assert_eq!(
        frame(&sequence, &parameters, 2).recipe().origin,
        [1.0, 2.0, 7.0]
    );
    assert!(sequence.get(FeatureId(2)).unwrap().is_normal_flipped());
}
#[test]
fn flip_metadata_requires_version_eight_and_double_flip_restores_old_layout() {
    let mut sequence = sequence();
    let original = sequence.clone();
    assert_eq!(sequence.work_plane_recipe().unwrap().version, 1);
    sequence.flip_plane_normal(FeatureId(2)).unwrap();
    let mut recipe = sequence.work_plane_recipe().unwrap();
    assert_eq!(recipe.version, 8);
    let restored = FeatureSequence::from_work_plane_recipe(&recipe).unwrap();
    assert!(sequence.same_work_plane_recipe(&restored));
    recipe.version = 7;
    assert!(FeatureSequence::from_work_plane_recipe(&recipe).is_err());
    sequence.flip_plane_normal(FeatureId(2)).unwrap();
    assert!(sequence.same_work_plane_recipe(&original));
    assert_eq!(sequence.work_plane_recipe().unwrap().version, 1);
}
#[test]
fn non_plane_flip_and_forged_non_plane_metadata_are_refused() {
    let mut sequence = sequence();
    let point = WorkGeometryDefinition::Point([DatumLength::Constant(0.0); 3]);
    sequence
        .append(AuthoredFeature::datum(FeatureId(3), "Point", point))
        .unwrap();
    let before = sequence.clone();
    assert!(sequence.flip_plane_normal(FeatureId(3)).is_err());
    assert!(sequence.flip_plane_normal(FeatureId(99)).is_err());
    assert!(sequence.same_work_plane_recipe(&before));
    let mut recipe = sequence.work_plane_recipe().unwrap();
    recipe.features[2].normal_flipped = true;
    recipe.version = 8;
    assert!(FeatureSequence::from_work_plane_recipe(&recipe).is_err());
}
#[test]
fn definition_replacement_retains_authored_orientation() {
    let mut sequence = sequence();
    sequence.flip_plane_normal(FeatureId(2)).unwrap();
    let definition = WorkGeometryDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: DatumLength::Parameter(1),
    };
    sequence
        .replace(FeatureId(2), definition.clone(), definition.reads())
        .unwrap();
    assert!(sequence.get(FeatureId(2)).unwrap().is_normal_flipped());
    assert_eq!(
        frame(&sequence, &parameters(), 2).normal(),
        [0.0, 0.0, -1.0]
    );
}
