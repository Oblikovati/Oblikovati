// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
use oblikovati_viewport::{
    GuiAtlas, GuiBatch, GuiFrame, GuiTextureKey, GuiTextureResources, GuiVertex, Viewport,
    ViewportCamera, ViewportCapture, ViewportOptions, ViewportPresentation, ViewportRuntime,
    ViewportScene, ViewportTriangle, ViewportWake,
};
use std::time::{Duration, Instant};
#[path = "native/background.rs"]
mod background;
#[path = "native/datum_plane.rs"]
mod datum_plane;
#[path = "native/lighting.rs"]
mod lighting;
#[path = "native/navigation.rs"]
mod navigation;
#[path = "native/prism.rs"]
mod prism;
#[path = "native/status_text.rs"]
mod status_text;
#[path = "native/wayland.rs"]
mod wayland;
#[path = "native/window_events.rs"]
mod window_events;
// The native suite shares the authored kernel-consumer corpus without running every fixture helper.
#[allow(dead_code)]
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
mod sketch_fixture;
fn scene() -> ViewportScene {
    ViewportScene {
        background: [0., 0., 1.],
        triangles: vec![ViewportTriangle {
            face_index: None,
            positions: [[-8., -8., 0.], [8., -8., 0.], [0., 8., 0.]],
            color: [1., 0., 0., 1.],
        }],
    }
}
fn verify_native(view: &mut Viewport) -> ViewportCapture {
    let capture = complete_surface_frame(view);
    await_presented_pixels(view, &capture);
    assert!(
        capture
            .rgba
            .as_chunks::<4>()
            .0
            .iter()
            .any(|p| p[0] > 0 && p[1] == 0 && p[2] == 0 && p[3] == 255)
    );
    assert!(capture.rgba.as_chunks::<4>().0.contains(&[0, 0, 255, 255]));
    capture
}
fn complete_surface_frame(view: &mut Viewport) -> ViewportCapture {
    let deadline = Instant::now() + Duration::from_secs(3);
    loop {
        match view.present().unwrap() {
            ViewportPresentation::Completed(capture) => return capture,
            ViewportPresentation::SurfaceChanged(_) => {
                assert!(view.capture().is_err());
                assert!(
                    Instant::now() < deadline,
                    "expected a stable native surface within three seconds"
                );
                view.poll_events(64).unwrap();
            }
        }
    }
}
fn await_presented_pixels(view: &Viewport, capture: &ViewportCapture) {
    let deadline = Instant::now() + Duration::from_secs(3);
    loop {
        let pixels = view.native_pixels();
        if pixels.as_ref().is_ok_and(|pixels| pixels == &capture.rgba) {
            return;
        }
        assert!(
            Instant::now() < deadline,
            "native viewport {} at {:?}: expected current GPU pixels, {}",
            view.id(),
            capture.extent,
            pixels.err().as_deref().unwrap_or("pixels differed"),
        );
        std::thread::sleep(Duration::from_millis(5));
    }
}
fn verify_resize_and_mapping(view: &mut Viewport) {
    let first = verify_native(view);
    resize_native(view, [800, 600]);
    let resized = verify_native(view);
    assert!(resized.surface_generation > first.surface_generation);
    view.set_camera(ViewportCamera::default()).unwrap();
    verify_native(view);
    view.set_mapped(false).unwrap();
    assert!(view.redraw().is_err());
    assert!(view.capture().is_err());
    assert!(view.native_pixels().is_err());
    view.set_mapped(true).unwrap();
    assert!(view.capture().is_err());
    assert!(verify_native(view).surface_generation > resized.surface_generation);
}
fn open_fixture(runtime: &ViewportRuntime, title: &str) -> Viewport {
    // The small pixel fixtures use the system-framed window, which has no app size hint.
    submit_fixture(runtime.open_document_window(title, [320, 240]).unwrap())
}
fn submit_fixture(mut view: Viewport) -> Viewport {
    view.submit(&scene()).unwrap();
    view
}

fn resize_native(view: &mut Viewport, extent: [u32; 2]) {
    view.resize(extent).unwrap();
    let wake = ViewportWake::new().unwrap();
    let deadline = Instant::now() + Duration::from_secs(2);
    loop {
        let actual = view.drawable_extent().unwrap();
        if actual == extent {
            return;
        }
        let remaining = deadline.saturating_duration_since(Instant::now());
        assert!(
            !remaining.is_zero(),
            "native extent {actual:?}: expected {extent:?}"
        );
        view.wait_events(&wake, Some(remaining)).unwrap();
        view.poll_events(64).unwrap();
    }
}

#[test]
fn native_present_can_defer_capture_without_losing_current_pixels() {
    let runtime = ViewportRuntime::new(ViewportOptions::default()).unwrap();
    let mut view = open_fixture(&runtime, "Oblikovati deferred capture");
    assert_eq!(view.completed_generations(), None);
    let expected = verify_native(&mut view);
    let completed = view.completed_generations().unwrap();
    assert!(view.present_without_capture().unwrap());
    assert_eq!(view.completed_generations(), Some(completed));
    assert_eq!(view.capture().unwrap(), expected);
    await_presented_pixels(&view, &expected);
    view.close();
    assert!(runtime.validation_errors().unwrap().is_empty());
}

#[test]
fn packed_native_preview_matches_full_capture_after_resize() {
    use oblikovati_viewport::ViewportPreviewPixels;

    let runtime = ViewportRuntime::new(ViewportOptions::default()).unwrap();
    let mut view = open_fixture(&runtime, "Oblikovati packed preview");
    for extent in [[320, 240], [319, 233]] {
        resize_native(&mut view, extent);
        let expected = verify_native(&mut view);
        assert!(
            view.present_with_preview(|size, generations, pixels| {
                assert_eq!(size, extent);
                assert_eq!(
                    generations,
                    [
                        expected.scene_generation,
                        expected.camera_generation,
                        expected.surface_generation,
                        expected.overlay_generation
                    ]
                );
                let ViewportPreviewPixels::Rgb(rgb) = pixels else {
                    panic!("expected the packed RGB path");
                };
                assert_eq!(rgb.len(), expected.rgba.len() / 4);
                assert!(
                    rgb.iter()
                        .zip(expected.rgba.as_chunks::<4>().0)
                        .all(|(rgb, rgba)| rgb.as_slice() == &rgba[..3])
                );
            })
            .unwrap()
        );
        assert_eq!(view.capture().unwrap(), expected);
    }
    view.close();
    assert!(runtime.validation_errors().unwrap().is_empty());
}
#[test]
fn application_owned_linux_windows_present_resize_capture_and_remove() {
    let runtime = ViewportRuntime::new(ViewportOptions::default()).unwrap();
    println!("OBLIKOVATI_VIEWPORT_DEVICE={}", runtime.device_name());
    let mut view = submit_fixture(
        runtime
            .open_window("Oblikovati Rust viewport acceptance", [640, 480])
            .unwrap(),
    );
    verify_title_metadata(&mut view);
    verify_resize_and_mapping(&mut view);
    let mut second = submit_fixture(
        runtime
            .open_window("Oblikovati Rust second viewport", [650, 490])
            .unwrap(),
    );
    verify_native(&mut second);
    view.close();
    verify_native(&mut second);
    second.close();
    prism::verify_meshed_prism(&runtime);
    assert!(
        runtime.validation_errors().unwrap().is_empty(),
        "{:?}",
        runtime.validation_errors().unwrap()
    );
}

fn verify_title_metadata(view: &mut Viewport) {
    view.set_title("Oblikovati - current part").unwrap();
    assert_eq!(view.native_title().unwrap(), "Oblikovati - current part");
    assert!(view.set_title("").is_err());
    assert!(view.set_title("invalid\0title").is_err());
}

#[path = "native/projection.rs"]
mod projection;

#[path = "native/sketch_preview.rs"]
mod sketch_preview;

#[path = "native/sketch_display.rs"]
mod sketch_display;

#[path = "native/point_preview.rs"]
mod point_preview;

#[path = "native/gui_composition.rs"]
mod gui_composition;
#[path = "gui_fixture.rs"]
mod gui_fixture;

#[path = "native/sketch_hover.rs"]
mod sketch_hover;
#[path = "native/sketch_palette.rs"]
mod sketch_palette;

#[path = "native/projection_performance.rs"]
mod projection_performance;
