// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{identity::RecipeIdAllocator, sketch::SketchCopyOperation};

#[test]
fn batch_copies_each_instance_from_the_original_selection() {
    let mut sketch = super::defining_transforms::hidden_geometry();
    let before = sketch.clone();
    let mut ids = RecipeIdAllocator::restore(Some(99), &[]).unwrap();
    let operations = [
        SketchCopyOperation::Translate([3.0, 4.0]),
        SketchCopyOperation::Rotate {
            center: [0.0; 2],
            radians: std::f64::consts::FRAC_PI_2,
        },
    ];
    let reports = sketch
        .copy_selected_entities_batch(
            &[SketchEntityId(11), SketchEntityId(7)],
            &operations,
            &mut || ids.allocate().map(SketchEntityId),
        )
        .unwrap();
    assert_eq!(reports.len(), 2);
    assert_eq!(&sketch.points()[..before.points().len()], before.points());
    for (report, expected) in reports.iter().zip([[5.0, 4.0], [0.0, 2.0]]) {
        let line = sketch.line(report.copies[0]).unwrap();
        let circle = sketch.circle(report.copies[1]).unwrap();
        assert_eq!(line.endpoints[0], circle.center);
        let endpoint = sketch.point(line.endpoints[1]).unwrap().position;
        for (actual, expected) in endpoint.into_iter().zip(expected) {
            assert!((actual - expected).abs() < 2e-14);
        }
    }
    super::copy::assert_reload(&sketch);
}

#[test]
fn later_invalid_instance_discards_the_batch_and_keeps_reservations_retired() {
    let mut sketch = super::defining_transforms::hidden_geometry();
    let before = sketch.clone();
    let mut ids = RecipeIdAllocator::restore(Some(99), &[]).unwrap();
    let operations = [
        SketchCopyOperation::Translate([1.0, 0.0]),
        SketchCopyOperation::Rotate {
            center: [0.0; 2],
            radians: f64::NAN,
        },
    ];
    let result =
        sketch.copy_selected_entities_batch(&[SketchEntityId(11)], &operations, &mut || {
            ids.allocate().map(SketchEntityId)
        });
    assert!(result.is_err());
    assert!(sketch.same_recipe(&before));
    assert_eq!(ids.high_water(), 102);
}

#[test]
fn empty_batch_validates_selection_without_reserving_ids() {
    let mut sketch = super::defining_transforms::hidden_geometry();
    let before = sketch.clone();
    let mut ids = RecipeIdAllocator::restore(Some(99), &[]).unwrap();
    assert!(
        sketch
            .copy_selected_entities_batch(&[SketchEntityId(999_999)], &[], &mut || ids
                .allocate()
                .map(SketchEntityId))
            .is_err()
    );
    assert!(
        sketch
            .copy_selected_entities_batch(&[SketchEntityId(11)], &[], &mut || ids
                .allocate()
                .map(SketchEntityId))
            .unwrap()
            .is_empty()
    );
    assert!(sketch.same_recipe(&before));
    assert_eq!(ids.high_water(), 99);
}
