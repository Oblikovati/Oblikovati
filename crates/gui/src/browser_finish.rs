// SPDX-License-Identifier: GPL-2.0-only
//! Model Browser `Finishes` folder and warning indicator for Part Modify Finish features (#3918).
use crate::application::workspace_layout::{browser_feature, browser_view::BrowserView};
use crate::{GuiBrowserFeature, GuiCommandState, GuiModelBrowser, GuiShellAction};
use imgui::{Condition, Ui};

const WARNING_AMBER: [f32; 4] = [0.95, 0.72, 0.16, 1.0];

/// Render the collapsible `Finishes` folder listing Finish features in program order.
pub(crate) fn finishes_folder(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    expansion: Option<bool>,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
) {
    let finishes: Vec<&GuiBrowserFeature> = browser
        .features
        .iter()
        .filter(|feature| feature.kind == "Finish")
        .collect();
    if finishes.is_empty() {
        return;
    }
    let node = ui.tree_node_config("Finishes").default_open(false);
    let token = match expansion {
        Some(open) => node.opened(open, Condition::Always).push(),
        None => node.push(),
    };
    if let Some(_folder) = token {
        for feature in finishes {
            let _id = ui.push_id(format!("finish-row-{}", feature.id.0));
            let _row =
                browser_feature::feature_row(ui, state, browser, feature, expansion, view, actions);
            if browser
                .finish_edits
                .get(&feature.id)
                .is_some_and(|seed| seed.has_warning)
            {
                let max = ui.item_rect_max();
                let min = ui.item_rect_min();
                let cx = max[0] - 12.0;
                let cy = (min[1] + max[1]) * 0.5;
                ui.get_window_draw_list()
                    .add_triangle(
                        [cx, cy - 6.0],
                        [cx - 6.0, cy + 5.0],
                        [cx + 6.0, cy + 5.0],
                        WARNING_AMBER,
                    )
                    .filled(true)
                    .build();
            }
        }
    }
}
