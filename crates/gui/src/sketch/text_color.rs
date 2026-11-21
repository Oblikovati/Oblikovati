// SPDX-License-Identifier: GPL-2.0-only
use super::{GuiSketchTextDraft, all_scalars, character_ranges::revise_runs};
use crate::application::model_browser::color_dialog::{ColorAnswer, ColorDialog, TITLE};
use imgui::{StyleColor, Ui};
use oblikovati_api::types::{Color, OVERRIDE_COLOR_SOURCE};
use oblikovati_model::sketch::uniform_sketch_text_color;
use std::ops::Range;

pub(super) fn draw(
    ui: &Ui,
    draft: &mut GuiSketchTextDraft,
    selection: &Range<usize>,
    dialog: &mut ColorDialog,
) -> bool {
    ui.same_line();
    let enabled = !draft.content.is_empty() && *selection == (0..draft.content.len());
    let current = current_rgb(draft);
    if color_button(ui, current, enabled) {
        dialog.begin_with_color(current.flatten());
        ui.open_popup(TITLE);
    }
    let Some(ColorAnswer::Chosen(rgb)) = dialog.panel(ui) else {
        return false;
    };
    apply_palette_color(draft, rgb)
}

fn color_button(ui: &Ui, color: Option<Option<[u8; 3]>>, enabled: bool) -> bool {
    let _disabled = ui.begin_disabled(!enabled);
    let _swatch = color.map(|color| {
        let swatch = color.map_or_else(
            || ui.style_color(StyleColor::CheckMark),
            |rgb| {
                let [r, g, b] = rgb.map(|channel| f32::from(channel) / 255.);
                [r, g, b, 1.]
            },
        );
        ui.push_style_color(StyleColor::Button, swatch)
    });
    ui.button("Color")
}

fn current_rgb(draft: &GuiSketchTextDraft) -> Option<Option<[u8; 3]>> {
    let runs = draft.formatting.runs.as_deref().unwrap_or_default();
    uniform_sketch_text_color(&draft.content, runs)
        .map(|uniform| uniform.map(|color| [color.r, color.g, color.b]))
}

fn apply_palette_color(draft: &mut GuiSketchTextDraft, [r, g, b]: [u8; 3]) -> bool {
    let Some(range) = all_scalars(&draft.content).filter(|range| !range.is_empty()) else {
        return false;
    };
    let color = Color {
        r,
        g,
        b,
        opacity: 1.,
        source: OVERRIDE_COLOR_SOURCE,
    };
    let runs = draft.formatting.runs.as_deref().unwrap_or_default();
    draft.formatting.runs = Some(revise_runs(runs, range, |run| {
        run.color = Some(Box::new(color.clone()));
    }));
    true
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_api::types::SketchTextRun;

    #[test]
    fn applying_color_preserves_existing_width() {
        let mut draft = GuiSketchTextDraft {
            content: "界".into(),
            ..Default::default()
        };
        draft.formatting.runs = Some(vec![SketchTextRun {
            end: 1,
            width_percent: Some(Box::new(50.)),
            ..Default::default()
        }]);
        assert!(apply_palette_color(&mut draft, [0, 128, 255]));
        assert_eq!(current_rgb(&draft), Some(Some([0, 128, 255])));
        assert_eq!(
            draft.formatting.runs.unwrap()[0].width_percent.as_deref(),
            Some(&50.)
        );
    }

    #[test]
    fn inherited_and_mixed_colors_have_different_swatch_states() {
        let mut draft = GuiSketchTextDraft {
            content: "ab".into(),
            ..Default::default()
        };
        assert_eq!(current_rgb(&draft), Some(None));
        assert!(apply_palette_color(&mut draft, [0, 128, 255]));
        draft.formatting.runs.as_mut().unwrap()[0].end = 1;
        assert_eq!(current_rgb(&draft), None);
    }
}
