// SPDX-License-Identifier: GPL-2.0-only
//! Navigate from an accepted Extrude definition into its source sketch.
use super::{ExtrudeDialog, GuiCommandState, GuiShellAction};
use crate::application::document_creation::workspaces::WorkspaceArea;
use crate::dockable_window::DockableWindow;
use imgui::{StyleColor, Ui};

impl ExtrudeDialog {
    pub(super) fn source_edit_active(&self, state: &GuiCommandState) -> bool {
        self.source_edit
            && self.document == state.document
            && self.sketch == state.active_sketch
            && state.authoring == crate::GuiAuthoringContext::Sketch
    }

    pub(super) fn source_breadcrumb_panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: WorkspaceArea,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.size = Default::default();
        let height = ui.frame_height() * 2. + ui.clone_style().window_padding[1] * 2.;
        super::panel::fit_properties(ui, area, height);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Extrude", &mut open)
            .scroll_bar(false)
            .scrollable(false)
            .build(|| {
                crate::theme::floating_property_shadow(ui);
                self.breadcrumb(ui, state, actions);
            });
        if !open {
            self.close(actions);
        }
    }

    pub(super) fn breadcrumb(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        let Some(browser) = &state.browser else {
            return false;
        };
        let feature = browser
            .features
            .iter()
            .find(|row| Some(row.id) == self.edit_feature);
        let sketch = browser
            .sketches
            .iter()
            .find(|row| Some(row.id) == self.sketch);
        ui.align_text_to_frame_padding();
        let (Some(feature), Some(sketch), Some(document)) = (feature, sketch, self.document) else {
            if let Some(sketch) = sketch {
                ui.text(format!("New Extrusion › {}", sketch.name));
            }
            return false;
        };

        ui.text(&feature.name);
        if !self.source_edit {
            underline(ui);
        }
        ui.same_line_with_spacing(0., 4.);
        ui.text("›");
        ui.same_line_with_spacing(0., 4.);

        let edit = GuiShellAction::EditSketch {
            document,
            sketch: sketch.id,
        };
        if !edit.dispatchable(state) {
            ui.text(&sketch.name);
            if self.source_edit {
                underline(ui);
            }
            return false;
        }

        let _ink = ui.push_style_color(StyleColor::Text, ui.style_color(StyleColor::NavHighlight));
        let _id = ui.push_id("extrude-source-sketch");
        if !ui
            .selectable_config(&sketch.name)
            .size(ui.calc_text_size(&sketch.name))
            .build()
        {
            return false;
        }
        self.source_edit = true;
        self.distance_editor = Default::default();
        self.reference.clear_picks();
        if std::mem::take(&mut self.previewed) {
            actions.push(GuiShellAction::PreviewExtrude(None));
        }
        actions.push(edit);
        true
    }
}

fn underline(ui: &Ui) {
    let [right, bottom] = ui.item_rect_max();
    crate::theme::tab_underline_at(ui, ui.item_rect_min(), [right, bottom + 3.]);
}
