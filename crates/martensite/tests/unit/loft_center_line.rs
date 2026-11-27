// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::sketch_loft::{ResolvedLoftCenterLine, SketchLoftCenterLine, SketchLoftSection},
    sketch::SketchEntityId,
};

#[test]
fn section_centers_accept_either_direction_of_the_same_straight_guide() {
    for endpoints in [[[0., 0., 0.], [0., 0., 3.]], [[0., 0., 3.], [0., 0., 0.]]] {
        require_center_line(&input_with_guide(endpoints), 1e-7).unwrap();
    }
}

#[test]
fn displaced_and_degenerate_lines_refuse_to_drive_the_loft() {
    for endpoints in [[[0., 0., 0.], [0.001, 0., 3.]], [[0., 0., 0.]; 2]] {
        let failure = require_center_line(&input_with_guide(endpoints), 1e-7).unwrap_err();
        assert_eq!(failure.kind(), Kind::InvalidInput);
    }
}

#[test]
fn closed_loop_rejects_the_native_wire_of_an_open_guide() {
    let input = input_with_guide([[0., 0., 0.], [0., 0., 2.]]);
    let failure = check_loft_guide_closure(&input).unwrap_err();
    assert_eq!(failure.kind(), Kind::InvalidInput);
    assert!(failure.to_string().contains("periodic Center Line"));
}

fn input_with_guide(endpoints: [[f64; 3]; 2]) -> ResolvedSketchLoft {
    let section = |sketch, z| ResolvedLoftCircle {
        source: SketchLoftSection {
            sketch: DocumentSketchId(sketch),
            circle: SketchEntityId(sketch),
        },
        center: [0., 0., z],
        normal: [0., 0., 1.],
        x_direction: [1., 0., 0.],
        radius: 1.,
    };
    ResolvedSketchLoft {
        sections: [section(1, 0.), section(2, 3.)],
        output: SketchLoftOutput::Solid,
        center_line: Some(ResolvedLoftCenterLine {
            source: SketchLoftCenterLine {
                sketch: DocumentSketchId(3),
                line: SketchEntityId(10),
            },
            endpoints,
        }),
        placed: None,
    }
}
