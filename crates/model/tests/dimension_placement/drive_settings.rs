// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn limits(lower: f64, upper: f64) -> SketchDimensionLimits {
    SketchDimensionLimits { lower, upper }
}
fn configured() -> Sketch2D {
    let mut geometry = fixture();
    for id in [3, 9] {
        geometry
            .set_dimension_reference(SketchConstraintId(id), true)
            .unwrap();
        geometry
            .set_dimension_limits(SketchConstraintId(id), limits(1.0, 2.0))
            .unwrap();
    }
    geometry
}
#[test]
fn bounds_clamp_only_the_new_drive_and_preserve_in_range_bits() {
    let bounds = limits(-0.0, 4.0);
    bounds.validate().unwrap();
    for (input, expected) in [
        (-1.0, -0.0_f64),
        (0.0, 0.0),
        (-0.0, -0.0),
        (3.0, 3.0),
        (5.0, 4.0),
    ] {
        assert_eq!(bounds.clamp(input).to_bits(), expected.to_bits());
    }
    assert_bad_bounds();
}
fn assert_bad_bounds() {
    for bounds in [
        limits(2.0, 1.0),
        limits(f64::NAN, 2.0),
        limits(0.0, f64::INFINITY),
    ] {
        let message = bounds.validate().unwrap_err().to_string();
        assert!(message.contains("finite ordered"));
    }
}
#[test]
fn settings_are_owned_and_invalid_changes_leave_the_recipe_intact() {
    let mut geometry = fixture();
    let dimension = SketchConstraintId(9);
    assert!(!geometry.dimension_is_reference(dimension).unwrap());
    assert_eq!(geometry.dimension_limits(dimension).unwrap(), None);
    let before = geometry.recipe();
    for id in [4, 99] {
        assert_bad_owner(&mut geometry, SketchConstraintId(id));
    }
    assert!(
        geometry
            .set_dimension_limits(dimension, limits(2.0, 1.0))
            .is_err()
    );
    assert_eq!(geometry.recipe(), before);
}
fn assert_bad_owner(geometry: &mut Sketch2D, id: SketchConstraintId) {
    assert!(geometry.dimension_is_reference(id).is_err());
    assert!(geometry.dimension_limits(id).is_err());
    assert!(geometry.set_dimension_reference(id, true).is_err());
    assert!(geometry.set_dimension_limits(id, limits(1.0, 2.0)).is_err());
}
#[test]
fn settings_round_trip_in_authored_order_with_explicit_earlier_metadata() {
    let mut geometry = configured();
    append_prior_metadata(&mut geometry);
    let recipe = geometry.recipe();
    assert_eq!(recipe.version, 6);
    assert_eq!(
        recipe.reference_dimensions.as_ref().unwrap(),
        &[SketchConstraintId(9), SketchConstraintId(3)]
    );
    let rows = recipe.dimension_limits.as_ref().unwrap();
    assert_eq!(
        rows.iter().map(|row| row.constraint.0).collect::<Vec<_>>(),
        [9, 3]
    );
    assert!(recipe.entity_order.is_some() && recipe.construction_curves.is_some());
    assert!(recipe.centerlines.is_some() && recipe.dimension_placements.is_some());
    let encoded = serde_json::to_vec(&recipe).unwrap();
    let decoded: SketchRecipe = serde_json::from_slice(&encoded).unwrap();
    assert!(geometry.same_recipe(&Sketch2D::from_recipe(&decoded).unwrap()));
}
#[test]
fn malformed_settings_refuse_duplicate_foreign_and_out_of_order_ids() {
    let valid = configured().recipe();
    for mode in 0..10 {
        let mut recipe = valid.clone();
        corrupt_settings(&mut recipe, mode);
        assert!(Sketch2D::from_recipe(&recipe).is_err(), "mode {mode}");
    }
}
fn corrupt_settings(recipe: &mut SketchRecipe, mode: usize) {
    if mode < 5 {
        corrupt_reference_fields(recipe, mode);
        return;
    }
    let rows = recipe.dimension_limits.as_mut().unwrap();
    match mode {
        5 => rows.reverse(),
        6 => rows.push(rows[0].clone()),
        7 => rows[0].constraint = SketchConstraintId(99),
        8 => rows[0].limits = limits(2.0, 1.0),
        9 => rows[0].limits = limits(0.0, f64::INFINITY),
        _ => unreachable!(),
    }
}
fn corrupt_reference_fields(recipe: &mut SketchRecipe, mode: usize) {
    match mode {
        0 => recipe.reference_dimensions = None,
        1 => recipe.dimension_limits = None,
        2 => recipe.reference_dimensions.as_mut().unwrap().reverse(),
        3 => recipe
            .reference_dimensions
            .as_mut()
            .unwrap()
            .push(SketchConstraintId(9)),
        4 => recipe.reference_dimensions = Some(vec![SketchConstraintId(4)]),
        _ => unreachable!(),
    }
}
#[test]
fn deletion_cleans_settings_and_bits_are_part_of_history_identity() {
    let mut geometry = configured();
    assert_limit_bit_identity(&mut geometry);
    geometry
        .remove_dimensional_constraint(SketchConstraintId(9))
        .unwrap();
    let recipe = geometry.recipe();
    assert_eq!(
        recipe.reference_dimensions.unwrap(),
        [SketchConstraintId(3)]
    );
    assert_eq!(
        recipe.dimension_limits.unwrap()[0].constraint,
        SketchConstraintId(3)
    );
}
fn assert_limit_bit_identity(geometry: &mut Sketch2D) {
    let mut changed = geometry.clone();
    changed
        .set_dimension_limits(SketchConstraintId(9), limits(-0.0, 2.0))
        .unwrap();
    geometry
        .set_dimension_limits(SketchConstraintId(9), limits(0.0, 2.0))
        .unwrap();
    assert!(!geometry.same_recipe(&changed));
    let encoded = serde_json::to_vec(&changed.recipe()).unwrap();
    let decoded: SketchRecipe = serde_json::from_slice(&encoded).unwrap();
    assert!(changed.same_recipe(&Sketch2D::from_recipe(&decoded).unwrap()));
}
#[test]
fn versions_one_through_five_restore_without_fabricating_settings() {
    for version in 1..=5 {
        let mut recipe = legacy_recipe(version);
        if version == 5 {
            recipe.dimension_placements = Some(Vec::new());
        }
        assert_legacy_settings_absent(recipe);
    }
}

fn assert_legacy_settings_absent(mut recipe: SketchRecipe) {
    let geometry = Sketch2D::from_recipe(&recipe).unwrap();
    assert!(
        !geometry
            .dimension_is_reference(SketchConstraintId(9))
            .unwrap()
    );
    assert_eq!(
        geometry.dimension_limits(SketchConstraintId(9)).unwrap(),
        None
    );
    recipe.reference_dimensions = Some(Vec::new());
    assert!(Sketch2D::from_recipe(&recipe).is_err());
    recipe.reference_dimensions = None;
    recipe.dimension_limits = Some(Vec::new());
    assert!(Sketch2D::from_recipe(&recipe).is_err());
}

fn append_prior_metadata(geometry: &mut Sketch2D) {
    geometry.add_point(SketchEntityId(3), [1.0, 0.0]).unwrap();
    geometry
        .add_line(SketchEntityId(4), [SketchEntityId(1), SketchEntityId(3)])
        .unwrap();
    geometry
        .set_line_centerline(SketchEntityId(4), true)
        .unwrap();
    geometry
        .set_curve_construction(&[SketchEntityId(2)], true)
        .unwrap();
    geometry
        .set_dimension_text_point(SketchConstraintId(9), [-0.0, 2.0])
        .unwrap();
}
