// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::identity::RecipeIdAllocator;
use oblikovati_model::sketch::SketchSelectionTransform;

#[test]
fn placed_copy_preserves_shared_points_and_scales_analytic_curves() {
    let source = super::defining_transforms::hidden_geometry();
    let before = source.clone();
    let mut target = Sketch2D::default();
    let mut ids = RecipeIdAllocator::restore(Some(99), &[]).unwrap();
    let selected = [11, 7, 15, 20, 21, 22].map(SketchEntityId);
    let report = target
        .copy_transformed_selection(
            &source,
            &selected,
            SketchSelectionTransform::Similarity {
                position: [10.0, 20.0],
                radians: std::f64::consts::FRAC_PI_2,
                scale: -2.0,
            },
            &mut || ids.allocate().map(SketchEntityId),
        )
        .unwrap();
    assert!(source.same_recipe(&before));
    assert_eq!(
        report.copies,
        [103, 104, 105, 106, 107, 108].map(SketchEntityId)
    );
    let line = target.line(SketchEntityId(103)).unwrap();
    assert_eq!(line.endpoints, [100, 101].map(SketchEntityId));
    assert_eq!(
        target.circle(SketchEntityId(104)).unwrap().center,
        line.endpoints[0]
    );
    assert_eq!(target.circle(SketchEntityId(104)).unwrap().radius, 4.0);
    assert_eq!(
        target.ellipse(SketchEntityId(106)).unwrap().radii(),
        [6.0, 2.0]
    );
    let endpoint = target.point(line.endpoints[1]).unwrap().position;
    assert!((endpoint[0] - 10.0).abs() < 1e-12 && (endpoint[1] - 16.0).abs() < 1e-12);
    assert!(target.arc(SketchEntityId(105)).unwrap().counter_clockwise);
    super::copy::assert_reload(&target);
}

#[test]
fn invalid_placement_does_not_change_target_or_reserve_identities() {
    let source = super::defining_transforms::hidden_geometry();
    for scale in [
        0.0,
        f64::INFINITY,
        f64::MAX,
        f64::MIN_POSITIVE * f64::MIN_POSITIVE,
    ] {
        let mut target = mixed_geometry();
        let before = target.clone();
        let mut ids = RecipeIdAllocator::restore(Some(99), &[]).unwrap();
        let result = target.copy_transformed_selection(
            &source,
            &[SketchEntityId(7)],
            SketchSelectionTransform::Similarity {
                position: [1.0; 2],
                radians: 0.0,
                scale,
            },
            &mut || ids.allocate().map(SketchEntityId),
        );
        assert!(result.is_err(), "scale {scale:?}");
        assert!(target.same_recipe(&before));
        assert_eq!(ids.high_water(), 99);
    }
}
