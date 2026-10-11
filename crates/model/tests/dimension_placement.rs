// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::*;
fn fixture() -> Sketch2D {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
    geometry
        .add_circle(SketchEntityId(2), SketchEntityId(1), 1.0)
        .unwrap();
    let radius = PlanarConstraint::Radius {
        curve: SketchEntityId(2),
        parameter: 1,
    };
    for id in [9, 3] {
        geometry
            .add_constraint(SketchConstraintId(id), radius.clone())
            .unwrap();
    }
    append_geometric_relation(&mut geometry);
    geometry
}
#[test]
fn annotation_metadata_is_owned_finite_and_independent_of_geometry() {
    let mut geometry = fixture();
    let before = geometry.recipe();
    let blank = geometry
        .dimension_text_point(SketchConstraintId(9))
        .unwrap();
    assert_eq!(blank, None);
    geometry
        .set_dimension_text_point(SketchConstraintId(9), [2.0, 3.0])
        .unwrap();
    let moved = geometry
        .dimension_text_point(SketchConstraintId(9))
        .unwrap();
    assert_eq!(moved, Some([2.0, 3.0]));
    assert_eq!(geometry.recipe().points, before.points);
    assert_eq!(geometry.recipe().circles, before.circles);
    assert_eq!(geometry.recipe().constraints, before.constraints);
    assert_invalid_placements(&mut geometry);
}
fn assert_invalid_placements(geometry: &mut Sketch2D) {
    let before = geometry.recipe();
    for id in [4, 99] {
        assert_invalid_owner(geometry, SketchConstraintId(id));
    }
    for position in [[f64::NAN, 0.0], [0.0, f64::INFINITY]] {
        assert!(
            geometry
                .set_dimension_text_point(SketchConstraintId(9), position)
                .is_err()
        );
    }
    assert_eq!(geometry.recipe(), before);
}
#[test]
fn annotation_recipe_records_authored_order_and_round_trips() {
    let mut geometry = fixture();
    for id in [3, 9] {
        geometry
            .set_dimension_text_point(SketchConstraintId(id), [2.0, 3.0])
            .unwrap();
    }
    let recipe = geometry.recipe();
    assert_eq!(recipe.version, 5);
    let rows = recipe.dimension_placements.as_ref().unwrap();
    let ids: Vec<_> = rows.iter().map(|row| row.constraint.0).collect();
    assert_eq!(ids, [9, 3]);
    let encoded = serde_json::to_vec(&recipe).unwrap();
    let decoded: SketchRecipe = serde_json::from_slice(&encoded).unwrap();
    let restored = Sketch2D::from_recipe(&decoded).unwrap();
    assert!(geometry.same_recipe(&restored));
}
#[test]
fn history_comparison_preserves_signed_zero_and_repeated_placement_is_a_noop() {
    let mut positive = fixture();
    positive
        .set_dimension_text_point(SketchConstraintId(9), [0.0, 3.0])
        .unwrap();
    let mut negative = positive.clone();
    negative
        .set_dimension_text_point(SketchConstraintId(9), [-0.0, 3.0])
        .unwrap();
    assert!(!positive.same_recipe(&negative));
    let before = negative.clone();
    negative
        .set_dimension_text_point(SketchConstraintId(9), [-0.0, 3.0])
        .unwrap();
    assert!(before.same_recipe(&negative));
}
#[test]
fn deleted_dimension_loses_its_placement_and_survivor_placement_stays_owned() {
    let mut geometry = fixture();
    for id in [9, 3] {
        geometry
            .set_dimension_text_point(SketchConstraintId(id), [2.0, 3.0])
            .unwrap();
    }
    geometry
        .remove_dimensional_constraint(SketchConstraintId(9))
        .unwrap();
    let saved = geometry.recipe().dimension_placements.unwrap();
    assert_eq!(saved.len(), 1);
    assert_eq!(saved[0].constraint, SketchConstraintId(3));
}
#[test]
fn legacy_versions_restore_without_fabricated_placements() {
    for version in 1..=4 {
        let mut recipe = legacy_recipe(version);
        let restored = Sketch2D::from_recipe(&recipe).unwrap();
        assert_eq!(
            restored
                .dimension_text_point(SketchConstraintId(9))
                .unwrap(),
            None
        );
        recipe.dimension_placements = Some(Vec::new());
        assert!(Sketch2D::from_recipe(&recipe).is_err());
    }
}
#[test]
fn corrupted_annotation_recipes_refuse_ids_order_finite_values_and_missing_metadata() {
    let mut geometry = fixture();
    for id in [9, 3] {
        geometry
            .set_dimension_text_point(SketchConstraintId(id), [2.0, 3.0])
            .unwrap();
    }
    let valid = geometry.recipe();
    for mode in 0..5 {
        let mut recipe = valid.clone();
        corrupt(&mut recipe, mode);
        assert!(Sketch2D::from_recipe(&recipe).is_err(), "mode {mode}");
    }
}
fn corrupt(recipe: &mut SketchRecipe, mode: usize) {
    if mode == 4 {
        recipe.dimension_placements = None;
        return;
    }
    let rows = recipe.dimension_placements.as_mut().unwrap();
    match mode {
        0 => rows.reverse(),
        1 => rows.push(rows[0].clone()),
        2 => rows[0].constraint = SketchConstraintId(4),
        3 => rows[0].position = [f64::INFINITY, 0.0],
        _ => unreachable!(),
    }
}

fn append_geometric_relation(geometry: &mut Sketch2D) {
    let relation = PlanarConstraint::Fix {
        point: SketchEntityId(1),
        position: [0.0, 0.0],
    };
    geometry
        .add_constraint(SketchConstraintId(4), relation)
        .unwrap();
}

fn assert_invalid_owner(geometry: &mut Sketch2D, id: SketchConstraintId) {
    assert!(geometry.dimension_text_point(id).is_err());
    assert!(geometry.set_dimension_text_point(id, [2.0, 3.0]).is_err());
}

fn legacy_recipe(version: u32) -> SketchRecipe {
    let mut recipe = fixture().recipe();
    recipe.version = version;
    if version >= 2 {
        recipe.entity_order = Some(vec![SketchEntityId(1), SketchEntityId(2)]);
    }
    if version >= 3 {
        recipe.construction_curves = Some(Vec::new());
    }
    if version >= 4 {
        recipe.centerlines = Some(Vec::new());
    }
    recipe
}
