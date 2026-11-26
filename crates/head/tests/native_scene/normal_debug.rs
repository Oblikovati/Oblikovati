// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{HeadSceneStatus, HeadSceneUpdate};
use oblikovati_viewport::{ViewportScene, ViewportTriangle};

#[test]
fn native_normal_debug_colors_both_windings_and_restores_shading() {
    let mut window = NativeHeadWindow::open().unwrap();
    let color = [0.25, 0.35, 0.8, 1.];
    window
        .submit_scene(HeadSceneUpdate {
            scene: ViewportScene {
                triangles: vec![
                    ViewportTriangle {
                        positions: [[-7., -2., 0.], [-3., -2., 0.], [-5., 2., 0.]],
                        color,
                    },
                    ViewportTriangle {
                        positions: [[3., -2., 0.], [5., 2., 0.], [7., -2., 0.]],
                        color,
                    },
                ],
                background: [0.05, 0.05, 0.05],
            },
            status: HeadSceneStatus {
                source: None,
                cached_assembly: false,
                visible_bodies: 0,
                health: vec![],
                presentation_declines: vec![],
                scene_errors: vec![],
            },
        })
        .unwrap();
    let before = capture_framebuffer(&mut window);
    toggle_normal_debug(&mut window, true);
    let debug = capture_framebuffer(&mut window);
    assert_eq!(before.camera_generation, debug.camera_generation);
    assert!(debug.scene_generation > before.scene_generation);
    let rgb = |point, capture: &ViewportCapture| {
        let [x, y] = position_pixel(point, &window);
        let i = ((y * capture.extent[0] + x) * 4) as usize;
        [capture.rgba[i], capture.rgba[i + 1], capture.rgba[i + 2]]
    };
    let front = rgb([-5., 0., 0.], &debug);
    let back = rgb([5., 0., 0.], &debug);
    assert!(front[1] > 90 && front[0] < 20 && front[2] < 20, "{front:?}");
    assert!(back[0] > 180 && back[1] < 20 && back[2] < 20, "{back:?}");
    toggle_normal_debug(&mut window, false);
    assert_eq!(capture_framebuffer(&mut window).rgba, before.rgba);
    window.finish().unwrap();
}

fn position_pixel(point: [f64; 3], window: &NativeHeadWindow) -> [u32; 2] {
    window
        .view
        .model_pixel(point)
        .unwrap()
        .map(|coordinate| u32::try_from(coordinate).unwrap())
}

fn toggle_normal_debug(window: &mut NativeHeadWindow, on: bool) {
    let request = serde_json::to_vec(&SetNormalDebugArgs { on }).unwrap();
    let reply =
        crate::viewport_automation::apply(window, METHOD_VIEWPORT_SET_NORMAL_DEBUG, &request)
            .unwrap()
            .unwrap();
    assert_eq!(
        serde_json::from_slice::<NormalDebugResult>(&reply)
            .unwrap()
            .on,
        on
    );
}
