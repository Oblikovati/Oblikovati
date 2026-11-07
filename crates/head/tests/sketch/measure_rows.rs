// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn segments() -> (Sketch2D, [SketchEntityId; 2]) {
    let mut sketch = Sketch2D::default();
    let points = [10, 11, 20, 21].map(SketchEntityId);
    for (id, at) in points.iter().zip([[0., 0.], [1., 0.], [0., 1.], [0., 2.]]) {
        sketch.add_point(*id, at).unwrap();
    }
    let lines = [SketchEntityId(1), SketchEntityId(2)];
    sketch.add_line(lines[0], [points[0], points[1]]).unwrap();
    sketch.add_line(lines[1], [points[2], points[3]]).unwrap();
    (sketch, lines)
}
fn rows(sketch: &Sketch2D, picks: &[MeasurePick]) -> Vec<String> {
    let units = UnitsOfMeasure::default();
    let context = MeasureContext {
        sketch,
        plane: SketchPlane::xy(),
        units: &units,
    };
    measure_rows(&context, picks).unwrap()
}
#[test]
fn one_curve_reads_as_a_selection_and_its_length() {
    let (sketch, [first, _]) = segments();
    assert_eq!(
        rows(&sketch, &[MeasurePick::Curve(first)]),
        ["Selection 1: Curve", "Length 10.000 mm"]
    );
}
#[test]
fn two_lines_add_their_angle_distances_and_delta() {
    let (sketch, [first, second]) = segments();
    let rows = rows(
        &sketch,
        &[MeasurePick::Curve(first), MeasurePick::Curve(second)],
    );
    assert!(rows.contains(&"Angle 90.00 deg".to_owned()), "{rows:?}");
    assert!(
        rows.contains(&"Minimum distance 10.000 mm".to_owned()),
        "{rows:?}"
    );
    assert!(
        rows.iter()
            .any(|row| row.starts_with("Maximum distance 22.361")),
        "{rows:?}"
    );
    assert!(
        rows.contains(&"Minimum delta X 0.000 Y 10.000 Z 0.000 mm".to_owned()),
        "{rows:?}"
    );
}
#[test]
fn a_point_reads_its_model_position_and_never_a_negative_zero() {
    let (sketch, _) = segments();
    let rows = rows(&sketch, &[MeasurePick::Point([-0.5, -0.])]);
    assert_eq!(rows[1], "Position X -5.000 Y 0.000 Z 0.000 mm");
}
#[test]
fn a_third_selection_starts_over_and_a_repeat_changes_nothing() {
    let (_, [first, second]) = segments();
    let mut state = SketchMeasureState {
        edit: crate::HeadSketchEdit {
            document: 1,
            sketch: oblikovati_model::document_sketch::DocumentSketchId(1),
            plane: SketchPlane::xy(),
        },
        generation: 0,
        picks: Vec::new(),
        rows: Vec::new(),
    };
    state.add_pick(MeasurePick::Curve(first));
    state.add_pick(MeasurePick::Curve(first));
    assert_eq!(state.picks.len(), 1);
    state.add_pick(MeasurePick::Curve(second));
    state.add_pick(MeasurePick::Point([0., 0.]));
    assert_eq!(state.picks, [MeasurePick::Point([0., 0.])]);
}

fn ellipse_for_measurement() -> (Sketch2D, SketchEntityId) {
    use oblikovati_model::sketch::SketchEllipse2D;
    let mut sketch = Sketch2D::default();
    let center = SketchEntityId(10);
    let curve = SketchEntityId(1);
    sketch.add_point(center, [0.2, -0.3]).unwrap();
    sketch
        .add_ellipse(SketchEllipse2D::new(curve, center, [0.6, 0.8], [0.8, 0.4]).unwrap())
        .unwrap();
    (sketch, curve)
}
#[test]
fn a_rotated_ellipse_reads_both_axis_sizes_and_its_exact_center() {
    let (sketch, ellipse) = ellipse_for_measurement();
    assert_eq!(
        rows(&sketch, &[MeasurePick::Curve(ellipse)]),
        [
            "Selection 1: Ellipse",
            "Major radius 8.000 mm",
            "Minor radius 4.000 mm",
            "Major diameter 16.000 mm",
            "Minor diameter 8.000 mm",
            "Length 38.754 mm",
            "Center X 2.000 Y -3.000 Z 0.000 mm",
        ]
    );
}

#[test]
fn circles_keep_their_existing_numeric_readout_and_full_turn_label() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(10), [0., 0.]).unwrap();
    sketch
        .add_circle(SketchEntityId(1), SketchEntityId(10), 0.5)
        .unwrap();
    assert_eq!(
        rows(&sketch, &[MeasurePick::Curve(SketchEntityId(1))]),
        [
            "Selection 1: Curve",
            "Radius 5.000 mm",
            "Diameter 10.000 mm",
            "Length 31.416 mm",
            "Angle 360.00 deg",
            "Center X 0.000 Y 0.000 Z 0.000 mm",
        ]
    );
}
#[test]
fn circular_arcs_keep_their_existing_numeric_readout_and_sweep_label() {
    let mut sketch = Sketch2D::default();
    for (id, point) in [(10, [0., 0.]), (11, [0.5, 0.]), (12, [0., 0.5])] {
        sketch.add_point(SketchEntityId(id), point).unwrap();
    }
    sketch
        .add_arc(
            SketchEntityId(1),
            SketchEntityId(10),
            [SketchEntityId(11), SketchEntityId(12)],
            true,
        )
        .unwrap();
    assert_eq!(
        rows(&sketch, &[MeasurePick::Curve(SketchEntityId(1))]),
        [
            "Selection 1: Curve",
            "Radius 5.000 mm",
            "Diameter 10.000 mm",
            "Length 7.854 mm",
            "Sweep 90.00 deg",
            "Center X 0.000 Y 0.000 Z 0.000 mm",
        ]
    );
}
