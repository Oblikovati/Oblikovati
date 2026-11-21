// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use GuiShellAction::{PlaceSketchTool as Place, ToggleSketchFormat as Toggle};
use GuiSketchTool as Tool;
use oblikovati_model::sketch::SketchCurveFormat::{Centerline, Construction};
fn format_ribbon_fixture(scale: f32) -> (GuiFrameEngine, GuiFrameRequest) {
    let (mut engine, mut request) = expanded_sketch_ribbon(scale);
    request.commands.sketch_format_available = true;
    settle_ribbon(&mut engine, &mut request);
    (engine, request)
}
#[test]
fn curve_formats_dispatch_at_both_scales_and_refuse_caller_transactions() {
    for scale in [1., 2.] {
        let (mut engine, mut request) = format_ribbon_fixture(scale);
        for (format, caption) in [(Construction, "Construction"), (Centerline, "Centerline")] {
            let point = ribbon_caption(&mut engine, &mut request, caption, false);
            expect_ribbon(&mut engine, &mut request, point, Toggle(format));
            request.commands.transaction_open = true;
            assert!(click_ribbon(&mut engine, &mut request, point).is_empty());
            request.commands.transaction_open = false;
        }
    }
}
#[test]
fn format_capability_and_construction_point_admission_are_owner_derived() {
    let (mut engine, mut request) = format_ribbon_fixture(1.);
    let construction = ribbon_caption(&mut engine, &mut request, "Construction", false);
    request.commands.sketch_format_available = false;
    assert!(click_ribbon(&mut engine, &mut request, construction).is_empty());
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
    caption: &str,
) -> Vec<[u8; 4]> {
    request.events = vec![GuiInputEvent::PointerMoved([1300., 800.])];
    engine.frame(request).unwrap();
    let [x, y] = ribbon_caption(engine, request, caption, false);
    engine
        .frame(request)
        .unwrap()
        .draws
        .batches
        .iter()
        .flat_map(|batch| &batch.vertices)
        .filter(|vertex| {
            (vertex.position[0] - x).abs() <= 54. && (vertex.position[1] - y).abs() <= 10.
        })
        .map(|vertex| vertex.srgb)
        .collect()
}
#[test]
fn an_armed_format_changes_only_its_button_color_and_style_is_restored() {
    let (mut engine, mut request) = format_ribbon_fixture(1.);
    let original = format_button_colors(&mut engine, &mut request, "Construction");
    let other = format_button_colors(&mut engine, &mut request, "Centerline");
    request.commands.sketch_formats.construction = true;
    assert_ne!(
        format_button_colors(&mut engine, &mut request, "Construction"),
        original
    );
    assert_eq!(
        format_button_colors(&mut engine, &mut request, "Centerline"),
        other
    );
    request.commands.sketch_formats.construction = false;
    assert_eq!(
        format_button_colors(&mut engine, &mut request, "Construction"),
        original
    );
}
