// SPDX-License-Identifier: GPL-2.0-only
//! Render applied commands through the same actions as the built-in ribbon.
use super::{Command, RibbonCustomization};
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;

impl RibbonCustomization {
    pub(crate) fn draw_panel(
        &self,
        ui: &Ui,
        tab: crate::ribbon::RibbonTab,
        state: &GuiCommandState,
        icons: &crate::icons::GuiRibbonIcons,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let entries = self.draft.applied(tab);
        for (index, entry) in entries.iter().enumerate() {
            if index > 0 {
                ui.same_line();
            }
            let _id = ui.push_id(format!("custom-command-{index}"));
            let label = if entry.text {
                entry.command.label()
            } else {
                "##icon-only"
            };
            let size = if entry.large { [90., 58.] } else { [90., 26.] };
            match entry.command {
                Command::Sketches => sketch_visibility(ui, label, size, state, icons, actions),
                Command::Tool(tool) => icons.command_button(
                    ui,
                    label,
                    GuiShellAction::PlaceSketchTool(tool),
                    state,
                    size,
                    actions,
                ),
                Command::Constraint(constraint) => icons.command_button(
                    ui,
                    label,
                    GuiShellAction::ConstrainSketch(constraint),
                    state,
                    size,
                    actions,
                ),
                Command::Dimension => icons.command_button(
                    ui,
                    label,
                    GuiShellAction::Dimension(crate::GuiDimensionAction::Start),
                    state,
                    size,
                    actions,
                ),
                Command::Measure => icons.command_button(
                    ui,
                    label,
                    GuiShellAction::Inspect(crate::GuiInspectAction::StartMeasure),
                    state,
                    size,
                    actions,
                ),
                Command::Annotations => {
                    let disabled = ui.begin_disabled(true);
                    ui.button(label);
                    drop(disabled);
                }
                Command::Separator => separator(ui),
            }
        }
    }
}
fn separator(ui: &Ui) {
    let [x, y] = ui.cursor_screen_pos();
    ui.get_window_draw_list()
        .add_line(
            [x + 3., y + 2.],
            [x + 3., y + 56.],
            ui.style_color(imgui::StyleColor::Separator),
        )
        .build();
    ui.dummy([8., 58.]);
}
fn sketch_visibility(
    ui: &Ui,
    label: &str,
    size: [f32; 2],
    state: &GuiCommandState,
    icons: &crate::icons::GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    let sketches = state
        .browser
        .as_ref()
        .map_or(&[][..], |browser| &browser.sketches);
    let disabled = ui.begin_disabled(sketches.is_empty() || state.transaction_open);
    let pressed = icons.button(ui, label, crate::icons::IconGlyph::CreateSketch, size);
    drop(disabled);
    let Some(document) = state.document.filter(|_| pressed) else {
        return;
    };
    let visible = !sketches.iter().any(|row| row.visible);
    actions.extend(
        sketches
            .iter()
            .map(|row| GuiShellAction::SetSketchVisibility {
                document,
                sketch: row.id,
                visible,
            }),
    );
}
