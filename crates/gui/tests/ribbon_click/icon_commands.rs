// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::ui_scale::UiScaleOptions;

#[test]
fn icon_scale_and_dpr_keep_the_entire_captioned_command_hit_area() {
    for (text, icon, backing) in [(1., 1., 1.), (1., 1., 2.), (1., 2., 1.), (2., 1., 1.)] {
        check_icon_command_hit_area(text, icon, backing);
    }
}
fn check_icon_command_hit_area(text: f32, icon: f32, backing: f32) {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [2880, 1800];
    request.framebuffer_scale = [backing; 2];
    request.commands.ui_scale = Some(UiScaleOptions::new(text, icon).unwrap());
    settle_ribbon(&mut engine, &mut request);
    let factor = (icon / text).max(1.);
    for y in [64. + 10. * factor, 60. + 53. * factor] {
        expect_ribbon(
            &mut engine,
            &mut request,
            [50. * text * backing, y * text * backing],
            NewPart,
        );
    }
}

#[test]
fn independent_icon_scaling_keeps_a_collapsed_sketch_create_panel_usable() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [700, 900];
    request.commands.ui_scale = Some(UiScaleOptions::new(1., 2.).unwrap());
    enter_part(&mut request);
    request.commands.authoring = GuiAuthoringContext::Sketch;
    settle_ribbon(&mut engine, &mut request);
    open_collapsed_panel(&mut engine, &mut request, "Create");
    expect_caption(
        &mut engine,
        &mut request,
        "Rectangle",
        true,
        GuiShellAction::PlaceSketchTool(GuiSketchTool::Rectangle),
    );
}
