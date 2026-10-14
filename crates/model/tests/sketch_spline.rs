// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::{
    MOVEABLE_FIXED, MOVEABLE_FREE, SKETCH_ENTITY_CONTROL_POINT_SPLINE, SKETCH_ENTITY_OFFSET_SPLINE,
    SKETCH_ENTITY_SPLINE, SPLINE_FIT_SMOOTH,
};
use oblikovati_model::{parameter::ParameterCollection, sketch::*};
fn id(value: u64) -> SketchEntityId {
    SketchEntityId(value)
}
fn parent(kind: SketchSplineKind) -> SketchSpline2D {
    SketchSpline2D::new(id(4), vec![id(1), id(2), id(3)], false, kind).unwrap()
}
#[test]
fn handle_record_requires_distinct_parent_and_end_identities() {
    assert!(SketchSplineHandle2D::new(id(4), 1, id(5)).is_ok());
    for (parent, end) in [(0, 5), (4, 0), (4, 4)] {
        assert!(SketchSplineHandle2D::new(id(parent), 0, id(end)).is_err());
    }
}
#[test]
fn handle_adoption_requires_one_fit_parent_point_and_owned_end() {
    let mut sketch = owned(
        SketchSplineKind::Fit {
            method: SPLINE_FIT_SMOOTH,
        },
        1.0,
    );
    sketch.add_point(id(6), [3.0, 0.0]).unwrap();
    let handle = SketchSplineHandle2D::new(id(4), 1, id(6)).unwrap();
    sketch.add_spline_handle(handle).unwrap();
    assert_eq!(sketch.spline_handles(), &[handle]);
    assert!(sketch.add_spline_handle(handle).is_err());
}
fn owned(kind: SketchSplineKind, distance: f64) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (point, position) in [(1, [0.0, 0.0]), (2, [1.0, 1.0]), (3, [2.0, 0.0])] {
        sketch.add_point(id(point), position).unwrap();
    }
    sketch.add_spline(parent(kind)).unwrap();
    sketch
        .add_offset_spline(SketchOffsetSpline2D::new(id(5), id(4), distance).unwrap())
        .unwrap();
    sketch
}
#[test]
fn offset_is_one_signed_distance_freedom_and_parents_keep_shared_point_variables() {
    for kind in spline_kinds() {
        assert_shared_freedoms(kind);
    }
}
fn assert_shared_freedoms(kind: SketchSplineKind) {
    let mut sketch = owned(kind, -2.0);
    let parameters = ParameterCollection::new(Default::default());
    let structure = sketch.analyze(&parameters, 1e-9).unwrap();
    assert_eq!((structure.variables, structure.degrees_of_freedom), (7, 7));
    sketch
        .add_spline(SketchSpline2D::new(id(6), vec![id(3), id(2), id(1)], true, kind).unwrap())
        .unwrap();
    assert_eq!(sketch.analyze(&parameters, 1e-9).unwrap().variables, 7);
    let outcome = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert!(sketch.same_recipe(outcome.candidate()));
    assert_eq!(
        outcome.candidate().offset_spline(id(5)).unwrap().distance(),
        -2.0
    );
    assert!(sketch.spline(id(6)).unwrap().closed());
}

#[test]
fn catalogue_preserves_control_fit_offset_kinds_and_authored_identity_order() {
    for (kind, expected) in spline_kinds()
        .into_iter()
        .zip([SKETCH_ENTITY_CONTROL_POINT_SPLINE, SKETCH_ENTITY_SPLINE])
    {
        assert_catalogue(kind, expected);
    }
}
fn assert_catalogue(kind: SketchSplineKind, expected: oblikovati_api::types::SketchEntityKind) {
    let sketch = owned(kind, 2.0);
    assert_eq!(sketch.entity_ids(), &[id(1), id(2), id(3), id(4), id(5)]);
    assert_eq!(sketch.entity_kind(id(4)).unwrap(), expected);
    assert_eq!(
        sketch.entity_kind(id(5)).unwrap(),
        SKETCH_ENTITY_OFFSET_SPLINE
    );
    let rows = sketch
        .inspect_entities(
            &ParameterCollection::new(Default::default()),
            &[],
            Default::default(),
        )
        .unwrap();
    assert_eq!(rows[3].points, vec![[0.0, 0.0], [1.0, 1.0], [2.0, 0.0]]);
    assert!(rows[4].points.is_empty());
    assert_eq!(rows[4].movement, MOVEABLE_FREE);
    assert_eq!(rows[4].radius, 0.0);
}

#[test]
fn parent_grounding_holds_points_and_leaves_only_offset_distance_free() {
    let mut sketch = owned(SketchSplineKind::Control, -2.0);
    let anchors = sketch.points().to_vec();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Ground {
                points: anchors.clone(),
            },
        )
        .unwrap();
    let parameters = ParameterCollection::new(Default::default());
    let outcome = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 1);
    assert_grounded_rows(outcome.candidate(), &parameters);
    assert_eq!(outcome.candidate().points(), anchors);
    assert_eq!(sketch.offset_spline(id(5)).unwrap().parent(), id(4));
}
fn assert_grounded_rows(sketch: &Sketch2D, parameters: &ParameterCollection<u64>) {
    let rows = sketch
        .inspect_entities(parameters, &[], Default::default())
        .unwrap();
    assert_eq!(rows[3].movement, MOVEABLE_FIXED);
    assert_eq!(rows[4].movement, MOVEABLE_FREE);
}
#[test]
fn shared_parent_translation_moves_each_point_once_without_changing_offset_or_role() {
    let mut sketch = owned(SketchSplineKind::Control, -2.0);
    add_shared_parent(&mut sketch, SketchSplineKind::Control, false);
    assert_eq!(
        sketch
            .translate_entities(&[id(4), id(6), id(5)], [1.0, 2.0])
            .unwrap(),
        vec![id(1), id(2), id(3)]
    );
    assert_eq!(sketch.point(id(2)).unwrap().position, [2.0, 3.0]);
    assert_eq!(sketch.offset_spline(id(5)).unwrap().distance(), -2.0);
    assert_eq!(sketch.profile_curve_ids(), vec![id(4), id(5), id(6)]);
    sketch.set_curve_construction(&[id(5)], true).unwrap();
    assert_eq!(sketch.profile_curve_ids(), vec![id(4), id(6)]);
    assert!(sketch.is_construction(id(5)).unwrap());
}
#[test]
fn wrong_parent_kind_missing_points_and_cross_kind_collisions_fail_atomically() {
    let mut sketch = owned(SketchSplineKind::Control, 1.0);
    let before = sketch.clone();
    assert_missing_owners(&mut sketch);
    assert!(
        sketch
            .add_spline(parent(SketchSplineKind::Control))
            .is_err()
    );
    assert!(sketch.add_point(id(5), [0.0; 2]).is_err());
    assert!(sketch.add_line(id(4), [id(1), id(2)]).is_err());
    assert!(
        sketch
            .offset_spline(id(99))
            .unwrap_err()
            .to_string()
            .contains("99")
    );
    assert!(sketch.same_recipe(&before));
}
#[path = "sketch_spline/storage.rs"]
mod storage;

fn spline_kinds() -> [SketchSplineKind; 2] {
    [
        SketchSplineKind::Control,
        SketchSplineKind::Fit {
            method: SPLINE_FIT_SMOOTH,
        },
    ]
}
fn add_shared_parent(sketch: &mut Sketch2D, kind: SketchSplineKind, closed: bool) {
    let shared = SketchSpline2D::new(id(6), vec![id(3), id(2), id(1)], closed, kind).unwrap();
    sketch.add_spline(shared).unwrap();
}
fn assert_missing_owners(sketch: &mut Sketch2D) {
    let missing =
        SketchSpline2D::new(id(6), vec![id(1), id(99)], false, SketchSplineKind::Control).unwrap();
    assert!(sketch.add_spline(missing).is_err());
    for parent in [id(1), id(99)] {
        let offset = SketchOffsetSpline2D::new(id(6), parent, 1.0).unwrap();
        assert!(sketch.add_offset_spline(offset).is_err());
    }
}

#[path = "sketch_spline/dimension.rs"]
mod dimension;
