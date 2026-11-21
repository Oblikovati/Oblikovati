// SPDX-License-Identifier: GPL-2.0-only
//! Show native line-spacing choices and convert their values for the editor display.
use super::GuiSketchTextDraft;
use imgui::Ui;
use oblikovati_api::types::{
    SketchTextLineSpacing, TEXT_LINE_SPACING_DOUBLE, TEXT_LINE_SPACING_EXACTLY,
    TEXT_LINE_SPACING_MULTIPLE, TEXT_LINE_SPACING_ONE_AND_HALF, TEXT_LINE_SPACING_SINGLE,
    TextLineSpacingMode,
};

const MODES: [(TextLineSpacingMode, &str, f64); 5] = [
    (TEXT_LINE_SPACING_SINGLE, "Single", 0.),
    (TEXT_LINE_SPACING_DOUBLE, "Double", 0.),
    (TEXT_LINE_SPACING_ONE_AND_HALF, "1.5 Lines", 0.),
    (TEXT_LINE_SPACING_MULTIPLE, "Multiple", 1.5),
    (TEXT_LINE_SPACING_EXACTLY, "Exactly", 0.3),
];

/// Apply an edit to the spacing, for example from Single to Double.
pub(super) fn draw(ui: &Ui, spacing: &mut Option<Box<SketchTextLineSpacing>>) -> bool {
    let selected = spacing
        .as_ref()
        .map_or(&TEXT_LINE_SPACING_SINGLE, |value| &value.mode);
    if let Some(choice) = choose_mode(ui, selected) {
        *spacing = Some(Box::new(choice));
        return true;
    }
    spacing.as_mut().is_some_and(|choice| edit_gap(ui, choice))
}

fn choose_mode(ui: &Ui, current: &TextLineSpacingMode) -> Option<SketchTextLineSpacing> {
    let active = MODES
        .iter()
        .find(|(mode, _, _)| mode == current)
        .map(|(_, name, _)| *name)?;
    let _list = ui.begin_combo("Line spacing", active)?;
    MODES.into_iter().find_map(|(mode, name, value)| {
        let selected = &mode == current;
        let clicked = ui.menu_item_config(name).selected(selected).build();
        (clicked && !selected).then_some(SketchTextLineSpacing { mode, value })
    })
}

fn edit_gap(ui: &Ui, spacing: &mut SketchTextLineSpacing) -> bool {
    let (label, factor) = if spacing.mode == TEXT_LINE_SPACING_MULTIPLE {
        ("Baseline factor", 1.)
    } else if spacing.mode == TEXT_LINE_SPACING_EXACTLY {
        ("Baseline gap (mm)", 10.)
    } else {
        return false;
    };
    let mut displayed = spacing.value * factor;
    let changed = ui.input_scalar(label, &mut displayed).build();
    let value = displayed / factor;
    if changed && value > 0. && value.is_finite() {
        spacing.value = value;
        return true;
    }
    false
}

/// Convert to an editor-font row factor, for example Double uses two native rows of height.
pub(super) fn editor_row_factor(draft: &GuiSketchTextDraft) -> f64 {
    let Some(spacing) = &draft.formatting.line_spacing else {
        return 1.;
    };
    if spacing.mode == TEXT_LINE_SPACING_DOUBLE {
        return 2.;
    }
    if spacing.mode == TEXT_LINE_SPACING_ONE_AND_HALF {
        return 1.5;
    }
    if spacing.mode == TEXT_LINE_SPACING_MULTIPLE {
        return spacing.value;
    }
    if spacing.mode == TEXT_LINE_SPACING_EXACTLY
        && let Ok(height) = draft.height_mm.trim().parse::<f64>()
        && height.is_finite()
        && height > 0.
    {
        return spacing.value * 10. / height;
    }
    1.
}
