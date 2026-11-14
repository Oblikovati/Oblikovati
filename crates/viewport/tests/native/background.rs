// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_viewport::ViewportBackground;
#[test]
fn native_application_background_survives_scene_refresh_and_clears_gradient() {
    let runtime = ViewportRuntime::new(ViewportOptions::default()).unwrap();
    let mut view = runtime
        .open_window("Oblikovati background acceptance", [128, 96])
        .unwrap();
    let empty = ViewportScene {
        background: [0., 0., 1.],
        triangles: Vec::new(),
    };
    view.submit(&empty).unwrap();
    verify_gradient(&mut view);
    let solid = ViewportBackground::Solid([45, 48, 54]);
    assert!(view.set_background(solid).unwrap());
    assert!(view.capture().is_err());
    let painted = verify_solid(&mut view, [45, 48, 54]);
    assert!(!view.set_background(solid).unwrap());
    assert_eq!(
        view.capture().unwrap().scene_generation,
        painted.scene_generation
    );
    view.submit(&empty).unwrap();
    verify_solid(&mut view, [45, 48, 54]);
    view.close();
    assert!(view.set_background(solid).is_err());
    assert!(runtime.validation_errors().unwrap().is_empty());
}
fn verify_gradient(view: &mut Viewport) {
    assert!(
        view.set_background(ViewportBackground::VerticalGradient {
            top: [255; 3],
            bottom: [0; 3]
        })
        .unwrap()
    );
    let capture = complete_surface_frame(view);
    let top = background_pixel(&capture, 8);
    let bottom = background_pixel(&capture, capture.extent[1] - 9);
    assert!(top[0] > bottom[0] + 100, "top={top:?}, bottom={bottom:?}");
    for pixel in [top, bottom] {
        assert_eq!(pixel[0], pixel[1]);
        assert_eq!(pixel[1], pixel[2]);
        assert_eq!(pixel[3], 255);
    }
}
fn verify_solid(view: &mut Viewport, expected: [u8; 3]) -> ViewportCapture {
    let capture = complete_surface_frame(view);
    for pixel in capture.rgba.as_chunks::<4>().0 {
        for (actual, expected) in pixel[..3].iter().zip(expected) {
            assert!(
                actual.abs_diff(expected) <= 1,
                "pixel={pixel:?}, expected={expected}"
            );
        }
        assert_eq!(pixel[3], 255);
    }
    capture
}
fn background_pixel(capture: &ViewportCapture, y: u32) -> [u8; 4] {
    let index = (y * capture.extent[0] + capture.extent[0] / 2) as usize;
    capture.rgba.as_chunks::<4>().0[index]
}
