// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use GuiShellAction::{
    ConstrainSketch as Constrain, PlaceSketchTool as Place, ToggleSketchFormat as Format,
};
use oblikovati_model::sketch::SketchCurveFormat::Construction;
fn narrow_fixture(width: u32, scale: f32) -> (GuiFrameEngine, GuiFrameRequest) {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [width * scale as u32, 900 * scale as u32];
    request.framebuffer_scale = [scale; 2];
    enter_part(&mut request);
    request.commands.authoring = GuiAuthoringContext::Sketch;
    request.commands.sketch_format_available = true;
    settle_ribbon(&mut engine, &mut request);
    (engine, request)
}
#[test]
fn a_collapsed_format_and_constrain_panel_dispatch_the_same_actions_at_both_scales() {
    for scale in [1., 2.] {
        let (mut engine, mut request) = narrow_fixture(1024, scale);
        open_collapsed_panel(&mut engine, &mut request, "Format");
        expect_caption(
            &mut engine,
            &mut request,
            "Construction",
            true,
            Format(Construction),
        );
        open_collapsed_panel(&mut engine, &mut request, "Constrain");
        expect_caption(
            &mut engine,
            &mut request,
            "Horizontal",
            true,
            Constrain(GuiSketchConstraint::Horizontal),
        );
    }
}

#[test]
fn the_narrow_model_ribbon_shows_and_opens_its_last_group_without_scrolling() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [780, 580];
    enter_part(&mut request);
    settle_ribbon(&mut engine, &mut request);
    open_collapsed_panel(&mut engine, &mut request, "Simplification");
    assert!(ribbon_caption(&mut engine, &mut request, "Simplify", true)[0] < 780.);
}

#[test]
fn sketch_compaction_is_visible_on_the_first_frame_and_keeps_the_viewport_height() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [920, 680];
    enter_part(&mut request);
    settle_ribbon(&mut engine, &mut request);
    let part_top = engine.frame(&request).unwrap().document_area.unwrap()[1];
    request.commands.authoring = GuiAuthoringContext::Sketch;
    for _ in 0..4 {
        let frame = engine.frame(&request).unwrap();
        assert_eq!(frame.document_area.unwrap()[1], part_top);
        let runs = drawn_text::rendered_frame();
        let finish = runs
            .runs
            .iter()
            .find(|run| run.text == "FinishSketch")
            .unwrap_or_else(|| {
                let labels: Vec<_> = runs.runs.iter().map(|run| run.text.as_str()).collect();
                panic!("expected FinishSketch in {labels:?}");
            });
        assert!(finish.maximum[0] < 920. && finish.maximum[1] < part_top as f32);
        assert!(runs.runs.iter().any(|run| run.text == "Create"));
    }
}

use GuiSketchTool::*;
const PRIMARY_TOOLS: [GuiSketchTool; 8] = [
    Line,
    Rectangle,
    Circle,
    Point,
    CenterPointArc,
    Ellipse,
    CenterRectangle,
    ThreePointRectangle,
];
#[test]
fn collapsed_create_keeps_all_primary_tools_reachable_at_both_scales() {
    for scale in [1., 2.] {
        let (mut engine, mut request) = narrow_fixture(640, scale);
        for tool in PRIMARY_TOOLS {
            open_collapsed_panel(&mut engine, &mut request, "Create");
            expect_caption(&mut engine, &mut request, tool.label(), true, Place(tool));
        }
        scroll_right(&mut engine, &mut request, [200., 88.], scale);
        expect_caption(
            &mut engine,
            &mut request,
            "Finish Sketch",
            false,
            FinishSketch,
        );
    }
}
#[test]
fn a_flyout_rechecks_owner_transaction_state_and_escape_restores_the_band() {
    let (mut engine, mut request) = narrow_fixture(1024, 1.);
    open_collapsed_panel(&mut engine, &mut request, "Constrain");
    request.commands.transaction_open = true;
    let horizontal = ribbon_caption(&mut engine, &mut request, "Horizontal", true);
    assert!(click_ribbon(&mut engine, &mut request, horizontal).is_empty());
    for pressed in [true, false] {
        request.events = vec![GuiInputEvent::Key {
            key: GuiKey::Escape,
            pressed,
        }];
        assert!(engine.frame(&request).unwrap().actions.is_empty());
    }
    request.commands.transaction_open = false;
    settle_ribbon(&mut engine, &mut request);
    open_collapsed_panel(&mut engine, &mut request, "Constrain");
    expect_caption(
        &mut engine,
        &mut request,
        "Horizontal",
        true,
        Constrain(GuiSketchConstraint::Horizontal),
    );
}
#[test]
fn widening_restores_expanded_controls_and_narrowing_collapses_them_again() {
    let (mut engine, mut request) = narrow_fixture(1024, 1.);
    request.extent[0] = 1920;
    expect_caption(
        &mut engine,
        &mut request,
        "Construction",
        false,
        Format(Construction),
    );
    request.extent[0] = 1024;
    settle_ribbon(&mut engine, &mut request);
    open_collapsed_panel(&mut engine, &mut request, "Format");
    expect_caption(
        &mut engine,
        &mut request,
        "Construction",
        true,
        Format(Construction),
    );
}

fn scroll_right(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    point: [f32; 2],
    scale: f32,
) {
    request.events = vec![GuiInputEvent::PointerMoved(
        point.map(|value| value * scale),
    )];
    engine.frame(request).unwrap();
    request.events = vec![GuiInputEvent::Wheel([-100., 0.])];
    engine.frame(request).unwrap();
    settle_ribbon(engine, request);
}
#[test]
fn the_scroll_tier_keeps_the_last_panel_reachable_below_its_minimum_row_width() {
    for scale in [1., 2.] {
        let (mut engine, mut request) = narrow_fixture(320, scale);
        scroll_right(&mut engine, &mut request, [200., 88.], scale);
        expect_caption(
            &mut engine,
            &mut request,
            "Finish Sketch",
            false,
            FinishSketch,
        );
    }
}
#[test]
fn a_panel_wider_than_the_window_keeps_its_last_command_reachable() {
    for scale in [1., 2.] {
        let (mut engine, mut request) = narrow_fixture(320, scale);
        open_collapsed_panel(&mut engine, &mut request, "Constrain");
        let point = ribbon_caption(&mut engine, &mut request, "Horizontal", true);
        scroll_right(
            &mut engine,
            &mut request,
            point.map(|value| value / scale),
            scale,
        );
        expect_caption(
            &mut engine,
            &mut request,
            "Tangent",
            true,
            Constrain(GuiSketchConstraint::Tangent),
        );
    }
}
#[test]
fn changing_context_cancels_a_panel_popup_before_the_next_command() {
    let (mut engine, mut request) = narrow_fixture(1024, 1.);
    open_collapsed_panel(&mut engine, &mut request, "Format");
    request.commands.authoring = GuiAuthoringContext::Part;
    settle_ribbon(&mut engine, &mut request);
    expect_caption(
        &mut engine,
        &mut request,
        "Start 2D Sketch",
        false,
        ChooseSketchPlane,
    );
}
#[test]
fn widening_with_a_popup_open_restores_its_inline_command_immediately() {
    let (mut engine, mut request) = narrow_fixture(1024, 1.);
    open_collapsed_panel(&mut engine, &mut request, "Format");
    request.extent[0] = 1920;
    expect_caption(
        &mut engine,
        &mut request,
        "Construction",
        false,
        Format(Construction),
    );
}
#[test]
fn a_clipped_child_is_closed_before_the_next_visible_frame() {
    let (mut engine, mut request) = narrow_fixture(640, 1.);
    request.extent[1] = 40;
    settle_ribbon(&mut engine, &mut request);
    request.extent[1] = 900;
    settle_ribbon(&mut engine, &mut request);
    open_collapsed_panel(&mut engine, &mut request, "Create");
    expect_caption(
        &mut engine,
        &mut request,
        "Rectangle",
        true,
        Place(GuiSketchTool::Rectangle),
    );
}
