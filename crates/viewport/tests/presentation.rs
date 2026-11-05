// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{PrismFaceMesh, PrismFaceRole};
fn fixture(coordinate: f64) -> PrismPresentation {
    PrismPresentation {
        faces: vec![PrismFaceMesh {
            role: PrismFaceRole::End,
            triangles: vec![[[coordinate, 1., 2.], [3., 4., 5.], [6., 7., 8.]]],
        }],
    }
}
#[test]
fn conversion_preserves_face_triangle_order_and_coordinates() {
    let mesh = fixture(0.5);
    let scene = ViewportScene::from_prism(&mesh, [0., 0., 1.], [0., 1., 0., 1.]).unwrap();
    assert_eq!(scene.triangles.len(), 1);
    assert_eq!(
        scene.triangles[0].positions,
        [[0.5, 1., 2.], [3., 4., 5.], [6., 7., 8.]]
    );
    assert_eq!(scene.triangles[0].color, [0., 1., 0., 1.]);
    assert_eq!(scene.background, [0., 0., 1.]);
}
#[test]
fn invalid_coordinates_or_display_channels_are_declined() {
    for coordinate in [f64::MAX, f64::NAN, f64::INFINITY] {
        assert!(
            ViewportScene::from_prism(&fixture(coordinate), [0.; 3], [1.; 4])
                .unwrap_err()
                .contains("representable as finite display floats")
        );
    }
    assert!(ViewportScene::from_prism(&fixture(1.), [-1.; 3], [1.; 4]).is_err());
    assert!(ViewportScene::from_prism(&fixture(1.), [0.; 3], [2.; 4]).is_err());
}
