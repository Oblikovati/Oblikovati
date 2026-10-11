// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::*;
fn mixed_geometry() -> Sketch2D {
    let mut geometry = Sketch2D::default();
    for (point, circle, first) in [(1, 2, 11), (3, 4, 13)] {
        let position = [point as f64, -0.0];
        geometry
            .add_defining_point(SketchEntityId(point), position)
            .unwrap();
        geometry
            .add_circle(SketchEntityId(circle), SketchEntityId(point), 1.0)
            .unwrap();
        attach_relations(&mut geometry, point, circle, first, position);
    }
    geometry
        .set_dimension_reference(SketchConstraintId(12), true)
        .unwrap();
    geometry
        .set_dimension_text_point(SketchConstraintId(12), [-2.0, 3.0])
        .unwrap();
    geometry
}
fn attach_relations(
    geometry: &mut Sketch2D,
    point: u64,
    circle: u64,
    first: u64,
    position: [f64; 2],
) {
    geometry
        .add_constraint(
            SketchConstraintId(first),
            PlanarConstraint::Fix {
                point: SketchEntityId(point),
                position,
            },
        )
        .unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(first + 1),
            PlanarConstraint::Radius {
                curve: SketchEntityId(circle),
                parameter: 42,
            },
        )
        .unwrap();
}
fn all_categories() -> SketchDeletionSelection {
    SketchDeletionSelection {
        entities: vec![SketchEntityId(2), SketchEntityId(2)],
        dimensions: vec![
            SketchConstraintId(14),
            SketchConstraintId(12),
            SketchConstraintId(12),
        ],
        geometric_constraints: vec![
            SketchConstraintId(13),
            SketchConstraintId(11),
            SketchConstraintId(13),
        ],
    }
}
#[test]
fn overlapping_mixed_selection_retires_each_identity_once_in_seed_order() {
    let mut geometry = mixed_geometry();
    let report = geometry.delete_selection(&all_categories()).unwrap();
    assert_eq!(report.entities, vec![SketchEntityId(2), SketchEntityId(1)]);
    assert_eq!(constraint_ids(&report.constraints), vec![11, 12, 13, 14]);
    assert_eq!(geometry.entity_ids(), &[SketchEntityId(4)]);
    assert_eq!(geometry.points()[0].id, SketchEntityId(3));
    assert!(geometry.constraints().is_empty());
    assert!(
        geometry
            .dimension_text_point(SketchConstraintId(12))
            .is_err()
    );
    assert!(geometry.same_recipe(&Sketch2D::from_recipe(&geometry.recipe()).unwrap()));
}
#[test]
fn relation_only_selection_keeps_every_geometry_identity_and_coordinate() {
    let mut geometry = mixed_geometry();
    let points = geometry.points().to_vec();
    let selection = SketchDeletionSelection {
        dimensions: vec![SketchConstraintId(12)],
        geometric_constraints: vec![SketchConstraintId(13)],
        ..Default::default()
    };
    let report = geometry.delete_selection(&selection).unwrap();
    assert!(report.entities.is_empty());
    assert_eq!(constraint_ids(&report.constraints), vec![12, 13]);
    assert_eq!(geometry.points(), points);
    assert_eq!(
        geometry.entity_ids(),
        &[SketchEntityId(2), SketchEntityId(4)]
    );
    assert_eq!(constraint_ids(geometry.constraints()), vec![11, 14]);
}
#[test]
fn wrong_relation_category_or_unknown_identity_refuses_the_whole_selection() {
    for selection in invalid_selections() {
        let mut geometry = mixed_geometry();
        let before = geometry.clone();
        assert!(geometry.delete_selection(&selection).is_err());
        assert!(geometry.same_recipe(&before));
    }
}
fn invalid_selections() -> Vec<SketchDeletionSelection> {
    let mut wrong_dimension = all_categories();
    wrong_dimension.dimensions.push(SketchConstraintId(11));
    let mut wrong_geometric = all_categories();
    wrong_geometric
        .geometric_constraints
        .push(SketchConstraintId(12));
    let mut unknown_relation = all_categories();
    unknown_relation.dimensions.push(SketchConstraintId(999));
    let mut unknown_entity = all_categories();
    unknown_entity.entities.push(SketchEntityId(999));
    vec![
        wrong_dimension,
        wrong_geometric,
        unknown_relation,
        unknown_entity,
    ]
}
#[test]
fn empty_mixed_selection_preserves_authored_recipe_bits() {
    let mut geometry = mixed_geometry();
    let before = geometry.clone();
    assert_eq!(
        geometry.delete_selection(&Default::default()).unwrap(),
        Default::default()
    );
    assert!(geometry.same_recipe(&before));
}

fn constraint_ids(constraints: &[SketchConstraint]) -> Vec<u64> {
    constraints.iter().map(|row| row.id.0).collect()
}
