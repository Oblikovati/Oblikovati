// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    identity::{RecipeIdAllocator, RecipeIdError},
    sketch::{PlanarConstraint, SketchConstraintId, SketchEntityEditError, SketchLineEnd},
};
use std::error::Error;
fn endpoint_fixture() -> Sketch2D {
    let mut sketch = mixed_geometry();
    sketch
        .set_line_centerline(SketchEntityId(11), true)
        .unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::LineHorizontal {
                line: SketchEntityId(11),
            },
        )
        .unwrap();
    sketch
}
#[test]
fn replacing_either_end_preserves_shared_points_roles_constraints_and_reload() {
    for (end, previous, endpoints) in [
        (SketchLineEnd::First, 9, [100, 3]),
        (SketchLineEnd::Last, 3, [9, 100]),
    ] {
        let mut sketch = endpoint_fixture();
        let before = sketch.clone();
        let report = sketch
            .replace_line_endpoint(SketchEntityId(11), end, [5.0, 0.0], &mut || {
                Ok(SketchEntityId(100))
            })
            .unwrap();
        assert_eq!(
            (report.line.0, report.previous.0, report.created.0),
            (11, previous, 100)
        );
        assert_eq!(
            sketch.line(report.line).unwrap().endpoints,
            endpoints.map(SketchEntityId)
        );
        assert_preserved(&sketch, &before);
    }
}
fn assert_preserved(sketch: &Sketch2D, before: &Sketch2D) {
    assert_eq!(sketch.entity_ids(), before.entity_ids());
    assert_eq!(sketch.constraints(), before.constraints());
    assert!(sketch.is_centerline(SketchEntityId(11)).unwrap());
    for point in before.points() {
        assert_eq!(sketch.point(point.id).unwrap(), point);
    }
    assert_eq!(sketch.recipe().circles, before.recipe().circles);
    assert_eq!(sketch.recipe().arcs, before.recipe().arcs);
    assert_eq!(
        sketch.point(SketchEntityId(100)).unwrap().position,
        [5.0, 0.0]
    );
    assert!(sketch.owned_entity_ids().contains(&SketchEntityId(100)));
    super::copy::assert_reload(sketch);
}
#[test]
fn invalid_endpoint_replacement_does_not_reserve_or_publish() {
    for (line, position) in [
        (11, [0.0, 0.0]),
        (11, [2.0, 0.0]),
        (11, [f64::NAN, 0.0]),
        (11, [0.0, f64::INFINITY]),
        (999, [5.0, 0.0]),
        (9, [5.0, 0.0]),
    ] {
        let mut sketch = endpoint_fixture();
        let before = sketch.clone();
        let mut ids = RecipeIdAllocator::restore(Some(99), &[]).unwrap();
        let error = sketch
            .replace_line_endpoint(
                SketchEntityId(line),
                SketchLineEnd::Last,
                position,
                &mut || ids.allocate().map(SketchEntityId),
            )
            .unwrap_err();
        assert!(matches!(error, SketchEntityEditError::Sketch(_)));
        assert!(error.source().is_some());
        assert_eq!(ids.high_water(), 99);
        assert!(sketch.same_recipe(&before));
    }
}
#[test]
fn invalid_reserved_identity_or_exhaustion_cannot_publish_a_detached_point() {
    for reserved in [
        Ok(SketchEntityId(0)),
        Ok(SketchEntityId(9)),
        Ok(SketchEntityId(11)),
        Err(RecipeIdError::Exhausted),
    ] {
        let mut sketch = endpoint_fixture();
        let before = sketch.clone();
        let error = sketch
            .replace_line_endpoint(
                SketchEntityId(11),
                SketchLineEnd::Last,
                [5.0, 0.0],
                &mut || reserved.clone(),
            )
            .unwrap_err();
        if let Err(expected) = reserved {
            assert_eq!(
                error.source().unwrap().downcast_ref::<RecipeIdError>(),
                Some(&expected)
            );
        }
        assert!(sketch.same_recipe(&before));
    }
}
#[test]
fn endpoint_seed_positions_are_not_rounded_projected_or_normalized() {
    for position in [[f64::MAX, -f64::MAX], [f64::from_bits(1), -0.0], [1.0, 7.0]] {
        let mut sketch = endpoint_fixture();
        let report = sketch
            .replace_line_endpoint(
                SketchEntityId(11),
                SketchLineEnd::First,
                position,
                &mut || Ok(SketchEntityId(100)),
            )
            .unwrap();
        assert_eq!(
            sketch
                .point(report.created)
                .unwrap()
                .position
                .map(f64::to_bits),
            position.map(f64::to_bits)
        );
    }
}
