// SPDX-License-Identifier: GPL-2.0-only
//! The File Open Options dialog: how one recent file would open.
use super::GuiRecentDocument;
use crate::{GuiShellAction, GuiTemplateKind};
use imgui::Ui;

const TITLE: &str = "File Open Options";
const LAST_ACTIVE: &str = "Last Active";
const PRIMARY: &str = "Primary";
const NOT_AVAILABLE: &str = "This opening mode is not available in this build.";

/// One file's choices while the dialog is up; Cancel simply drops them.
#[derive(Default)]
pub(super) struct FileOpenOptions {
    target: Option<OpenTarget>,
    opening: bool,
}
struct OpenTarget {
    path: String,
    name: String,
    kind: GuiTemplateKind,
    modes: Vec<&'static str>,
    mode: usize,
    views: Vec<String>,
    view: usize,
    skip_unresolved: bool,
}
impl OpenTarget {
    fn of(row: &GuiRecentDocument) -> Self {
        let mut views = vec![LAST_ACTIVE.to_owned(), PRIMARY.to_owned()];
        views.extend(row.design_views.iter().cloned());
        Self {
            path: row.path.clone(),
            name: row.name.clone(),
            kind: row.kind,
            modes: modes_of(row.kind),
            mode: 0,
            views,
            view: 0,
            skip_unresolved: false,
        }
    }
    fn mode_name(&self) -> &'static str {
        self.modes.get(self.mode).copied().unwrap_or("Full")
    }
    fn full(&self) -> bool {
        self.mode == 0
    }
}
fn modes_of(kind: GuiTemplateKind) -> Vec<&'static str> {
    match kind {
        GuiTemplateKind::Assembly => vec!["Full", "Express"],
        GuiTemplateKind::Drawing => vec!["Full", "Defer"],
        _ => Vec::new(),
    }
}
impl FileOpenOptions {
    /// Ask for the dialog; example: from a recent row's Open with options.
    pub(super) fn request(&mut self, row: &GuiRecentDocument) {
        self.target = Some(OpenTarget::of(row));
        self.opening = true;
    }
    /// Draw the dialog; OK opens with the file's own state in Full mode.
    pub(super) fn panel(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        if std::mem::take(&mut self.opening) {
            ui.open_popup(TITLE);
        }
        let Some(_dialog) = ui.begin_modal_popup(TITLE) else {
            return;
        };
        let Some(target) = &mut self.target else {
            ui.close_current_popup();
            return;
        };
        ui.text(&target.name);
        option_rows(ui, target);
        if option_buttons(ui, target, actions) {
            ui.close_current_popup();
        }
    }
}
fn option_rows(ui: &Ui, target: &mut OpenTarget) {
    if !target.modes.is_empty() {
        mode_row(ui, target);
    }
    if target.kind != GuiTemplateKind::Drawing {
        ui.text(format!("Model State: {PRIMARY}"));
        view_row(ui, target);
    }
    if target.kind == GuiTemplateKind::Assembly {
        let _disabled = ui.begin_disabled(!target.full());
        ui.text(format!("Position View: {PRIMARY}"));
    }
    if target.kind != GuiTemplateKind::Part {
        let _disabled = ui.begin_disabled(!target.full());
        ui.checkbox("Skip all unresolved files", &mut target.skip_unresolved);
    }
}
fn mode_row(ui: &Ui, target: &mut OpenTarget) {
    let label = format!("Mode: {}", target.mode_name());
    let modes = target.modes.clone();
    if let Some(mode) = crate::widget_abi::popup_choice(ui, &label, "##open-mode", &modes) {
        target.mode = mode;
    }
}
fn view_row(ui: &Ui, target: &mut OpenTarget) {
    let _disabled = ui.begin_disabled(!target.full());
    let label = format!("Design View: {}", target.views[target.view]);
    let views = target.views.clone();
    if let Some(view) = crate::widget_abi::popup_choice(ui, &label, "##open-view", &views) {
        target.view = view;
    }
}
/// True when the dialog closes. Only Full mode opens: the others have no loader behind them.
fn option_buttons(ui: &Ui, target: &OpenTarget, actions: &mut Vec<GuiShellAction>) -> bool {
    ui.separator();
    let mut closing = false;
    {
        let _muted = ui.begin_disabled(!target.full());
        if crate::widget_abi::button(ui, "OK", [80., 0.]) {
            actions.push(GuiShellAction::OpenDocument(
                oblikovati_api::wire::OpenDocumentArgs {
                    full_document_name: target.path.clone(),
                    ..Default::default()
                },
            ));
            closing = true;
        }
    }
    if !target.full() && ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED)
    {
        ui.tooltip_text(NOT_AVAILABLE);
    }
    ui.same_line();
    closing |= crate::widget_abi::button(ui, "Cancel", [80., 0.]);
    closing || ui.is_key_pressed_no_repeat(imgui::Key::Escape)
}
