// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{parameter::ParameterCollection, sketch::*};
#[path = "sketch_entity_delete/ownership.rs"]
mod ownership;
fn id(value: u64) -> SketchEntityId {
    SketchEntityId(value)
}
fn populated() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (point, position) in [
        (1, [0.0, 0.0]),
        (2, [1.0, 0.0]),
        (3, [1.0, 1.0]),
        (4, [2.0, 1.0]),
    ] {
        sketch.add_defining_point(id(point), position).unwrap();
    }
    sketch.add_point(id(90), [9.0, -0.0]).unwrap();
    sketch.add_line(id(10), [id(1), id(2)]).unwrap();
    sketch.add_line(id(11), [id(2), id(3)]).unwrap();
    add_curved_entities(&mut sketch);
    sketch
}
fn add_curved_entities(sketch: &mut Sketch2D) {
    sketch.add_circle(id(12), id(2), 1.0).unwrap();
    sketch.add_arc(id(13), id(2), [id(1), id(3)], true).unwrap();
    sketch
        .add_ellipse(SketchEllipse2D::new(id(14), id(2), [1.0, 0.0], [2.0, 1.0]).unwrap())
        .unwrap();
    sketch
        .add_spline(
            SketchSpline2D::new(id(15), vec![id(1), id(4)], false, SketchSplineKind::Control)
                .unwrap(),
        )
        .unwrap();
    sketch
        .add_offset_spline(SketchOffsetSpline2D::new(id(16), id(15), -0.25).unwrap())
        .unwrap();
}
#[test]
fn deleting_a_shared_point_removes_all_dependent_curve_kinds_and_only_orphan_defining_points() {
    let mut sketch = populated();
    let parent = sketch.spline(id(15)).unwrap().clone();
    let offset = *sketch.offset_spline(id(16)).unwrap();
    let retired = sketch.delete_entities(&[id(2)]).unwrap();
    assert_eq!(
        retired.entities,
        vec![id(10), id(11), id(12), id(13), id(14), id(2), id(3)]
    );
    assert!(retired.constraints.is_empty());
    assert_eq!(sketch.entity_ids(), &[id(90), id(15), id(16)]);
    assert_eq!(point_ids(&sketch), [id(1), id(4), id(90)]);
    assert_eq!(sketch.spline(id(15)).unwrap(), &parent);
    assert_eq!(*sketch.offset_spline(id(16)).unwrap(), offset);
    assert_valid_candidate(&sketch);
}
#[test]
fn parent_deletion_cascades_offsets_and_preserves_a_point_shared_with_other_curves() {
    let mut sketch = populated();
    let before = *sketch.point(id(1)).unwrap();
    let retired = sketch.delete_entities(&[id(15), id(15)]).unwrap();
    assert_eq!(retired.entities, vec![id(15), id(16), id(4)]);
    assert_eq!(*sketch.point(id(1)).unwrap(), before);
    assert_eq!(
        sketch.entity_ids(),
        &[id(90), id(10), id(11), id(12), id(13), id(14)]
    );
    assert_valid_candidate(&sketch);
}
#[test]
fn deleting_all_curves_retains_the_independent_point_marker_and_its_coordinate_bits() {
    let mut sketch = populated();
    let marker = *sketch.point(id(90)).unwrap();
    let selected: Vec<SketchEntityId> = (10..=16).map(id).collect();
    let retired = sketch.delete_entities(&selected).unwrap();
    assert_eq!(retired.entities.len(), 11);
    assert_eq!(sketch.entity_ids(), &[id(90)]);
    assert_eq!(sketch.points(), &[marker]);
    assert_eq!(
        sketch.points()[0].position[1].to_bits(),
        (-0.0_f64).to_bits()
    );
    assert_valid_candidate(&sketch);
}
#[test]
fn removing_an_offset_cleans_dimension_metadata_and_leaves_the_parent_unchanged() {
    let mut sketch = populated();
    install_offset_relation(&mut sketch);
    let parent = sketch.spline(id(15)).unwrap().clone();
    let points = sketch.points().to_vec();
    let retired = sketch.delete_entities(&[id(16)]).unwrap();
    assert_eq!(retired.entities, vec![id(16)]);
    assert_eq!(retired.constraints.len(), 1);
    assert_eq!(retired.constraints[0].id, SketchConstraintId(1));
    assert_eq!(sketch.spline(id(15)).unwrap(), &parent);
    assert_eq!(sketch.points(), points);
    assert!(sketch.dimension_source(SketchConstraintId(1)).is_err());
    assert_empty_dimension_metadata(&sketch.recipe());
    assert_valid_candidate(&sketch);
}
fn install_offset_relation(sketch: &mut Sketch2D) {
    let relation = PlanarConstraint::OffsetSplineDistance {
        offset: id(16),
        parameter: 99,
        side: LineContactSide::Right,
    };
    let constraint = SketchConstraintId(1);
    sketch.add_constraint(constraint, relation).unwrap();
    configure_retired_dimension(sketch, constraint);
    sketch.set_curve_construction(&[id(16)], true).unwrap();
}
fn configure_retired_dimension(sketch: &mut Sketch2D, constraint: SketchConstraintId) {
    sketch.set_dimension_reference(constraint, true).unwrap();
    sketch
        .set_dimension_text_point(constraint, [3.0, 4.0])
        .unwrap();
    sketch
        .set_dimension_limits(
            constraint,
            SketchDimensionLimits {
                lower: 0.0,
                upper: 10.0,
            },
        )
        .unwrap();
}
fn assert_empty_dimension_metadata(recipe: &SketchRecipe) {
    assert!(recipe.reference_dimensions.as_ref().unwrap().is_empty());
    assert!(recipe.dimension_placements.as_ref().unwrap().is_empty());
    assert!(recipe.dimension_limits.as_ref().unwrap().is_empty());
    assert!(
        !recipe
            .construction_curves
            .as_ref()
            .unwrap()
            .contains(&id(16))
    );
}
#[test]
fn direct_curve_relations_are_removed_even_when_all_their_points_survive() {
    let mut sketch = populated();
    install_shared_point_relations(&mut sketch);
    let retired = sketch.delete_entities(&[id(10)]).unwrap();
    assert_eq!(
        retired
            .constraints
            .iter()
            .map(|row| row.id)
            .collect::<Vec<_>>(),
        [SketchConstraintId(1)]
    );
    assert_eq!(sketch.constraints().len(), 1);
    assert_eq!(sketch.constraints()[0].id, SketchConstraintId(2));
    assert!(sketch.point(id(1)).is_ok());
    assert!(sketch.point(id(2)).is_ok());
}
fn install_shared_point_relations(sketch: &mut Sketch2D) {
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::LineHorizontal { line: id(10) },
        )
        .unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(2),
            PlanarConstraint::Horizontal {
                points: [id(1), id(2)],
            },
        )
        .unwrap();
}
#[test]
fn invalid_selection_is_atomic_and_empty_selection_does_not_prune_unattached_points() {
    let mut sketch = populated();
    sketch.add_defining_point(id(100), [5.0, 6.0]).unwrap();
    let before = sketch.clone();
    for missing in [id(0), id(999)] {
        let error = sketch.delete_entities(&[id(15), missing]).unwrap_err();
        assert!(error.to_string().contains(&format!("{missing:?}")));
        assert!(sketch.same_recipe(&before));
    }
    assert_eq!(
        sketch.delete_entities(&[]).unwrap(),
        SketchEntityDeletion::default()
    );
    assert!(sketch.same_recipe(&before));
}
fn assert_valid_candidate(sketch: &Sketch2D) {
    assert!(sketch.same_recipe(&Sketch2D::from_recipe(&sketch.recipe()).unwrap()));
    let parameters = ParameterCollection::new(Default::default());
    let candidate = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert!(sketch.same_recipe(candidate.candidate()));
    assert_eq!(
        candidate.numerical().termination,
        oblikovati_solve::nonlinear::SolveTermination::Converged
    );
}
fn point_ids(sketch: &Sketch2D) -> Vec<SketchEntityId> {
    sketch.points().iter().map(|point| point.id).collect()
}
#[test]
fn dependency_closure_is_independent_of_numeric_id_order_and_working_scale() {
    for scale in [1e-6, 1.0, 1e6] {
        let mut sketch = reversed_identity_parent(scale);
        let retired = sketch.delete_entities(&[id(1000)]).unwrap();
        assert_eq!(retired.entities, [id(900), id(2), id(1000), id(2000)]);
        assert!(sketch.owned_entity_ids().is_empty());
        assert_valid_candidate(&sketch);
    }
}
fn reversed_identity_parent(scale: f64) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    sketch.add_defining_point(id(1000), [0.0, 0.0]).unwrap();
    sketch.add_defining_point(id(2000), [scale, scale]).unwrap();
    let kind = SketchSplineKind::Fit {
        method: oblikovati_api::types::SPLINE_FIT_SMOOTH,
    };
    let parent = SketchSpline2D::new(id(900), vec![id(1000), id(2000)], false, kind).unwrap();
    sketch.add_spline(parent).unwrap();
    sketch
        .add_offset_spline(SketchOffsetSpline2D::new(id(2), id(900), -scale).unwrap())
        .unwrap();
    sketch
}
