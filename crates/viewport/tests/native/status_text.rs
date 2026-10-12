// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn native_status_uses_the_same_view_and_keeps_pixel_position_during_camera_navigation() {
    let runtime = ViewportRuntime::new(ViewportOptions::default()).unwrap();
    let mut view = runtime
        .open_window("Oblikovati status acceptance", [256, 128])
        .unwrap();
    view.submit(&scene()).unwrap();
    view.set_status_text("Saved").unwrap();
    let before = verify_native(&mut view);
    let ink = status_ink(&before);
    assert!(!ink.is_empty());
    view.navigate(oblikovati_viewport::ViewportNavigation::Pan([30, 10]))
        .unwrap();
    let moved = complete_surface_frame(&mut view);
    assert_eq!(status_ink(&moved), ink);
    assert_eq!(moved.overlay_generation, before.overlay_generation);
    verify_status_fit_and_clear(&mut view);
    view.close();
    assert!(runtime.validation_errors().is_empty());
}
fn status_ink(capture: &ViewportCapture) -> Vec<[u32; 2]> {
    capture
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .enumerate()
        .filter(|(_, pixel)| **pixel == [255, 255, 255, 255])
        .map(|(index, _)| {
            [
                index as u32 % capture.extent[0],
                index as u32 / capture.extent[0],
            ]
        })
        .collect()
}
fn verify_status_fit_and_clear(view: &mut Viewport) {
    let fit = oblikovati_viewport::ViewportNavigation::Fit { margin: 0.1 };
    view.navigate(fit).unwrap();
    let camera = view.camera_state().unwrap();
    assert!(view.set_status_text("").unwrap());
    let cleared = verify_native(view);
    assert!(status_ink(&cleared).is_empty());
    assert!(!view.navigate(fit).unwrap());
    assert_eq!(view.camera_state().unwrap(), camera);
    view.set_status_text("Changed").unwrap();
    let replaced = verify_native(view);
    assert!(!status_ink(&replaced).is_empty());
    assert!(replaced.overlay_generation > cleared.overlay_generation);
}
