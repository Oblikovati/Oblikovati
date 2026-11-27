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
                face_index: None,
                positions: [[-8., -4., 0.], [-2., -4., 0.], [-5., 5., 0.]],
                normals: None,
                color,
            },
            ViewportTriangle {
                face_index: None,
                positions: [[2., -4., -3.], [8., -4., 3.], [5., 5., 0.]],
                normals: None,
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
                face_index: None,
                positions: [[-8., -4., 0.], [-2., -4., 0.], [-5., 5., 0.]],
                normals: None,
                color: [1., 0., 0., 1.],
            },
            ViewportTriangle {
                face_index: None,
                positions: [[2., -4., 0.], [8., -4., 0.], [5., 5., 0.]],
                normals: None,
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
    println!("viewport lighting device: {}", runtime.device_name());
    let mut view = runtime.open_offscreen([320, 240]).unwrap();
    let mut triangles = Vec::new();
    let color = [0.7, 0.7, 0.7, 1.];
    for quad in [
        [[5., -5., -5.], [5., 5., -5.], [5., 5., 5.], [5., -5., 5.]],
        [[-5., 5., -5.], [-5., 5., 5.], [5., 5., 5.], [5., 5., -5.]],
        [[-5., -5., 5.], [5., -5., 5.], [5., 5., 5.], [-5., 5., 5.]],
    ] {
        for positions in [[quad[0], quad[1], quad[2]], [quad[0], quad[2], quad[3]]] {
            triangles.push(ViewportTriangle {
                positions,
                normals: None,
                color,
                face_index: None,
            });
        }
    }
    let mut scene = ViewportScene {
        background: [0., 0., 0.],
        triangles,
    };
    view.submit(&scene).unwrap();
    view.set_camera(ViewportCamera {
        eye: [20.; 3],
        up: [0., 0., 1.],
        ..Default::default()
    })
    .unwrap();
    let capture = view.redraw().unwrap();
    let faces = [[5., -1., 1.], [-1., 5., 1.], [1., -1., 5.]]
        .map(|point| face_pixel(&view, &capture, point));
    // Go's shaded-mode ambient term for neutral 0.7 is about 90/255 (#4044).
    // Allow ten levels for backend and color-conversion differences.
    assert!(
        faces.iter().all(|face| face[0] >= 80),
        "dark face below Go ambient floor: {faces:?}"
    );
    assert!(
        faces.iter().all(|face| face[0] > 0),
        "faces {faces:?} must be visible"
    );
    assert!(
        faces[0] != faces[1] && faces[1] != faces[2] && faces[0] != faces[2],
        "faces {faces:?} must be distinct"
    );
    for channel in 0..3 {
        let mut color = [0., 0., 0., 1.];
        color[channel] = 0.7;
        for triangle in &mut scene.triangles {
            triangle.color = color;
        }
        view.submit(&scene).unwrap();
        let capture = view.redraw().unwrap();
        let dark = face_pixel(&view, &capture, [5., -1., 1.]);
        assert!(dark[channel] >= 80, "channel {channel}: {dark:?}");
        assert!(
            (0..3)
                .filter(|other| *other != channel)
                .all(|other| dark[other] == 0),
            "channel {channel}: {dark:?}"
        );
    }
    for triangle in &mut scene.triangles {
        triangle.color = [0.05, 0., 0., 1.];
    }
    view.submit(&scene).unwrap();
    let capture = view.redraw().unwrap();
    let dim = face_pixel(&view, &capture, [5., -1., 1.]);
    assert!(
        dim[0] > 0 && dim[0] < 80 && dim[1] == 0 && dim[2] == 0,
        "dim color {dim:?}"
    );
    view.close();
    assert!(runtime.validation_errors().unwrap().is_empty());
}

#[test]
fn reversed_visible_face_keeps_ambient_color() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = runtime.open_offscreen([320, 240]).unwrap();
    let triangle = ViewportTriangle {
        face_index: None,
        positions: [[-8., -4., 0.], [0., 8., 0.], [8., -4., 0.]],
        normals: None,
        color: [0.7, 0., 0., 1.],
    };
    view.submit(&ViewportScene {
        background: [0., 0., 0.],
        triangles: vec![triangle],
    })
    .unwrap();
    let capture = view.redraw().unwrap();
    let color = face_pixel(&view, &capture, [0., 0., 0.]);
    assert!(
        color[0] >= 80 && color[1] == 0 && color[2] == 0,
        "back face {color:?}"
    );
    view.close();
    assert!(runtime.validation_errors().unwrap().is_empty());
}

#[test]
fn per_vertex_normals_produce_smooth_phong_gradient_across_one_triangle() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = runtime.open_offscreen([320, 240]).unwrap();
    let tilt = 0.5_f32.sqrt();
    let triangle = ViewportTriangle {
        face_index: Some(0),
        positions: [[-8., -4., 0.], [8., -4., 0.], [0., 6., 0.]],
        normals: Some([[-tilt, 0., tilt], [tilt, 0., tilt], [0., 0., 1.]]),
        color: [0.7, 0.7, 0.7, 1.],
    };
    view.submit(&ViewportScene {
        background: [0., 0., 0.],
        triangles: vec![triangle],
    })
    .unwrap();
    let capture = view.redraw().unwrap();
    let left = face_pixel(&view, &capture, [-4., -2., 0.]);
    let center = face_pixel(&view, &capture, [0., -2., 0.]);
    let right = face_pixel(&view, &capture, [4., -2., 0.]);
    assert!(left[0] > 0 && center[0] > 0 && right[0] > 0);
    assert!(
        left != center || center != right,
        "varying vertex normals must shade smoothly across one triangle: {left:?} {center:?} {right:?}"
    );
    view.close();
    assert!(runtime.validation_errors().unwrap().is_empty());
}
