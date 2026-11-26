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
    ungrouped: BTreeSet<String>,
    staged_ungrouped: BTreeSet<String>,
    large: BTreeSet<String>,
    staged_large: BTreeSet<String>,
}
impl BuiltinLayout {
    pub(super) fn load(
        &mut self,
        locations: BTreeSet<String>,
        groups: BTreeSet<String>,
        large: BTreeSet<String>,
    ) {
        self.applied = locations;
        self.ungrouped = groups;
        self.large = large;
        self.open();
    }
    pub(super) fn open(&mut self) {
        self.staged = self.applied.clone();
        self.staged_ungrouped = self.ungrouped.clone();
        self.staged_large = self.large.clone();
    }
    pub(super) fn apply(&mut self) {
        self.applied = self.staged.clone();
        self.ungrouped = self.staged_ungrouped.clone();
        self.large = self.staged_large.clone();
    }
    pub(super) fn import(
        &mut self,
        locations: BTreeSet<String>,
        groups: BTreeSet<String>,
        large: BTreeSet<String>,
    ) {
        self.staged = locations;
        self.staged_ungrouped = groups;
        self.staged_large = large;
    }
    pub(super) fn changed(&self) -> bool {
        self.applied != self.staged
            || self.ungrouped != self.staged_ungrouped
            || self.large != self.staged_large
    }
}
impl RibbonCustomization {
    pub(super) fn staged_document(&self) -> SettingsDocument {
        SettingsDocument {
            commands: self.draft.snapshot().clone(),
            expanded_builtins: self.builtins.staged.clone(),
            ungrouped_builtins: self.builtins.staged_ungrouped.clone(),
            large_builtins: self.builtins.staged_large.clone(),
            appearance: self.staged_appearance,
        }
    }
    pub(super) fn applied_document(&self) -> SettingsDocument {
        SettingsDocument {
            commands: self.draft.applied_snapshot().clone(),
            expanded_builtins: self.builtins.applied.clone(),
            ungrouped_builtins: self.builtins.ungrouped.clone(),
            large_builtins: self.builtins.large.clone(),
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
            self.builtins.load(
                document.expanded_builtins,
                document.ungrouped_builtins,
                document.large_builtins,
            );
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
    pub(super) fn has_pending_changes(&self) -> bool {
        self.draft.changed()
            || self.builtins.changed()
            || self.staged_appearance != self.appearance
            || self.keyboard_changed()
            || self.marking_changed()
    }
    pub(super) fn apply(&mut self, retain_empty_panels: bool) -> Result<(), String> {
        let document = self.staged_document();
        Self::save_settings(&mut self.store, &document)?;
        self.draft.apply();
        if !retain_empty_panels {
            self.draft.retained_panels.clear();
        }
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
        self.builtins.load(
            defaults.expanded_builtins,
            defaults.ungrouped_builtins,
            defaults.large_builtins,
        );
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
    pub(crate) fn builtin_ungrouped(&self, id: &str) -> bool {
        self.builtins.ungrouped.contains(id)
    }
    pub(crate) fn builtin_large(&self, id: &str) -> bool {
        self.builtins.large.contains(id)
    }
    pub(super) fn set_builtin_large(&mut self, id: &str, large: bool) -> Result<(), String> {
        let mut document = self.applied_document();
        if large {
            document.large_builtins.insert(id.to_owned());
        } else {
            document.large_builtins.remove(id);
        }
        Self::save_settings(&mut self.store, &document)?;
        if self.builtins.staged_large.contains(id) == self.builtin_large(id) {
            if large {
                self.builtins.staged_large.insert(id.to_owned());
            } else {
                self.builtins.staged_large.remove(id);
            }
        }
        self.builtins.large = document.large_builtins;
        Ok(())
    }
    pub(super) fn set_builtin_ungrouped(
        &mut self,
        id: &str,
        ungrouped: bool,
    ) -> Result<(), String> {
        let mut document = self.applied_document();
        if ungrouped {
            document.ungrouped_builtins.insert(id.to_owned());
        } else {
            document.ungrouped_builtins.remove(id);
        }
        Self::save_settings(&mut self.store, &document)?;
        for groups in [
            &mut self.builtins.ungrouped,
            &mut self.builtins.staged_ungrouped,
        ] {
            if ungrouped {
                groups.insert(id.to_owned());
            } else {
                groups.remove(id);
            }
        }
        Ok(())
    }
    pub(crate) fn relocate_builtin(&mut self, id: &str, expanded: bool) -> Result<(), String> {
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
    pub(crate) fn report_command_error(&mut self, error: String) {
        self.command_error = Some(error);
        self.command_error_opening = true;
    }
    pub(crate) fn relocate_builtin_reported(&mut self, id: &str, expanded: bool) -> bool {
        if let Err(error) = self.relocate_builtin(id, expanded) {
            self.command_error = Some(error);
            self.command_error_opening = true;
            return false;
        }
        true
    }
    /// Draw a built-in command and its location menu; example: `layout.builtin_draw(ui, id, false, draw)`.
    pub(crate) fn builtin_draw(
        &mut self,
        ui: &imgui::Ui,
        id: &str,
        expanded: bool,
        draw: impl FnOnce(),
    ) -> bool {
        self.draw_builtin_with_menu(
            ui,
            id,
            expanded,
            false,
            |_| draw(),
            || unavailable_shortcut(ui),
        )
    }
    pub(crate) fn builtin_draw_with_shortcut(
        &mut self,
        ui: &imgui::Ui,
        id: &str,
        draw: impl FnOnce(),
        shortcut: impl FnOnce(),
    ) -> bool {
        self.draw_builtin_with_menu(ui, id, false, false, |_| draw(), shortcut)
    }
    pub(crate) fn builtin_group_draw(
        &mut self,
        ui: &imgui::Ui,
        id: &str,
        expanded: bool,
        draw: impl FnOnce(bool),
    ) -> bool {
        self.draw_builtin_with_menu(ui, id, expanded, true, draw, || unavailable_shortcut(ui))
    }
    fn draw_builtin_with_menu(
        &mut self,
        ui: &imgui::Ui,
        id: &str,
        expanded: bool,
        group: bool,
        draw: impl FnOnce(bool),
        shortcut: impl FnOnce(),
    ) -> bool {
        if self.builtin_expanded(id) != expanded {
            return false;
        }
        let _command_id = ui.push_id(id);
        if group {
            ui.group(|| draw(self.builtin_ungrouped(id)));
        } else {
            draw(false);
        }
        let hovered = ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED);
        let right = hovered
            && (ui.is_mouse_clicked(imgui::MouseButton::Right)
                || ui.is_mouse_released(imgui::MouseButton::Right));
        if hovered && ui.is_mouse_released(imgui::MouseButton::Right) {
            ui.open_popup("##builtin-command-context");
        }
        self.builtin_menu(ui, id, expanded, group, shortcut);
        right
    }
    fn builtin_menu(
        &mut self,
        ui: &imgui::Ui,
        id: &str,
        expanded: bool,
        group: bool,
        shortcut: impl FnOnce(),
    ) {
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
        if group {
            let ungrouped = self.builtin_ungrouped(id);
            let label = if ungrouped {
                "Group Commands into Drop-Down Menu"
            } else {
                "Ungroup Commands from Drop-Down Menu"
            };
            if ui.menu_item(label)
                && let Err(error) = self.set_builtin_ungrouped(id, !ungrouped)
            {
                self.report_command_error(error);
            }
        }
        if id == crate::shell_ribbon::ID
            && !expanded
            && let Some(_size) = ui.begin_menu("Button Size")
        {
            for (label, large) in [("Standard", false), ("Large", true)] {
                if ui
                    .menu_item_config(label)
                    .selected(self.builtin_large(id) == large)
                    .build()
                    && let Err(error) = self.set_builtin_large(id, large)
                {
                    self.report_command_error(error);
                }
            }
        }
        shortcut();
    }
}

fn unavailable_shortcut(ui: &imgui::Ui) {
    ui.menu_item_config("Add to Quick Access Toolbar")
        .enabled(false)
        .build();
}
