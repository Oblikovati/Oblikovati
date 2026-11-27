// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn line_literals_and_expression_points_use_their_contract_units() {
    for scale in [0.1, 2.0, 100.0] {
        let (mut files, _, mut source) = configured_scaled(scale);
        create_empty_sketch(&mut files, &mut source);
        set_inference(&mut files, &mut source, false, false, "none");
        let input = line_input([[1.0, 2.0], [4.0, 3.0]]);
        assert_line_coordinates(&mut files, &mut source, &input, [[1.0, 2.0], [4.0, 3.0]]);
        let mut expression = input;
        expression.points = Some(vec![None]);
        expression.point_exprs = Some(unit_expression_points());
        assert_line_coordinates(
            &mut files,
            &mut source,
            &expression,
            [[1.0, 2.0], [4.0, 3.0]],
        );
    }
}
fn assert_line_coordinates(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    input: &AddSketchEntityArgs,
    expected: [[f64; 2]; 2],
) {
    let reply = add_call(files, source, input).unwrap();
    assert_eq!(
        row(files, reply.entity_id).points,
        expected_points(&expected)
    );
}
#[test]
fn repaired_expressions_use_current_parameters_without_inventing_live_dimensions() {
    let (mut files, document, mut source) = fresh();
    set_inference(&mut files, &mut source, false, false, "none");
    let mut input = line_input([[0.0; 2], [1.0, 1.0]]);
    input.point_exprs = Some(parameter_expression_points());
    set_width(&mut files, &mut source, "missing");
    assert_refused(
        &mut files,
        document,
        &mut source,
        &input,
        "healthy finite length",
    );
    set_width(&mut files, &mut source, "2 cm");
    let first = add_call(&mut files, &mut source, &input).unwrap();
    let saved = row(&files, first.entity_id);
    set_width(&mut files, &mut source, "3 cm");
    assert_eq!(row(&files, first.entity_id), saved);
    assert_line_coordinates(&mut files, &mut source, &input, [[3.0, 2.0], [12.0, 3.0]]);
}

fn unit_expression_points() -> Vec<Option<Vec<String>>> {
    vec![
        Some(vec!["10 mm".into(), "20 mm".into()]),
        Some(vec!["40 mm".into(), "30 mm".into()]),
    ]
}

fn parameter_expression_points() -> Vec<Option<Vec<String>>> {
    vec![
        Some(vec!["width".into(), "2 cm".into()]),
        Some(vec!["width * 4".into(), "3 cm".into()]),
    ]
}
