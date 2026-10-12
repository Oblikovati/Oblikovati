// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn camera_vectors_and_projection_decline_invalid_inputs() {
    let changes: [fn(&mut ViewportCamera); 8] = [
        |camera| camera.eye = [f64::NAN, 0., 0.],
        |camera| camera.eye = [0.; 3],
        |camera| camera.up = [0.; 3],
        |camera| camera.eye = [f64::MAX; 3],
        |camera| camera.up = [f64::MAX; 3],
        |camera| camera.scale = f64::INFINITY,
        |camera| camera.near = 0.,
        |camera| camera.far = 1.,
    ];
    for change in changes {
        let mut camera = ViewportCamera::default();
        change(&mut camera);
        assert!(camera.validate().unwrap_err().contains("expected"));
    }
    assert!(ViewportCamera::default().validate().is_ok());
}
#[test]
fn presentation_coordinates_and_channels_decline_nonfinite_and_out_of_range_values() {
    let mut scene = ViewportScene {
        background: [0.; 3],
        triangles: vec![ViewportTriangle {
            positions: [[0.; 3]; 3],
            color: [1.; 4],
        }],
    };
    assert!(scene.validate().is_ok());
    scene.background[1] = -1.;
    assert!(scene.validate().unwrap_err().contains("background"));
    scene.background[1] = 0.;
    scene.triangles[0].positions[0][2] = f32::INFINITY;
    assert!(scene.validate().unwrap_err().contains("positions"));
    scene.triangles[0].positions[0][2] = 0.;
    scene.triangles[0].color[3] = 2.;
    assert!(scene.validate().unwrap_err().contains("color"));
}
