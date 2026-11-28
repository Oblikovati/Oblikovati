// SPDX-License-Identifier: GPL-2.0-only
//! Dockable properties for Fillet v2 and Chamfer blend drafts.
use super::*;
use crate::fillet_panel::PartialFilletModal;
use crate::{GuiFilletV2Action, GuiFilletV2Preview};

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
        if self.chamfer {
            self.draw_chamfer_panel(ui, state, actions, area);
            return;
        }
        self.draw_fillet_panel(ui, state, actions, area);
    }

    fn draw_chamfer_panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
    ) {
        let style = ui.clone_style();
        let height = 3. * ui.frame_height_with_spacing() + 2. * style.window_padding[1] + 28.;
        crate::widget_abi::fit_viewport_properties(ui, "Chamfer", area, [280., height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Chamfer", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            self.draw_chamfer_properties(ui, state);
            self.refresh_preview(state, actions);
            ui.separator();
            self.controls(ui, state, actions);
        });
        if !open {
            self.close(actions);
        }
    }

    fn draw_chamfer_properties(&mut self, ui: &Ui, state: &GuiCommandState) {
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
        crate::theme::property_label(ui, "Distance");
        let _numeric = crate::ui_scale::numeric_font(ui);
        let _border = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.));
        if ui.input_text("##blend-value", &mut self.radius).build() {
            self.submitted = None;
        }
    }

    fn draw_fillet_panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
    ) {
        self.v2
            .sync_ambient_edges(state, &self.radius, self.edit.is_some());
        self.v2.consume_pick(state, &self.radius);

        let height = ((area.bottom - area.top) - 24.0).clamp(280.0, 520.0);
        crate::widget_abi::fit_viewport_properties(ui, "Fillet", area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Fillet", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let footer_h = 38.0;
            let body_h = (ui.content_region_avail()[1] - footer_h).max(140.0);
            ui.child_window("##fillet-body")
                .size([0.0, body_h])
                .build(|| {
                    self.v2.draw_body(
                        ui,
                        state,
                        &mut self.radius,
                        self.edit.is_some(),
                        self.edited_edges,
                        actions,
                    );
                });
            if self.v2.is_v2_active() {
                self.refresh_v2_preview(state, actions);
            } else {
                self.refresh_preview(state, actions);
            }
            ui.separator();
            if self.v2.is_v2_active() {
                self.draw_v2_footer(ui, state, actions);
            } else if self.v2.local_error(&self.radius).is_some() {
                let _ = crate::theme::property_accept(ui, false);
                ui.same_line_with_spacing(0.0, 6.0);
                if crate::theme::property_cancel(ui)
                    || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape))
                {
                    self.close(actions);
                }
            } else {
                self.controls(ui, state, actions);
            }
        });
        if self.v2.draw_partial_modal(ui, actions) {
            self.refresh_v2_preview(state, actions);
        }
        if !open {
            self.close(actions);
        }
    }

    fn refresh_v2_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let Some(document) = self.document else {
            return;
        };
        let feature = self.edit.map(|(id, _)| id);
        let request = self.v2.request(document, state, feature).filter(|req| {
            GuiShellAction::FilletV2(GuiFilletV2Action::Preview(Box::new(req.clone())))
                .dispatchable(state)
        });
        if self.v2.attempted_v2 != request {
            self.v2.attempted_v2 = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::FilletV2(GuiFilletV2Action::Preview(
                    Box::new(req),
                ))),
                None => actions.push(GuiShellAction::FilletV2(GuiFilletV2Action::Clear)),
            }
        }
        if self.v2.pending_commit_after_partial
            && let Some(req) = self.v2.request(document, state, feature)
            && state
                .fillet_v2_preview
                .as_ref()
                .is_some_and(|p| p.request == req && p.result == GuiFilletV2Preview::Ready)
        {
            self.v2.pending_commit_after_partial = false;
            self.v2.submitted_v2 = Some(req.clone());
            actions.push(GuiShellAction::FilletV2(GuiFilletV2Action::Commit(
                Box::new(req),
            )));
        }
    }

    fn draw_v2_footer(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let local_ok = self.v2.local_error(&self.radius).is_none();
        let feature = self.edit.map(|(id, _)| id);
        let req = local_ok
            .then(|| {
                self.document
                    .and_then(|doc| self.v2.request(doc, state, feature))
            })
            .flatten()
            .filter(|req| {
                GuiShellAction::FilletV2(GuiFilletV2Action::Commit(Box::new(req.clone())))
                    .dispatchable(state)
            });
        let can_ok = req.is_some();
        let accepted = crate::theme::property_accept(ui, can_ok)
            || (can_ok && ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Enter));
        if accepted && let Some(req) = req {
            if let Some(preview) = &state.fillet_v2_preview
                && let GuiFilletV2Preview::PartialFailure {
                    failed_count,
                    total_count,
                    kept_sets,
                } = &preview.result
            {
                self.v2.partial_modal = Some(PartialFilletModal {
                    failed_count: *failed_count,
                    total_count: *total_count,
                    kept_sets: kept_sets.clone(),
                    accept_successful: true,
                });
            } else {
                self.v2.submitted_v2 = Some(req.clone());
                actions.push(GuiShellAction::FilletV2(GuiFilletV2Action::Commit(
                    Box::new(req),
                )));
            }
        }
        ui.same_line_with_spacing(0.0, 6.0);
        if crate::theme::property_cancel(ui)
            || (self.v2.partial_modal.is_none()
                && ui.is_window_focused()
                && ui.is_key_pressed_no_repeat(imgui::Key::Escape))
        {
            self.close(actions);
        }
    }
}
