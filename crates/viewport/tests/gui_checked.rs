// SPDX-License-Identifier: GPL-2.0-only
//! #3851: checked target changes must reach the owner as fresh-frame requests.
use super::*;
use martensite_vulkan::RenderGuiSurfacePolicy;
fn checked_fixture() -> (Viewport, GuiFrameCompiler, RecordingDevice) {
    gui_resize_fixture_with_policy(RenderGuiSurfacePolicy::CheckedV2)
}
#[test]
fn checked_gui_resize_during_redraw_requires_a_fresh_packet() {
    let (mut view, compiler, recorder) = checked_fixture();
    schedule_recorded_gui_resize(&view, &recorder);
    assert_eq!(
        view.present().unwrap(),
        ViewportPresentation::SurfaceChanged([80, 60])
    );
    assert_gui_surface_replacement(&mut view, &compiler, &recorder, 1);
}
#[test]
fn checked_gui_resize_before_capture_does_not_publish_a_stale_frame() {
    let (mut view, compiler, recorder) = checked_fixture();
    assert!(view.draw_current_surface().unwrap());
    assert!(view.observe_draw_extent().unwrap().is_none());
    let Some(ViewTarget::Offscreen(window)) = &view.target else {
        panic!("expected offscreen target");
    };
    window.set_size(80, 60);
    assert_eq!(
        view.capture_completed_surface().unwrap(),
        ViewportPresentation::SurfaceChanged([80, 60])
    );
    assert_gui_surface_replacement(&mut view, &compiler, &recorder, 1);
}
#[test]
fn checked_gui_target_revision_without_extent_change_invalidates_the_old_packet() {
    let (mut view, compiler, recorder) = checked_fixture();
    recorder.state.lock().unwrap().revision_once = Some(view.live().unwrap().renderer.downgrade());
    assert_eq!(
        view.present().unwrap(),
        ViewportPresentation::SurfaceChanged([64, 48])
    );
    assert!(view.capture().is_err());
    assert_eq!(
        view.present().unwrap(),
        ViewportPresentation::SurfaceChanged([64, 48])
    );
    assert_eq!(recorder.state.lock().unwrap().frames.len(), 1);
    view.submit_gui_frame(&compiler, &frame([64, 48], [255; 4], [0., 0., 64., 48.]))
        .unwrap();
    let fresh = view.redraw().unwrap();
    assert_eq!(fresh.extent, [64, 48]);
    assert_eq!(fresh.surface_generation, 1);
    assert_eq!(fresh.camera_generation, 0);
}
#[test]
fn checked_gui_resize_does_not_hide_a_primary_backend_fault() {
    let (mut view, compiler, recorder) = checked_fixture();
    schedule_recorded_gui_resize(&view, &recorder);
    recorder.state.lock().unwrap().fail_after_resize = true;
    let error = view.present().unwrap_err();
    assert!(error.contains("recorded device fault"), "{error}");
    assert!(view.capture().is_err());
    recorder.state.lock().unwrap().fail_after_resize = false;
    assert_gui_surface_replacement(&mut view, &compiler, &recorder, 1);
}

#[test]
fn checked_gui_attachment_reports_target_change_without_accepting_the_old_packet() {
    let (mut view, compiler, recorder) = checked_fixture();
    view.redraw().unwrap();
    view.resize([80, 60]).unwrap();
    let old = frame([64, 48], [255; 4], [0., 0., 64., 48.]);
    assert_eq!(
        view.submit_gui_frame_checked(&compiler, &old).unwrap(),
        crate::ViewportGuiSubmission::SurfaceChanged([80, 60])
    );
    assert!(view.capture().is_err());
    assert_gui_surface_replacement(&mut view, &compiler, &recorder, 1);
}
#[test]
fn checked_gui_attachment_accepts_a_current_packet_and_rejects_invalid_draw_input() {
    let (mut view, compiler, _) = checked_fixture();
    let mut current = frame([64, 48], [255; 4], [0., 0., 64., 48.]);
    assert_eq!(
        view.submit_gui_frame_checked(&compiler, &current).unwrap(),
        crate::ViewportGuiSubmission::Accepted
    );
    current.batches[0].indices.push(u32::MAX);
    assert!(view.submit_gui_frame_checked(&compiler, &current).is_err());
}

#[test]
fn checked_gui_replacement_keeps_the_newer_packet_current() {
    let (mut view, compiler, recorder) = checked_fixture();
    recorder.state.lock().unwrap().replace_gui_once =
        Some(super::super::recorded_device::RecordedGuiReplacement {
            target: view.live().unwrap().renderer.downgrade(),
            layer: compiler
                .layer(&frame([64, 48], [0, 255, 0, 255], [0., 0., 64., 48.]))
                .unwrap(),
            extent: [64, 48],
        });
    assert_eq!(
        view.present().unwrap(),
        ViewportPresentation::SurfaceChanged([64, 48])
    );
    let ViewportPresentation::Completed(capture) = view.present().unwrap() else {
        panic!("expected the newer packet to remain current");
    };
    assert_eq!(capture.surface_generation, 0);
    assert_eq!(recorder.state.lock().unwrap().frames.len(), 2);
    assert_current_green_gui(&recorder);
}
#[test]
fn checked_gui_clear_retains_the_model_without_requiring_an_absent_packet() {
    let (mut view, _, recorder) = checked_fixture();
    view.submit(&scene()).unwrap();
    recorder.state.lock().unwrap().clear_gui_once = Some(view.live().unwrap().renderer.downgrade());
    assert_eq!(
        view.present().unwrap(),
        ViewportPresentation::SurfaceChanged([64, 48])
    );
    let ViewportPresentation::Completed(capture) = view.present().unwrap() else {
        panic!("expected model presentation after GUI clear");
    };
    assert_eq!(capture.surface_generation, 0);
    let state = recorder.state.lock().unwrap();
    assert_eq!(state.frames.len(), 2);
    assert_eq!(state.frames.last().unwrap().draws().len(), 1);
}

fn assert_current_green_gui(recorder: &RecordingDevice) {
    let state = recorder.state.lock().unwrap();
    let draw = state.frames.last().unwrap().draws().last().unwrap();
    assert!(
        draw.vertices()
            .iter()
            .all(|vertex| vertex.color == [0., 1., 0., 1.])
    );
}
