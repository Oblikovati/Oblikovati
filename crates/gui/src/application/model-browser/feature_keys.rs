// SPDX-License-Identifier: GPL-2.0-only
//! Arrow navigation through the feature rows that the browser actually displays.
use crate::application::model_browser::browser_keys::pressed_arrow;
use crate::application::workspace_layout::browser_view::BrowserView;
use crate::{GuiCommandState, GuiModelBrowser, GuiShellAction};
use imgui::{Key, Ui};
use oblikovati_model::{document_sketch::DocumentSketchId, feature::FeatureId};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum FeatureRow {
    Feature(FeatureId),
    Sketch(DocumentSketchId),
}
fn selected(browser: &GuiModelBrowser) -> Option<FeatureRow> {
    browser
        .selected_feature
        .map(FeatureRow::Feature)
        .or_else(|| {
            (browser.selected_sketches.len() == 1)
                .then(|| FeatureRow::Sketch(browser.selected_sketches[0]))
        })
}
fn neighbour(rows: &[FeatureRow], current: FeatureRow, key: Key) -> Option<FeatureRow> {
    let index = rows.iter().position(|row| *row == current)?;
    let next = match key {
        Key::DownArrow => (index + 1).min(rows.len() - 1),
        Key::UpArrow => index.saturating_sub(1),
        _ => return None,
    };
    Some(rows[next])
}
pub(crate) fn follow(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(key) = pressed_arrow(ui) else {
        return;
    };
    let Some(row) = selected(browser).and_then(|row| neighbour(&view.feature_rows, row, key))
    else {
        return;
    };
    view.arrow_consumed = true;
    let document = state.document.unwrap_or(0);
    let action = match row {
        FeatureRow::Feature(feature) => GuiShellAction::SelectBrowserFeature { document, feature },
        FeatureRow::Sketch(sketch) => GuiShellAction::SelectBrowserSketch {
            document,
            sketch,
            toggle: false,
        },
    };
    if action.dispatchable(state) {
        actions.push(action);
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn navigation_uses_visible_order_and_stops_at_the_ends() {
        let rows = [
            FeatureRow::Feature(FeatureId(1)),
            FeatureRow::Sketch(DocumentSketchId(2)),
            FeatureRow::Feature(FeatureId(3)),
        ];
        assert_eq!(neighbour(&rows, rows[0], Key::DownArrow), Some(rows[1]));
        assert_eq!(neighbour(&rows, rows[1], Key::UpArrow), Some(rows[0]));
        assert_eq!(neighbour(&rows, rows[2], Key::DownArrow), Some(rows[2]));
        assert_eq!(neighbour(&rows, rows[0], Key::UpArrow), Some(rows[0]));
        assert_eq!(neighbour(&rows, rows[0], Key::LeftArrow), None);
        assert_eq!(
            neighbour(&rows, FeatureRow::Feature(FeatureId(9)), Key::DownArrow),
            None
        );
    }
}
