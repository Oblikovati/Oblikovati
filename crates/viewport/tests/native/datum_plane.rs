// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::theme::Rgba;
use oblikovati_viewport::ViewportDatumPlane;

fn plane(alpha: f32, z: f64) -> ViewportDatumPlane {
    ViewportDatumPlane {
        corners: [[-5., -5., z], [5., -5., z], [5., 5., z], [-5., 5., z]],
        fill: Rgba {
            r: 0.,
            g: 1.,
            b: 0.,
            a: alpha,
        },
        border: Rgba {
            r: 1.,
            g: 1.,
            b: 0.,
            a: 1.,
        },
    }
}

fn pixel(capture: &ViewportCapture, x: u32, y: u32) -> [u8; 4] {
    let index = ((y * capture.extent[0] + x) * 4) as usize;
    capture.rgba[index..index + 4].try_into().unwrap()
}

#[test]
fn datum_fill_blends_and_removes_without_changing_the_solid_scene() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = runtime.open_offscreen([320, 240]).unwrap();
    view.submit(&ViewportScene {
        background: [0., 0., 1.],
        triangles: Vec::new(),
    })
    .unwrap();
    let initial = pixel(&view.redraw().unwrap(), 160, 120);
    assert!(view.set_datum_planes(&[plane(0.5, 0.)]).unwrap());
    let blended = pixel(&view.redraw().unwrap(), 160, 120);
    assert!(
        blended[1] > 0 && blended[2] > 0,
        "expected green fill over blue: {blended:?}"
    );
    assert!(!view.set_datum_planes(&[plane(0.5, 0.)]).unwrap());
    assert!(view.set_datum_planes(&[]).unwrap());
    assert_eq!(pixel(&view.redraw().unwrap(), 160, 120), initial);
    view.close();
    assert!(runtime.validation_errors().unwrap().is_empty());
}

#[test]
fn solid_in_front_of_datum_occludes_its_fill() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = runtime.open_offscreen([320, 240]).unwrap();
    let scene = ViewportScene {
        background: [0.; 3],
        triangles: vec![ViewportTriangle {
            face_index: None,
            positions: [[-8., -8., 1.], [8., -8., 1.], [0., 8., 1.]],
            normals: None,
            color: [1., 0., 0., 1.],
        }],
    };
    view.submit(&scene).unwrap();
    let solid = pixel(&view.redraw().unwrap(), 160, 120);
    view.set_datum_planes(&[plane(0.5, 0.)]).unwrap();
    assert_eq!(pixel(&view.redraw().unwrap(), 160, 120), solid);
    view.close();
    assert!(runtime.validation_errors().unwrap().is_empty());
}

#[test]
fn datum_in_front_blends_with_the_solid_behind_it() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = runtime.open_offscreen([320, 240]).unwrap();
    let scene = ViewportScene {
        background: [0.; 3],
        triangles: vec![ViewportTriangle {
            face_index: None,
            positions: [[-8., -8., 0.], [8., -8., 0.], [0., 8., 0.]],
            normals: None,
            color: [1., 0., 0., 1.],
        }],
    };
    view.submit(&scene).unwrap();
    let solid = pixel(&view.redraw().unwrap(), 160, 120);
    view.set_datum_planes(&[plane(0.5, 1.)]).unwrap();
    let mixed = pixel(&view.redraw().unwrap(), 160, 120);
    assert!(
        mixed[0] > 0 && mixed[1] > 0 && mixed != solid,
        "expected blended solid and plane: {mixed:?}"
    );
    view.close();
    assert!(runtime.validation_errors().unwrap().is_empty());
}
