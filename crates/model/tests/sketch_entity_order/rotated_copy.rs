// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{identity::RecipeIdAllocator, sketch::SketchCopyOperation};

#[test]
fn rotated_copy_preserves_shared_geometry_curve_frames_and_source() {
    let mut sketch = super::defining_transforms::hidden_geometry();
    let before = sketch.clone();
    let mut ids = RecipeIdAllocator::restore(Some(99), &[]).unwrap();
    let selected = [11, 7, 15, 20, 21, 22, 11].map(SketchEntityId);
    let report = sketch
        .copy_selected_entities(
            &selected,
            SketchCopyOperation::Rotate {
                center: [1.0; 2],
                radians: std::f64::consts::FRAC_PI_2,
            },
            &mut || ids.allocate().map(SketchEntityId),
        )
        .unwrap();
    assert_eq!(
        report.copies,
        [103, 104, 105, 106, 107, 108].map(SketchEntityId)
    );
    assert_eq!(&sketch.points()[..before.points().len()], before.points());
    let sources = [9, 3, 1, 11, 7, 15, 20, 21, 22].map(SketchEntityId);
    assert_eq!(
        report.relations,
        sources
            .into_iter()
            .zip((100..109).map(SketchEntityId))
            .collect::<Vec<_>>()
    );
    assert_eq!(
        sketch.line(SketchEntityId(11)).unwrap(),
        before.line(SketchEntityId(11)).unwrap()
    );
    assert_eq!(
        sketch.arc(SketchEntityId(15)).unwrap(),
        before.arc(SketchEntityId(15)).unwrap()
    );
    assert_eq!(
        sketch.ellipse(SketchEntityId(20)).unwrap(),
        before.ellipse(SketchEntityId(20)).unwrap()
    );
    for (id, expected) in [(100, [2.0, 0.0]), (101, [2.0, 2.0]), (102, [0.0, 0.0])] {
        near(sketch.point(SketchEntityId(id)).unwrap().position, expected);
    }
    let line = sketch.line(SketchEntityId(103)).unwrap();
    assert_eq!(line.endpoints, [100, 101].map(SketchEntityId));
    assert_eq!(
        sketch.circle(SketchEntityId(104)).unwrap().center,
        SketchEntityId(100)
    );
    assert!(sketch.arc(SketchEntityId(105)).unwrap().counter_clockwise);
    near(
        sketch.ellipse(SketchEntityId(106)).unwrap().major_axis(),
        [0.0, 1.0],
    );
    for id in [107, 108] {
        assert_eq!(
            sketch.spline(SketchEntityId(id)).unwrap().points(),
            [100, 101, 102].map(SketchEntityId)
        );
    }
    super::copy::assert_reload(&sketch);
}

fn near(actual: [f64; 2], expected: [f64; 2]) {
    for (actual, expected) in actual.into_iter().zip(expected) {
        assert!(
            (actual - expected).abs() <= 2e-14,
            "{actual:?} != {expected:?}"
        );
    }
}

#[test]
fn invalid_rotation_does_not_change_geometry_or_reserve_identities() {
    for (center, radians) in [
        ([f64::NAN, 0.0], 1.0),
        ([0.0; 2], f64::INFINITY),
        ([f64::MAX; 2], std::f64::consts::PI),
    ] {
        let mut sketch = super::defining_transforms::hidden_geometry();
        let before = sketch.clone();
        let mut ids = RecipeIdAllocator::restore(Some(99), &[]).unwrap();
        let result = sketch.copy_selected_entities(
            &[SketchEntityId(11)],
            SketchCopyOperation::Rotate { center, radians },
            &mut || ids.allocate().map(SketchEntityId),
        );
        assert!(result.is_err());
        assert!(sketch.same_recipe(&before));
        assert_eq!(ids.high_water(), 99);
    }
}
