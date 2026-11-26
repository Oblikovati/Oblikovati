// SPDX-License-Identifier: GPL-2.0-only
//! Browse part assets and assign a selected material or appearance.
use crate::dockable_window::DockableWindow;
use crate::{GuiAssetChoice, GuiCommandState, GuiPartAssetKind, GuiShellAction};
use imgui::Ui;

#[derive(Default)]
pub(crate) struct MaterialBrowser {
    open: crate::dockable_lifecycle::DockableState,
    appearance_browser: bool,
    document: Option<u64>,
    material: Option<String>,
    appearance: Option<String>,
    browser_view: crate::asset_browser_view::AssetBrowserView,
    editor: crate::appearance_editor::AppearanceEditor,
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
        if open != self.open.is_requested() {
            self.open.set_open(open);
        }
    }
    pub(crate) fn is_open(&self) -> bool {
        self.open.is_open()
    }
    pub(crate) fn is_requested(&self) -> bool {
        self.open.is_requested()
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
    }
    pub(crate) fn sync_before_docking(&self, state: &GuiCommandState, home_in_front: bool) {
        use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING};
        let unavailable = state
            .document_type
            .as_ref()
            .is_some_and(|kind| *kind == DOCUMENT_DRAWING || *kind == DOCUMENT_ASSEMBLY);
        self.open.set_suspended(unavailable || home_in_front);
    }
    /// Draw the current part's asset choices and queue explicit assignments.
    /// Example: `browser.panel(ui, state, false, actions)` in a Part workspace.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        home_in_front: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.sync_before_docking(state, home_in_front);
        if !home_in_front && self.appearance_browser {
            self.editor.panel(ui, state, &mut self.appearance, actions);
        }
        if !self.is_open() {
            return;
        }
        self.sync_document(state);
        let mut open = true;
        let title = if self.appearance_browser {
            "Appearance Browser"
        } else {
            "Material Browser"
        };
        Self::fit_dockable(
            ui,
            title,
            [ui.io().display_size[0] - 460., 200.],
            [440., 430.],
        );
        Self::dockable_window(ui, title, &mut open).build(|| self.content(ui, state, actions));
        self.open.set_open(open);
    }
    fn sync_document(&mut self, state: &GuiCommandState) {
        if self.document == state.document {
            return;
        }
        self.document = state.document;
        self.material = None;
        self.appearance = None;
        self.browser_view = if self.appearance_browser {
            crate::asset_browser_view::AssetBrowserView::appearances()
        } else {
            Default::default()
        };
    }
    fn content(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let Some(browser) = state
            .browser
            .as_ref()
            .filter(|_| state.authoring == crate::GuiAuthoringContext::Part)
        else {
            ui.text("Select a part to browse its assets.");
            return;
        };
        let can_assign = browser.root_selected && !state.transaction_open;
        if self.appearance_browser {
            if let Some(id) = self.browser_view.draw(
                ui,
                &browser.appearances,
                &mut self.appearance,
                Some(can_assign),
            ) {
                crate::ribbon_part_assets::queue_asset(
                    state,
                    GuiPartAssetKind::Appearance,
                    &id,
                    actions,
                );
            }
            asset_tab(
                ui,
                state,
                &browser.appearances,
                &mut self.appearance,
                browser.part_appearance.as_deref(),
                GuiPartAssetKind::Appearance,
                actions,
            );
            self.editor
                .create_menu(ui, state, &mut self.appearance, actions);
        } else {
            if let Some(id) =
                self.browser_view
                    .draw(ui, &browser.materials, &mut self.material, Some(can_assign))
            {
                crate::ribbon_part_assets::queue_asset(
                    state,
                    GuiPartAssetKind::Material,
                    &id,
                    actions,
                );
            }
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
        crate::ribbon_part_assets::dock_asset_combo(ui, label, selected.as_deref(), choices)
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
            swatch: None,
            ..Default::default()
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
