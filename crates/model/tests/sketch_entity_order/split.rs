// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    identity::{RecipeIdAllocator, RecipeIdError},
    sketch::{SketchEntityEditError, SketchLineSplitReport},
};
use std::error::Error;
fn split_fixture(scale: f64) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    sketch
        .add_defining_point(SketchEntityId(1), [0.0; 2])
        .unwrap();
    sketch
        .add_defining_point(SketchEntityId(2), [4.0 * scale, 0.0])
        .unwrap();
    sketch
        .add_line(SketchEntityId(3), [1, 2].map(SketchEntityId))
        .unwrap();
    sketch
}
fn split_at(sketch: &mut Sketch2D, pick: [f64; 2]) -> SketchLineSplitReport {
    let mut ids = RecipeIdAllocator::restore(Some(99), &[]).unwrap();
    sketch
        .split_line(SketchEntityId(3), pick, &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap()
}
#[test]
fn split_projects_off_line_picks_and_retains_the_original_line_and_endpoint_identity() {
    let mut sketch = split_fixture(1.0);
    let report = split_at(&mut sketch, [2.0, 7.0]);
    assert_eq!(report.parts, [3, 101].map(SketchEntityId));
    assert_eq!(report.junction, SketchEntityId(100));
    assert_eq!(
        sketch.line(report.parts[0]).unwrap().endpoints,
        [1, 100].map(SketchEntityId)
    );
    assert_eq!(
        sketch.line(report.parts[1]).unwrap().endpoints,
        [100, 2].map(SketchEntityId)
    );
    assert_eq!(sketch.point(report.junction).unwrap().position, [2.0, 0.0]);
    verify_split_ownership(&sketch);
    super::copy::assert_reload(&sketch);
}
fn verify_split_ownership(sketch: &Sketch2D) {
    assert_eq!(
        sketch.point(SketchEntityId(2)).unwrap().position,
        [4.0, 0.0]
    );
    assert_eq!(
        sketch.entity_ids(),
        &[SketchEntityId(3), SketchEntityId(101)]
    );
}
#[test]
fn split_preserves_shared_endpoints_roles_and_authored_constraints() {
    let mut sketch = split_fixture_with_roles();
    let before = sketch.constraints().to_vec();
    let report = split_at(&mut sketch, [1.0, 1.0]);
    assert_eq!(sketch.constraints(), before);
    assert_eq!(
        sketch.line(SketchEntityId(4)).unwrap().endpoints,
        [2, 1].map(SketchEntityId)
    );
    assert!(sketch.is_centerline(report.parts[0]).unwrap());
    assert!(sketch.is_centerline(report.parts[1]).unwrap());
    assert!(
        !sketch
            .recipe()
            .construction_curves
            .unwrap_or_default()
            .contains(&report.parts[1])
    );
}
fn split_fixture_with_roles() -> Sketch2D {
    use oblikovati_model::sketch::{PlanarConstraint, SketchConstraintId};
    let mut sketch = split_fixture(1.0);
    sketch
        .add_line(SketchEntityId(4), [2, 1].map(SketchEntityId))
        .unwrap();
    sketch.set_line_centerline(SketchEntityId(3), true).unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::LineHorizontal {
                line: SketchEntityId(3),
            },
        )
        .unwrap();
    sketch
}
#[test]
fn split_projection_is_scale_safe_without_a_fixed_endpoint_epsilon() {
    for scale in [1e-150, 1.0, 1e150] {
        let mut sketch = split_fixture(scale);
        let report = split_at(&mut sketch, [2.0 * scale, scale]);
        assert_eq!(
            sketch.point(report.junction).unwrap().position,
            [2.0 * scale, 0.0]
        );
    }
    let mut sketch = split_fixture(1.0);
    let report = split_at(&mut sketch, [1e-12, 0.0]);
    assert_eq!(
        sketch.point(report.junction).unwrap().position,
        [1e-12, 0.0]
    );
}
#[test]
fn invalid_split_input_reserves_nothing_and_preserves_all_geometry() {
    let mut sketch = split_fixture(1.0);
    for (line, pick) in [
        (3, [0.0, 1.0]),
        (3, [4.0, 1.0]),
        (3, [-1.0, 0.0]),
        (3, [f64::NAN, 0.0]),
        (999, [2.0, 0.0]),
        (1, [2.0, 0.0]),
    ] {
        assert_split_refused(&mut sketch, line, pick);
    }
    sketch.set_position(SketchEntityId(2), [0.0; 2]).unwrap();
    assert_split_refused(&mut sketch, 3, [0.0; 2]);
}
fn assert_split_refused(sketch: &mut Sketch2D, line: u64, pick: [f64; 2]) {
    let before = sketch.clone();
    let mut ids = RecipeIdAllocator::restore(Some(99), &[]).unwrap();
    let error = sketch
        .split_line(SketchEntityId(line), pick, &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap_err();
    assert!(matches!(error, SketchEntityEditError::Sketch(_)));
    assert!(error.source().is_some());
    assert_eq!(ids.high_water(), 99);
    assert!(sketch.same_recipe(&before));
}
#[test]
fn split_identity_exhaustion_burns_reservations_without_publishing_half_a_split() {
    let mut sketch = split_fixture(1.0);
    let before = sketch.clone();
    let mut ids = RecipeIdAllocator::restore(Some(u64::MAX - 1), &[]).unwrap();
    let error = sketch
        .split_line(SketchEntityId(3), [2.0; 2], &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap_err();
    assert_eq!(
        error,
        SketchEntityEditError::Identity(RecipeIdError::Exhausted)
    );
    assert_eq!(
        error.source().unwrap().downcast_ref::<RecipeIdError>(),
        Some(&RecipeIdError::Exhausted)
    );
    assert_eq!(ids.high_water(), u64::MAX);
    assert!(sketch.same_recipe(&before));
}
#[test]
fn split_rejects_unrepresentable_differences_and_rounded_endpoint_results() {
    let mut sketch = split_fixture(1.0);
    sketch
        .set_position(SketchEntityId(1), [-f64::MAX, 0.0])
        .unwrap();
    sketch
        .set_position(SketchEntityId(2), [f64::MAX, 0.0])
        .unwrap();
    assert_split_refused(&mut sketch, 3, [0.0; 2]);
    let next = f64::from_bits(1.0_f64.to_bits() + 1);
    sketch.set_position(SketchEntityId(1), [1.0; 2]).unwrap();
    sketch.set_position(SketchEntityId(2), [next; 2]).unwrap();
    assert_split_refused(&mut sketch, 3, [1.0, next]);
}
#[test]
fn split_diagonal_projection_uses_the_line_support_instead_of_the_pick_coordinates() {
    let mut sketch = split_fixture(1.0);
    sketch.set_position(SketchEntityId(2), [4.0, 4.0]).unwrap();
    let report = split_at(&mut sketch, [0.0, 4.0]);
    assert_eq!(sketch.point(report.junction).unwrap().position, [2.0, 2.0]);
}
