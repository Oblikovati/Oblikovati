// SPDX-License-Identifier: GPL-2.0-only
//! The bottom status line: the current prompt, and how much an edited sketch still needs.
use crate::GuiFrameRequest;
use imgui::{StyleColor, Ui, WindowFlags};
use oblikovati_api::types::{TOKEN_CHROME_WINDOW_BG, TOKEN_DIMENSION_DRIVEN};

pub(crate) const STATUS_BAR_HEIGHT: f32 = 24.;
const STATUS_LEFT_INSET: f32 = 10.;
const STATUS_RIGHT_INSET: f32 = 12.;
const DOT_RADIUS: f32 = 3.;
const CELL_GAP: f32 = 9.;

/// Draw the status line across the window's foot; example: `Ready` and `4 dimensions needed`.
pub(crate) fn status_bar(
    ui: &Ui,
    request: &GuiFrameRequest,
    home: bool,
    alias: Option<&str>,
    tool_prompt: Option<&str>,
) {
    let [width, height] = ui.io().display_size;
    let _colors = request.commands.theme.as_ref().map(|theme| {
        let palette = theme.palette();
        [
            (StyleColor::WindowBg, TOKEN_CHROME_WINDOW_BG),
            (StyleColor::Text, TOKEN_DIMENSION_DRIVEN),
        ]
        .map(|(style, token)| {
            let color = &palette[&token];
            crate::widget_abi::push_style_color(ui, style, [color.r, color.g, color.b, color.a])
        })
    });
    // The status line is shorter than ImGui's default minimum window height (#4141).
    let _minimum = crate::widget_abi::window_min_size(ui, [1., 1.]);
    crate::widget_abi::window_geometry(
        ui,
        [0., height - STATUS_BAR_HEIGHT],
        [width, STATUS_BAR_HEIGHT],
        imgui::Condition::Always,
    );
    ui.window("##status-bar").flags(flags()).build(|| {
        let prompt = if home {
            command_status(request.status.as_deref(), None)
        } else {
            command_status(
                tool_prompt
                    .or(request.commands.work_axis_prompt.as_deref())
                    .or(request.status.as_deref()),
                alias,
            )
        };
        let origin = ui.window_pos();
        let y = ((STATUS_BAR_HEIGHT - ui.current_font_size()) * 0.5).floor();
        crate::widget_abi::filled_circle(
            ui,
            [
                origin[0] + STATUS_LEFT_INSET + DOT_RADIUS,
                origin[1] + STATUS_BAR_HEIGHT / 2.,
            ],
            DOT_RADIUS,
            ui.style_color(StyleColor::NavHighlight),
        );
        let prompt_x = STATUS_LEFT_INSET + 2. * DOT_RADIUS + 7.;
        crate::widget_abi::cursor_position(ui, [prompt_x, y]);
        ui.text(&prompt);
        let prompt_right = prompt_x + ui.calc_text_size(&prompt)[0] + CELL_GAP;
        if !home {
            telemetry(ui, &request.commands, width, prompt_right);
        }
    });
}
fn telemetry(ui: &Ui, commands: &crate::GuiCommandState, width: f32, prompt_right: f32) {
    let _numeric = crate::ui_scale::mono_micro_font(ui);
    let units = commands.document_units.as_ref();
    let coordinates = commands
        .viewport_coordinates
        .zip(units)
        .map(|([x, y], units)| {
            let precision = units.length_display_precision.clamp(0, 6) as usize;
            format!("X {x:.precision$}  Y {y:.precision$}")
        });
    let zoom = commands
        .viewport_zoom_percent
        .map(|percent| format!("{percent}%"));
    let selection_or_constraint = commands.dimensions_needed.map_or_else(
        || {
            selected_entities_text(
                commands.selected_face_appearances.len(),
                commands.selected_edge_count,
            )
        },
        |needed| Some(dimensions_needed_text(needed)),
    );
    let dof = commands
        .dimensions_needed
        .map(|needed| format!("DOF {needed}"));
    let cells = [
        (coordinates.as_deref(), false),
        (dof.as_deref(), false),
        (
            selection_or_constraint.as_deref(),
            commands.dimensions_needed == Some(0),
        ),
        (units.map(|units| units.length_unit.as_str()), false),
        (zoom.as_deref(), false),
    ];
    draw_telemetry(ui, cells, width - STATUS_RIGHT_INSET, prompt_right);
}

fn draw_telemetry(
    ui: &Ui,
    cells: [(Option<&str>, bool); 5],
    trailing_edge: f32,
    prompt_right: f32,
) {
    let mut right = trailing_edge;
    let y = ((STATUS_BAR_HEIGHT - ui.current_font_size()) * 0.5).floor();
    for (cell, successful) in cells.into_iter().rev() {
        let Some(cell) = cell else { continue };
        let left = right - ui.calc_text_size(cell)[0];
        if left < prompt_right {
            continue;
        }
        if right < trailing_edge {
            telemetry_separator(ui, right + CELL_GAP);
        }
        crate::widget_abi::cursor_position(ui, [left, y]);
        let _color = successful.then(|| {
            crate::widget_abi::push_style_color(ui, StyleColor::Text, constraint_success_color(ui))
        });
        ui.text(cell);
        right = left - 2. * CELL_GAP;
    }
}

fn telemetry_separator(ui: &Ui, offset: f32) {
    let [left, top] = ui.window_pos();
    let x = left + offset;
    crate::widget_abi::line(
        ui,
        [x, top + 5.],
        [x, top + STATUS_BAR_HEIGHT - 5.],
        ui.style_color(StyleColor::Border),
    );
}

fn constraint_success_color(ui: &Ui) -> [f32; 4] {
    let [r, g, b, _] = ui.style_color(StyleColor::WindowBg);
    let surface = r.max(g).max(b);
    crate::theme::hex_rgb(if surface < 0.01 {
        0x4ade80
    } else if surface > 0.5 {
        0x16a34a
    } else {
        0x22c55e
    })
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
