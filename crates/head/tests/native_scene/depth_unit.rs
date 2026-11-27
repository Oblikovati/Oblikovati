// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_viewport::ViewportTriangle;
use std::cell::Cell;

struct FakeDepthTarget {
    camera: ViewportCamera,
    fitted: ViewportCamera,
    calls: usize,
    failure: bool,
    camera_reads: Cell<usize>,
}
impl DepthTarget for FakeDepthTarget {
    fn depth_camera(&self) -> Result<ViewportCamera, String> {
        self.camera_reads.set(self.camera_reads.get() + 1);
        Ok(self.camera)
    }
    fn fit_depth(&mut self) -> Result<(), String> {
        self.calls += 1;
        if self.failure {
            return Err("injected depth failure".into());
        }
        self.camera = self.fitted;
        Ok(())
    }
}
fn scene(z: f32) -> ViewportScene {
    ViewportScene {
        background: [0.; 3],
        triangles: vec![ViewportTriangle {
            face_index: None,
            positions: [[0., 0., z], [1., 0., z], [0., 1., z]],
            normals: None,
            color: [1.; 4],
        }],
    }
}
fn target() -> FakeDepthTarget {
    let camera = ViewportCamera::default();
    FakeDepthTarget {
        camera,
        fitted: camera,
        calls: 0,
        failure: false,
        camera_reads: Cell::new(0),
    }
}
#[test]
fn depth_refresh_skips_visible_and_empty_scenes() {
    let mut target = target();
    let mut empty = scene(0.);
    empty.triangles.clear();
    refresh_if_clipped(&mut target, &empty).unwrap();
    refresh_if_clipped(&mut target, &scene(0.)).unwrap();
    assert_eq!(target.calls, 0);
}
#[test]
fn depth_refresh_propagates_fit_failure_without_mutating_camera() {
    let mut target = target();
    target.failure = true;
    let before = target.camera;
    let error = refresh_if_clipped(&mut target, &scene(19.5)).unwrap_err();
    assert_eq!(error, "injected depth failure");
    assert_eq!(target.camera, before);
    assert_eq!(target.calls, 1);
}
#[test]
fn depth_refresh_rejects_a_fit_that_leaves_geometry_clipped() {
    let mut target = target();
    assert!(
        refresh_if_clipped(&mut target, &scene(19.5))
            .unwrap_err()
            .contains("scene remains outside")
    );
    assert_eq!(target.calls, 1);
}
#[test]
fn depth_refresh_accepts_a_fit_that_contains_the_scene() {
    let mut target = target();
    target.fitted.near = 0.1;
    refresh_if_clipped(&mut target, &scene(19.5)).unwrap();
    assert_eq!(target.calls, 1);
    assert_eq!(target.camera.near, 0.1);
}

#[test]
fn scene_depth_detects_near_far_and_boundary_clipping() {
    for perspective in [false, true] {
        let camera = ViewportCamera {
            perspective,
            ..ViewportCamera::default()
        };
        assert!(!scene_needs_depth_fit(&scene(0.), camera).unwrap());
        assert!(scene_needs_depth_fit(&scene(19.), camera).unwrap());
        assert!(scene_needs_depth_fit(&scene(-80.), camera).unwrap());
        assert!(scene_needs_depth_fit(&scene(19.5), camera).unwrap());
        assert!(scene_needs_depth_fit(&scene(-81.), camera).unwrap());
    }
}
#[test]
fn scene_depth_rejects_an_invalid_view_axis() {
    let camera = ViewportCamera {
        eye: [0.; 3],
        target: [0.; 3],
        ..ViewportCamera::default()
    };
    assert!(
        scene_needs_depth_fit(&scene(0.), camera)
            .unwrap_err()
            .contains("finite distinct points")
    );
}

#[test]
fn scene_depth_follows_rotated_view_direction() {
    let camera = ViewportCamera {
        eye: [20., 0., 0.],
        target: [0., 0., 0.],
        up: [0., 1., 0.],
        ..ViewportCamera::default()
    };
    let mut clipped = scene(0.);
    for point in &mut clipped.triangles[0].positions {
        point[0] = 19.5;
    }
    assert!(scene_needs_depth_fit(&clipped, camera).unwrap());
    assert!(!scene_needs_depth_fit(&scene(0.), camera).unwrap());
}

#[test]
fn scene_depth_rejects_invalid_ranges() {
    let camera = ViewportCamera {
        near: 0.,
        ..ViewportCamera::default()
    };
    assert!(
        scene_needs_depth_fit(&scene(0.), camera)
            .unwrap_err()
            .contains("0 < near < far")
    );
}

#[test]
fn geometry_crossing_the_eye_keeps_the_user_camera() {
    let mut target = target();
    target.failure = true;
    let mut across_eye = scene(19.5);
    across_eye.triangles[0].positions[1][2] = 21.;
    refresh_if_clipped(&mut target, &across_eye).unwrap();
    assert_eq!(target.calls, 0);
}

#[test]
fn cached_bounds_skip_scene_scan_when_fully_visible() {
    let scene = scene(0.);
    let bounds = SceneBounds::from_scene(&scene).unwrap();
    let mut target = target();
    target.failure = true;
    refresh_with_bounds(&mut target, &scene, bounds).unwrap();
    assert_eq!(target.calls, 0);
    assert_eq!(target.camera_reads.get(), 1);
}

#[test]
fn conservative_bounds_fall_back_to_exact_vertices() {
    let mut scene = scene(0.);
    scene.triangles[0].positions = [[10., 18., 0.], [18., 10., 0.], [14., 14., 0.]];
    let bounds = SceneBounds::from_scene(&scene).unwrap();
    let camera = ViewportCamera {
        eye: [20., 20., 0.],
        target: [0.; 3],
        near: 3.,
        ..ViewportCamera::default()
    };
    assert!(!bounds.fits(camera).unwrap());
    let mut target = target();
    target.camera = camera;
    target.failure = true;
    refresh_with_bounds(&mut target, &scene, bounds).unwrap();
    assert_eq!(target.calls, 0);
    assert_eq!(target.camera_reads.get(), 2);
}

#[test]
fn empty_scene_has_no_cached_bounds() {
    let mut empty = scene(0.);
    empty.triangles.clear();
    assert!(SceneBounds::from_scene(&empty).is_none());
}
