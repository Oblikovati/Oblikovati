// SPDX-License-Identifier: GPL-2.0-only
//! Dockable properties for Fillet and Chamfer blend drafts.
use super::*;

impl BlendDialog {
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        origin: [f32; 2],
    ) {
        self.sync_before_docking(state, actions);
        if self.document.is_none() {
            return;
        }
        let title = if self.chamfer { "Chamfer" } else { "Fillet" };
        Self::fit_dockable(ui, title, origin, [280., 138.]);
        let mut open = true;
        Self::dockable_window(ui, title, &mut open).build(|| {
            self.draw_properties(ui, state);
            self.refresh_preview(state, actions);
            ui.separator();
            self.controls(ui, state, actions);
        });
        if !open {
            self.close(actions);
        }
    }
    fn draw_properties(&mut self, ui: &Ui, state: &GuiCommandState) {
        let label = if self.chamfer { "Distance" } else { "Radius" };
        let count = if self.edit.is_some() {
            self.edited_edges
        } else {
            state.selected_edge_count
        };
        let selection = match count {
            0 => "Select Edges".to_owned(),
            1 => "1 Edge".to_owned(),
            count => format!("{count} Edges"),
        };
        crate::theme::property_label(ui, "Edges");
        ui.text(selection);
        crate::theme::property_label(ui, label);
        let _numeric = crate::ui_scale::numeric_font(ui);
        if ui.input_text("##blend-value", &mut self.radius).build() {
            self.submitted = None;
        }
    }
}
