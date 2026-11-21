// SPDX-License-Identifier: GPL-2.0-only
//! Preserve temporary panels across selected-tab presses (#3856).
use super::*;
impl PanelExpansions {
    pub(in crate::ribbon_panels) fn preserves_tab_press(&self) -> bool {
        self.preserve_tab_press
    }
    pub(in crate::ribbon_panels) fn temporary_tab_press(&mut self) -> Option<bool> {
        if self.open.is_empty() {
            return None;
        }
        if self.temporary_tab_armed {
            self.clear();
            return Some(true);
        }
        self.temporary_tab_armed = true;
        self.preserve_tab_press = true;
        Some(false)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn expanded() -> PanelExpansions {
        let mut panels = PanelExpansions::default();
        panels.open.insert(
            (RibbonTab::Model, RibbonPanel::PartModify),
            Expansion {
                anchor: [0.; 2],
                trigger: [[0.; 2]; 2],
                pinned: false,
                focus: false,
                bounds: [[0.; 2]; 2],
                bend_notice: Default::default(),
            },
        );
        panels
    }
    #[test]
    fn selected_tab_preserves_then_closes_a_temporary_expansion() {
        let mut panels = expanded();
        assert_eq!(panels.temporary_tab_press(), Some(false));
        assert!(panels.preserves_tab_press());
        assert_eq!(panels.temporary_tab_press(), Some(true));
        assert!(!panels.has_open());
        assert_eq!(panels.temporary_tab_press(), None);
    }
    #[test]
    fn clearing_an_expansion_resets_the_next_tab_sequence() {
        let mut panels = expanded();
        assert_eq!(panels.temporary_tab_press(), Some(false));
        panels.clear();
        panels.open = expanded().open;
        assert_eq!(panels.temporary_tab_press(), Some(false));
    }
}
