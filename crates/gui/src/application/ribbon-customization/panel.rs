// SPDX-License-Identifier: GPL-2.0-only
//! Render applied commands through the same actions as the built-in ribbon.
use super::{Command, RibbonCustomization};
use crate::application::viewport_orientation::ZoomAction;
use crate::icons::{GuiRibbonIcons, IconGlyph};
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;

impl RibbonCustomization {
    /// Report whether the ribbon still shows the applied panel, such as on Tools during Apply.
    pub(crate) fn has_panel(&self, tab: crate::ribbon::RibbonTab) -> bool {
        self.open.is_open() && self.draft.retained_panels.contains(&tab)
            || self
                .draft
                .applied(tab)
                .iter()
                .any(|entry| entry.command != Command::Separator)
    }

    pub(crate) fn draw_panel(
        &mut self,
        ui: &Ui,
        tab: crate::ribbon::RibbonTab,
        expanded: Option<bool>,
        state: &GuiCommandState,
        icons: &crate::icons::GuiRibbonIcons,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        let entries = self.draft.applied(tab).to_vec();
        let mut right_clicked = false;
        let origin = ui.cursor_pos();
        let normal = icons.appearance == crate::ribbon_command_layout::RibbonAppearance::Normal;
        for (count, (index, entry)) in entries
            .iter()
            .enumerate()
            .filter(|(_, entry)| expanded.is_none_or(|shown| entry.expanded == shown))
            .enumerate()
        {
            if normal && count > 0 {
                ui.same_line();
            } else if !normal {
                icons.cursor(ui, origin, icons.appearance.cell(count).0);
            }
            let _id = ui.push_id(format!("custom-command-{index}"));
            let label = if entry.text || !normal {
                entry.command.label()
            } else {
                "##icon-only"
            };
            let size = if normal {
                if entry.large { [90., 58.] } else { [90., 26.] }
            } else {
                icons.appearance.cell(count).1
            };
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
                Command::FinishSketch => {
                    icons.command_button(
                        ui,
                        label,
                        GuiShellAction::FinishSketch,
                        state,
                        size,
                        actions,
                    );
                }
                Command::CancelSketch => {
                    let _disabled = ui.begin_disabled(true);
                    let caption = if label == "##icon-only" { "×" } else { label };
                    crate::widget_abi::button(ui, caption, size);
                    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
                        ui.tooltip_text("Cancel Sketch is unavailable in this build.");
                    }
                }
                Command::Zoom(zoom) => {
                    if zoom_button(ui, label, size, state, icons, zoom) {
                        self.pending_zoom = Some(zoom);
                    }
                }
                Command::Annotations => {
                    let _disabled = ui.begin_disabled(true);
                    icons.button(ui, label, IconGlyph::Annotation, size);
                }
                Command::Drawing2D => {
                    let _disabled = ui.begin_disabled(true);
                    icons.button(ui, label, IconGlyph::NewDrawing, size);
                    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
                        ui.tooltip_text("2-D Drawing is unavailable in this build.");
                    }
                }
                Command::Separator => separator(ui, if normal { [8., 58.] } else { size }),
            }
            if entry.command != Command::Separator {
                right_clicked |= self.command_menu(ui, tab, entry.command, entry.expanded);
            }
        }
        right_clicked
    }
    fn command_menu(
        &mut self,
        ui: &Ui,
        tab: crate::ribbon::RibbonTab,
        command: Command,
        expanded: bool,
    ) -> bool {
        let right = ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED)
            && ui.is_mouse_clicked(imgui::MouseButton::Right);
        if right {
            ui.open_popup("##custom-command-context");
        }
        if let Some(_popup) = ui.begin_popup("##custom-command-context") {
            let label = if expanded {
                "Move to Main Panel"
            } else {
                "Move to Expanded Panel"
            };
            if ui.menu_item(label)
                && let Err(error) = self.relocate_command(tab, command, !expanded)
            {
                self.command_error = Some(error);
                self.command_error_opening = true;
            }
        }
        right
    }
}
fn zoom_button(
    ui: &Ui,
    label: &str,
    size: [f32; 2],
    state: &GuiCommandState,
    icons: &GuiRibbonIcons,
    zoom: ZoomAction,
) -> bool {
    let supported = state.view_axes.is_some()
        && matches!(
            state.authoring,
            crate::GuiAuthoringContext::Part | crate::GuiAuthoringContext::Sketch
        );
    let _disabled = ui.begin_disabled(!supported);
    let glyph = match zoom {
        ZoomAction::All => IconGlyph::ZoomAll,
        ZoomAction::Window => IconGlyph::ZoomWindow,
        ZoomAction::Zoom | ZoomAction::Selected => IconGlyph::Find,
    };
    let pressed = icons.button(ui, label, glyph, size);
    if label == "##icon-only" && ui.is_item_hovered() {
        ui.tooltip_text(zoom.label());
    }
    pressed
}
fn separator(ui: &Ui, size: [f32; 2]) {
    let [x, y] = ui.cursor_screen_pos();
    ui.get_window_draw_list()
        .add_line(
            [x + 3., y + 2.],
            [x + 3., y + size[1] - 2.],
            ui.style_color(imgui::StyleColor::Separator),
        )
        .build();
    ui.dummy([8., size[1]]);
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
