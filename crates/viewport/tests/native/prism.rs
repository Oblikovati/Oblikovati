// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{
    PrismMeshOptions, PrismProbeDefinition, PrismProbeLoop, retain_planar_prism,
};
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
    let snapshot = retain_planar_prism(&holed_profile()).unwrap();
    let analytic = snapshot.report().clone();
    let scene = prism_scene(&snapshot);
    let mut view = runtime
        .open_window("Oblikovati kernel prism presentation", [320, 240])
        .unwrap();
    view.set_camera(prism_camera()).unwrap();
    view.submit(&scene).unwrap();
    let capture = complete_surface_frame(&mut view);
    await_presented_pixels(&view, &capture);
    verify_hole_pixels(&capture);
    assert_eq!(snapshot.report(), &analytic);
    view.close();
}
fn prism_camera() -> ViewportCamera {
    ViewportCamera {
        eye: [2., 1.5, 20.],
        target: [2., 1.5, 1.],
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
