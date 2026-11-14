// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn shared_equations_cover_both_orientations_and_representable_scales() {
    for scale in [1e-6, 1.0, 1e6] {
        for sign in [-1.0, 1.0] {
            for vertical in [false, true] {
                let start = [2.0 * scale, -3.0 * scale];
                let offset = if vertical {
                    [scale, 4.0 * scale]
                } else {
                    [4.0 * scale, scale]
                };
                let end = [start[0] + sign * offset[0], start[1] + sign * offset[1]];
                assert_shared_solution(line_geometry(start, end), vertical);
            }
        }
    }
}
#[test]
fn parameter_driven_line_length_with_horizontal_alignment_is_fully_constrained() {
    let mut parameters = parameters();
    parameters
        .insert_expression(1, "length", oblikovati_api::types::USER_PARAM, "4 cm")
        .unwrap();
    let mut geometry = line_geometry([0.0; 2], [4.0, 0.0]);
    append(&mut geometry, 1, line_relation(false));
    fix_start(&mut geometry, 2);
    append_length_dimension(&mut geometry);
    let solved = geometry
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(solved.numerical().structure.degrees_of_freedom, 0);
    assert_eq!(solved.numerical().termination, SolveTermination::Converged);
    assert_eq!(solved.candidate().parameter_dependencies(), [1]);
    assert_eq!(geometry.geometric_constraints().count(), 2);
}
fn append_length_dimension(sketch: &mut Sketch2D) {
    append(
        sketch,
        3,
        PlanarConstraint::Distance {
            points: [entity(1), entity(2)],
            measure: DistanceMeasure::Horizontal,
            parameter: 1,
        },
    );
}
