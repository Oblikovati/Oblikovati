// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{
    DIM_CONSTRAINT_ELLIPSE_MINOR_RADIUS, DIM_CONSTRAINT_ELLIPSE_RADIUS, DimensionConstraintKind,
};

#[test]
fn ellipse_axis_drafts_reach_the_same_editor_and_enter_dispatch() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [1200, 800];
    enter_part(&mut request);
    request.commands.authoring = GuiAuthoringContext::Sketch;
    for (id, kind) in [
        (44, DIM_CONSTRAINT_ELLIPSE_RADIUS),
        (45, DIM_CONSTRAINT_ELLIPSE_MINOR_RADIUS),
    ] {
        request.commands.dimension_draft = Some(ellipse_axis_draft(id, kind));
        assert_ellipse_draft_enter(&mut engine, &mut request, id);
    }
}
fn ellipse_axis_draft(id: u64, kind: DimensionConstraintKind) -> GuiDimensionDraft {
    GuiDimensionDraft {
        id,
        kind: kind.0.to_string(),
        orientation: String::new(),
        expression: "8 mm".into(),
    }
}
fn assert_ellipse_draft_enter(engine: &mut GuiFrameEngine, request: &mut GuiFrameRequest, id: u64) {
    settle_ribbon(engine, request);
    let actions = dimension_editor_key(engine, request, GuiKey::Enter);
    assert_eq!(
        actions,
        [GuiShellAction::Dimension(GuiDimensionAction::Apply {
            id,
            expression: "8 mm".into(),
            driven: false,
        })]
    );
}
