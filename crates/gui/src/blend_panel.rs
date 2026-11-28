// SPDX-License-Identifier: GPL-2.0-only
//! Dockable properties for Fillet v2 and Chamfer blend drafts.
use super::*;
use crate::fillet_panel::PartialFilletModal;
use crate::{GuiChamferV2Action, GuiChamferV2Preview, GuiFilletV2Action, GuiFilletV2Preview};

impl BlendDialog {
    pub(super) fn reset_chamfer_v2(&mut self, state: &GuiCommandState) {
        let pick_seq = state.chamfer_v2_pick.as_ref().map_or(0, |p| p.sequence);
        self.chamfer_v2.reset_new(pick_seq);
    }

    pub(super) fn hydrate_chamfer_v2(
        &mut self,
        state: &GuiCommandState,
        feature: oblikovati_model::feature::FeatureId,
        chamfer: bool,
    ) {
        let pick_seq = state.chamfer_v2_pick.as_ref().map_or(0, |p| p.sequence);
        if chamfer
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.chamfer_v2_edits.get(&feature))
        {
            self.radius = seed.distance.clone();
            self.chamfer_v2.hydrate_edit(seed, pick_seq);
        } else {
            self.chamfer_v2.reset_new(pick_seq);
        }
    }

    pub(super) fn clear_chamfer_v2(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.chamfer
            && (self.chamfer_v2.attempted.is_some()
                || self.chamfer_v2.face_pick_active
                || self.chamfer_v2.tab == crate::GuiChamferTab::Partial)
        {
            actions.push(GuiShellAction::ChamferV2(GuiChamferV2Action::Clear));
        }
        self.chamfer_v2.attempted = None;
        self.chamfer_v2.submitted_close = None;
        self.chamfer_v2.submitted_apply = None;
        self.chamfer_v2.failure_modal = None;
    }

    pub(super) fn sync_chamfer_v2(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self
            .chamfer_v2
            .submitted_close
            .as_ref()
            .is_some_and(|req| state.document_generation != Some(req.generation))
        {
            self.close(actions);
        }
        if self
            .chamfer_v2
            .submitted_apply
            .as_ref()
            .is_some_and(|req| state.document_generation != Some(req.generation))
        {
            self.chamfer_v2.reset_after_apply();
        }
    }

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
        self.chamfer_v2
            .sync_ambient_edges(state, self.edit.is_some());
        self.chamfer_v2.consume_pick(state);

        let height = self
            .chamfer_v2
            .fitted_height(ui)
            .min((area.bottom - area.top - 28.0).max(160.0));
        crate::widget_abi::fit_viewport_properties(ui, "Chamfer", area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Chamfer", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            self.chamfer_v2.draw_body(
                ui,
                state,
                &mut self.radius,
                self.edit.is_some(),
                self.edited_edges,
                actions,
            );
            self.refresh_chamfer_v2_preview(state, actions);
            ui.separator();
            self.draw_chamfer_v2_footer(ui, state, actions);
        });
        self.chamfer_v2.draw_failure_modal(ui);
        if !open {
            self.close(actions);
        }
    }

    fn refresh_chamfer_v2_preview(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(document) = self.document else {
            return;
        };
        let feature = self.edit.map(|(id, _)| id);
        let request = self
            .chamfer_v2
            .request(document, state, feature, &self.radius)
            .filter(|req| {
                GuiShellAction::ChamferV2(GuiChamferV2Action::Preview(Box::new(req.clone())))
                    .dispatchable(state)
            });
        if self.chamfer_v2.attempted != request {
            self.chamfer_v2.attempted = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::ChamferV2(GuiChamferV2Action::Preview(
                    Box::new(req),
                ))),
                None => actions.push(GuiShellAction::ChamferV2(GuiChamferV2Action::Clear)),
            }
        }
    }

    fn draw_chamfer_v2_footer(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let local_ok = self.chamfer_v2.local_error(&self.radius).is_none();
        let feature = self.edit.map(|(id, _)| id);
        let req = self
            .document
            .and_then(|doc| self.chamfer_v2.request(doc, state, feature, &self.radius));
        let has_edges = req.as_ref().is_some_and(|r| !r.edges.is_empty());
        let commit_ready = local_ok
            && req.as_ref().is_some_and(|r| {
                GuiShellAction::ChamferV2(GuiChamferV2Action::Commit(Box::new(r.clone())))
                    .dispatchable(state)
            });
        let preview_failed = state.chamfer_v2_preview.as_ref().and_then(|p| {
            req.as_ref()
                .filter(|r| p.request == **r)
                .and_then(|_| match &p.result {
                    GuiChamferV2Preview::Failed(msg) => Some(msg.clone()),
                    _ => None,
                })
        });
        let can_ok = local_ok && has_edges && (commit_ready || preview_failed.is_some());
        let accepted = crate::theme::property_accept(ui, can_ok)
            || (can_ok && ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Enter));
        if accepted && let Some(req) = req.clone() {
            if let Some(msg) = preview_failed {
                self.chamfer_v2.failure_modal = Some(msg);
            } else if commit_ready {
                self.chamfer_v2.submitted_close = Some(req.clone());
                actions.push(GuiShellAction::ChamferV2(GuiChamferV2Action::Commit(
                    Box::new(req),
                )));
            }
        }
        ui.same_line_with_spacing(0.0, 6.0);
        let can_apply = self.edit.is_none() && commit_ready;
        let _dis = ui.begin_disabled(!can_apply);
        if ui.button("Apply##chamfer-apply")
            && can_apply
            && let Some(req) = req
        {
            self.chamfer_v2.submitted_apply = Some(req.clone());
            actions.push(GuiShellAction::ChamferV2(GuiChamferV2Action::Apply(
                Box::new(req),
            )));
        }
        drop(_dis);
        ui.same_line_with_spacing(0.0, 6.0);
        if crate::theme::property_cancel(ui)
            || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape))
        {
            self.close(actions);
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
