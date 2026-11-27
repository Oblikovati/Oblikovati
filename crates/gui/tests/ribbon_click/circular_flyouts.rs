// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use GuiShellAction::PlaceSketchTool as Place;
use GuiSketchTool as Tool;
fn circular_ribbon_fixture(scale: f32) -> (GuiFrameEngine, GuiFrameRequest) {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [1200 * scale as u32, 800 * scale as u32];
    request.framebuffer_scale = [scale; 2];
    enter_part(&mut request);
    request.commands.authoring = GuiAuthoringContext::Sketch;
    settle_ribbon(&mut engine, &mut request);
    (engine, request)
}

#[test]
fn circular_flyouts_dispatch_both_choices_at_both_scales() {
    for scale in [1., 2.] {
        let (mut engine, mut request) = circular_ribbon_fixture(scale);
        for (primary, tool) in [
            (Tool::Circle, Tool::ThreePointCircle),
            (Tool::CenterPointArc, Tool::ThreePointArc),
        ] {
            open_tool_choices(&mut engine, &mut request, primary.label());
            expect_caption(
                &mut engine,
                &mut request,
                tool.menu_label(),
                true,
                Place(tool),
            );
        }
    }
}

#[test]
fn circular_flyouts_refuse_transactions_and_stale_open_choices() {
    let (mut engine, mut request) = circular_ribbon_fixture(1.);
    request.commands.transaction_open = true;
    assert!(click_ribbon(&mut engine, &mut request, [240., 68.]).is_empty());
    assert!(click_ribbon(&mut engine, &mut request, [300., 98.]).is_empty());
    request.commands.transaction_open = false;
    assert!(click_ribbon(&mut engine, &mut request, [240., 68.]).is_empty());
    request.commands.transaction_open = true;
    settle_ribbon(&mut engine, &mut request);
    assert!(click_ribbon(&mut engine, &mut request, [300., 98.]).is_empty());
    request.commands.transaction_open = false;
    expect_ribbon(&mut engine, &mut request, [180., 68.], Place(Tool::Circle));
}

#[test]
fn circular_primary_menu_choices_preserve_the_original_commands() {
    let (mut engine, mut request) = circular_ribbon_fixture(1.);
    for tool in [Tool::Circle, Tool::CenterPointArc] {
        open_tool_choices(&mut engine, &mut request, tool.label());
        expect_caption(
            &mut engine,
            &mut request,
            tool.menu_label(),
            true,
            Place(tool),
        );
    }
}
#[test]
fn a_context_change_retires_the_circular_menu_without_dispatch() {
    let (mut engine, mut request) = circular_ribbon_fixture(1.);
    assert!(click_ribbon(&mut engine, &mut request, [240., 68.]).is_empty());
    request.commands.authoring = GuiAuthoringContext::Part;
    assert!(settle_ribbon(&mut engine, &mut request).is_empty());
    request.commands.authoring = GuiAuthoringContext::Sketch;
    settle_ribbon(&mut engine, &mut request);
    expect_ribbon(&mut engine, &mut request, [180., 68.], Place(Tool::Circle));
}

#[test]
fn circle_flyout_dispatches_ellipse_at_both_scales() {
    for scale in [1., 2.] {
        let (mut engine, mut request) = circular_ribbon_fixture(scale);
        open_tool_choices(&mut engine, &mut request, Tool::Circle.label());
        expect_caption(
            &mut engine,
            &mut request,
            Tool::Ellipse.menu_label(),
            true,
            Place(Tool::Ellipse),
        );
    }
}
