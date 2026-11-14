// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn native_light_distinguishes_faces_with_the_same_material() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = runtime.open_offscreen([320, 240]).unwrap();
    let color = [0.7, 0.6, 0.5, 1.];
    let scene = ViewportScene {
        background: [0., 0., 0.],
        triangles: vec![
            ViewportTriangle {
                positions: [[-8., -4., 0.], [-2., -4., 0.], [-5., 5., 0.]],
                color,
            },
            ViewportTriangle {
                positions: [[2., -4., -3.], [8., -4., 3.], [5., 5., 0.]],
                color,
            },
        ],
    };
    view.submit(&scene).unwrap();
    let capture = view.redraw().unwrap();
    let left = face_pixel(&view, &capture, [-5., -1., 0.]);
    let right = face_pixel(&view, &capture, [5., -1., 0.]);
    assert_ne!(left, [0, 0, 0, 255], "left face must be visible");
    assert_ne!(right, [0, 0, 0, 255], "right face must be visible");
    assert_ne!(left, right, "different face normals must change lighting");
    view.close();
    assert!(runtime.validation_errors().unwrap().is_empty());
}

fn face_pixel(view: &Viewport, capture: &ViewportCapture, point: [f64; 3]) -> [u8; 4] {
    let [x, y] = view.model_pixel(point).unwrap();
    let offset = (y as usize * capture.extent[0] as usize + x as usize) * 4;
    capture.rgba[offset..offset + 4].try_into().unwrap()
}

#[test]
fn lighting_retains_each_faces_material_color() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = runtime.open_offscreen([320, 240]).unwrap();
    let scene = ViewportScene {
        background: [0.; 3],
        triangles: vec![
            ViewportTriangle {
                positions: [[-8., -4., 0.], [-2., -4., 0.], [-5., 5., 0.]],
                color: [1., 0., 0., 1.],
            },
            ViewportTriangle {
                positions: [[2., -4., 0.], [8., -4., 0.], [5., 5., 0.]],
                color: [0., 1., 0., 1.],
            },
        ],
    };
    view.submit(&scene).unwrap();
    let capture = view.redraw().unwrap();
    let red = face_pixel(&view, &capture, [-5., -1., 0.]);
    let green = face_pixel(&view, &capture, [5., -1., 0.]);
    assert!(red[0] > 0 && red[1..] == [0, 0, 255], "{red:?}");
    assert!(
        green[1] > 0 && green[0] == 0 && green[2..] == [0, 255],
        "{green:?}"
    );
    view.close();
    assert!(runtime.validation_errors().unwrap().is_empty());
}

#[test]
fn isometric_cube_has_three_visible_face_tones() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = runtime.open_offscreen([320, 240]).unwrap();
    let mut triangles = Vec::new();
    let color = [0.7, 0.7, 0.7, 1.];
    for quad in [
        [[5., -5., -5.], [5., 5., -5.], [5., 5., 5.], [5., -5., 5.]],
        [[-5., 5., -5.], [-5., 5., 5.], [5., 5., 5.], [5., 5., -5.]],
        [[-5., -5., 5.], [5., -5., 5.], [5., 5., 5.], [-5., 5., 5.]],
    ] {
        for positions in [[quad[0], quad[1], quad[2]], [quad[0], quad[2], quad[3]]] {
            triangles.push(ViewportTriangle { positions, color });
        }
    }
    view.submit(&ViewportScene {
        background: [0., 0., 0.],
        triangles,
    })
    .unwrap();
    view.set_camera(ViewportCamera {
        eye: [20.; 3],
        up: [0., 0., 1.],
        ..Default::default()
    })
    .unwrap();
    let capture = view.redraw().unwrap();
    let faces = [[5., -1., 1.], [-1., 5., 1.], [1., -1., 5.]]
        .map(|point| face_pixel(&view, &capture, point));
    assert!(
        faces.iter().all(|face| face[0] > 0),
        "faces {faces:?} must be visible"
    );
    assert!(
        faces[0] != faces[1] && faces[1] != faces[2] && faces[0] != faces[2],
        "faces {faces:?} must be distinct"
    );
    view.close();
    assert!(runtime.validation_errors().unwrap().is_empty());
}
