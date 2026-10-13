// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    identity::RecipeIdAllocator,
    sketch::{SketchCopyOperation, SketchCopyReport, SketchSelectionTransform},
};
fn mirror_fixture(start: [f64; 2], end: [f64; 2]) -> Sketch2D {
    let mut sketch = super::defining_transforms::hidden_geometry();
    sketch
        .add_defining_point(SketchEntityId(24), start)
        .unwrap();
    sketch.add_defining_point(SketchEntityId(25), end).unwrap();
    sketch
        .add_line(SketchEntityId(23), [24, 25].map(SketchEntityId))
        .unwrap();
    sketch
}
fn mirror_selected(sketch: &mut Sketch2D, selected: &[u64]) -> SketchCopyReport {
    let mut ids = RecipeIdAllocator::restore(Some(99), &[]).unwrap();
    let selected: Vec<_> = selected.iter().copied().map(SketchEntityId).collect();
    sketch
        .copy_selected_entities(
            &selected,
            SketchCopyOperation::Mirror(SketchEntityId(23)),
            &mut || ids.allocate().map(SketchEntityId),
        )
        .unwrap()
}
fn near_pair(actual: [f64; 2], expected: [f64; 2]) {
    for (actual, expected) in actual.into_iter().zip(expected) {
        assert!(
            (actual - expected).abs() <= 2e-14,
            "{actual:?} != {expected:?}"
        );
    }
}
#[test]
fn reflection_copies_mixed_shared_geometry_and_reverses_only_copied_arc_sweep() {
    let mut sketch = mirror_fixture([-2.0, 1.0], [2.0, 1.0]);
    let before = sketch.clone();
    let report = mirror_selected(&mut sketch, &[11, 7, 15, 20, 21, 22, 15]);
    assert_eq!(
        report.copies,
        [103, 104, 105, 106, 107, 108].map(SketchEntityId)
    );
    assert_eq!(&sketch.points()[..before.points().len()], before.points());
    assert_eq!(
        *sketch.arc(SketchEntityId(15)).unwrap(),
        *before.arc(SketchEntityId(15)).unwrap()
    );
    verify_reflected_curves(&sketch);
    for (id, expected) in [(100, [0.0, 2.0]), (101, [2.0, 2.0]), (102, [0.0, 0.0])] {
        assert_eq!(sketch.point(SketchEntityId(id)).unwrap().position, expected);
    }
}
fn verify_reflected_curves(sketch: &Sketch2D) {
    verify_reflected_incidence(sketch);
    near_pair(
        sketch.arc_midpoint(SketchEntityId(105)).unwrap(),
        [2.0_f64.sqrt(), 2.0 - 2.0_f64.sqrt()],
    );
    assert!(!sketch.arc(SketchEntityId(105)).unwrap().counter_clockwise);
    super::copy::assert_reload(sketch);
}
fn verify_reflected_incidence(sketch: &Sketch2D) {
    assert_eq!(
        sketch.line(SketchEntityId(103)).unwrap().endpoints,
        [100, 101].map(SketchEntityId)
    );
    assert_eq!(
        sketch.circle(SketchEntityId(104)).unwrap().center,
        SketchEntityId(100)
    );
    for id in [107, 108] {
        assert_eq!(
            sketch.spline(SketchEntityId(id)).unwrap().points(),
            [100, 101, 102].map(SketchEntityId)
        );
    }
}
#[test]
fn oblique_reflection_turns_ellipse_axis_once_and_can_copy_the_axis_itself() {
    let mut sketch = mirror_fixture([1.0, 0.0], [2.0, 1.0]);
    let report = mirror_selected(&mut sketch, &[20, 20, 23]);
    let ellipse = sketch.ellipse(report.copies[0]).unwrap();
    near_pair(ellipse.major_axis(), [0.0, 1.0]);
    near_pair(
        sketch.point(ellipse.center()).unwrap().position,
        [1.0, -1.0],
    );
    let line = sketch.line(report.copies[1]).unwrap();
    near_pair(
        sketch.point(line.endpoints[0]).unwrap().position,
        [1.0, 0.0],
    );
    near_pair(
        sketch.point(line.endpoints[1]).unwrap().position,
        [2.0, 1.0],
    );
}
#[test]
fn reflection_direction_handles_subnormal_and_overflowing_endpoint_differences() {
    for (start, end) in [
        ([0.0; 2], [f64::from_bits(1), 0.0]),
        ([-f64::MAX, 0.0], [f64::MAX, 0.0]),
    ] {
        let mut sketch = mirror_fixture(start, end);
        let copy = mirror_selected(&mut sketch, &[1]).copies[0];
        assert_eq!(sketch.point(copy).unwrap().position, [0.0, -2.0]);
    }
}
#[test]
fn invalid_mirror_axis_or_selection_never_reserves_or_publishes_a_prefix() {
    let mut sketch = mirror_fixture([1.0; 2], [1.0; 2]);
    for (axis, selected) in [
        (23, vec![11]),
        (7, vec![11]),
        (999, vec![11]),
        (11, vec![11, 999]),
    ] {
        assert_mirror_refused(&mut sketch, axis, &selected);
    }
}
fn assert_mirror_refused(sketch: &mut Sketch2D, axis: u64, selected: &[u64]) {
    let before = sketch.clone();
    let mut ids = RecipeIdAllocator::restore(Some(99), &[]).unwrap();
    let selected: Vec<_> = selected.iter().copied().map(SketchEntityId).collect();
    let result = sketch.copy_selected_entities(
        &selected,
        SketchCopyOperation::Mirror(SketchEntityId(axis)),
        &mut || ids.allocate().map(SketchEntityId),
    );
    assert!(result.is_err());
    assert!(sketch.same_recipe(&before));
    assert_eq!(ids.high_water(), 99);
}
#[test]
fn in_place_reflection_is_atomic_and_two_reflections_restore_arc_orientation() {
    let mut sketch = mirror_fixture([0.0; 2], [1.0, 0.0]);
    let before = sketch.clone();
    let transform = SketchSelectionTransform::Reflect {
        start: [0.0; 2],
        end: [1.0, 0.0],
    };
    for _ in 0..2 {
        sketch
            .transform_entities(&[SketchEntityId(15); 2], transform)
            .unwrap();
    }
    assert!(sketch.same_recipe(&before));
    assert_eq!(transform.label(), "Mirror sketch entities");
}
#[test]
fn invalid_reflection_inputs_are_atomic() {
    let mut sketch = mirror_fixture([0.0; 2], [1.0, 0.0]);
    let before = sketch.clone();
    let bad = SketchSelectionTransform::Reflect {
        start: [f64::NAN; 2],
        end: [1.0; 2],
    };
    assert!(
        sketch
            .transform_entities(&[SketchEntityId(15)], bad)
            .is_err()
    );
    assert!(sketch.same_recipe(&before));
}
#[test]
fn reflection_preserves_both_minor_and_major_arc_lengths_and_roles() {
    for clockwise in [false, true] {
        let mut sketch = mirror_fixture([-2.0, 1.0], [2.0, 1.0]);
        let mut recipe = sketch.recipe();
        recipe.arcs[0].counter_clockwise = !clockwise;
        sketch = Sketch2D::from_recipe(&recipe).unwrap();
        sketch
            .set_line_centerline(SketchEntityId(11), true)
            .unwrap();
        let source_midpoint = sketch.arc_midpoint(SketchEntityId(15)).unwrap();
        let copies = mirror_selected(&mut sketch, &[15, 11]).copies;
        let reflected = sketch.arc_midpoint(copies[0]).unwrap();
        near_pair(reflected, [source_midpoint[0], 2.0 - source_midpoint[1]]);
        assert!(sketch.is_centerline(copies[1]).unwrap());
    }
}
#[test]
fn mirror_rejects_unrepresentable_affine_offsets_before_reservation() {
    let mut sketch = mirror_fixture([f64::MAX, 0.0], [f64::MAX, 1.0]);
    assert_mirror_refused(&mut sketch, 23, &[11]);
}
