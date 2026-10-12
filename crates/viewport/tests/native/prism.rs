// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{
    PrismMeshOptions, PrismProbeDefinition, PrismProbeLoop, retain_prism_on_plane,
};
use oblikovati_model::sketch::SketchPlane;
fn holed_profile() -> PrismProbeDefinition {
    PrismProbeDefinition {
        outer: PrismProbeLoop {
            edge_keys: vec![1, 2, 3, 4],
            vertices: vec![[0., 0.], [4., 0.], [4., 3.], [0., 3.]],
        },
        holes: vec![PrismProbeLoop {
            edge_keys: vec![11, 12, 13, 14],
            vertices: vec![[1., 1.], [1., 2.], [3., 2.], [3., 1.]],
        }],
        displacement: [0., 0., 2.],
    }
}
pub(super) fn verify_meshed_prism(runtime: &ViewportRuntime) {
    let tilted = tilted_frame();
    let mut view = runtime
        .open_window("Oblikovati placed kernel prism presentation", [320, 240])
        .unwrap();
    for plane in [
        SketchPlane::xy(),
        SketchPlane::xz(),
        SketchPlane::yz(),
        tilted,
        tilted.flipped_normal().unwrap(),
    ] {
        verify_placed_frame(&mut view, plane);
    }
    view.close();
}
fn verify_placed_frame(view: &mut Viewport, plane: SketchPlane) {
    let snapshot = retain_prism_on_plane(&holed_profile(), plane).unwrap();
    let analytic = snapshot.report().clone();
    view.set_camera(prism_camera(plane)).unwrap();
    view.submit(&prism_scene(&snapshot)).unwrap();
    let capture = complete_surface_frame(view);
    await_presented_pixels(view, &capture);
    verify_hole_pixels(&capture);
    assert_eq!(snapshot.report(), &analytic);
}
fn prism_camera(plane: SketchPlane) -> ViewportCamera {
    let center = plane.to_model([2., 1.5]).unwrap();
    let normal = plane.normal();
    ViewportCamera {
        eye: std::array::from_fn(|i| center[i] + normal[i] * 20.),
        target: std::array::from_fn(|i| center[i] + normal[i]),
        up: plane.recipe().y_axis,
        scale: 6.,
        ..ViewportCamera::default()
    }
}
fn verify_hole_pixels(capture: &ViewportCapture) {
    let [width, height] = capture.extent;
    assert_eq!(sample(capture, width / 2, height / 2), [0, 0, 255, 255]);
    assert_eq!(
        sample(capture, width * 3 / 10, height / 2),
        [0, 255, 0, 255]
    );
    assert_eq!(sample(capture, 0, 0), [0, 0, 255, 255]);
}
fn sample(capture: &ViewportCapture, x: u32, y: u32) -> [u8; 4] {
    let offset = ((y * capture.extent[0] + x) * 4) as usize;
    capture.rgba[offset..offset + 4].try_into().unwrap()
}

fn prism_scene(snapshot: &oblikovati_martensite::PrismProbeSnapshot) -> ViewportScene {
    let mesh = snapshot
        .presentation(PrismMeshOptions {
            chord_error: 0.05,
            angular_error: 0.5,
        })
        .unwrap();
    ViewportScene::from_prism(&mesh, [0., 0., 1.], [0., 1., 0., 1.]).unwrap()
}

fn tilted_frame() -> SketchPlane {
    SketchPlane::new(
        [10., -20., 30.],
        [0.6, 0.8, 0.],
        [0., 0., 1.],
        Default::default(),
    )
    .unwrap()
}
