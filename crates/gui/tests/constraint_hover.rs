// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::{
    GuiAuthoringContext, GuiCommandState, GuiConstraintGlyph, GuiFrameEngine, GuiFrameRequest,
    GuiInputEvent,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchConstraintId, SketchEntityId},
};

static GUI_CONTEXT: std::sync::Mutex<()> = std::sync::Mutex::new(());

fn request() -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [1280, 720],
        events: Vec::new(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1., 1.],
        commands: GuiCommandState {
            document: Some(1),
            authoring: GuiAuthoringContext::Sketch,
            constraint_glyphs: vec![GuiConstraintGlyph {
                sketch: DocumentSketchId(3),
                constraint: SketchConstraintId(4),
                related: vec![SketchEntityId(5), SketchEntityId(6)],
                pixel: [600, 400],
                text: "//".into(),
            }],
            ..Default::default()
        },
    }
}

#[test]
fn hovering_the_rendered_glyph_reports_both_lines_and_clears_on_exit() {
    let _guard = GUI_CONTEXT.lock().unwrap();
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = request();
    request.events = vec![GuiInputEvent::PointerMoved([400., 400.])];
    assert!(engine.frame(&request).unwrap().constraint_hover.is_none());
    request.events = vec![GuiInputEvent::PointerMoved([610., 410.])];
    let hover = engine
        .frame(&request)
        .unwrap()
        .constraint_hover
        .expect("expected the glyph hover");
    assert_eq!(hover.sketch, DocumentSketchId(3));
    assert_eq!(hover.constraint, SketchConstraintId(4));
    assert_eq!(hover.related, [SketchEntityId(5), SketchEntityId(6)]);
    assert!(request.commands.sketch_selection.entities.is_empty());
    request.events = vec![GuiInputEvent::PointerMoved([400., 400.])];
    assert!(engine.frame(&request).unwrap().constraint_hover.is_none());
}

#[test]
fn overlapping_glyphs_do_not_choose_an_arbitrary_constraint() {
    let _guard = GUI_CONTEXT.lock().unwrap();
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = request();
    let mut second = request.commands.constraint_glyphs[0].clone();
    second.constraint = SketchConstraintId(9);
    request.commands.constraint_glyphs.push(second);
    request.events = vec![GuiInputEvent::PointerMoved([610., 410.])];
    assert!(engine.frame(&request).unwrap().constraint_hover.is_none());
}
