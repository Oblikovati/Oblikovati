// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn scene() -> ViewportScene {
    ViewportScene {
        background: [0., 0., 1.],
        triangles: vec![ViewportTriangle {
            face_index: None,
            positions: [[1., 0., 0.], [0., 1., 0.], [0., 0., 1.]],
            normals: Some([[1., 0., 0.], [0., 1., 0.], [0., 0., 1.]]),
            color: [0., 1., 0., 1.],
        }],
    }
}

#[test]
fn places_a_copy_with_row_major_rotation_and_translation() {
    let source = scene();
    let occurrence = oblikovati_model::assembly::AssemblyOccurrenceRecipe {
        transform: vec![
            0., -1., 0., 10., 1., 0., 0., 20., 0., 0., 1., 30., 0., 0., 0., 1.,
        ],
        ..Default::default()
    };
    let placed = source.placed(&occurrence.placement().unwrap()).unwrap();
    assert_eq!(
        placed.triangles[0].positions,
        [[10., 21., 30.], [9., 20., 30.], [10., 20., 31.]]
    );
    assert_eq!(source.triangles[0].positions[0], [1., 0., 0.]);
    assert_eq!(placed.triangles[0].color, source.triangles[0].color);
    let normals = placed.triangles[0].normals.unwrap();
    assert!((normals[0][1] - 1.).abs() < 1e-5, "{normals:?}");
    assert!((normals[1][0] + 1.).abs() < 1e-5, "{normals:?}");
    assert!((normals[2][2] - 1.).abs() < 1e-5, "{normals:?}");
}

#[test]
fn refuses_projective_and_overflowing_placements() {
    let mut cells = [
        1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1., 0., 1., 0., 0., 1.,
    ];
    assert!(
        scene()
            .placed(&Matrix { cells })
            .unwrap_err()
            .contains("affine")
    );
    cells[12] = 0.;
    cells[3] = f64::MAX;
    assert!(
        scene()
            .placed(&Matrix { cells })
            .unwrap_err()
            .contains("finite f32")
    );
    cells[3] = f64::NAN;
    assert!(
        scene()
            .placed(&Matrix { cells })
            .unwrap_err()
            .contains("affine")
    );
}
