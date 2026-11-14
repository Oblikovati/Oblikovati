// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    sketch::{Sketch2D, SketchEntityId, SketchInferencePolicy},
    sketch_settings::{SketchSettings, default_sketch_settings},
    units::UnitsOfMeasure,
};

fn coincident_candidates() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(9), [2., 3.]).unwrap();
    sketch.add_point(SketchEntityId(3), [2., 3.]).unwrap();
    sketch
}
#[test]
fn point_proposals_keep_shared_authored_order_and_exclude_fresh_points() {
    let sketch = coincident_candidates();
    let settings = default_sketch_settings();
    let policy = SketchInferencePolicy::for_units(&UnitsOfMeasure::default());
    assert_eq!(
        sketch
            .infer_point_coincidence([2., 3.], &[], &settings, policy)
            .unwrap(),
        Some(SketchEntityId(3))
    );
    assert_eq!(
        sketch
            .infer_point_coincidence([2., 3.], &[SketchEntityId(3)], &settings, policy)
            .unwrap(),
        Some(SketchEntityId(9))
    );
    assert_eq!(
        sketch
            .infer_point_coincidence(
                [2., 3.],
                &[SketchEntityId(3), SketchEntityId(9)],
                &settings,
                policy
            )
            .unwrap(),
        None
    );
}
#[test]
fn disabled_inference_and_points_outside_reach_do_not_propose_coincidence() {
    let sketch = coincident_candidates();
    let policy = SketchInferencePolicy::for_units(&UnitsOfMeasure::default());
    let settings = SketchSettings {
        infer_constraints: false,
        ..default_sketch_settings()
    };
    assert_eq!(
        sketch
            .infer_point_coincidence([2., 3.], &[], &settings, policy)
            .unwrap(),
        None
    );
    let outside = [2. + 2. * policy.snap_distance_working, 3.];
    assert_eq!(
        sketch
            .infer_point_coincidence(outside, &[], &default_sketch_settings(), policy)
            .unwrap(),
        None
    );
    assert!(
        sketch
            .infer_point_coincidence([f64::NAN, 3.], &[], &settings, policy)
            .is_err()
    );
}
