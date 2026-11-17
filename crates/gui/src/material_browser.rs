// SPDX-License-Identifier: GPL-2.0-only
//! Browse part assets and assign a selected material or appearance.
use crate::dockable_window::DockableWindow;
use crate::{GuiAssetChoice, GuiCommandState, GuiPartAssetKind, GuiShellAction};
use imgui::Ui;

#[derive(Default)]
pub(crate) struct MaterialBrowser {
    open: crate::dockable_lifecycle::DockableState,
    appearance_browser: bool,
    select_preferred: bool,
    document: Option<u64>,
    material: Option<String>,
    appearance: Option<String>,
}
impl DockableWindow for MaterialBrowser {}
impl MaterialBrowser {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        let title = if self.appearance_browser {
            "Appearance Browser"
        } else {
            "Material Browser"
        };
        registry.register::<Self>(title, &self.open)
    }
    pub(crate) fn set_open(&mut self, open: bool) {
        if open && !self.open.is_open() {
            self.open();
        } else {
            self.open.set_open(open);
        }
    }
    pub(crate) fn is_open(&self) -> bool {
        self.open.is_open()
    }
    /// Create a separate appearance browser; example: `MaterialBrowser::appearances()`.
    pub(crate) fn appearances() -> Self {
        Self {
            appearance_browser: true,
            ..Default::default()
        }
    }
    /// Open the browser; example: `browser.open()` from the quick access toolbar.
    pub(crate) fn open(&mut self) {
        self.open.set_open(true);
        self.select_preferred = true;
    }
    /// Draw the current part's asset choices and queue explicit assignments.
    /// Example: `browser.panel(ui, state, actions)` once per frame.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if !self.open.is_open() {
            return;
        }
        self.sync_document(state);
        let mut open = self.open.is_open();
        let mut close_requested = false;
        let title = if self.appearance_browser {
            "Appearance Browser"
        } else {
            "Material Browser"
        };
        Self::fit_dockable(
            ui,
            title,
            [ui.io().display_size[0] - 460., 200.],
            [440., 200.],
        );
        Self::dockable_window(ui, title, &mut open)
            .build(|| close_requested = self.content(ui, state, actions));
        self.open.set_open(open && !close_requested);
    }
    fn sync_document(&mut self, state: &GuiCommandState) {
        if self.document == state.document {
            return;
        }
        self.document = state.document;
        self.material = None;
        self.appearance = None;
    }
    fn content(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        let Some(browser) = state
            .browser
            .as_ref()
            .filter(|_| state.authoring == crate::GuiAuthoringContext::Part)
        else {
            ui.text("Select a part to browse its assets.");
            return ui.button("Close");
        };
        self.tabs(ui, state, browser, actions);
        ui.separator();
        ui.button("Close")
    }
    fn tabs(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        browser: &crate::GuiModelBrowser,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(_tabs) = ui.tab_bar("##material-browser-tabs") else {
            return;
        };
        let requested = std::mem::take(&mut self.select_preferred);
        let material_flags = if requested && !self.appearance_browser {
            imgui::TabItemFlags::SET_SELECTED
        } else {
            imgui::TabItemFlags::empty()
        };
        if let Some(_tab) = ui.tab_item_with_flags("Materials", None, material_flags) {
            asset_tab(
                ui,
                state,
                &browser.materials,
                &mut self.material,
                browser.part_material.as_deref(),
                GuiPartAssetKind::Material,
                actions,
            );
        }
        let appearance_flags = if requested && self.appearance_browser {
            imgui::TabItemFlags::SET_SELECTED
        } else {
            imgui::TabItemFlags::empty()
        };
        if let Some(_tab) = ui.tab_item_with_flags("Appearances", None, appearance_flags) {
            asset_tab(
                ui,
                state,
                &browser.appearances,
                &mut self.appearance,
                browser.part_appearance.as_deref(),
                GuiPartAssetKind::Appearance,
                actions,
            );
        }
    }
}
fn asset_tab(
    ui: &Ui,
    state: &GuiCommandState,
    choices: &[GuiAssetChoice],
    selected: &mut Option<String>,
    assigned: Option<&str>,
    kind: GuiPartAssetKind,
    actions: &mut Vec<GuiShellAction>,
) {
    initialize_selection(selected, assigned, choices);
    let label = match kind {
        GuiPartAssetKind::Material => "Material",
        GuiPartAssetKind::Appearance => "Appearance",
    };
    if let Some(id) =
        crate::ribbon_part_assets::asset_combo(ui, label, selected.as_deref(), choices)
    {
        *selected = Some(id);
    }
    assign_button(
        ui,
        state,
        choices,
        selected.as_deref(),
        assigned,
        kind,
        actions,
    );
}
fn initialize_selection(
    selected: &mut Option<String>,
    assigned: Option<&str>,
    choices: &[GuiAssetChoice],
) {
    if selected.is_none() {
        *selected = assigned
            .map(str::to_owned)
            .or_else(|| choices.first().map(|choice| choice.id.clone()));
    }
}
fn assign_button(
    ui: &Ui,
    state: &GuiCommandState,
    choices: &[GuiAssetChoice],
    selected: Option<&str>,
    assigned: Option<&str>,
    kind: GuiPartAssetKind,
    actions: &mut Vec<GuiShellAction>,
) {
    let usable = selected
        .filter(|id| Some(*id) != assigned && choices.iter().any(|choice| choice.id == *id));
    let disabled = ui.begin_disabled(usable.is_none() || state.transaction_open);
    if ui.button("Assign to part")
        && let Some(id) = usable
    {
        crate::ribbon_part_assets::queue_asset(state, kind, id, actions);
    }
    drop(disabled);
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn selection_keeps_user_choice_and_prefers_the_assigned_asset() {
        let choices = [GuiAssetChoice {
            id: "steel".into(),
            name: "Steel".into(),
        }];
        let mut chosen = None;
        initialize_selection(&mut chosen, Some("assigned"), &choices);
        assert_eq!(chosen.as_deref(), Some("assigned"));
        initialize_selection(&mut chosen, Some("other"), &choices);
        assert_eq!(chosen.as_deref(), Some("assigned"));
        chosen = None;
        initialize_selection(&mut chosen, None, &choices);
        assert_eq!(chosen.as_deref(), Some("steel"));
    }
}
