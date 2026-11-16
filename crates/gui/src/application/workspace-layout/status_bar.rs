// SPDX-License-Identifier: GPL-2.0-only
//! The bottom status line: the current prompt, and how much an edited sketch still needs.
use crate::GuiFrameRequest;
use imgui::{Ui, WindowFlags};

pub(crate) const STATUS_BAR_HEIGHT: f32 = 22.;

/// Draw the status line across the window's foot; example: `Ready` and `4 dimensions needed`.
pub(crate) fn status_bar(ui: &Ui, request: &GuiFrameRequest, alias: Option<&str>) {
    let [width, height] = ui.io().display_size;
    ui.window("##status-bar")
        .position([0., height - STATUS_BAR_HEIGHT], imgui::Condition::Always)
        .size([width, STATUS_BAR_HEIGHT], imgui::Condition::Always)
        .flags(flags())
        .build(|| {
            crate::widget_abi::cursor_position(ui, [8., 4.]);
            ui.text(command_status(request.status.as_deref(), alias));
            if let Some(needed) = request.commands.dimensions_needed {
                let text = dimensions_needed_text(needed);
                let advance = ui.calc_text_size(&text)[0];
                crate::widget_abi::cursor_position(ui, [width - advance - 12., 4.]);
                ui.text(text);
            } else if let Some(text) = selected_edge_text(request.commands.selected_edge_count) {
                let advance = ui.calc_text_size(&text)[0];
                crate::widget_abi::cursor_position(ui, [width - advance - 12., 4.]);
                ui.text(text);
            }
        });
}
fn selected_edge_text(count: usize) -> Option<String> {
    match count {
        0 => None,
        1 => Some("1 Edge".into()),
        count => Some(format!("{count} Edges")),
    }
}
fn command_status<'a>(prompt: Option<&'a str>, alias: Option<&str>) -> std::borrow::Cow<'a, str> {
    match alias {
        Some(text) => format!("Command alias input: {text}").into(),
        None => prompt.unwrap_or("Ready").into(),
    }
}
fn flags() -> WindowFlags {
    WindowFlags::NO_DECORATION
        | WindowFlags::NO_MOVE
        | WindowFlags::NO_DOCKING
        | WindowFlags::NO_SAVED_SETTINGS
        | WindowFlags::NO_BRING_TO_FRONT_ON_FOCUS
        | WindowFlags::NO_FOCUS_ON_APPEARING
        | WindowFlags::NO_NAV
        // It only reads: the pointer passes through to the view beneath its edge.
        | WindowFlags::NO_INPUTS
}
/// A count of free directions reads as the dimensions it would take to remove them.
pub(crate) fn dimensions_needed_text(needed: usize) -> String {
    match needed {
        0 => "Fully Constrained".into(),
        1 => "1 dimension needed".into(),
        count => format!("{count} dimensions needed"),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn edge_selection_status_reports_count() {
        assert_eq!(selected_edge_text(0), None);
        assert_eq!(selected_edge_text(1).as_deref(), Some("1 Edge"));
        assert_eq!(selected_edge_text(2).as_deref(), Some("2 Edges"));
    }
    #[test]
    fn status_feedback_reads_the_existing_alias_before_the_tool_prompt() {
        assert_eq!(
            command_status(Some("Select first axis point"), Some("8")),
            "Command alias input: 8"
        );
        assert_eq!(
            command_status(Some("Select first axis point"), None),
            "Select first axis point"
        );
        assert_eq!(command_status(None, None), "Ready");
    }
    #[test]
    fn the_count_reads_as_dimensions_with_a_named_finished_state() {
        assert_eq!(dimensions_needed_text(0), "Fully Constrained");
        assert_eq!(dimensions_needed_text(1), "1 dimension needed");
        assert_eq!(dimensions_needed_text(215), "215 dimensions needed");
    }
}
