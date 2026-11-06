// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchPlane;
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
