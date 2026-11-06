// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use GuiShellAction::{PlaceSketchTool as Place, ToggleSketchFormat as Toggle};
use GuiSketchTool as Tool;
use oblikovati_model::sketch::SketchCurveFormat::{Centerline, Construction};
fn format_ribbon_fixture(scale: f32) -> (GuiFrameEngine, GuiFrameRequest) {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [1440 * scale as u32, 900 * scale as u32];
    request.framebuffer_scale = [scale; 2];
    enter_part(&mut request);
    request.commands.authoring = GuiAuthoringContext::Sketch;
    request.commands.sketch_format_available = true;
    settle_ribbon(&mut engine, &mut request);
    (engine, request)
}
#[test]
fn curve_formats_dispatch_at_both_scales_and_refuse_caller_transactions() {
    for scale in [1., 2.] {
        let (mut engine, mut request) = format_ribbon_fixture(scale);
        for (format, y) in [(Construction, 68.), (Centerline, 88.)] {
            let point = [1130. * scale, y * scale];
            expect_ribbon(&mut engine, &mut request, point, Toggle(format));
            request.commands.transaction_open = true;
            assert!(click_ribbon(&mut engine, &mut request, point).is_empty());
            request.commands.sketch_tool_transaction = true;
            expect_ribbon(&mut engine, &mut request, point, Toggle(format));
            request.commands.sketch_tool_transaction = false;
            request.commands.transaction_open = false;
        }
    }
}
#[test]
fn format_capability_and_construction_point_admission_are_owner_derived() {
    let (mut engine, mut request) = format_ribbon_fixture(1.);
    request.commands.sketch_format_available = false;
    assert!(click_ribbon(&mut engine, &mut request, [1130., 68.]).is_empty());
    request.commands.sketch_formats.construction = true;
    assert!(click_ribbon(&mut engine, &mut request, [300., 68.]).is_empty());
    expect_ribbon(&mut engine, &mut request, [180., 68.], Place(Tool::Circle));
    request.commands.sketch_formats.construction = false;
    request.commands.sketch_formats.centerline = true;
    expect_ribbon(&mut engine, &mut request, [300., 68.], Place(Tool::Point));
}
fn format_button_colors(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    y: f32,
) -> Vec<[u8; 4]> {
    request.events = vec![GuiInputEvent::PointerMoved([1300., 800.])];
    engine.frame(request).unwrap();
    settle_ribbon(engine, request);
    engine
        .frame(request)
        .unwrap()
        .draws
        .batches
        .iter()
        .flat_map(|batch| &batch.vertices)
        .filter(|vertex| {
            vertex.position[0] >= 1084.
                && vertex.position[0] <= 1192.
                && vertex.position[1] >= y
                && vertex.position[1] <= y + 18.
        })
        .map(|vertex| vertex.srgb)
        .collect()
}
#[test]
fn an_armed_format_changes_only_its_button_color_and_style_is_restored() {
    let (mut engine, mut request) = format_ribbon_fixture(1.);
    let original = format_button_colors(&mut engine, &mut request, 59.);
    let other = format_button_colors(&mut engine, &mut request, 79.);
    request.commands.sketch_formats.construction = true;
    assert_ne!(
        format_button_colors(&mut engine, &mut request, 59.),
        original
    );
    assert_eq!(format_button_colors(&mut engine, &mut request, 79.), other);
    request.commands.sketch_formats.construction = false;
    assert_eq!(
        format_button_colors(&mut engine, &mut request, 59.),
        original
    );
}
