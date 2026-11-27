// SPDX-License-Identifier: GPL-2.0-only
use super::{
    GuiFrameEngine, GuiFrameRequest, GuiShellAction, click_ribbon, drawn_text, settle_ribbon,
};
use std::ops::Range;

/// Find a rendered ribbon caption in native pointer pixels.
/// Example: `ribbon_caption(engine, request, "Finish Sketch", false)` finds the inline control.
pub(super) fn ribbon_caption(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    label: &str,
    popup: bool,
) -> [f32; 2] {
    let frame = settled_frame(engine, request);
    pointer_pixels(request, caption_run(&frame, label, popup).center())
}

fn settled_frame(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
) -> drawn_text::RenderedFrame {
    assert!(settle_ribbon(engine, request).is_empty());
    drawn_text::rendered_frame()
}

fn caption_run<'a>(
    frame: &'a drawn_text::RenderedFrame,
    label: &str,
    popup: bool,
) -> &'a drawn_text::RenderedRun {
    let requested = visible_words(label);
    frame
        .runs
        .iter()
        .rev()
        .find(|row| frame.is_popup_row(row) == popup && matches_caption(&row.text, &requested))
        .unwrap_or_else(|| {
            let rows: Vec<_> = frame
                .runs
                .iter()
                .filter(|row| frame.is_popup_row(row) == popup)
                .map(|row| row.text.as_str())
                .collect();
            panic!("caption {label:?}, popup {popup}: expected a visible row in {rows:?}");
        })
}

fn pointer_pixels(request: &GuiFrameRequest, [x, y]: [f32; 2]) -> [f32; 2] {
    [
        x * request.framebuffer_scale[0],
        y * request.framebuffer_scale[1],
    ]
}

/// Open a collapsed panel. Example: `open_collapsed_panel(engine, request, "Format")`.
pub(super) fn open_collapsed_panel(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    panel: &str,
) {
    let frame = settled_frame(engine, request);
    let caption = caption_run(&frame, panel, false);
    assert!(click_ribbon(engine, request, pointer_pixels(request, caption.center())).is_empty());
}

/// Open the arrow beside an inline tool. Example: `open_tool_choices(engine, request, "Circle")`.
pub(super) fn open_tool_choices(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    label: &str,
) {
    let frame = settled_frame(engine, request);
    let caption = caption_run(&frame, label, false);
    let [x, y] = caption.center();
    let position = if caption.text.ends_with(['v', '▼']) {
        // Adjacent text can form one run. Its last glyph is the separate arrow button.
        [caption.maximum[0] - 2., y]
    } else {
        arrow_in_region(&frame, caption, [x..x + 100., y - 6. ..y + 6.])
            .unwrap_or_else(|| panic!("{label:?}: expected the arrow beside its caption"))
    };
    assert!(click_ribbon(engine, request, pointer_pixels(request, position)).is_empty());
}

fn arrow_in_region(
    frame: &drawn_text::RenderedFrame,
    caption: &drawn_text::RenderedRun,
    [columns, rows]: [Range<f32>; 2],
) -> Option<[f32; 2]> {
    frame.runs.iter().find_map(|run| {
        let [x, y] = run.center();
        (run.list == caption.list
            && matches!(run.text.as_str(), "v" | "▼")
            && columns.contains(&x)
            && rows.contains(&y))
        .then_some([x, y])
    })
}

/// Click a rendered caption and check its dispatch.
/// Example: `expect_caption(engine, request, "New Part", false, GuiShellAction::NewPart)`.
pub(super) fn expect_caption(
    engine: &mut GuiFrameEngine,
    request: &mut GuiFrameRequest,
    label: &str,
    popup: bool,
    action: GuiShellAction,
) {
    let position = ribbon_caption(engine, request, label, popup);
    assert_eq!(
        click_ribbon(engine, request, position),
        [action],
        "caption {label:?} at {position:?}"
    );
}

fn visible_words(label: &str) -> String {
    label
        .chars()
        .filter(|letter| !letter.is_whitespace())
        .collect()
}

fn matches_caption(text: &str, requested: &str) -> bool {
    let visible = visible_words(text);
    let label = visible.strip_suffix(['v', '▼']).unwrap_or(&visible);
    visible == requested || label.eq_ignore_ascii_case(requested)
}
