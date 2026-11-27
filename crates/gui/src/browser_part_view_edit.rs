// SPDX-License-Identifier: GPL-2.0-only
//! Inline part-view names retain their draft across duplicate-name choices (#4118).
use super::*;
use oblikovati_api::contract::PartDesignView as _;
use oblikovati_model::part_design_view::{PartDesignView, PartDesignViews};

#[derive(Debug, Default)]
pub(crate) struct PartViewEdit {
    core: browser_inline_edit::BrowserInlineEdit,
    selected: Option<(u64, u64)>,
    duplicate: bool,
}
impl PartViewEdit {
    pub(super) fn request_copy(&mut self, document: u64, views: &PartDesignViews) {
        self.core
            .request(document, views.list().iter().map(|view| view.id()));
    }
    pub(super) fn observe(&mut self, document: Option<u64>, views: &PartDesignViews) {
        self.core.observe(document, || {
            views
                .list()
                .iter()
                .map(|view| (view.id(), view.name()))
                .collect()
        });
        if self
            .selected
            .is_some_and(|(owner, id)| Some(owner) != document || views.get(id).is_err())
        {
            self.selected = None;
        }
    }
    pub(super) fn needs_open_branch(&self) -> bool {
        self.core.needs_open_branch()
    }
    pub(super) fn select(&mut self, document: u64, id: u64) {
        self.selected = Some((document, id));
    }
    pub(super) fn rename_key(&mut self, ui: &Ui, document: Option<u64>, entry: &PartDesignView) {
        let Some(document) = document else {
            return;
        };
        if self.core.draft.is_none()
            && self.selected == Some((document, entry.id()))
            && !entry.built_in()
            && ui.is_window_focused()
            && ui.is_key_pressed_no_repeat(imgui::Key::F2)
        {
            self.core.begin(document, entry.id(), &entry.name());
        }
    }
    pub(super) fn draw(
        &mut self,
        ui: &Ui,
        document: Option<u64>,
        entry: &PartDesignView,
        views: &PartDesignViews,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        let Some((accept, cancel)) = self.core.input(
            ui,
            document,
            entry.id(),
            "part-view-edit",
            "##part-view-name",
        ) else {
            return false;
        };
        if accept && !self.duplicate {
            self.accept(ui, entry, views, actions);
        } else if cancel && !self.duplicate {
            self.core.draft = None;
        }
        self.duplicate_dialog(ui);
        true
    }
    fn accept(
        &mut self,
        ui: &Ui,
        entry: &PartDesignView,
        views: &PartDesignViews,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(draft) = self.core.draft.as_ref() else {
            return;
        };
        if draft.name.trim().is_empty() {
            self.core.draft = None;
            return;
        }
        if draft.name.contains('\0')
            || views
                .list()
                .iter()
                .any(|row| row.id() != draft.id && row.name() == draft.name)
        {
            self.duplicate = true;
            ui.open_popup("View Name Already Used");
            return;
        }
        if draft.name != entry.name() {
            actions.push(GuiShellAction::BrowserChoice {
                document: draft.document,
                choice: crate::GuiBrowserChoice::RenamePartView {
                    id: draft.id,
                    name: draft.name.clone(),
                },
            });
        }
        self.core.draft = None;
    }
    fn duplicate_dialog(&mut self, ui: &Ui) {
        if !self.duplicate {
            return;
        }
        let Some(_popup) = ui.begin_modal_popup("View Name Already Used") else {
            return;
        };
        ui.text_wrapped("The new name is already used.");
        if crate::widget_abi::button(ui, "Edit", [80., 0.]) {
            if let Some(draft) = self.core.draft.as_mut() {
                draft.focus = true;
            }
            self.duplicate = false;
            ui.close_current_popup();
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "Cancel", [80., 0.])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape)
        {
            self.core.draft = None;
            self.duplicate = false;
            ui.close_current_popup();
        }
    }
}
