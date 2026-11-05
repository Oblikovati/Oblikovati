// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    identity::{RecipeIdAllocator, RecipeIdError},
    sketch::{
        SketchCopyError, SketchCopyReport, SketchEllipse2D, SketchOffsetSpline2D, SketchSpline2D,
        SketchSplineKind,
    },
};
use std::error::Error;
fn ids_after(last: u64) -> RecipeIdAllocator {
    RecipeIdAllocator::restore(Some(last), &[]).unwrap()
}
fn copy_mixed(sketch: &mut Sketch2D) -> SketchCopyReport {
    let mut ids = ids_after(99);
    let selected = [11, 7, 15, 20, 21, 22, 9, 11].map(SketchEntityId);
    sketch
        .copy_entities(&selected, [3.0, 4.0], &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap()
}
#[test]
fn complete_copy_keeps_shared_points_and_returns_selection_and_full_provenance() {
    let mut sketch = super::defining_transforms::hidden_geometry();
    let before = sketch.clone();
    let report = copy_mixed(&mut sketch);
    assert_eq!(
        report.copies,
        [103, 104, 105, 106, 107, 108, 100].map(SketchEntityId)
    );
    let sources = [9, 3, 1, 11, 7, 15, 20, 21, 22].map(SketchEntityId);
    let copies = (100..109).map(SketchEntityId);
    assert_eq!(
        report.relations,
        sources.into_iter().zip(copies).collect::<Vec<_>>()
    );
    assert_eq!(&sketch.points()[..before.points().len()], before.points());
    assert_copied_shapes(&sketch, &before);
    assert_copy_visibility(&sketch);
}
fn assert_copy_visibility(sketch: &Sketch2D) {
    let recipe = sketch.recipe();
    let hidden = recipe.defining_points.as_ref().unwrap();
    assert!(hidden.contains(&SketchEntityId(101)));
    assert!(!hidden.contains(&SketchEntityId(100)));
    assert!(sketch.entity_ids().contains(&SketchEntityId(100)));
}

fn assert_copied_shapes(sketch: &Sketch2D, before: &Sketch2D) {
    assert_eq!(
        sketch.line(SketchEntityId(103)).unwrap().endpoints,
        [100, 101].map(SketchEntityId)
    );
    assert_eq!(
        sketch.circle(SketchEntityId(104)).unwrap().center,
        SketchEntityId(100)
    );
    assert_eq!(
        sketch.arc(SketchEntityId(105)).unwrap().end,
        SketchEntityId(102)
    );
    assert_copied_parametric_shapes(sketch, before);
}
fn assert_copied_parametric_shapes(sketch: &Sketch2D, before: &Sketch2D) {
    assert_eq!(
        sketch.ellipse(SketchEntityId(106)).unwrap().major_axis(),
        before.ellipse(SketchEntityId(20)).unwrap().major_axis()
    );
    for (source, target) in [(21, 107), (22, 108)] {
        let copied = sketch.spline(SketchEntityId(target)).unwrap();
        assert_eq!(copied.points(), [100, 101, 102].map(SketchEntityId));
        assert_eq!(
            copied.kind(),
            before.spline(SketchEntityId(source)).unwrap().kind()
        );
    }
    assert_eq!(
        sketch.point(SketchEntityId(100)).unwrap().position,
        [3.0, 4.0]
    );
}
#[test]
fn curve_roles_are_copied_but_source_constraints_are_not_duplicated() {
    let mut sketch = copy_source_with_roles();
    copy_mixed(&mut sketch);
    assert!(sketch.is_construction(SketchEntityId(104)).unwrap());
    assert!(sketch.is_centerline(SketchEntityId(103)).unwrap());
    assert!(
        !sketch
            .recipe()
            .construction_curves
            .unwrap()
            .contains(&SketchEntityId(103))
    );
    assert_eq!(sketch.constraints().len(), 1);
    assert_reload(&sketch);
}
fn copy_source_with_roles() -> Sketch2D {
    use oblikovati_model::sketch::{PlanarConstraint, SketchConstraintId};
    let mut sketch = super::defining_transforms::hidden_geometry();
    sketch
        .set_curve_construction(&[SketchEntityId(7)], true)
        .unwrap();
    sketch
        .set_line_centerline(SketchEntityId(11), true)
        .unwrap();
    let fixed = PlanarConstraint::Fix {
        point: SketchEntityId(9),
        position: [0.0; 2],
    };
    sketch.add_constraint(SketchConstraintId(1), fixed).unwrap();
    sketch
}
pub(super) fn assert_reload(sketch: &Sketch2D) {
    let encoded = serde_json::to_vec(&sketch.recipe()).unwrap();
    let recipe: SketchRecipe = serde_json::from_slice(&encoded).unwrap();
    assert!(sketch.same_recipe(&Sketch2D::from_recipe(&recipe).unwrap()));
}
#[test]
fn source_and_copied_points_are_independent_after_publication() {
    let mut sketch = super::defining_transforms::hidden_geometry();
    copy_mixed(&mut sketch);
    sketch.set_position(SketchEntityId(9), [-1.0; 2]).unwrap();
    assert_eq!(
        sketch.point(SketchEntityId(100)).unwrap().position,
        [3.0, 4.0]
    );
    sketch.set_position(SketchEntityId(100), [7.0; 2]).unwrap();
    assert_eq!(sketch.point(SketchEntityId(9)).unwrap().position, [-1.0; 2]);
}
#[test]
fn invalid_copy_input_reserves_nothing_and_never_copies_a_valid_prefix() {
    let mut sketch = super::defining_transforms::hidden_geometry();
    let before = sketch.clone();
    let mut ids = ids_after(99);
    for (selection, vector) in [(vec![11, 999], [1.0; 2]), (vec![11], [f64::NAN, 0.0])] {
        let selection: Vec<_> = selection.into_iter().map(SketchEntityId).collect();
        let error = sketch
            .copy_entities(&selection, vector, &mut || {
                ids.allocate().map(SketchEntityId)
            })
            .unwrap_err();
        assert!(matches!(error, SketchCopyError::Sketch(_)));
        assert!(error.source().is_some());
        assert!(sketch.same_recipe(&before));
        assert_eq!(ids.high_water(), 99);
    }
}
#[test]
fn identity_exhaustion_preserves_geometry_and_retains_the_typed_cause() {
    let mut sketch = super::defining_transforms::hidden_geometry();
    let before = sketch.clone();
    let mut ids = ids_after(u64::MAX - 2);
    let error = sketch
        .copy_entities(&[SketchEntityId(11)], [1.0; 2], &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap_err();
    assert_eq!(error, SketchCopyError::Identity(RecipeIdError::Exhausted));
    assert_eq!(
        error.source().unwrap().downcast_ref::<RecipeIdError>(),
        Some(&RecipeIdError::Exhausted)
    );
    assert!(error.to_string().contains("identity"));
    assert_eq!(ids.high_water(), u64::MAX);
    assert!(sketch.same_recipe(&before));
}
#[test]
fn direct_offset_selection_is_refused_before_any_copy_or_reservation() {
    let mut sketch = super::defining_transforms::hidden_geometry();
    let offset = SketchOffsetSpline2D::new(SketchEntityId(23), SketchEntityId(21), 1.0).unwrap();
    sketch.add_offset_spline(offset).unwrap();
    let before = sketch.clone();
    let mut ids = ids_after(99);
    let error = sketch
        .copy_entities(
            &[SketchEntityId(11), SketchEntityId(23)],
            [1.0; 2],
            &mut || ids.allocate().map(SketchEntityId),
        )
        .unwrap_err();
    assert!(error.to_string().contains("23"));
    assert!(error.to_string().contains("offset"));
    assert_eq!(ids.high_water(), 99);
    assert!(sketch.same_recipe(&before));
}
struct CopyIdSequence(std::vec::IntoIter<SketchEntityId>);
impl CopyIdSequence {
    fn new(ids: &[u64]) -> Self {
        Self(
            ids.iter()
                .copied()
                .map(SketchEntityId)
                .collect::<Vec<_>>()
                .into_iter(),
        )
    }
    fn reserve(&mut self) -> Result<SketchEntityId, RecipeIdError> {
        self.0.next().ok_or(RecipeIdError::Exhausted)
    }
}
#[test]
fn a_broken_identity_source_cannot_publish_zero_reused_or_duplicate_ids() {
    for values in [vec![0], vec![9], vec![100, 100]] {
        let mut sketch = super::defining_transforms::hidden_geometry();
        let before = sketch.clone();
        let mut ids = CopyIdSequence::new(&values);
        let error = sketch
            .copy_entities(&[SketchEntityId(11)], [1.0; 2], &mut || ids.reserve())
            .unwrap_err();
        assert!(matches!(error, SketchCopyError::Sketch(_)));
        assert!(error.to_string().contains("identity") || error.to_string().contains("entity"));
        assert!(sketch.same_recipe(&before));
    }
}
#[test]
fn empty_copy_is_a_checked_noop_but_zero_offset_still_creates_independent_geometry() {
    let mut sketch = mixed_geometry();
    let before = sketch.clone();
    let mut ids = ids_after(99);
    let empty = sketch
        .copy_entities(&[], [0.0; 2], &mut || ids.allocate().map(SketchEntityId))
        .unwrap();
    assert!(empty.copies.is_empty() && empty.relations.is_empty());
    assert!(sketch.same_recipe(&before));
    assert_eq!(ids.high_water(), 99);
    assert_zero_copy(&mut sketch, &before, &mut ids);
}
fn assert_zero_copy(sketch: &mut Sketch2D, before: &Sketch2D, ids: &mut RecipeIdAllocator) {
    let report = sketch
        .copy_entities(&[SketchEntityId(3)], [0.0; 2], &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap();
    assert_eq!(report.copies, [SketchEntityId(100)]);
    assert_eq!(
        sketch.point(SketchEntityId(100)).unwrap().position,
        before.point(SketchEntityId(3)).unwrap().position
    );
}
#[test]
fn copy_output_overflow_is_refused_before_any_reservation() {
    let mut sketch = mixed_geometry();
    sketch
        .set_position(SketchEntityId(3), [f64::MAX, 0.0])
        .unwrap();
    let before = sketch.clone();
    let mut ids = ids_after(99);
    assert!(
        sketch
            .copy_entities(&[SketchEntityId(11)], [f64::MAX, 0.0], &mut || ids
                .allocate()
                .map(SketchEntityId))
            .is_err()
    );
    assert_eq!(ids.high_water(), 99);
    assert!(sketch.same_recipe(&before));
}
#[test]
fn copy_preserves_oblique_axis_bits_and_closed_spline_intent() {
    let mut sketch = mixed_geometry();
    let ellipse = SketchEllipse2D::new(
        SketchEntityId(30),
        SketchEntityId(3),
        [2.0, 7.0],
        [5.0, 2.0],
    )
    .unwrap();
    sketch.add_ellipse(ellipse).unwrap();
    add_closed_copy_source(&mut sketch);
    assert_parametric_copy_bits(&mut sketch, ellipse.major_axis());
}
fn assert_parametric_copy_bits(sketch: &mut Sketch2D, axis: [f64; 2]) {
    let mut ids = ids_after(99);
    let selected = [30, 31].map(SketchEntityId);
    let report = sketch
        .copy_entities(&selected, [1.0; 2], &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap();
    let ellipse = sketch.ellipse(report.copies[0]).unwrap();
    assert_eq!(
        ellipse.major_axis().map(f64::to_bits),
        axis.map(f64::to_bits)
    );
    assert!(sketch.spline(report.copies[1]).unwrap().closed());
    assert_reload(sketch);
}
fn add_closed_copy_source(sketch: &mut Sketch2D) {
    let points = vec![SketchEntityId(3), SketchEntityId(9)];
    let spline =
        SketchSpline2D::new(SketchEntityId(31), points, true, SketchSplineKind::Control).unwrap();
    sketch.add_spline(spline).unwrap();
}
