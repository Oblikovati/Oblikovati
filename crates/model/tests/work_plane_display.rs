// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureSequence,
        work_geometry::{DatumLength, WorkGeometryDefinition, WorkPlaneDisplaySettings},
    },
    quantity::Dimension,
    sketch::SketchPlane,
    units::UnitsOfMeasure,
};

fn corners() -> [[f64; 3]; 2] {
    [[1.0, -0.0, 0.0], [3.0, 4.0, 2.0]]
}
fn settings() -> WorkPlaneDisplaySettings {
    WorkPlaneDisplaySettings::default()
        .edited(Some(true), Some(true), Some(corners()))
        .unwrap()
}
fn program() -> FeatureSequence<WorkGeometryDefinition> {
    let mut program = FeatureSequence::default();
    program
        .append(AuthoredFeature::datum(
            FeatureId(1),
            "Plane",
            WorkGeometryDefinition::Fixed(SketchPlane::xy()),
        ))
        .unwrap();
    program
}
#[test]
fn explicit_size_has_precedence_and_flags_can_be_edited_later() {
    let settings = settings();
    assert!(!settings.auto_resize() && settings.grounded());
    let edited = settings.edited(Some(true), None, None).unwrap();
    assert!(edited.auto_resize() && edited.grounded());
    assert_eq!(
        edited.current_corners(None, &Default::default()).unwrap(),
        Some(corners())
    );
    assert!(!WorkPlaneDisplaySettings::default().auto_resize());
    assert!(!WorkPlaneDisplaySettings::default().grounded());
}
#[test]
fn derived_corners_use_physical_centimetres_and_ignore_display_preferences() {
    let mut units = UnitsOfMeasure::default().with_working_scale(0.1).unwrap();
    units.set_preferred(Dimension::Length, "in").unwrap();
    let frame = SketchPlane::new(
        [10.0, 20.0, 30.0],
        [1.0, 0.0, 0.0],
        [0.0, 1.0, 0.0],
        Default::default(),
    )
    .unwrap();
    let settings = WorkPlaneDisplaySettings::default();
    assert_eq!(
        settings.current_corners(Some(frame), &units).unwrap(),
        Some([[-4.0, -3.0, 3.0], [6.0, 7.0, 3.0]])
    );
    assert_eq!(settings.current_corners(None, &units).unwrap(), None);
}
#[test]
fn derived_corners_follow_orientation_and_explicit_corners_remain_authored() {
    let frame = SketchPlane::xy().flipped_normal().unwrap();
    let ordinary = WorkPlaneDisplaySettings::default();
    assert_eq!(
        ordinary
            .current_corners(Some(frame), &Default::default())
            .unwrap(),
        Some([[-5.0, 5.0, 0.0], [5.0, -5.0, 0.0]])
    );
    assert_eq!(
        settings()
            .current_corners(Some(frame), &Default::default())
            .unwrap(),
        Some(corners())
    );
}
#[test]
fn rectangle_edits_compare_exact_bits_including_signed_zero() {
    let original = settings();
    let mut changed = corners();
    changed[0][1] = 0.0;
    let changed = original.edited(None, None, Some(changed)).unwrap();
    assert!(!original.same_recipe(&changed));
    assert!(original.same_recipe(&original.clone()));
    assert!(!original.same_recipe(&WorkPlaneDisplaySettings::default()));
    assert!(!WorkPlaneDisplaySettings::default().same_recipe(&original));
}
#[test]
fn nonfinite_corners_are_named_refusals() {
    for value in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        let mut invalid = corners();
        invalid[0][0] = value;
        let error = settings().edited(None, None, Some(invalid)).unwrap_err();
        assert!(
            error
                .to_string()
                .contains("expected two finite three-component")
        );
        assert!(std::error::Error::source(&error).is_none());
    }
}

#[test]
fn derived_corner_overflow_is_a_named_refusal() {
    let units = UnitsOfMeasure::default().with_working_scale(100.0).unwrap();
    let frame = SketchPlane::new(
        [f64::MAX, 0.0, 0.0],
        [1.0, 0.0, 0.0],
        [0.0, 1.0, 0.0],
        Default::default(),
    )
    .unwrap();
    assert!(
        WorkPlaneDisplaySettings::default()
            .current_corners(Some(frame), &units)
            .is_err()
    );
}
#[test]
fn nondefault_metadata_requires_version_nine_and_restores_exactly() {
    let mut program = program();
    program.set_plane_display(FeatureId(1), settings()).unwrap();
    let mut recipe = program.work_plane_recipe().unwrap();
    assert_eq!(recipe.version, 9);
    let restored = FeatureSequence::from_work_plane_recipe(&recipe).unwrap();
    assert!(program.same_work_plane_recipe(&restored));
    recipe.version = 8;
    assert!(FeatureSequence::from_work_plane_recipe(&recipe).is_err());
    program
        .set_plane_display(FeatureId(1), Default::default())
        .unwrap();
    assert_eq!(program.work_plane_recipe().unwrap().version, 1);
}
#[test]
fn points_and_unknown_ids_cannot_receive_plane_display_metadata() {
    let mut program = program();
    let point = WorkGeometryDefinition::Point([DatumLength::Constant(0.0); 3]);
    program
        .append(AuthoredFeature::datum(FeatureId(2), "Point", point))
        .unwrap();
    let before = program.clone();
    assert!(program.set_plane_display(FeatureId(2), settings()).is_err());
    assert!(
        program
            .set_plane_display(FeatureId(99), settings())
            .is_err()
    );
    assert!(program.same_work_plane_recipe(&before));
    let mut recipe = program.work_plane_recipe().unwrap();
    recipe.features[1].plane_display = settings();
    recipe.version = 9;
    assert!(FeatureSequence::from_work_plane_recipe(&recipe).is_err());
}
#[test]
fn display_changes_preserve_definition_reads_and_replacement_retains_settings() {
    let mut program = program();
    let before = *program.get(FeatureId(1)).unwrap().definition();
    let reads = program.get(FeatureId(1)).unwrap().reads().clone();
    program.set_plane_display(FeatureId(1), settings()).unwrap();
    program
        .replace(FeatureId(1), before, reads.clone())
        .unwrap();
    let feature = program.get(FeatureId(1)).unwrap();
    assert!(feature.definition().same_recipe(before));
    assert_eq!(feature.reads(), &reads);
    assert!(feature.plane_display_settings().same_recipe(&settings()));
}

#[path = "work_plane_display/history.rs"]
mod history;
