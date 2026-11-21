// SPDX-License-Identifier: GPL-2.0-only
//! Scene edits must not leave parts clipped against an old camera depth range (#4044).
use super::*;
use oblikovati_martensite::{
    PrismMeshOptions, PrismProbeDefinition, PrismProbeLoop, retain_planar_prism,
};
use oblikovati_viewport::{ViewportCamera, ViewportNavigation, ViewportScene};

#[test]
fn native_scene_refreshes_depth_after_geometry_edit() {
    let (mut window, before_home, fitted) = fitted_window();
    let first = capture_native_scene(&mut window);
    let pixel = window.view.model_pixel([4., 0., 0.]).unwrap();
    window.submit_scene(update(2.)).unwrap();
    assert_same_pose(window.view.camera_state().unwrap(), fitted);
    assert_eq!(window.view.model_pixel([4., 0., 0.]).unwrap(), pixel);
    assert_fitted_edit(&mut window, first.camera_generation);
    assert!(window.view.previous_view().unwrap());
    assert_eq!(window.view.camera_state().unwrap(), before_home);
    window.finish().unwrap();
}

#[test]
fn empty_native_scene_preserves_camera_depth_and_generation() {
    let (mut window, _, camera) = fitted_window();
    let first = capture_native_scene(&mut window);
    let mut empty = update(1.);
    empty.scene.triangles.clear();
    window.submit_scene(empty).unwrap();
    let after = capture_native_scene(&mut window);
    assert_eq!(window.view.camera_state().unwrap(), camera);
    assert_eq!(after.camera_generation, first.camera_generation);
    assert!(after.scene_generation > first.scene_generation);
    window.finish().unwrap();
}

#[test]
fn native_scene_depth_refresh_survives_a_resize() {
    let (mut window, _, _) = fitted_window();
    window.view.resize([800, 600]).unwrap();
    window.view.poll_events(64).unwrap();
    let first = capture_native_scene(&mut window);
    let camera = window.view.camera_state().unwrap();
    window.submit_scene(update(2.)).unwrap();
    assert_same_pose(window.view.camera_state().unwrap(), camera);
    assert_fitted_edit(&mut window, first.camera_generation);
    window.finish().unwrap();
}

#[test]
fn native_scene_refreshes_depth_for_negative_extrusion() {
    let (mut window, _, camera) = fitted_window();
    let first = capture_native_scene(&mut window);
    window.submit_scene(update(-2.)).unwrap();
    assert_same_pose(window.view.camera_state().unwrap(), camera);
    assert_fitted_edit(&mut window, first.camera_generation);
    window.finish().unwrap();
}

fn fitted_window() -> (NativeHeadWindow, ViewportCamera, ViewportCamera) {
    let mut window = NativeHeadWindow::open().unwrap();
    window.submit_scene(update(1.)).unwrap();
    let previous = window.view.camera_state().unwrap();
    window
        .view
        .navigate(ViewportNavigation::Home { margin: 0.1 })
        .unwrap();
    let fitted = window.view.camera_state().unwrap();
    (window, previous, fitted)
}

fn assert_same_pose(actual: ViewportCamera, expected: ViewportCamera) {
    assert_eq!(
        (actual.eye, actual.target, actual.up, actual.scale),
        (expected.eye, expected.target, expected.up, expected.scale)
    );
}

fn assert_fitted_edit(window: &mut NativeHeadWindow, previous_generation: u64) {
    let edited = capture_native_scene(window);
    assert!(edited.camera_generation > previous_generation);
    assert!(!window.view.navigate(ViewportNavigation::FitDepth).unwrap());
    assert_eq!(edited.rgba, capture_native_scene(window).rgba);
}

fn update(depth: f64) -> crate::HeadSceneUpdate {
    let mesh = prism(depth)
        .presentation(PrismMeshOptions {
            chord_error: 0.05,
            angular_error: 0.5,
        })
        .unwrap();
    let scene =
        ViewportScene::from_prism(&mesh, [0.2, 0.21, 0.24], [0.702, 0.722, 0.741, 1.]).unwrap();
    crate::HeadSceneUpdate {
        scene,
        status: status(),
    }
}

fn prism(depth: f64) -> oblikovati_martensite::PrismProbeSnapshot {
    retain_planar_prism(&PrismProbeDefinition {
        outer: PrismProbeLoop::polygon(
            vec![1, 2, 3, 4],
            vec![
                [0.8888889, -2.4666667],
                [7.5555553, -2.4666667],
                [7.5555553, 1.5333333],
                [0.8888889, 1.5333333],
            ],
        ),
        holes: Vec::new(),
        displacement: [0., 0., depth],
    })
    .unwrap()
}

fn status() -> crate::HeadSceneStatus {
    crate::HeadSceneStatus {
        source: None,
        cached_assembly: false,
        visible_bodies: 1,
        health: Vec::new(),
        presentation_declines: Vec::new(),
        scene_errors: Vec::new(),
    }
}

#[test]
fn previous_view_refreshes_depth_for_changed_scene_without_new_history() {
    let (mut window, before_home, fitted) = fitted_window();
    window
        .view
        .navigate(ViewportNavigation::Fit { margin: 0.2 })
        .unwrap();
    window.submit_scene(update(2.)).unwrap();
    assert!(window.previous_view().unwrap());
    assert_same_pose(window.view.camera_state().unwrap(), fitted);
    let restored = capture_native_scene(&mut window);
    assert!(!window.view.navigate(ViewportNavigation::FitDepth).unwrap());
    assert_eq!(capture_native_scene(&mut window).rgba, restored.rgba);
    assert!(window.previous_view().unwrap());
    assert_same_pose(window.view.camera_state().unwrap(), before_home);
    assert!(!window.previous_view().unwrap());
    window.finish().unwrap();
}

#[test]
fn marked_view_refreshes_depth_for_changed_scene() {
    let (mut window, _, fitted) = fitted_window();
    window.mark_view().unwrap();
    window.submit_scene(update(2.)).unwrap();
    assert!(window.restore_marked_view().unwrap());
    assert_same_pose(window.view.camera_state().unwrap(), fitted);
    let restored = capture_native_scene(&mut window);
    assert!(!window.view.navigate(ViewportNavigation::FitDepth).unwrap());
    assert_eq!(capture_native_scene(&mut window).rgba, restored.rgba);
    window.finish().unwrap();
}
