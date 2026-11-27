// SPDX-License-Identifier: GPL-2.0-only
//! Read applied interface state from the owners of each preference.
use crate::GuiCommandState;
use crate::application::{
    quick_access_toolbar::QuickAccessToolbar,
    ribbon_display::{RibbonDisplay, RibbonDisplayState, RibbonPlacement},
};
use serde_json::{Value, json};

pub(super) const SECTIONS: [&str; 5] = [
    "ribbon-state",
    "quick-access-toolbar",
    "keyboard-mappings",
    "marking-menu",
    "mini-toolbar",
];
pub(super) type Sections = [Value; SECTIONS.len()];

pub(crate) struct InterfaceExport<'a> {
    pub ribbon: &'a RibbonDisplayState,
    pub quick_access: &'a QuickAccessToolbar,
    pub marking_menu_enabled: bool,
    pub mini_toolbar_visible: bool,
}

impl InterfaceExport<'_> {
    pub(super) fn sections(self, commands: &GuiCommandState) -> Sections {
        let mode = match self.ribbon.mode {
            RibbonDisplay::Full => "full",
            RibbonDisplay::PanelButtons => "panel-buttons",
            RibbonDisplay::PanelTitles => "panel-titles",
            RibbonDisplay::Tabs => "tabs",
        };
        let placement = match self.ribbon.placement {
            RibbonPlacement::Top => "top",
            RibbonPlacement::Left => "left",
            RibbonPlacement::Right => "right",
            RibbonPlacement::Floating => "floating",
        };
        let hidden: Vec<_> = self
            .ribbon
            .hidden_panels
            .iter()
            .map(|(tab, panel)| json!({"tab":tab.label(), "panel":panel.label()}))
            .collect();
        let keyboard = commands.keymap.as_ref().map(|map| map.preferences());
        let marking = commands.marking_menus.as_ref().map(|menus| menus.all());
        [
            json!({
                "mode": mode,
                "placement": placement,
                "show-panel-titles": self.ribbon.show_panel_titles,
                "hidden-panels": hidden,
            }),
            self.quick_access.export_settings(commands.document_type),
            json!({
                "default-multi-character-aliases": keyboard.as_ref().map(|settings| settings.default_aliases),
                "bindings": keyboard.map(|settings| settings.delta),
            }),
            json!({
                "enabled": self.marking_menu_enabled,
                "classic": commands.sketch_classic_menu,
                "style": commands.marking_menu_display.style,
                "overflow": commands.marking_menu_display.overflow,
                "menus": marking,
            }),
            json!({"visible": self.mini_toolbar_visible}),
        ]
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ribbon::RibbonTab;
    use crate::ribbon_panels::RibbonPanel;
    use oblikovati_model::{keymap::Keymap, marking_menu::MarkingMenus};

    #[test]
    fn interface_sections_capture_live_preference_values() {
        let mut ribbon = RibbonDisplayState::default();
        ribbon.mode = RibbonDisplay::Tabs;
        ribbon.placement = RibbonPlacement::Left;
        ribbon
            .hidden_panels
            .insert((RibbonTab::Model, RibbonPanel::PartExplore));
        let mut keymap = Keymap::desktop_defaults();
        keymap.set_chord("edit.undo", "Ctrl+U").unwrap();
        keymap.enable_default_aliases(true);
        let commands = GuiCommandState {
            document_type: Some(oblikovati_api::types::DOCUMENT_PART),
            keymap: Some(keymap),
            marking_menus: Some(MarkingMenus::default()),
            sketch_classic_menu: true,
            ..Default::default()
        };
        let mut toolbar = QuickAccessToolbar::default();
        toolbar.below_ribbon = true;
        toolbar.add_ribbon_command(&commands, "Shell");
        let [ribbon, toolbar, keyboard, marking, mini] = InterfaceExport {
            ribbon: &ribbon,
            quick_access: &toolbar,
            marking_menu_enabled: false,
            mini_toolbar_visible: false,
        }
        .sections(&commands);
        assert_eq!(ribbon["mode"], "tabs");
        assert_eq!(ribbon["placement"], "left");
        assert!(
            ribbon["hidden-panels"]
                .as_array()
                .unwrap()
                .contains(&json!({
                    "tab": "3D Model",
                    "panel": "Explore",
                }))
        );
        assert_eq!(toolbar["placement"], "below-ribbon");
        assert_eq!(toolbar["active-context"], "part");
        assert_eq!(toolbar["contexts"]["part"]["added"][0]["command"], "Shell");
        assert_eq!(keyboard["default-multi-character-aliases"], true);
        assert_eq!(keyboard["bindings"]["chords"]["edit.undo"], "Ctrl+U");
        assert_eq!(marking["classic"], true);
        assert_eq!(marking["enabled"], false);
        assert_eq!(marking["menus"].as_array().unwrap().len(), 3);
        assert_eq!(mini["visible"], false);
    }
}
