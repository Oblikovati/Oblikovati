// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
use oblikovati_viewport::{
    Viewport, ViewportCamera, ViewportCapture, ViewportOptions, ViewportPresentation,
    ViewportRuntime, ViewportScene, ViewportTriangle,
};
use std::time::{Duration, Instant};
#[path = "native/navigation.rs"]
mod navigation;
#[path = "native/prism.rs"]
mod prism;
#[path = "native/status_text.rs"]
mod status_text;
// The native suite shares the authored kernel-consumer corpus without running every fixture helper.
#[allow(dead_code)]
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
mod sketch_fixture;
fn scene() -> ViewportScene {
    ViewportScene {
        background: [0., 0., 1.],
        triangles: vec![ViewportTriangle {
            positions: [[-8., -8., 0.], [8., -8., 0.], [0., 8., 0.]],
            color: [1., 0., 0., 1.],
        }],
    }
}
fn verify_native(view: &mut Viewport) -> ViewportCapture {
    let capture = complete_surface_frame(view);
    await_presented_pixels(view, &capture);
    for expected in [[255, 0, 0, 255], [0, 0, 255, 255]] {
        assert!(capture.rgba.as_chunks::<4>().0.contains(&expected));
    }
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
    while view.native_pixels().unwrap() != capture.rgba {
        assert!(
            Instant::now() < deadline,
            "native viewport {} at {:?}: expected current GPU pixels",
            view.id(),
            capture.extent
        );
        std::thread::sleep(Duration::from_millis(5));
    }
}
fn verify_resize_and_mapping(view: &mut Viewport) {
    let first = verify_native(view);
    view.resize([400, 300]).unwrap();
    view.poll_events(64).unwrap();
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
    let mut view = runtime.open_window(title, [320, 240]).unwrap();
    view.submit(&scene()).unwrap();
    view
}
#[test]
fn application_owned_linux_windows_present_resize_capture_and_remove() {
    let runtime = ViewportRuntime::new(ViewportOptions::default()).unwrap();
    println!("OBLIKOVATI_VIEWPORT_DEVICE={}", runtime.device_name());
    let mut view = open_fixture(&runtime, "Oblikovati Rust viewport acceptance");
    verify_title_metadata(&mut view);
    verify_resize_and_mapping(&mut view);
    let mut second = open_fixture(&runtime, "Oblikovati Rust second viewport");
    verify_native(&mut second);
    view.close();
    verify_native(&mut second);
    second.close();
    prism::verify_meshed_prism(&runtime);
    assert!(
        runtime.validation_errors().is_empty(),
        "{:?}",
        runtime.validation_errors()
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
