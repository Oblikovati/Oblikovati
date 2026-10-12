// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::ViewportTriangle;
use martensite_vulkan::{FrameCapture, RenderBackend, RenderError, RenderFrame};
use rustycad_tkmath::gp::Point3;
use rustycad_tkservice::aspect::AspectNeutralWindow;
use std::sync::{Arc, Mutex};
#[derive(Default)]
struct RecordedDevice {
    frames: Vec<RenderFrame>,
    fail: bool,
    resize_once: Option<ResizeDuringDraw>,
    reported_extent: Option<[u32; 2]>,
}
struct ResizeDuringDraw {
    window: Handle<AspectNeutralWindow>,
    extent: [i32; 2],
}
#[derive(Clone, Default)]
struct RecordingDevice {
    state: Arc<Mutex<RecordedDevice>>,
}
impl RenderBackend for RecordingDevice {
    fn render(&mut self, frame: &RenderFrame) -> Result<FrameCapture, RenderError> {
        let mut state = self.state.lock().unwrap();
        state.frames.push(frame.clone());
        if state.fail {
            return Err(RenderError::rejected(
                "recorded device fault",
                "a successful draw",
            ));
        }
        if let Some(resize) = state.resize_once.take() {
            resize.window.set_size(resize.extent[0], resize.extent[1]);
        }
        let [width, height] = state.reported_extent.unwrap_or(frame.extent());
        Ok(FrameCapture {
            width,
            height,
            rgba: vec![255; width as usize * height as usize * 4],
            depth: vec![1.; width as usize * height as usize],
        })
    }
}
fn runtime() -> (ViewportRuntime, RecordingDevice) {
    let recorder = RecordingDevice::default();
    let driver = Graphic3dRenderDriver::with_backend(recorder.clone(), 1024).unwrap();
    (
        ViewportRuntime {
            state: Rc::new(RuntimeState {
                driver,
                x11: None,
                next_view: Cell::new(1),
            }),
        },
        recorder,
    )
}
fn scene() -> ViewportScene {
    ViewportScene {
        background: [0., 0., 1.],
        triangles: vec![ViewportTriangle {
            positions: [[-8., -8., 0.], [8., -8., 0.], [0., 8., 0.]],
            color: [1., 0., 0., 1.],
        }],
    }
}
#[test]
fn retained_scene_camera_resize_and_capture_generations_use_one_kernel_path() {
    let (runtime, recorder) = runtime();
    let mut view = runtime.open_offscreen([64, 48]).unwrap();
    assert!(view.capture().is_err());
    view.submit(&scene()).unwrap();
    let first = view.redraw().unwrap();
    assert_eq!(
        (
            first.scene_generation,
            first.camera_generation,
            first.extent
        ),
        (1, 0, [64, 48])
    );
    assert_eq!(draw_count(&recorder), 1);
    verify_camera_resize(&mut view, first.view_id);
}
fn verify_camera_resize(view: &mut Viewport, identity: u64) {
    view.set_camera(ViewportCamera {
        scale: 25.,
        ..Default::default()
    })
    .unwrap();
    assert!(view.capture().is_err());
    assert_eq!(view.resize([80, 60]).unwrap(), [80, 60]);
    let resized = view.redraw().unwrap();
    assert_eq!(
        (
            resized.camera_generation,
            resized.surface_generation,
            resized.extent
        ),
        (2, 1, [80, 60])
    );
    assert_eq!(view.id(), identity);
}
#[test]
fn failed_draw_never_returns_the_previous_success_as_current() {
    let (runtime, recorder) = runtime();
    let mut view = runtime.open_offscreen([8, 8]).unwrap();
    view.submit(&scene()).unwrap();
    view.redraw().unwrap();
    recorder.state.lock().unwrap().fail = true;
    assert!(view.redraw().unwrap_err().contains("recorded device fault"));
    assert!(view.capture().is_err());
    recorder.state.lock().unwrap().fail = false;
    assert_eq!(view.redraw().unwrap().scene_generation, 1);
}
#[test]
fn invalid_scene_and_camera_inputs_leave_previous_committed_view_unchanged() {
    let (runtime, recorder) = runtime();
    let mut view = runtime.open_offscreen([8, 8]).unwrap();
    view.submit(&scene()).unwrap();
    let before = view.redraw().unwrap();
    let mut bad = scene();
    bad.triangles[0].color[0] = f32::NAN;
    assert!(view.submit(&bad).unwrap_err().contains("color"));
    assert!(
        view.set_camera(ViewportCamera {
            scale: 0.,
            ..Default::default()
        })
        .is_err()
    );
    assert_eq!(view.capture().unwrap(), before);
    assert_eq!(recorder.state.lock().unwrap().frames.len(), 1);
}
#[test]
fn scene_replacement_and_removal_do_not_accumulate_rendered_structures() {
    let (runtime, recorder) = runtime();
    let mut view = runtime.open_offscreen([8, 8]).unwrap();
    view.submit(&scene()).unwrap();
    view.submit(&ViewportScene {
        background: [0., 1., 0.],
        triangles: Vec::new(),
    })
    .unwrap();
    view.redraw().unwrap();
    assert_eq!(draw_count(&recorder), 0);
    assert_eq!(view.live().unwrap().viewer.active_views().len(), 1);
    let viewer = view.live().unwrap().viewer.clone();
    view.close();
    view.close();
    assert!(viewer.active_views().is_empty());
    assert!(view.capture().is_err());
    assert!(view.redraw().is_err());
}
#[test]
fn target_classes_and_view_ids_are_explicit_and_runtime_outlives_its_creator() {
    let (runtime, _) = runtime();
    let mut first = runtime.open_offscreen([8, 8]).unwrap();
    let second = runtime.open_offscreen([8, 8]).unwrap();
    assert_ne!(first.id(), second.id());
    assert!(first.native_pixels().is_err());
    assert!(first.set_mapped(false).is_err());
    assert!(first.poll_events(32).unwrap().is_empty());
    assert!(runtime.open_window("fixture", [8, 8]).is_err());
    assert!(runtime.device_name().contains("injected"));
    assert!(runtime.validation_errors().is_empty());
    drop(runtime);
    assert_eq!(first.redraw().unwrap().extent, [8, 8]);
}
#[test]
fn size_and_generation_limits_decline_before_view_creation_or_draw() {
    let (runtime, _) = runtime();
    for extent in [[0, 8], [8, 0], [u32::MAX, 1]] {
        assert!(runtime.open_offscreen(extent).is_err());
    }
    runtime.state.next_view.set(u64::MAX);
    assert!(runtime.open_offscreen([8, 8]).is_err());
    assert!(next_generation(u64::MAX).is_err());
    assert_eq!(next_generation(0).unwrap(), 1);
}

#[test]
fn external_resize_invalidates_capture_and_is_observed_before_redraw() {
    let (runtime, _) = runtime();
    let mut view = runtime.open_offscreen([8, 8]).unwrap();
    view.redraw().unwrap();
    view.target.as_ref().unwrap().resize([16, 12]).unwrap();
    assert!(view.capture().is_err());
    let capture = view.redraw().unwrap();
    assert_eq!(capture.extent, [16, 12]);
    assert_eq!(capture.surface_generation, 1);
    assert_eq!(capture.camera_generation, 1);
    assert!(view.require_frame_extent([8, 8]).is_err());
}
#[test]
fn exhausted_generation_does_not_resize_target_or_return_an_external_stale_frame() {
    let (runtime, _) = runtime();
    let mut view = runtime.open_offscreen([8, 8]).unwrap();
    view.surface = u64::MAX;
    view.redraw().unwrap();
    assert!(view.resize([16, 12]).is_err());
    assert_eq!(view.target.as_ref().unwrap().extent(), [8, 8]);
    assert!(view.capture().is_ok());
    view.target.as_ref().unwrap().resize([16, 12]).unwrap();
    assert!(view.poll_events(0).is_err());
    assert!(view.capture().is_err());
    assert!(view.redraw().is_err());
}
#[test]
fn changed_extent_during_draw_is_a_decline_and_never_a_current_capture() {
    let (runtime, _) = runtime();
    let mut view = runtime.open_offscreen([8, 8]).unwrap();
    view.redraw().unwrap();
    view.target.as_ref().unwrap().resize([16, 12]).unwrap();
    assert_eq!(view.observe_draw_extent().unwrap(), Some([16, 12]));
    assert!(view.capture().is_err());
    assert_eq!(view.redraw().unwrap().extent, [16, 12]);
}

fn draw_count(recorder: &RecordingDevice) -> usize {
    recorder
        .state
        .lock()
        .unwrap()
        .frames
        .last()
        .unwrap()
        .draws()
        .len()
}
#[test]
fn camera_target_change_at_equal_radius_is_not_lost_by_kernel_center_setter() {
    let (runtime, _) = runtime();
    let mut view = runtime.open_offscreen([8, 8]).unwrap();
    let camera = ViewportCamera {
        eye: [0., 0., 0.],
        target: [20., 0., 0.],
        ..Default::default()
    };
    view.set_camera(camera).unwrap();
    let configured = view.live().unwrap().view.camera();
    assert_eq!(configured.center(), Point3::new(20., 0., 0.));
    assert_eq!(configured.eye(), Point3::new(0., 0., 0.));
    verify_parallel_camera_decline(&mut view, camera);
}
fn verify_parallel_camera_decline(view: &mut Viewport, camera: ViewportCamera) {
    let before = view.redraw().unwrap();
    assert!(
        view.set_camera(ViewportCamera {
            up: [1., 0., 0.],
            ..camera
        })
        .is_err()
    );
    assert_eq!(view.capture().unwrap(), before);
}
#[test]
fn native_wait_declines_offscreen_and_retired_targets() {
    let (runtime, _) = runtime();
    let wake = crate::ViewportWake::new().unwrap();
    let mut view = runtime.open_offscreen([8, 8]).unwrap();
    assert!(
        view.wait_events(&wake, None)
            .unwrap_err()
            .contains("native event source")
    );
    view.close();
    assert!(
        view.wait_events(&wake, None)
            .unwrap_err()
            .contains("live view")
    );
}

#[test]
fn changing_surface_during_backend_draw_reaches_the_public_call_boundary() {
    let (runtime, recorder) = runtime();
    let mut view = runtime.open_offscreen([8, 8]).unwrap();
    let Some(ViewTarget::Offscreen(window)) = &view.target else {
        panic!("expected offscreen fixture");
    };
    recorder.state.lock().unwrap().resize_once = Some(ResizeDuringDraw {
        window: window.clone(),
        extent: [16, 12],
    });
    assert_eq!(
        view.present().unwrap(),
        ViewportPresentation::SurfaceChanged([16, 12])
    );
    assert!(view.capture().is_err());
    verify_new_surface_capture(&mut view, &recorder);
}
fn verify_new_surface_capture(view: &mut Viewport, recorder: &RecordingDevice) {
    let ViewportPresentation::Completed(capture) = view.present().unwrap() else {
        panic!("expected new surface frame");
    };
    assert_eq!(capture.extent, [16, 12]);
    assert_eq!(capture.surface_generation, 1);
    assert_eq!(recorder.state.lock().unwrap().frames.len(), 2);
}

#[test]
fn wrong_backend_capture_extent_is_an_error_not_a_surface_change() {
    let (runtime, recorder) = runtime();
    let mut view = runtime.open_offscreen([8, 8]).unwrap();
    recorder.state.lock().unwrap().reported_extent = Some([9, 8]);
    assert!(view.present().unwrap_err().contains("expected extent"));
    assert!(view.capture().is_err());
    assert_eq!(view.surface, 0);
    recorder.state.lock().unwrap().reported_extent = None;
    assert_eq!(view.redraw().unwrap().extent, [8, 8]);
}
#[test]
fn visibility_event_invalidates_previous_capture_even_when_extent_is_unchanged() {
    let (runtime, _) = runtime();
    let mut view = runtime.open_offscreen([8, 8]).unwrap();
    view.redraw().unwrap();
    view.observe_visibility(&[crate::ViewportEvent::Mapped(false)])
        .unwrap();
    assert!(view.capture().is_err());
    assert_eq!(view.surface, 1);
    view.surface = u64::MAX;
    assert!(
        view.observe_visibility(&[crate::ViewportEvent::Mapped(true)])
            .is_err()
    );
    assert!(view.capture().is_err());
}
