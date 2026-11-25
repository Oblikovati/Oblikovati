// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard and mouse selection of the customized ribbon target.
use super::{RibbonCustomization, RibbonTab, category::move_cursor};
use imgui::{Key, Ui};

const TARGETS: [RibbonTab; 7] = [
    RibbonTab::Model,
    RibbonTab::Sketch,
    RibbonTab::Annotate,
    RibbonTab::Tools,
    RibbonTab::Manage,
    RibbonTab::View,
    RibbonTab::Assemble,
];

const UNAVAILABLE_PART_TABS: &[&str] = &[
    "Environments",
    "Collaborate",
    "Fusion",
    "3D Sketch",
    "Construction",
    "Edit Base Solid",
    "Repair",
    "Route",
    "Render",
    "Analysis",
    "3D Print",
    "Freeform",
    "Hidden Tab",
    "BIM Content",
];

const UNAVAILABLE_ASSEMBLY_TABS: &[&str] =
    &["Mold Layout", "Core/Cavity", "Mold Assembly", "Assemble"];

pub(super) fn target_index(target: RibbonTab) -> usize {
    TARGETS.iter().position(|tab| *tab == target).unwrap_or(0)
}

pub(super) fn target_caption(tab: RibbonTab, suffix: &str) -> String {
    if tab == RibbonTab::Assemble {
        format!("Assembly | Design{suffix}")
    } else {
        format!("Part | {}{suffix}", tab.label())
    }
}

impl RibbonCustomization {
    pub(super) fn target_menu(&mut self, ui: &Ui) {
        self.target_cursor = move_cursor(ui, self.target_cursor, TARGETS.len());
        let enter = ui.is_key_pressed_no_repeat(Key::Enter);
        for (index, target) in TARGETS.into_iter().enumerate() {
            let clicked = ui
                .selectable_config(target_caption(target, ""))
                .selected(index == self.target_cursor)
                .build();
            if clicked || (enter && index == self.target_cursor) {
                self.draft.switch(target);
                self.scroll_target_to_selection = false;
                self.scroll_source_to_selection = self.draft.source.is_some();
                ui.close_current_popup();
            }
        }
        for (environment, tabs) in [
            ("Part", UNAVAILABLE_PART_TABS),
            ("Assembly", UNAVAILABLE_ASSEMBLY_TABS),
        ] {
            for tab in tabs {
                ui.selectable_config(format!("{environment} | {tab}"))
                    .disabled(true)
                    .build();
                if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
                    ui.tooltip_text("This ribbon target is unavailable in this build.");
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn target_indices_follow_the_visible_order() {
        assert_eq!(target_index(RibbonTab::Tools), 3);
        assert_eq!(target_index(RibbonTab::Manage), 4);
        assert_eq!(target_index(RibbonTab::View), 5);
        assert_eq!(target_index(RibbonTab::Assemble), 6);
    }
}
