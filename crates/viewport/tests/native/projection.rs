// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchPlane;
use oblikovati_viewport::ViewportNavigation;
#[test]
fn native_sketch_projection_tracks_pixels_camera_history_and_surface_sizes() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = runtime.open_window("Sketch projection", [128, 96]).unwrap();
    view.submit(&scene()).unwrap();
    verify_mapping(&mut view);
    for command in [
        ViewportNavigation::Pan([13, -7]),
        ViewportNavigation::Zoom {
            factor: 0.7,
            anchor: Some([70, 40]),
        },
        ViewportNavigation::Orbit {
            yaw: 0.2,
            pitch: 0.1,
        },
    ] {
        view.navigate(command).unwrap();
        verify_mapping(&mut view);
    }
    view.navigate(ViewportNavigation::Fit { margin: 0.1 })
        .unwrap();
    view.resize([192, 120]).unwrap();
    verify_mapping(&mut view);
    assert!(view.previous_view().unwrap());
    verify_mapping(&mut view);
    verify_rejections(&view);
    view.close();
    assert!(view.sketch_point([1, 1], SketchPlane::xy()).is_err());
    assert!(view.sketch_pixel(SketchPlane::xy(), [0.; 2]).is_err());
    assert!(runtime.validation_errors().is_empty());
}
fn verify_mapping(view: &mut Viewport) {
    let capture = complete_surface_frame(view);
    let plane = SketchPlane::xy();
    let pixel = view.sketch_pixel(plane, [0.; 2]).unwrap();
    let index = (pixel[1] as usize * capture.extent[0] as usize + pixel[0] as usize) * 4;
    assert_eq!(&capture.rgba[index..index + 4], &[255, 0, 0, 255]);
    let [w, h] = capture.extent.map(|x| i32::try_from(x).unwrap());
    for pixel in [
        [w / 2, h / 2],
        [w / 3, h / 3],
        [0, 0],
        [w - 1, h - 1],
        [-10, h + 10],
    ] {
        let local = view.sketch_point(pixel, plane).unwrap().unwrap();
        let restored = view.sketch_pixel(plane, local).unwrap();
        for (a, b) in restored.into_iter().zip(pixel) {
            assert!((a - b).abs() <= 1, "{restored:?} != {pixel:?}");
        }
    }
    assert_eq!(view.capture().unwrap(), capture);
}
fn verify_rejections(view: &Viewport) {
    assert!(view.sketch_point([0, i32::MIN], SketchPlane::xy()).is_err());
    for point in [[f64::NAN, 0.], [f64::MAX, 0.], [1e12, 0.]] {
        assert!(view.sketch_pixel(SketchPlane::xy(), point).is_err());
    }
}
#[test]
fn native_projection_matches_independent_orthographic_coordinates_and_tilted_frames() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = runtime.open_offscreen([200, 100]).unwrap();
    let point = view
        .sketch_point([125, 24], SketchPlane::xy())
        .unwrap()
        .unwrap();
    assert!((point[0] - 5.).abs() < 1e-10);
    assert!((point[1] - 5.).abs() < 1e-10);
    assert_eq!(view.sketch_point([0, 0], SketchPlane::xz()).unwrap(), None);
    let tilted = SketchPlane::new(
        [0., 0., 3.],
        [0.6, 0., 0.8],
        [0., 1., 0.],
        Default::default(),
    )
    .unwrap();
    let point = view.sketch_point([125, 24], tilted).unwrap().unwrap();
    assert!((point[0] - 5. / 0.6).abs() < 1e-10);
    assert!((point[1] - 5.).abs() < 1e-10);
    view.close();
    assert!(runtime.validation_errors().is_empty());
}
