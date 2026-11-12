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
fn horizontal(
    view: &mut BrowserView,
    browser: &GuiModelBrowser,
    row: FeatureRow,
    key: Key,
) -> Option<FeatureRow> {
    match (row, key) {
        (FeatureRow::Feature(feature), Key::LeftArrow | Key::RightArrow) => {
            let open = *view.feature_open.get(&feature)?;
            if (key == Key::LeftArrow && open) || (key == Key::RightArrow && !open) {
                view.feature_expansion = Some((feature, !open));
                return Some(row);
            }
            if key == Key::RightArrow && open {
                let next = neighbour(&view.feature_rows, row, Key::DownArrow)?;
                return Some(if matches!(next, FeatureRow::Sketch(_)) {
                    next
                } else {
                    row
                });
            }
            Some(row)
        }
        (FeatureRow::Sketch(sketch), Key::LeftArrow) => {
            let parent = browser
                .sketches
                .iter()
                .find(|item| item.id == sketch)?
                .consumed_by?;
            if view.feature_open.get(&parent) == Some(&false) {
                view.feature_expansion = Some((parent, true));
            }
            Some(FeatureRow::Feature(parent))
        }
        (FeatureRow::Sketch(_), Key::RightArrow) => Some(row),
        _ => None,
    }
}
fn belongs_to_feature_tree(browser: &GuiModelBrowser, view: &BrowserView, row: FeatureRow) -> bool {
    if view.feature_rows.contains(&row) {
        return true;
    }
    let FeatureRow::Sketch(sketch) = row else {
        return false;
    };
    browser
        .sketches
        .iter()
        .find(|item| item.id == sketch)
        .and_then(|item| item.consumed_by)
        .is_some_and(|feature| view.feature_rows.contains(&FeatureRow::Feature(feature)))
}
pub(crate) fn follow(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
) {
    let focus = imgui::WindowFocusedFlags::ROOT_AND_CHILD_WINDOWS;
    let current = selected(browser).filter(|row| belongs_to_feature_tree(browser, view, *row));
    view.feature_keyboard_focus = current.is_some()
        && ui.is_window_focused_with_flags(focus)
        && !crate::widget_abi::any_popup_open(ui);
    if !view.feature_keyboard_focus {
        return;
    }
    let Some(key) = pressed_arrow(ui) else {
        return;
    };
    let Some(current) = current else {
        return;
    };
    let target = neighbour(&view.feature_rows, current, key)
        .or_else(|| horizontal(view, browser, current, key));
    let Some(row) = target else {
        return;
    };
    view.arrow_consumed = true;
    if row == current {
        return;
    }
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
    fn horizontal_keys_open_close_and_enter_visible_children() {
        let feature = FeatureId(1);
        let parent = FeatureRow::Feature(feature);
        let child = FeatureRow::Sketch(DocumentSketchId(2));
        let mut view = BrowserView::default();
        view.feature_rows = vec![parent, child];
        view.feature_open.insert(feature, true);
        let browser = GuiModelBrowser::default();
        assert_eq!(
            horizontal(&mut view, &browser, parent, Key::RightArrow),
            Some(child)
        );
        assert_eq!(
            horizontal(&mut view, &browser, parent, Key::LeftArrow),
            Some(parent)
        );
        assert_eq!(view.feature_expansion, Some((feature, false)));
        view.feature_open.insert(feature, false);
        assert_eq!(
            horizontal(&mut view, &browser, parent, Key::RightArrow),
            Some(parent)
        );
        assert_eq!(view.feature_expansion, Some((feature, true)));
    }
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
