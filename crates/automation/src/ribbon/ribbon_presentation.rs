// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::wire::{RibbonPanelInfo, RibbonTabInfo};

use super::{panel, tab, unavailable_compact_panel, unavailable_large, view_tab};

/// Build Presentation document tabs; example: `tabs()` returns Presentation and View.
pub(super) fn tabs() -> Vec<RibbonTabInfo> {
    vec![
        tab(
            "Presentation",
            vec![create_panel(), storyboard_panel(), view_panel()],
        ),
        view_tab(),
    ]
}

fn create_panel() -> RibbonPanelInfo {
    panel(
        "Create",
        vec![unavailable_large(
            "Presentation.Tweak",
            "Tweak Components",
            "presentation-tweak",
        )],
    )
}

fn storyboard_panel() -> RibbonPanelInfo {
    unavailable_compact_panel("Storyboard", &["Add Action", "Animate", "Publish Video"])
}

fn view_panel() -> RibbonPanelInfo {
    panel(
        "View",
        vec![unavailable_large(
            "Presentation.Snapshot",
            "Snapshot View",
            "presentation-snapshot",
        )],
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn presentation_tabs_keep_the_create_storyboard_and_view_panels() {
        let tabs = tabs();
        let panels = tabs[0].panels.as_ref().unwrap();
        assert_eq!(tabs[0].name, "Presentation");
        assert_eq!(tabs[1].name, "View");
        assert_eq!(
            panels
                .iter()
                .map(|panel| panel.name.as_str())
                .collect::<Vec<_>>(),
            ["Create", "Storyboard", "View"]
        );
    }
}
