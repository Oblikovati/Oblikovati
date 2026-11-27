// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::SketchTextStyle;
use oblikovati_model::parameter::ParameterCollection;
use oblikovati_model::sketch::SketchText2D;
use oblikovati_viewport::ViewportSketchPreview;

const ANCHOR: [f64; 2] = [-3., -1.];
const WORKING_SCALE: f64 = 0.1;

#[test]
fn native_linked_text_redraws_on_parameter_change_and_reports_missing_values() {
    use oblikovati_api::types::MODEL_PARAM;
    let mut parameters = ParameterCollection::new(Default::default());
    parameters
        .insert_expression(42, "d0", MODEL_PARAM, "10 mm")
        .unwrap();
    let authored = linked_geometry();
    let (runtime, mut view) = open_display("Linked sketch text");
    let blank = view.redraw().unwrap().rgba;
    let initial = draw_linked_text(&mut view, &authored, &parameters);
    assert_ne!(initial, blank, "initial linked text has visible glyphs");
    parameters.set_expression(42, "12 mm").unwrap();
    let updated = draw_linked_text(&mut view, &authored, &parameters);
    assert_ne!(
        updated, initial,
        "the changed parameter redraws visible glyphs"
    );
    parameters.set_expression(42, "10 mm").unwrap();
    assert_eq!(draw_linked_text(&mut view, &authored, &parameters), initial);
    parameters.remove_for_owner(42).unwrap();
    assert_eq!(draw_linked_text(&mut view, &authored, &parameters), blank);
    let declines = view.sketch_display_declines().unwrap();
    assert_eq!(declines[0].entity, Id(2));
    assert!(declines[0].cause.contains("42"));
    close_display(&runtime, &mut view);
}

fn linked_geometry() -> Sketch2D {
    use oblikovati_api::types::{
        SKETCH_TEXT_FIELD_MODEL_PARAMETER, SKETCH_TEXT_FIELD_PLACEHOLDER, SketchTextField,
    };
    use oblikovati_model::sketch::SketchTextParameterBinding;
    let mut geometry = Sketch2D::default();
    geometry.add_defining_point(Id(1), ANCHOR).unwrap();
    let mut style = fixture_text(SKETCH_TEXT_FIELD_PLACEHOLDER);
    style.linked_fields = Some(vec![SketchTextField {
        at: 0,
        kind: SKETCH_TEXT_FIELD_MODEL_PARAMETER,
        parameter: "d0".into(),
        ..Default::default()
    }]);
    geometry
        .add_text(SketchText2D {
            id: Id(2),
            anchor: Id(1),
            style,
            parameter_bindings: vec![SketchTextParameterBinding {
                at: 0,
                parameter_id: 42,
            }],
        })
        .unwrap();
    geometry
}

fn draw_linked_text(
    view: &mut oblikovati_viewport::Viewport,
    authored: &Sketch2D,
    parameters: &ParameterCollection<u64>,
) -> Vec<u8> {
    let (geometry, text_declines) = authored.text_display_geometry(parameters, "Part.opd");
    view.set_sketch_display(&[ViewportSketchDisplay {
        working_scale: WORKING_SCALE,
        sketch: DocumentSketchId(1),
        plane: SketchPlane::xy(),
        geometry,
        text_declines,
        blocks: vec![],
        curvature: Vec::new(),
        line_color: None,
    }])
    .unwrap();
    view.redraw().unwrap().rgba
}

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
        replaces: None,
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
            parameter_bindings: Vec::new(),
        })
        .unwrap();
    ViewportSketchDisplay {
        text_declines: Vec::new(),
        working_scale: WORKING_SCALE,
        sketch: DocumentSketchId(1),
        plane: SketchPlane::xy(),
        geometry,
        blocks: vec![],
        curvature: Vec::new(),
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

#[test]
fn native_text_edit_preview_masks_the_original_and_cancel_restores_it() {
    let (runtime, mut view) = open_display("Edit sketch text");
    let original = committed_text(fixture_text("OO"));
    view.set_sketch_display(std::slice::from_ref(&original))
        .unwrap();
    let before = view.redraw().unwrap().rgba;
    let preview = replacement_preview(fixture_text("I"));
    view.set_sketch_preview(Some(&preview)).unwrap();
    let during = lit_pixels(&view.redraw().unwrap());
    assert!(during.iter().filter(|&&pixel| pixel).count() > 0);
    assert_ne!(view.redraw().unwrap().rgba, before);
    view.set_sketch_preview(None).unwrap();
    assert_eq!(view.redraw().unwrap().rgba, before);
    assert_only_replacement_is_visible(&mut view, &preview, &during);
    close_display(&runtime, &mut view);
}

fn fixture_text(content: &str) -> SketchTextStyle {
    SketchTextStyle {
        content: content.into(),
        family: "DejaVu Sans".into(),
        height: 0.8,
        ..Default::default()
    }
}

fn replacement_preview(style: SketchTextStyle) -> ViewportSketchPreview {
    ViewportSketchPreview::Text {
        plane: SketchPlane::xy(),
        anchor: ANCHOR,
        style: Box::new(style),
        working_scale: WORKING_SCALE,
        replaces: Some((DocumentSketchId(1), Id(2))),
    }
}

fn assert_only_replacement_is_visible(
    view: &mut Viewport,
    preview: &ViewportSketchPreview,
    during: &[bool],
) {
    view.set_sketch_display(&[]).unwrap();
    view.set_sketch_preview(Some(preview)).unwrap();
    let alone = lit_pixels(&view.redraw().unwrap());
    assert!(
        alone == during,
        "original text was visible during its replacement preview"
    );
}
