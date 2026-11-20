// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{SketchTextStyle, TEXT_ALIGN_UPPER};
use oblikovati_martensite::planar_curve::sketch::sample_sketch_text;
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
