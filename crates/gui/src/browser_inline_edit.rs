// SPDX-License-Identifier: GPL-2.0-only
//! Shared identity and input handling for browser inline names (#4118).
use imgui::Ui;

#[derive(Debug, Default)]
pub(crate) struct BrowserInlineEdit {
    pending: Option<(u64, Vec<u64>)>,
    pub(super) draft: Option<BrowserInlineDraft>,
}
#[derive(Debug)]
pub(super) struct BrowserInlineDraft {
    pub document: u64,
    pub id: u64,
    pub name: String,
    pub focus: bool,
    pub error: Option<String>,
}
impl BrowserInlineEdit {
    pub(super) fn request(&mut self, document: u64, ids: impl Iterator<Item = u64>) {
        self.pending = Some((document, ids.collect()));
        self.draft = None;
    }
    pub(super) fn begin(&mut self, document: u64, id: u64, name: &str) {
        self.pending = None;
        self.draft = Some(BrowserInlineDraft {
            document,
            id,
            name: name.into(),
            focus: true,
            error: None,
        });
    }
    pub(super) fn observe(
        &mut self,
        document: Option<u64>,
        rows: impl FnOnce() -> Vec<(u64, String)>,
    ) {
        if self.pending.is_none() && self.draft.is_none() {
            return;
        }
        let rows = rows();
        if self.draft.as_ref().is_some_and(|draft| {
            Some(draft.document) != document || !rows.iter().any(|row| row.0 == draft.id)
        }) {
            self.draft = None;
        }
        let Some((owner, before)) = self.pending.take() else {
            return;
        };
        if Some(owner) != document {
            return;
        }
        if let Some((id, name)) = rows.iter().find(|row| !before.contains(&row.0)) {
            self.begin(owner, *id, name);
        }
    }
    pub(super) fn needs_open_branch(&self) -> bool {
        self.draft.is_some() || self.pending.is_some()
    }
    pub(super) fn input(
        &mut self,
        ui: &Ui,
        document: Option<u64>,
        id: u64,
        prefix: &str,
        label: &str,
    ) -> Option<(bool, bool)> {
        let draft = self.draft.as_mut()?;
        if draft.id != id || Some(draft.document) != document {
            return None;
        }
        let _id = ui.push_id(format!("{prefix}-{id}"));
        if std::mem::take(&mut draft.focus) {
            ui.set_keyboard_focus_here();
        }
        let accept = ui
            .input_text(label, &mut draft.name)
            .auto_select_all(true)
            .enter_returns_true(true)
            .build();
        Some((accept, ui.is_key_pressed_no_repeat(imgui::Key::Escape)))
    }
}
