// SPDX-License-Identifier: GPL-2.0-only
//! Part material and appearance selections call the shared asset operations.
use crate::{GuiCommandState, GuiPartAssetKind, GuiShellAction};
use imgui::Ui;

pub(crate) fn draw(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    let disabled = ui.begin_disabled(
        state.authoring != crate::GuiAuthoringContext::Part || state.transaction_open,
    );
    if let Some(browser) = &state.browser {
        asset_row(ui, state, browser, GuiPartAssetKind::Material, actions);
        crate::widget_abi::cursor_position(ui, [origin[0], origin[1] + 30.]);
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
    let preview = asset_preview(choices, selected);
    ui.set_next_item_width(150.);
    let Some(_list) = ui.begin_combo(label, preview) else {
        return;
    };
    for asset in choices {
        if select_asset(ui, asset, selected) {
            queue_asset(state, kind, &asset.id, actions);
        }
    }
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
    ui.selectable_config(&asset.name)
        .selected(already_selected)
        .build()
        && !already_selected
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
fn queue_asset(
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
