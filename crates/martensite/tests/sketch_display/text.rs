// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{
    SketchTextFraction, SketchTextLineSpacing, SketchTextRun, SketchTextStyle, TEXT_ALIGN_UPPER,
    TEXT_LINE_SPACING_DOUBLE, TEXT_LINE_SPACING_EXACTLY,
};
use oblikovati_martensite::planar_curve::sketch::{pick_sketch_text, sample_sketch_text};
use oblikovati_model::sketch::SketchText2D;

#[test]
fn glyph_edges_share_the_same_placement_for_draft_and_committed_text() {
    let style = SketchTextStyle {
        content: "O8".into(),
        family: "DejaVu Sans".into(),
        height: 0.8,
        rotation: std::f64::consts::FRAC_PI_2,
        v_align: TEXT_ALIGN_UPPER,
        ..Default::default()
    };
    let metrics = SketchSamplingOptionsV1 {
        working_scale: 0.1,
        ..options()
    };
    let anchor = [7., -2.];
    let draft = sample_sketch_text(&style, anchor, metrics).unwrap();
    assert!(draft.len() >= 3);
    assert!(draft.iter().all(|edge| edge.len() >= 2));
    let sketch = authored_text(style, anchor);
    let report = sample_sketch_display(&sketch, metrics).unwrap();
    assert!(report.declines.is_empty());
    assert_eq!(report.entities.len(), 1);
    assert_eq!(
        report.entities[0].primitive,
        SketchDisplayPrimitive::GlyphEdges(draft)
    );
}

#[test]
fn glyph_placement_preserves_working_scale_and_rotation() {
    let mut style = SketchTextStyle {
        content: "O".into(),
        family: "DejaVu Sans".into(),
        height: 0.8,
        ..Default::default()
    };
    let baseline = sample_sketch_text(&style, [0.; 2], options()).unwrap();
    let anchor = [2., 3.];
    style.rotation = std::f64::consts::FRAC_PI_2;
    let sampled = SketchSamplingOptionsV1 {
        working_scale: 0.5,
        deflection: options().deflection * 2.,
        ..options()
    };
    let rotated = sample_sketch_text(&style, anchor, sampled).unwrap();
    assert_rotation(&baseline, &rotated, anchor);
}

#[test]
fn line_spacing_changes_baselines_without_scaling_the_glyphs() {
    let metrics = SketchSamplingOptionsV1 {
        working_scale: 0.1,
        ..options()
    };
    let mut style = SketchTextStyle {
        content: "O".into(),
        family: "DejaVu Sans".into(),
        height: 0.8,
        ..Default::default()
    };
    let measure = |style: &SketchTextStyle| {
        outline_height(&sample_sketch_text(style, [0.; 2], metrics).unwrap())
    };
    let glyph_height = measure(&style);
    style.content = "O\n\nO".into();
    let natural_height = measure(&style);
    style.line_spacing = Some(Box::new(SketchTextLineSpacing {
        mode: TEXT_LINE_SPACING_DOUBLE,
        value: 0.,
    }));
    assert!((measure(&style) - glyph_height - 2. * (natural_height - glyph_height)).abs() < 1e-6);
    style.line_spacing = Some(Box::new(SketchTextLineSpacing {
        mode: TEXT_LINE_SPACING_EXACTLY,
        value: 0.7,
    }));
    assert!((measure(&style) - glyph_height - 14.).abs() < 1e-6);
}

fn outline_height(paths: &[Vec<[f64; 2]>]) -> f64 {
    let [min, max] = paths
        .iter()
        .flatten()
        .fold([f64::INFINITY, f64::NEG_INFINITY], |range, point| {
            [range[0].min(point[1]), range[1].max(point[1])]
        });
    max - min
}

fn assert_rotation(before: &[Vec<[f64; 2]>], after: &[Vec<[f64; 2]>], anchor: [f64; 2]) {
    assert_eq!(before.len(), after.len());
    for (first, second) in before.iter().zip(after) {
        assert_eq!(first.len(), second.len());
        for (&[x, y], &point) in first.iter().zip(second) {
            let expected = [anchor[0] - 2. * y, anchor[1] + 2. * x];
            assert!((point[0] - expected[0]).hypot(point[1] - expected[1]) < 1e-6);
        }
    }
}

#[test]
fn invalid_or_empty_text_does_not_yield_fake_outline_geometry() {
    let mut style = SketchTextStyle {
        height: 0.3,
        content: " ".into(),
        ..Default::default()
    };
    assert!(
        sample_sketch_text(&style, [0.; 2], options())
            .unwrap()
            .is_empty()
    );
    style.content = "O".into();
    style.family = "no-oblikovati-font-with-this-name".into();
    assert!(
        sample_sketch_text(&style, [0.; 2], options())
            .unwrap_err()
            .contains("font")
    );
}

#[test]
fn width_scales_the_native_outline_horizontally_and_picking_uses_the_same_shape() {
    let mut style = SketchTextStyle {
        height: 0.8,
        content: "O".into(),
        family: "DejaVu Sans".into(),
        ..Default::default()
    };
    let options = options();
    let before = sample_sketch_text(&style, [0.; 2], options).unwrap();
    style.formatting_runs = Some(vec![SketchTextRun {
        end: 1,
        width_percent: Some(Box::new(50.)),
        ..Default::default()
    }]);
    let after = sample_sketch_text(&style, [0.; 2], options).unwrap();
    assert_scaled_width(&before, &after, options.deflection);
    let point = after[0][0];
    let picked = pick_sketch_text(&style, [0.; 2], options.working_scale, point)
        .unwrap()
        .unwrap();
    assert!((picked[0] - point[0]).hypot(picked[1] - point[1]) <= options.deflection);
}

fn assert_scaled_width(before: &[Vec<[f64; 2]>], after: &[Vec<[f64; 2]>], tolerance: f64) {
    let original = glyph_extents(before);
    let scaled = glyph_extents(after);
    assert!((scaled[0] - original[0] * 0.5).abs() <= 2. * tolerance);
    assert!((scaled[1] - original[1]).abs() <= 2. * tolerance);
}

fn glyph_extents(paths: &[Vec<[f64; 2]>]) -> [f64; 2] {
    std::array::from_fn(|axis| {
        let (low, high) = paths
            .iter()
            .flatten()
            .fold((f64::INFINITY, f64::NEG_INFINITY), |(low, high), point| {
                (low.min(point[axis]), high.max(point[axis]))
            });
        high - low
    })
}

fn authored_text(style: SketchTextStyle, anchor: [f64; 2]) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    sketch.add_defining_point(Id(1), anchor).unwrap();
    sketch
        .add_text(SketchText2D {
            id: Id(2),
            anchor: Id(1),
            style,
        })
        .unwrap();
    sketch
}

#[test]
fn native_fraction_outlines_include_a_separator_between_the_two_terms() {
    let metrics = SketchSamplingOptionsV1 {
        working_scale: 0.1,
        ..options()
    };
    let mut style = SketchTextStyle {
        content: "1/2".into(),
        family: "DejaVu Sans".into(),
        height: 0.8,
        ..Default::default()
    };
    let inline = sample_sketch_text(&style, [0.; 2], metrics).unwrap();
    style.stacked_fractions = Some(vec![SketchTextFraction { start: 0, end: 3 }]);
    let fraction = sample_sketch_text(&style, [0.; 2], metrics).unwrap();
    let height = |paths: &[Vec<[f64; 2]>]| outline_height(paths);
    assert!(height(&fraction) > height(&inline));
    let [a, b] = widest_horizontal_edge(&fraction).unwrap();
    assert!(fraction.iter().flatten().any(|point| point[1] < a[1]));
    assert!(fraction.iter().flatten().any(|point| point[1] > a[1]));
    let sketch = authored_text(style, [0.; 2]);
    let report = sample_sketch_display(&sketch, metrics).unwrap();
    assert!(report.declines.is_empty());
    assert_eq!(
        report.entities[0].primitive,
        SketchDisplayPrimitive::GlyphEdges(fraction)
    );
    let center = [a[0].midpoint(b[0]), a[1]];
    let hit = pick_sketch_text(
        &sketch.texts()[0].style,
        [0.; 2],
        metrics.working_scale,
        center,
    )
    .unwrap()
    .unwrap();
    assert!((hit[0] - center[0]).hypot(hit[1] - center[1]) < 1e-6);
}

fn widest_horizontal_edge(paths: &[Vec<[f64; 2]>]) -> Option<[[f64; 2]; 2]> {
    paths
        .iter()
        .filter_map(|path| {
            let (&first, &last) = (path.first()?, path.last()?);
            (first[1] == last[1]).then_some([first, last])
        })
        .max_by(|[a, b], [c, d]| (a[0] - b[0]).abs().total_cmp(&(c[0] - d[0]).abs()))
}

#[test]
fn stacked_fraction_width_scales_the_separator_without_moving_the_rows() {
    let mut style = SketchTextStyle {
        content: "1/2".into(),
        family: "DejaVu Sans".into(),
        height: 0.8,
        stacked_fractions: Some(vec![SketchTextFraction { start: 0, end: 3 }]),
        ..Default::default()
    };
    let metrics = SketchSamplingOptionsV1 {
        working_scale: 0.1,
        ..options()
    };
    let regular = sample_sketch_text(&style, [0.; 2], metrics).unwrap();
    style.formatting_runs = Some(vec![SketchTextRun {
        start: 0,
        end: 3,
        width_percent: Some(Box::new(50.)),
        ..Default::default()
    }]);
    let narrow = sample_sketch_text(&style, [0.; 2], metrics).unwrap();
    let [a, b] = widest_horizontal_edge(&regular).unwrap();
    let [c, d] = widest_horizontal_edge(&narrow).unwrap();
    assert!(((a[0] - b[0]).abs() * 0.5 - (c[0] - d[0]).abs()).abs() < 1e-6);
    assert!((outline_height(&regular) - outline_height(&narrow)).abs() < 1e-6);
}
