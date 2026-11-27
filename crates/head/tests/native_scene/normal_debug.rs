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
                        face_index: None,
                        positions: [[-7., -2., 0.], [-3., -2., 0.], [-5., 2., 0.]],
                        normals: None,
                        color,
                    },
                    ViewportTriangle {
                        face_index: None,
                        positions: [[3., -2., 0.], [5., 2., 0.], [7., -2., 0.]],
                        normals: None,
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

#[test]
fn native_mesh_colors_group_faces_and_restore_the_previous_render_mode() {
    let mut window = NativeHeadWindow::open().unwrap();
    let triangles = [-6., 0., 6.]
        .into_iter()
        .zip([0, 0, 1])
        .map(|(x, face_index)| ViewportTriangle {
            positions: [[x - 1., -2., 0.], [x + 1., -2., 0.], [x, 2., 0.]],
            normals: None,
            color: [0.25, 0.35, 0.8, 1.],
            face_index: Some(face_index),
        })
        .collect();
    window
        .submit_scene(HeadSceneUpdate {
            scene: ViewportScene {
                triangles,
                background: [0.05; 3],
            },
            status: HeadSceneStatus {
                source: None,
                cached_assembly: false,
                visible_bodies: 1,
                health: vec![],
                presentation_declines: vec![],
                scene_errors: vec![],
            },
        })
        .unwrap();
    let shaded = capture_framebuffer(&mut window);
    toggle_normal_debug(&mut window, true);
    let normals = capture_framebuffer(&mut window);

    toggle_mesh_colors(&mut window, true, false);
    let faces = capture_framebuffer(&mut window);
    toggle_mesh_colors(&mut window, true, true);
    let triangles = capture_framebuffer(&mut window);
    let sample = |capture: &ViewportCapture, x| {
        let [column, row] = position_pixel([x, 0., 0.], &window);
        let offset = ((row * capture.extent[0] + column) * 4) as usize;
        <[u8; 4]>::try_from(&capture.rgba[offset..offset + 4]).unwrap()
    };
    let face_color = sample(&faces, -6.);
    assert_eq!(face_color, sample(&faces, 0.));
    assert_ne!(face_color, sample(&faces, 6.));
    assert_ne!(face_color, sample(&normals, -6.));

    assert_eq!(shaded.camera_generation, triangles.camera_generation);
    assert_ne!(sample(&triangles, -6.), sample(&triangles, 0.));
    assert_ne!(sample(&triangles, 0.), sample(&triangles, 6.));

    toggle_mesh_colors(&mut window, false, false);
    assert_eq!(capture_framebuffer(&mut window).rgba, normals.rgba);
    toggle_normal_debug(&mut window, false);
    assert_eq!(capture_framebuffer(&mut window).rgba, shaded.rgba);
    window.finish().unwrap();
}

fn toggle_mesh_colors(window: &mut NativeHeadWindow, on: bool, per_triangle: bool) {
    let request = serde_json::to_vec(&SetMeshColorsArgs { on, per_triangle }).unwrap();
    let reply =
        crate::viewport_automation::apply(window, METHOD_VIEWPORT_SET_MESH_COLORS, &request)
            .unwrap()
            .unwrap();
    let state = serde_json::from_slice::<MeshColorsResult>(&reply).unwrap();
    assert_eq!((state.on, state.per_triangle), (on, per_triangle));
}
