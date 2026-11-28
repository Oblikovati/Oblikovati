// SPDX-License-Identifier: GPL-2.0-only
//! Dockable properties for Fillet and Chamfer blend drafts.
use super::*;

impl BlendDialog {
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
    ) {
        self.sync_before_docking(state, actions);
        if self.document.is_none() {
            return;
        }
        let title = if self.chamfer { "Chamfer" } else { "Fillet" };
        let style = ui.clone_style();
        // The title bar and property rows each take one frame, followed by the footer.
        let height = 3. * ui.frame_height_with_spacing() + 2. * style.window_padding[1] + 28.;
        crate::widget_abi::fit_viewport_properties(ui, title, area, [280., height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, title, &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
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
        crate::feature_properties::draw_pick_pill(ui, &selection, count > 0);
        crate::theme::property_label(ui, label);
        let _numeric = crate::ui_scale::numeric_font(ui);
        let _border = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.));
        if ui.input_text("##blend-value", &mut self.radius).build() {
            self.submitted = None;
        }
    }
}
