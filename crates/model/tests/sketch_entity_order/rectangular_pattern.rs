// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    identity::RecipeIdAllocator,
    sketch::{SketchCopyOperation, SketchRectangularPatternLayout},
};
use std::collections::BTreeSet;

fn grid() -> SketchRectangularPatternLayout {
    SketchRectangularPatternLayout {
        counts: [3, 2],
        vectors: [[-10., 0.], [0., 5.]],
        fitted: false,
        suppressed: BTreeSet::new(),
    }
}
fn offsets(layout: &SketchRectangularPatternLayout) -> Vec<([u32; 2], [f64; 2])> {
    layout
        .occurrences()
        .unwrap()
        .into_iter()
        .map(|item| match item.operation {
            SketchCopyOperation::Translate(vector) => (item.index, vector),
            _ => panic!("rectangular layout must use translations"),
        })
        .collect()
}
#[test]
fn reverse_grid_and_fitted_span_keep_occurrence_identity_and_suppression() {
    let mut layout = grid();
    assert_eq!(
        offsets(&layout),
        vec![
            ([0, 1], [0., 5.]),
            ([1, 0], [-10., 0.]),
            ([1, 1], [-10., 5.]),
            ([2, 0], [-20., 0.]),
            ([2, 1], [-20., 5.])
        ]
    );
    layout.suppressed.insert([1, 1]);
    layout.fitted = true;
    assert_eq!(
        offsets(&layout),
        vec![
            ([0, 1], [0., 5.]),
            ([1, 0], [-5., 0.]),
            ([2, 0], [-10., 0.]),
            ([2, 1], [-10., 5.])
        ]
    );
    layout.fitted = false;
    assert_eq!(offsets(&layout)[2], ([2, 0], [-20., 0.]));
    layout.suppressed.remove(&[1, 1]);
    assert_eq!(offsets(&layout).len(), 5);
}
#[test]
fn grid_layout_creates_six_circles_from_one_seed() {
    let mut sketch = super::defining_transforms::hidden_geometry();
    let operations: Vec<_> = grid()
        .occurrences()
        .unwrap()
        .into_iter()
        .map(|item| item.operation)
        .collect();
    let mut ids = RecipeIdAllocator::restore(Some(200), &[]).unwrap();
    let reports = sketch
        .copy_selected_entities_batch(&[SketchEntityId(7)], &operations, &mut || {
            ids.allocate().map(SketchEntityId)
        })
        .unwrap();
    let centers: Vec<_> = reports
        .iter()
        .map(|report| {
            sketch
                .point(sketch.circle(report.copies[0]).unwrap().center)
                .unwrap()
                .position
        })
        .collect();
    assert_eq!(
        centers,
        [[0., 5.], [-10., 0.], [-10., 5.], [-20., 0.], [-20., 5.]]
    );
    assert_eq!(sketch.circles().len(), 6);
    super::copy::assert_reload(&sketch);
}
#[test]
fn invalid_grid_is_rejected_before_generating_any_occurrences() {
    let mut layout = grid();
    layout.counts[0] = 0;
    assert!(layout.occurrences().is_err());
    layout = grid();
    layout.suppressed.insert([0, 0]);
    assert!(layout.occurrences().is_err());
    layout.suppressed.clear();
    layout.suppressed.insert([3, 0]);
    assert!(layout.occurrences().is_err());
    layout = grid();
    layout.vectors[0][0] = f64::INFINITY;
    assert!(layout.occurrences().is_err());
    layout.vectors[0][0] = f64::MAX;
    assert!(layout.occurrences().is_err());
}
#[test]
fn singleton_fitted_direction_does_not_divide_by_zero() {
    let mut layout = grid();
    layout.counts = [1, 2];
    layout.fitted = true;
    assert_eq!(offsets(&layout), [([0, 1], [0., 5.])]);
}
