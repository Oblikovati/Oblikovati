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

#[test]
fn custom_coincident_selection_controls_point_proposals() {
    use oblikovati_api::types::{SKETCH_INFER_COINCIDENT, SKETCH_INFER_HORIZONTAL};
    let sketch = coincident_candidates();
    let policy = SketchInferencePolicy::for_units(&UnitsOfMeasure::default());
    let mut settings = default_sketch_settings();
    settings.custom_inference_types = true;
    settings.inference_types = SKETCH_INFER_HORIZONTAL;
    assert_eq!(
        sketch
            .infer_point_coincidence([2., 3.], &[], &settings, policy)
            .unwrap(),
        None
    );
    settings.inference_types = SKETCH_INFER_COINCIDENT;
    assert_eq!(
        sketch
            .infer_point_coincidence([2., 3.], &[], &settings, policy)
            .unwrap(),
        Some(SketchEntityId(3))
    );
}

#[test]
fn proposals_before_new_points_match_scoped_proposals_with_all_fresh_points_excluded() {
    use oblikovati_api::types::{
        INFERENCE_SCOPE_ALL_GEOMETRY, INFERENCE_SCOPE_CURRENT_COMMAND,
        INFERENCE_SCOPE_SELECTED_GEOMETRY,
    };
    use oblikovati_model::sketch::SketchInferenceCandidates;
    let source = coincident_candidates();
    let mut appended = source.clone();
    let positions = [[2.0, 3.0], [2.0001, 3.0]];
    let fresh = [SketchEntityId(101), SketchEntityId(102)];
    for (id, position) in fresh.into_iter().zip(positions) {
        appended.add_point(id, position).unwrap();
    }
    let scopes = [
        SketchInferenceCandidates::new(INFERENCE_SCOPE_ALL_GEOMETRY, &[]).unwrap(),
        SketchInferenceCandidates::new(INFERENCE_SCOPE_CURRENT_COMMAND, &[]).unwrap(),
        SketchInferenceCandidates::new(INFERENCE_SCOPE_SELECTED_GEOMETRY, &[SketchEntityId(9)])
            .unwrap(),
    ];
    let settings = default_sketch_settings();
    let policy = SketchInferencePolicy::for_units(&UnitsOfMeasure::default());
    for (index, scope) in scopes.iter().enumerate() {
        for position in positions {
            let early = source
                .infer_point_coincidence_scoped(position, &[], &settings, policy, scope, &[])
                .unwrap();
            let late = appended
                .infer_point_coincidence_scoped(position, &fresh, &settings, policy, scope, &fresh)
                .unwrap();
            assert_eq!(early, late);
            assert_eq!(
                early,
                [Some(SketchEntityId(3)), None, Some(SketchEntityId(9))][index]
            );
        }
    }
    let future =
        SketchInferenceCandidates::new(INFERENCE_SCOPE_SELECTED_GEOMETRY, &[fresh[0]]).unwrap();
    assert!(
        source
            .infer_point_coincidence_scoped(positions[0], &[], &settings, policy, &future, &[])
            .is_err()
    );
    assert_eq!(
        appended
            .infer_point_coincidence_scoped(
                positions[0],
                &fresh,
                &settings,
                policy,
                &future,
                &fresh
            )
            .unwrap(),
        None
    );
}
