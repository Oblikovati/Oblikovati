// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_viewport::ViewportNavigation;
fn close_vector(actual: [f64; 3], expected: [f64; 3]) {
    for (actual, expected) in actual.into_iter().zip(expected) {
        assert!(
            (actual - expected).abs() < 1e-10,
            "{actual}: expected {expected}"
        );
    }
}
#[test]
fn native_pan_and_orbit_preserve_camera_invariants_and_reach_presented_pixels() {
    let runtime = ViewportRuntime::new(ViewportOptions::default()).unwrap();
    let mut view = runtime
        .open_window("Oblikovati navigation acceptance", [128, 96])
        .unwrap();
    view.submit(&scene()).unwrap();
    let before = verify_native(&mut view);
    view.navigate(ViewportNavigation::Pan([12, 6])).unwrap();
    close_vector(view.camera_state().unwrap().target, [-2.5, 1.25, 0.]);
    let after = verify_native(&mut view);
    assert_eq!(after.scene_generation, before.scene_generation);
    assert_eq!(after.camera_generation, before.camera_generation + 1);
    assert_ne!(after.rgba, before.rgba);
    verify_native_orbit(&mut view);
    view.close();
    assert!(view.camera_state().is_err());
    assert!(view.navigate(ViewportNavigation::Pan([1, 0])).is_err());
    assert!(runtime.validation_errors().is_empty());
}
fn verify_native_orbit(view: &mut Viewport) {
    view.set_camera(ViewportCamera::default()).unwrap();
    view.navigate(ViewportNavigation::Orbit {
        yaw: 0.2,
        pitch: -0.1,
    })
    .unwrap();
    let camera = view.camera_state().unwrap();
    close_vector(camera.target, [0.; 3]);
    close_vector(
        camera.eye,
        [
            20. * 0.2_f64.sin() * 0.1_f64.cos(),
            20. * 0.1_f64.sin(),
            20. * 0.2_f64.cos() * 0.1_f64.cos(),
        ],
    );
    verify_native(view);
}
fn anchored_navigation(view: &mut Viewport, factor: f64) {
    let command = ViewportNavigation::Zoom {
        factor,
        anchor: Some([96, 24]),
    };
    view.navigate(command).unwrap();
}
#[test]
fn anchored_zoom_round_trips_and_invalid_changes_preserve_completed_frames() {
    let runtime = ViewportRuntime::new(ViewportOptions::default()).unwrap();
    let mut view = runtime
        .open_window("Oblikovati zoom acceptance", [128, 96])
        .unwrap();
    view.submit(&scene()).unwrap();
    let before = verify_native(&mut view);
    anchored_navigation(&mut view, 0.9);
    let zoomed = view.camera_state().unwrap();
    assert!((zoomed.scale - 18.).abs() < 1e-10);
    close_vector(zoomed.target, [2. / 3., 0.5, 0.]);
    verify_native(&mut view);
    anchored_navigation(&mut view, 0.9_f64.recip());
    let restored = verify_native(&mut view);
    assert_eq!(restored.rgba, before.rgba);
    verify_camera_rejection(&mut view, &restored);
    view.close();
    assert!(runtime.validation_errors().is_empty());
}
fn centered_navigation(factor: f64) -> ViewportNavigation {
    ViewportNavigation::Zoom {
        factor,
        anchor: None,
    }
}
fn verify_camera_rejection(view: &mut Viewport, completed: &ViewportCapture) {
    let camera = view.camera_state().unwrap();
    assert!(!view.navigate(ViewportNavigation::Pan([0, 0])).unwrap());
    for factor in [f64::NAN, 1e308] {
        assert!(view.navigate(centered_navigation(factor)).is_err());
    }
    assert_eq!(view.camera_state().unwrap(), camera);
    assert_eq!(view.capture().unwrap(), *completed);
    view.navigate(centered_navigation(0.9)).unwrap();
    assert!(view.capture().is_err());
    verify_native(view);
}

#[test]
fn fit_restores_offscreen_geometry_and_home_sets_the_go_isometric_direction() {
    let runtime = ViewportRuntime::new(ViewportOptions::default()).unwrap();
    let mut view = runtime
        .open_window("Oblikovati Home acceptance", [128, 96])
        .unwrap();
    view.submit(&scene()).unwrap();
    let fitted = recover_panned_geometry(&mut view);
    verify_home_direction(&mut view);
    let home = verify_native(&mut view);
    assert_eq!(home.scene_generation, fitted.scene_generation);
    assert!(home.camera_generation > fitted.camera_generation);
    view.close();
    assert!(runtime.validation_errors().is_empty());
}
fn verify_home_direction(view: &mut Viewport) {
    assert!(
        view.navigate(ViewportNavigation::Home { margin: 0.1 })
            .unwrap()
    );
    let camera = view.camera_state().unwrap();
    close_vector(camera.target, [0.; 3]);
    close_vector(camera.up, [0., 1., 0.]);
    let offset = std::array::from_fn::<_, 3, _>(|axis| camera.eye[axis] - camera.target[axis]);
    assert!(offset[0] > 0.);
    close_vector(offset, [offset[0]; 3]);
    assert!(
        view.navigate(ViewportNavigation::Home { margin: f64::NAN })
            .is_err()
    );
    assert_eq!(view.camera_state().unwrap(), camera);
}

fn recover_panned_geometry(view: &mut Viewport) -> ViewportCapture {
    let original = view.camera_state().unwrap();
    view.navigate(ViewportNavigation::Pan([2000, 2000]))
        .unwrap();
    assert!(
        view.navigate(ViewportNavigation::Fit { margin: 0.1 })
            .unwrap()
    );
    let fitted = verify_native(view);
    close_vector(view.camera_state().unwrap().target, [0.; 3]);
    close_vector(view.camera_state().unwrap().up, original.up);
    fitted
}
