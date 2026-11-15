// SPDX-License-Identifier: GPL-2.0-only
//! Built-in command locations share the customization transaction.
use super::{
    Command, GuiRibbonSettingsStore, RibbonCustomization,
    settings_xml::{self, SettingsDocument},
};
use crate::ribbon::RibbonTab;
use std::collections::BTreeSet;

#[derive(Default)]
pub(super) struct BuiltinLayout {
    applied: BTreeSet<String>,
    staged: BTreeSet<String>,
}
impl BuiltinLayout {
    pub(super) fn load(&mut self, locations: BTreeSet<String>) {
        self.applied = locations;
        self.open();
    }
    pub(super) fn open(&mut self) {
        self.staged = self.applied.clone();
    }
    pub(super) fn apply(&mut self) {
        self.applied = self.staged.clone();
    }
    pub(super) fn import(&mut self, locations: BTreeSet<String>) {
        self.staged = locations;
    }
    pub(super) fn changed(&self) -> bool {
        self.applied != self.staged
    }
}
impl RibbonCustomization {
    pub(super) fn staged_document(&self) -> SettingsDocument {
        SettingsDocument {
            commands: self.draft.snapshot().clone(),
            expanded_builtins: self.builtins.staged.clone(),
            appearance: self.staged_appearance,
        }
    }
    pub(super) fn applied_document(&self) -> SettingsDocument {
        SettingsDocument {
            commands: self.draft.applied_snapshot().clone(),
            expanded_builtins: self.builtins.applied.clone(),
            appearance: self.appearance,
        }
    }
}

impl RibbonCustomization {
    pub(crate) fn set_store(
        &mut self,
        mut store: Box<dyn GuiRibbonSettingsStore>,
    ) -> Result<(), String> {
        if let Some(bytes) = store.load()? {
            if bytes.len() > settings_xml::MAX_SETTINGS_BYTES {
                return Err(format!(
                    "ribbon settings {} bytes: expected at most {} bytes",
                    bytes.len(),
                    settings_xml::MAX_SETTINGS_BYTES
                ));
            }
            let document = settings_xml::decode(&bytes)?;
            self.builtins.load(document.expanded_builtins);
            self.appearance = document.appearance;
            self.staged_appearance = document.appearance;
            self.draft.load_applied(document.commands);
        }
        self.store = Some(store);
        Ok(())
    }
    pub(crate) fn appearance(&self) -> crate::ribbon_command_layout::RibbonAppearance {
        self.appearance
    }
    pub(crate) fn select_appearance(
        &mut self,
        appearance: crate::ribbon_command_layout::RibbonAppearance,
    ) {
        if let Err(error) = self.save_appearance(appearance) {
            self.command_error = Some(error);
            self.command_error_opening = true;
        }
    }
    fn save_appearance(
        &mut self,
        appearance: crate::ribbon_command_layout::RibbonAppearance,
    ) -> Result<(), String> {
        let mut document = self.applied_document();
        document.appearance = appearance;
        Self::save_settings(&mut self.store, &document)?;
        if self.staged_appearance == self.appearance {
            self.staged_appearance = appearance;
        }
        self.appearance = appearance;
        Ok(())
    }
    pub(super) fn apply(&mut self) -> Result<(), String> {
        let document = self.staged_document();
        Self::save_settings(&mut self.store, &document)?;
        self.draft.apply();
        self.builtins.apply();
        self.appearance = self.staged_appearance;
        Ok(())
    }
    /// Persist the default ribbon before replacing its applied settings.
    /// Example: `customization.reset()?`.
    pub(crate) fn reset(&mut self) -> Result<(), String> {
        let defaults = settings_xml::SettingsDocument::default();
        Self::save_settings(&mut self.store, &defaults)?;
        self.draft.load_applied(defaults.commands);
        self.builtins.load(defaults.expanded_builtins);
        self.appearance = defaults.appearance;
        self.staged_appearance = defaults.appearance;
        Ok(())
    }
    pub(super) fn save_settings(
        store: &mut Option<Box<dyn GuiRibbonSettingsStore>>,
        settings: &settings_xml::SettingsDocument,
    ) -> Result<(), String> {
        let Some(store) = store.as_mut() else {
            return Ok(());
        };
        let bytes = settings_xml::encode(settings)?;
        if bytes.len() > settings_xml::MAX_SETTINGS_BYTES {
            return Err(format!(
                "ribbon settings {} bytes: expected at most {} bytes",
                bytes.len(),
                settings_xml::MAX_SETTINGS_BYTES
            ));
        }
        store.save(&bytes)
    }
    pub(super) fn relocate_command(
        &mut self,
        tab: RibbonTab,
        command: Command,
        expanded: bool,
    ) -> Result<(), String> {
        let mut settings = self.applied_document();
        let entry = settings
            .commands
            .get_mut(&tab)
            .and_then(|rows| rows.iter_mut().find(|row| row.command == command))
            .ok_or_else(|| {
                format!(
                    "ribbon command {:?}: expected an applied command",
                    command.id()
                )
            })?;
        entry.expanded = expanded;
        Self::save_settings(&mut self.store, &settings)?;
        self.draft.relocate(tab, command, expanded);
        Ok(())
    }
}

impl RibbonCustomization {
    /// Check a built-in command's applied location; example: `customization.builtin_expanded(id)`.
    pub(crate) fn builtin_expanded(&self, id: &str) -> bool {
        self.builtins.applied.contains(id)
    }
    pub(super) fn relocate_builtin(&mut self, id: &str, expanded: bool) -> Result<(), String> {
        let mut document = self.applied_document();
        if expanded {
            document.expanded_builtins.insert(id.to_owned());
        } else {
            document.expanded_builtins.remove(id);
        }
        Self::save_settings(&mut self.store, &document)?;
        for locations in [&mut self.builtins.applied, &mut self.builtins.staged] {
            if expanded {
                locations.insert(id.to_owned());
            } else {
                locations.remove(id);
            }
        }
        Ok(())
    }
    /// Draw a built-in command and its location menu; example: `layout.builtin_draw(ui, id, false, draw)`.
    pub(crate) fn builtin_draw(
        &mut self,
        ui: &imgui::Ui,
        id: &str,
        expanded: bool,
        draw: impl FnOnce(),
    ) -> bool {
        if self.builtin_expanded(id) != expanded {
            return false;
        }
        let _command_id = ui.push_id(id);
        draw();
        let hovered = ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED);
        let right = hovered
            && (ui.is_mouse_clicked(imgui::MouseButton::Right)
                || ui.is_mouse_released(imgui::MouseButton::Right));
        if hovered && ui.is_mouse_released(imgui::MouseButton::Right) {
            ui.open_popup("##builtin-command-context");
        }
        self.builtin_menu(ui, id, expanded);
        right
    }
    fn builtin_menu(&mut self, ui: &imgui::Ui, id: &str, expanded: bool) {
        let Some(_menu) = ui.begin_popup("##builtin-command-context") else {
            return;
        };
        let label = if expanded {
            "Move to Main Panel"
        } else {
            "Move to Expanded Panel"
        };
        if ui.menu_item(label)
            && let Err(error) = self.relocate_builtin(id, !expanded)
        {
            self.command_error = Some(error);
            self.command_error_opening = true;
        }
        let disabled = ui.begin_disabled(true);
        ui.menu_item("Add to Quick Access Toolbar");
        drop(disabled);
    }
}
