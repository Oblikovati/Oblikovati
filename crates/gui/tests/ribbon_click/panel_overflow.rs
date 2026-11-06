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
fn open_panel(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    point: [f32; 2],
    scale: f32,
) {
    assert!(click_ribbon(engine, request, point.map(|value| value * scale)).is_empty());
    settle_ribbon(engine, request);
}
#[test]
fn a_collapsed_format_and_constrain_panel_dispatch_the_same_actions_at_both_scales() {
    for scale in [1., 2.] {
        let (mut engine, mut request) = narrow_fixture(1024, scale);
        open_panel(&mut engine, &mut request, [790., 88.], scale);
        expect_ribbon(
            &mut engine,
            &mut request,
            [798. * scale, 175. * scale],
            Format(Construction),
        );
        open_panel(&mut engine, &mut request, [550., 88.], scale);
        expect_ribbon(
            &mut engine,
            &mut request,
            [566. * scale, 175. * scale],
            Constrain(GuiSketchConstraint::Horizontal),
        );
    }
}
use GuiSketchTool::*;
const PRIMARY_TOOL_POINTS: [(GuiSketchTool, [f32; 2]); 8] = [
    (Line, [202., 235.]),
    (Rectangle, [78., 195.]),
    (Circle, [193., 175.]),
    (Point, [318., 175.]),
    (CenterPointArc, [193., 195.]),
    (Ellipse, [318., 195.]),
    (CenterRectangle, [202., 215.]),
    (ThreePointRectangle, [318., 215.]),
];
#[test]
fn collapsed_create_keeps_all_primary_tools_reachable_at_both_scales() {
    for scale in [1., 2.] {
        let (mut engine, mut request) = narrow_fixture(640, scale);
        for (tool, point) in PRIMARY_TOOL_POINTS {
            open_panel(&mut engine, &mut request, [70., 88.], scale);
            expect_ribbon(
                &mut engine,
                &mut request,
                point.map(|value| value * scale),
                Place(tool),
            );
        }
        open_panel(&mut engine, &mut request, [186., 88.], scale);
        expect_ribbon(
            &mut engine,
            &mut request,
            [194. * scale, 195. * scale],
            FinishSketch,
        );
    }
}
#[test]
fn a_flyout_rechecks_owner_transaction_state_and_escape_restores_the_band() {
    let (mut engine, mut request) = narrow_fixture(1024, 1.);
    open_panel(&mut engine, &mut request, [550., 88.], 1.);
    request.commands.transaction_open = true;
    assert!(click_ribbon(&mut engine, &mut request, [566., 175.]).is_empty());
    for pressed in [true, false] {
        request.events = vec![GuiInputEvent::Key {
            key: GuiKey::Escape,
            pressed,
        }];
        assert!(engine.frame(&request).unwrap().actions.is_empty());
    }
    request.commands.transaction_open = false;
    settle_ribbon(&mut engine, &mut request);
    open_panel(&mut engine, &mut request, [550., 88.], 1.);
    expect_ribbon(
        &mut engine,
        &mut request,
        [566., 175.],
        Constrain(GuiSketchConstraint::Horizontal),
    );
}
#[test]
fn widening_restores_expanded_controls_and_narrowing_collapses_them_again() {
    let (mut engine, mut request) = narrow_fixture(1024, 1.);
    request.extent[0] = 1440;
    settle_ribbon(&mut engine, &mut request);
    expect_ribbon(
        &mut engine,
        &mut request,
        [1130., 68.],
        Format(Construction),
    );
    request.extent[0] = 1024;
    settle_ribbon(&mut engine, &mut request);
    open_panel(&mut engine, &mut request, [790., 88.], 1.);
    expect_ribbon(
        &mut engine,
        &mut request,
        [798., 175.],
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
        open_panel(&mut engine, &mut request, [250., 88.], scale);
        expect_ribbon(
            &mut engine,
            &mut request,
            [250. * scale, 190. * scale],
            Format(Construction),
        );
    }
}
#[test]
fn a_panel_wider_than_the_window_keeps_its_last_command_reachable() {
    for scale in [1., 2.] {
        let (mut engine, mut request) = narrow_fixture(320, scale);
        open_panel(&mut engine, &mut request, [280., 88.], scale);
        scroll_right(&mut engine, &mut request, [200., 200.], scale);
        expect_ribbon(
            &mut engine,
            &mut request,
            [250. * scale, 229. * scale],
            Constrain(GuiSketchConstraint::Tangent),
        );
    }
}
#[test]
fn changing_context_cancels_a_panel_popup_before_the_next_command() {
    let (mut engine, mut request) = narrow_fixture(1024, 1.);
    open_panel(&mut engine, &mut request, [790., 88.], 1.);
    request.commands.authoring = GuiAuthoringContext::Part;
    settle_ribbon(&mut engine, &mut request);
    expect_ribbon(&mut engine, &mut request, [50., 90.], StartSketchXY);
}
#[test]
fn widening_with_a_popup_open_restores_its_inline_command_immediately() {
    let (mut engine, mut request) = narrow_fixture(1024, 1.);
    open_panel(&mut engine, &mut request, [790., 88.], 1.);
    request.extent[0] = 1440;
    settle_ribbon(&mut engine, &mut request);
    expect_ribbon(
        &mut engine,
        &mut request,
        [1130., 68.],
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
    open_panel(&mut engine, &mut request, [70., 88.], 1.);
    expect_ribbon(
        &mut engine,
        &mut request,
        [78., 195.],
        Place(GuiSketchTool::Rectangle),
    );
}
