// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchPlane;

#[test]
fn explicit_projection_synchronization_observes_resize_and_refreshes_camera() {
    let (runtime, _) = runtime();
    let mut view = runtime.open_offscreen([100, 100]).unwrap();
    assert!(!view.synchronize_projection_extent().unwrap());
    view.target.as_ref().unwrap().resize([200, 100]).unwrap();
    assert!(view.synchronize_projection_extent().unwrap());
    assert!(!view.synchronize_projection_extent().unwrap());
    let point = view
        .sketch_point([125, 24], SketchPlane::xy())
        .unwrap()
        .unwrap();
    assert!((point[0] - 5.).abs() < 1e-10);
    assert!((point[1] - 5.).abs() < 1e-10);
    view.close();
    assert!(view.synchronize_projection_extent().is_err());
}
#[test]
fn unobserved_surface_resize_cannot_mix_old_camera_aspect_with_new_pixels() {
    let (runtime, _) = runtime();
    let mut view = runtime.open_offscreen([100, 100]).unwrap();
    view.target.as_ref().unwrap().resize([200, 100]).unwrap();
    assert!(
        view.sketch_point([125, 24], SketchPlane::xy())
            .unwrap_err()
            .contains("poll events")
    );
    assert!(view.sketch_pixel(SketchPlane::xy(), [0.; 2]).is_err());
    view.poll_events(1).unwrap();
    let point = view
        .sketch_point([125, 24], SketchPlane::xy())
        .unwrap()
        .unwrap();
    assert!((point[0] - 5.).abs() < 1e-10);
    assert!((point[1] - 5.).abs() < 1e-10);
}

#[test]
fn cached_batch_refuses_unobserved_resize_and_refreshes_after_poll() {
    let (runtime, _) = runtime();
    let mut view = runtime.open_offscreen([100, 100]).unwrap();
    let plane = SketchPlane::xy();
    let points = [[1., 0.], [-2., 1.]];
    let first = view.sketch_pixels(plane, &points).unwrap();
    assert_eq!(view.sketch_pixels(plane, &points).unwrap(), first);
    view.target.as_ref().unwrap().resize([200, 100]).unwrap();
    assert!(
        view.sketch_pixels(plane, &points)
            .unwrap_err()
            .contains("poll events")
    );
    view.poll_events(1).unwrap();
    let actual = view.sketch_pixels(plane, &points).unwrap();
    let expected: Vec<_> = points
        .iter()
        .map(|p| view.sketch_pixel(plane, *p).unwrap())
        .collect();
    assert_eq!(actual, expected);
    assert_ne!(actual, first);
    view.close();
    assert!(view.sketch_pixels(plane, &points).is_err());
}

#[test]
fn cached_batch_tracks_changed_points_and_camera() {
    let (runtime, _) = runtime();
    let mut view = runtime.open_offscreen([200, 100]).unwrap();
    let plane = SketchPlane::xy();
    let original = [[1., 0.], [-2., 1.]];
    let changed = [[1., 0.], [-3., 1.]];
    let first = view.sketch_pixels(plane, &original).unwrap();
    assert_eq!(view.sketch_pixels(plane, &original).unwrap(), first);
    let moved = view.sketch_pixels(plane, &changed).unwrap();
    assert_ne!(first, moved);
    assert_matches_scalar(&view, plane, &changed);
    view.navigate(crate::ViewportNavigation::Pan([12, -3]))
        .unwrap();
    assert_matches_scalar(&view, plane, &original);
    let perspective = crate::ViewportCamera {
        perspective: true,
        ..view.camera_state().unwrap()
    };
    view.set_camera(perspective).unwrap();
    assert_matches_scalar(&view, plane, &original);
    view.close();
}

fn assert_matches_scalar(view: &Viewport, plane: SketchPlane, points: &[[f64; 2]]) {
    let expected: Vec<_> = points
        .iter()
        .map(|point| view.sketch_pixel(plane, *point).unwrap())
        .collect();
    assert_eq!(view.sketch_pixels(plane, points).unwrap(), expected);
    assert_eq!(view.sketch_pixels(plane, points).unwrap(), expected);
}

#[test]
fn failed_camera_change_retires_cached_projection() {
    let (runtime, _) = runtime();
    let mut view = runtime.open_offscreen([100, 100]).unwrap();
    let plane = SketchPlane::xy();
    let points = [[1., 2.]];
    let before = view.sketch_pixels(plane, &points).unwrap();
    assert_eq!(view.projection_cache.borrow().retained(), 1);
    let invalid = crate::ViewportCamera {
        scale: 0.,
        ..view.camera_state().unwrap()
    };
    assert!(view.set_camera(invalid).is_err());
    assert_eq!(view.projection_cache.borrow().retained(), 0);
    assert_eq!(view.sketch_pixels(plane, &points).unwrap(), before);
    view.close();
}
