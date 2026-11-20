// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::SketchTextStyle;
use oblikovati_model::sketch::SketchText2D;
use oblikovati_viewport::ViewportSketchPreview;

const ANCHOR: [f64; 2] = [-3., -1.];
const WORKING_SCALE: f64 = 0.1;

#[test]
fn native_sketch_text_draws_matching_preview_and_committed_pixels() {
    let (runtime, mut view) = open_display("Sketch text");
    let style = SketchTextStyle {
        content: "O8".into(),
        family: "DejaVu Sans".into(),
        height: 0.8,
        rotation: 0.2,
        ..Default::default()
    };
    let baseline = view.redraw().unwrap();
    let preview = ViewportSketchPreview::Text {
        plane: SketchPlane::xy(),
        anchor: ANCHOR,
        style: Box::new(style.clone()),
        working_scale: WORKING_SCALE,
    };
    view.set_sketch_preview(Some(&preview)).unwrap();
    let draft = lit_pixels(&view.redraw().unwrap());
    assert!(draft.iter().filter(|&&pixel| pixel).count() > 25);
    view.set_sketch_preview(None).unwrap();
    assert_eq!(view.redraw().unwrap().rgba, baseline.rgba);
    view.set_sketch_display(&[committed_text(style)]).unwrap();
    assert!(view.sketch_display_declines().unwrap().is_empty());
    assert!(
        lit_pixels(&view.redraw().unwrap()) == draft,
        "glyph pixels differ between preview and committed display"
    );
    close_display(&runtime, &mut view);
}

fn committed_text(style: SketchTextStyle) -> ViewportSketchDisplay {
    let mut geometry = Sketch2D::default();
    geometry.add_defining_point(Id(1), ANCHOR).unwrap();
    geometry
        .add_text(SketchText2D {
            id: Id(2),
            anchor: Id(1),
            style,
        })
        .unwrap();
    ViewportSketchDisplay {
        working_scale: WORKING_SCALE,
        sketch: DocumentSketchId(1),
        plane: SketchPlane::xy(),
        geometry,
        blocks: vec![],
        line_color: None,
    }
}

fn lit_pixels(capture: &ViewportCapture) -> Vec<bool> {
    capture
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .map(|pixel| pixel[..3].iter().any(|&channel| channel != 0))
        .collect()
}
