// SPDX-License-Identifier: GPL-2.0-only
//! The bottom status line: the current prompt, and how much an edited sketch still needs.
use crate::GuiFrameRequest;
use imgui::{Ui, WindowFlags};

pub(crate) const STATUS_BAR_HEIGHT: f32 = 22.;

/// Draw the status line across the window's foot; example: `Ready` and `4 dimensions needed`.
pub(crate) fn status_bar(
    ui: &Ui,
    request: &GuiFrameRequest,
    alias: Option<&str>,
    tool_prompt: Option<&str>,
) {
    let [width, height] = ui.io().display_size;
    // The status line is shorter than ImGui's default minimum window height (#4141).
    let _minimum = ui.push_style_var(imgui::StyleVar::WindowMinSize([1., 1.]));
    ui.window("##status-bar")
        .position([0., height - STATUS_BAR_HEIGHT], imgui::Condition::Always)
        .size([width, STATUS_BAR_HEIGHT], imgui::Condition::Always)
        .flags(flags())
        .build(|| {
            crate::widget_abi::cursor_position(ui, [8., 4.]);
            ui.text(command_status(
                tool_prompt.or(request.status.as_deref()),
                alias,
            ));
            let _numeric = crate::ui_scale::numeric_font(ui);
            if let Some(needed) = request.commands.dimensions_needed {
                let text = dimensions_needed_text(needed);
                let advance = ui.calc_text_size(&text)[0];
                crate::widget_abi::cursor_position(ui, [width - advance - 12., 4.]);
                ui.text(text);
            } else if let Some(text) = selected_entities_text(
                request.commands.selected_face_appearances.len(),
                request.commands.selected_edge_count,
            ) {
                let advance = ui.calc_text_size(&text)[0];
                crate::widget_abi::cursor_position(ui, [width - advance - 12., 4.]);
                ui.text(text);
            }
        });
}
fn selected_entities_text(faces: usize, edges: usize) -> Option<String> {
    let face_text = selection_count_text(faces, "Face", "Faces");
    let edge_text = selection_count_text(edges, "Edge", "Edges");
    match (face_text, edge_text) {
        (Some(faces), Some(edges)) => Some(format!("{faces}, {edges}")),
        (Some(text), None) | (None, Some(text)) => Some(text),
        (None, None) => None,
    }
}
fn selection_count_text(count: usize, singular: &str, plural: &str) -> Option<String> {
    match count {
        0 => None,
        1 => Some(format!("1 {singular}")),
        count => Some(format!("{count} {plural}")),
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
        assert_eq!(selected_entities_text(0, 0), None);
        assert_eq!(selected_entities_text(0, 1).as_deref(), Some("1 Edge"));
        assert_eq!(selected_entities_text(0, 2).as_deref(), Some("2 Edges"));
        assert_eq!(selected_entities_text(1, 0).as_deref(), Some("1 Face"));
        assert_eq!(
            selected_entities_text(2, 1).as_deref(),
            Some("2 Faces, 1 Edge")
        );
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
