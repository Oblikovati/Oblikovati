// SPDX-License-Identifier: GPL-2.0-only
//! Model Browser 3rd Party folder, Reference Model context menu, and crossed-link badge.
use crate::{GuiBrowserFeature, GuiCommandState, GuiImportAction, GuiModelBrowser, GuiShellAction};
use imgui::{Condition, Ui};
use oblikovati_model::feature::imported_model::ImportMode;

/// Draw the 3rd Party folder listing imported mesh entries (e.g. STL files).
pub(crate) fn third_party_folder(ui: &Ui, browser: &GuiModelBrowser, expansion: Option<bool>) {
    if browser.third_party_entries.is_empty() {
        return;
    }
    let node = ui.tree_node_config("3rd Party").default_open(true);
    let token = match expansion {
        Some(open) => node.opened(open, Condition::Always).push(),
        None => node.push(),
    };
    if let Some(_folder) = token {
        for entry in &browser.third_party_entries {
            ui.bullet_text(entry);
        }
    }
}

/// Draw a crossed-link visual badge on a reference feature row when its link is suppressed.
pub(crate) fn draw_link_indicator(
    ui: &Ui,
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
    label_pos: [f32; 2],
) {
    let Some(seed) = browser.import_edits.get(&feature.id) else {
        return;
    };
    if seed.import_mode != ImportMode::ReferenceModel || seed.link_broken || !seed.link_suppressed {
        return;
    }
    let x = label_pos[0] + 1.;
    let y = label_pos[1] + (ui.text_line_height() - 14.) * 0.5;
    let draw = ui.get_window_draw_list();
    let red = [0.92, 0.28, 0.28, 1.0];
    draw.add_line([x, y + 13.], [x + 14., y + 1.], red)
        .thickness(2.0)
        .build();
    draw.add_line([x, y + 1.], [x + 14., y + 13.], red)
        .thickness(2.0)
        .build();
}

/// Format a browser feature row's plain text label without reference link suffixes.
pub(crate) fn plain_feature_label(feature: &GuiBrowserFeature, extended_names: bool) -> String {
    let mut label = feature.name.clone();
    if feature.suppressed {
        label.push_str(" (suppressed)");
    }
    if let (true, Some(extended)) = (extended_names, &feature.extended) {
        label.push_str(&format!(" ({extended})"));
    }
    label
}

/// Format a browser feature row's text label, including suppression and link-suppressed cues.
pub(crate) fn feature_label(
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
    extended_names: bool,
) -> String {
    let mut label = plain_feature_label(feature, extended_names);
    label.push_str(reference_label_suffix(browser, feature));
    label
}

/// Append a suffix cue to a suppressed reference feature label if applicable.
pub(crate) fn reference_label_suffix(
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
) -> &'static str {
    let Some(seed) = browser.import_edits.get(&feature.id) else {
        return "";
    };
    if seed.import_mode == ImportMode::ReferenceModel && !seed.link_broken && seed.link_suppressed {
        " [Link Suppressed]"
    } else {
        ""
    }
}

/// Draw the reference-model context menu (`Edit Import`, `Suppress/Unsuppress Link`, `Break Link`).
pub(crate) fn reference_menu(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    let Some(seed) = browser.import_edits.get(&feature.id) else {
        return false;
    };
    if seed.import_mode != ImportMode::ReferenceModel || seed.link_broken {
        return false;
    }
    let document = state.document.unwrap_or(0);
    let edit_action = GuiShellAction::Import(GuiImportAction::OpenEdit(feature.id));
    if ui
        .menu_item_config("Edit Import")
        .enabled(edit_action.enabled(state))
        .build()
    {
        actions.push(edit_action);
    }
    let (link_label, next_suppressed) = if seed.link_suppressed {
        ("Unsuppress Link", false)
    } else {
        ("Suppress Link", true)
    };
    let link_action = GuiShellAction::Import(GuiImportAction::SetLinkSuppressed {
        document,
        feature: feature.id,
        suppressed: next_suppressed,
    });
    if ui
        .menu_item_config(link_label)
        .enabled(link_action.enabled(state))
        .build()
    {
        actions.push(link_action);
    }
    let break_action = GuiShellAction::Import(GuiImportAction::BreakLink {
        document,
        feature: feature.id,
    });
    if ui
        .menu_item_config("Break Link")
        .enabled(break_action.enabled(state))
        .build()
    {
        actions.push(break_action);
    }
    true
}
