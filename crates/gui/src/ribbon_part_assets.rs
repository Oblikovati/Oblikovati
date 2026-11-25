// SPDX-License-Identifier: GPL-2.0-only
//! Part material and appearance selections call the shared asset operations.
use crate::{GuiCommandState, GuiPartAssetKind, GuiShellAction};
use imgui::Ui;

pub(crate) fn draw(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    icons: &crate::icons::GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    let disabled = ui.begin_disabled(
        state.authoring != crate::GuiAuthoringContext::Part || state.transaction_open,
    );
    if let Some(browser) = &state.browser {
        asset_row(ui, state, browser, GuiPartAssetKind::Material, actions);
        let position = match icons.appearance {
            crate::ribbon_command_layout::RibbonAppearance::Compact => [200., 0.],
            crate::ribbon_command_layout::RibbonAppearance::Small => [0., 24.],
            _ => [0., 30.],
        };
        icons.cursor(ui, origin, position);
        asset_row(ui, state, browser, GuiPartAssetKind::Appearance, actions);
    }
    drop(disabled);
}
fn asset_row(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &crate::GuiModelBrowser,
    kind: GuiPartAssetKind,
    actions: &mut Vec<GuiShellAction>,
) {
    let (label, selected, choices) = row_choices(browser, kind);
    if let Some(id) = asset_combo(ui, label, selected, choices) {
        queue_asset(state, kind, &id, actions);
    }
}

/// Show the active Part's current assignments in Document Settings.
pub(crate) fn document_asset_summary(ui: &Ui, state: &GuiCommandState) {
    let Some(browser) = state.browser.as_ref().filter(|browser| {
        matches!(
            browser.kind,
            crate::GuiBrowserKind::Part | crate::GuiBrowserKind::SheetMetal
        )
    }) else {
        ui.text_disabled("Part assignments unavailable");
        return;
    };
    for kind in [GuiPartAssetKind::Material, GuiPartAssetKind::Appearance] {
        let (label, selected, choices) = row_choices(browser, kind);
        ui.text(format!(
            "Part {label}: {}",
            asset_preview(choices, selected)
        ));
    }
}
/// Choose one named asset; example: `asset_combo(ui, "Material", current, choices)`.
pub(crate) fn asset_combo(
    ui: &Ui,
    label: &str,
    selected: Option<&str>,
    choices: &[crate::GuiAssetChoice],
) -> Option<String> {
    let preview = asset_preview(choices, selected);
    if let Some(rgb) = choices
        .iter()
        .find(|asset| Some(asset.id.as_str()) == selected)
        .and_then(|asset| asset.swatch)
    {
        ui.dummy([16., ui.frame_height()]);
        paint_swatch(ui, rgb);
        ui.same_line();
    }
    ui.set_next_item_width(150.);
    let _list = ui.begin_combo(label, preview)?;
    choices
        .iter()
        .find(|asset| select_asset(ui, asset, selected))
        .map(|asset| asset.id.clone())
}
fn row_choices(
    browser: &crate::GuiModelBrowser,
    kind: GuiPartAssetKind,
) -> (&str, Option<&str>, &[crate::GuiAssetChoice]) {
    match kind {
        GuiPartAssetKind::Material => (
            "Material",
            browser.part_material.as_deref(),
            &browser.materials,
        ),
        GuiPartAssetKind::Appearance => (
            "Appearance",
            browser.part_appearance.as_deref(),
            &browser.appearances,
        ),
    }
}
fn select_asset(ui: &Ui, asset: &crate::GuiAssetChoice, selected: Option<&str>) -> bool {
    let _id = ui.push_id(&asset.id);
    let already_selected = Some(asset.id.as_str()) == selected;
    let caption = if asset.swatch.is_some() {
        format!("       {}", asset.name)
    } else {
        asset.name.clone()
    };
    let clicked = ui
        .selectable_config(caption)
        .selected(already_selected)
        .build();
    if let Some(rgb) = asset.swatch {
        paint_swatch(ui, rgb);
    }
    clicked && !already_selected
}
fn paint_swatch(ui: &Ui, [r, g, b]: [u8; 3]) {
    let [x, top] = ui.item_rect_min();
    let y = (top + ui.item_rect_max()[1]) * 0.5;
    let minimum = [x + 2., y - 5.];
    let maximum = [x + 14., y + 5.];
    let color = [r, g, b, 255].map(|channel| f32::from(channel) / 255.);
    let draw = ui.get_window_draw_list();
    draw.add_rect(minimum, maximum, color).filled(true).build();
    draw.add_rect(minimum, maximum, crate::theme::strong_border(ui))
        .build();
}
fn asset_preview<'a>(choices: &'a [crate::GuiAssetChoice], selected: Option<&'a str>) -> &'a str {
    choices
        .iter()
        .find(|asset| Some(asset.id.as_str()) == selected)
        .map_or_else(
            || selected.unwrap_or("Default"),
            |asset| asset.name.as_str(),
        )
}
/// Queue an asset assignment; example: `queue_asset(state, kind, id, actions)`.
pub(crate) fn queue_asset(
    state: &GuiCommandState,
    kind: GuiPartAssetKind,
    id: &str,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(document) = state.document else {
        return;
    };
    let action = GuiShellAction::AssignPartAsset {
        document,
        kind,
        id: id.to_owned(),
    };
    if action.enabled(state) {
        actions.push(action);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn row_choices_match_the_asset_kind() {
        let browser = crate::GuiModelBrowser {
            part_material: Some("steel".into()),
            part_appearance: Some("black".into()),
            ..Default::default()
        };
        assert_eq!(
            row_choices(&browser, GuiPartAssetKind::Material).1,
            Some("steel")
        );
        assert_eq!(
            row_choices(&browser, GuiPartAssetKind::Appearance).1,
            Some("black")
        );
    }
    #[test]
    fn preview_shows_unknown_identity_instead_of_a_false_default() {
        let choices = [crate::GuiAssetChoice {
            id: "steel".into(),
            name: "Steel".into(),
            swatch: None,
        }];
        assert_eq!(asset_preview(&choices, Some("steel")), "Steel");
        assert_eq!(asset_preview(&choices, Some("missing")), "missing");
        assert_eq!(asset_preview(&choices, None), "Default");
    }
    #[test]
    fn asset_actions_require_an_active_part_without_a_transaction() {
        let mut state = GuiCommandState {
            document: Some(3),
            authoring: crate::GuiAuthoringContext::Part,
            ..Default::default()
        };
        let mut actions = Vec::new();
        queue_asset(&state, GuiPartAssetKind::Material, "steel", &mut actions);
        assert_eq!(
            actions,
            [GuiShellAction::AssignPartAsset {
                document: 3,
                kind: GuiPartAssetKind::Material,
                id: "steel".into()
            }]
        );
        state.transaction_open = true;
        queue_asset(&state, GuiPartAssetKind::Appearance, "black", &mut actions);
        assert_eq!(actions.len(), 1);
        state.transaction_open = false;
        state.authoring = crate::GuiAuthoringContext::Sketch;
        queue_asset(&state, GuiPartAssetKind::Appearance, "black", &mut actions);
        assert_eq!(actions.len(), 1);
    }
}
