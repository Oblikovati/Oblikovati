// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Condition, MouseButton, StyleColor, Ui};
use oblikovati_model::{document_sketch::DocumentSketchId, feature::FeatureId};

/// Immutable document structure; only the desktop owner can activate an edit.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiModelBrowser {
    pub name: String,
    pub has_origin: bool,
    pub sketches: Vec<GuiBrowserSketch>,
    pub features: Vec<GuiBrowserFeature>,
    pub selected_sketches: Vec<DocumentSketchId>,
}
/// Stable sketch identity paired with its current canonical API inspection or local decline.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiBrowserSketch {
    pub id: DocumentSketchId,
    pub inspection: Result<oblikovati_api::wire::SketchInfo, String>,
    pub name: String,
    pub visible: bool,
    pub consumed_by: Option<FeatureId>,
}
/// One authored feature in model program order, with its actual suppression state.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiBrowserFeature {
    pub id: FeatureId,
    pub name: String,
    pub suppressed: bool,
}
impl GuiModelBrowser {
    pub(crate) fn validate(&self) -> Result<(), String> {
        self.validate_labels()?;
        self.validate_identities()?;
        for id in &self.selected_sketches {
            if !self.sketches.iter().any(|row| row.id == *id) {
                return Err(format!(
                    "browser selected sketch {id:?}: expected a current sketch row"
                ));
            }
        }
        for row in &self.sketches {
            row.validate(&self.features)?;
        }
        Ok(())
    }
    fn validate_labels(&self) -> Result<(), String> {
        for text in std::iter::once(&self.name)
            .chain(self.sketches.iter().map(|row| &row.name))
            .chain(self.features.iter().map(|row| &row.name))
        {
            if text.contains('\0') {
                return Err(format!("browser label {text:?}: expected text without NUL"));
            }
        }
        Ok(())
    }
    fn validate_identities(&self) -> Result<(), String> {
        let sketches: std::collections::BTreeSet<_> =
            self.sketches.iter().map(|row| row.id.0).collect();
        let features: std::collections::BTreeSet<_> =
            self.features.iter().map(|row| row.id.0).collect();
        if sketches.len() != self.sketches.len()
            || features.len() != self.features.len()
            || sketches.contains(&0)
            || features.contains(&0)
        {
            return Err(format!(
                "browser identities sketches={sketches:?}, features={features:?}: expected unique nonzero IDs within each namespace"
            ));
        }
        Ok(())
    }
}
impl GuiBrowserSketch {
    fn validate(&self, features: &[GuiBrowserFeature]) -> Result<(), String> {
        self.validate_consumer(features)?;
        let tooltip = match &self.inspection {
            Ok(summary) => &summary.plane,
            Err(cause) => cause,
        };
        if tooltip.contains('\0') {
            return Err(format!(
                "browser sketch {:?} inspection {tooltip:?}: expected text without NUL",
                self.id
            ));
        }
        Ok(())
    }
    fn validate_consumer(&self, features: &[GuiBrowserFeature]) -> Result<(), String> {
        if self
            .consumed_by
            .is_some_and(|owner| !features.iter().any(|feature| feature.id == owner))
        {
            return Err(format!(
                "browser sketch {:?}, owner {:?}: expected an owner in the current feature program",
                self.id, self.consumed_by
            ));
        }
        Ok(())
    }
}
pub(crate) fn panel(
    ui: &Ui,
    state: &GuiCommandState,
    height: f32,
    band_height: f32,
    actions: &mut Vec<GuiShellAction>,
) {
    crate::widget_abi::window_geometry(
        ui,
        [8., band_height + 10.],
        [
            model_browser_width(ui.io().display_size[0]),
            (height - band_height - 18.).max(120.),
        ],
        Condition::Always,
    );
    ui.window("Model")
        .movable(false)
        .resizable(false)
        .build(|| browser_contents(ui, state, actions));
}
fn browser_contents(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some(browser) = &state.browser else {
        ui.text("(no document)");
        return;
    };
    let _document = ui.push_id(format!("document-{:?}", state.document));
    ui.text(&browser.name);
    origin_branch(ui, browser);
    parameter_branch(ui, state);
    sketch_branches(ui, state, browser, actions);
    feature_branch(ui, state, browser, actions);
}
fn origin_branch(ui: &Ui, browser: &GuiModelBrowser) {
    if !browser.has_origin {
        return;
    }
    if let Some(_origin) = ui.tree_node_config("Origin").default_open(true).push() {
        for label in [
            "Center Point",
            "X Axis",
            "Y Axis",
            "Z Axis",
            "XY Plane",
            "XZ Plane",
            "YZ Plane",
        ] {
            ui.bullet_text(label);
        }
    }
}
fn parameter_branch(ui: &Ui, state: &GuiCommandState) {
    if let Some(_parameters) = ui.tree_node_config("Parameters").default_open(false).push() {
        for row in &state.parameters {
            ui.bullet_text(format!("{} = {}", row.name, row.value));
        }
    }
}
fn sketch_branches(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    actions: &mut Vec<GuiShellAction>,
) {
    if let Some(_sketches) = ui.tree_node_config("Sketches").default_open(true).push() {
        for row in browser
            .sketches
            .iter()
            .filter(|row| row.consumed_by.is_none())
        {
            sketch_row(ui, state, row, actions);
        }
    }
}
fn feature_branch(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    actions: &mut Vec<GuiShellAction>,
) {
    if let Some(_features) = ui.tree_node_config("Features").default_open(true).push() {
        for feature in &browser.features {
            feature_row(ui, state, browser, feature, actions);
        }
    }
}
fn feature_row(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
    actions: &mut Vec<GuiShellAction>,
) {
    let _feature = ui.push_id(format!("feature-{}", feature.id.0));
    ui.text(if feature.suppressed {
        format!("{} (suppressed)", feature.name)
    } else {
        feature.name.clone()
    });
    ui.indent();
    for row in browser
        .sketches
        .iter()
        .filter(|row| row.consumed_by == Some(feature.id))
    {
        sketch_row(ui, state, row, actions);
    }
    ui.unindent();
}
fn sketch_row(
    ui: &Ui,
    state: &GuiCommandState,
    row: &GuiBrowserSketch,
    actions: &mut Vec<GuiShellAction>,
) {
    let _sketch = ui.push_id(format!("sketch-{}", row.id.0));
    let action = GuiShellAction::EditSketch {
        document: state.document.unwrap_or(0),
        sketch: row.id,
    };
    if sketch_selectable(
        ui,
        row,
        state
            .browser
            .as_ref()
            .is_some_and(|browser| browser.selected_sketches.contains(&row.id)),
    ) {
        sketch_activation(ui, state, row.id, &action, actions);
    }
    sketch_context_menu(ui, state, row, action, actions);
}
fn sketch_activation(
    ui: &Ui,
    state: &GuiCommandState,
    sketch: DocumentSketchId,
    edit: &GuiShellAction,
    actions: &mut Vec<GuiShellAction>,
) {
    let select = GuiShellAction::SelectBrowserSketch {
        document: state.document.unwrap_or(0),
        sketch,
        // #3811: edit activation must retain its sketch even after a modified first click.
        toggle: !ui.is_mouse_double_clicked(MouseButton::Left)
            && (ui.io().key_shift || ui.io().key_ctrl),
    };
    if select.dispatchable(state) {
        actions.push(select);
    }
    if ui.is_mouse_double_clicked(MouseButton::Left) && edit.dispatchable(state) {
        actions.push(edit.clone());
    }
}
fn sketch_selectable(ui: &Ui, row: &GuiBrowserSketch, selected: bool) -> bool {
    let position = ui.cursor_screen_pos();
    let activated = crate::widget_abi::selectable(ui, "##sketch-row", selected, true);
    // Literal labels are drawn separately so imported names containing ## remain visible.
    crate::widget_abi::literal_text(
        ui,
        position,
        ui.style_color(StyleColor::Text),
        &sketch_caption(row),
    );
    if ui.is_item_hovered() {
        ui.tooltip_text(sketch_tooltip(row));
    }
    activated
}
fn sketch_context_menu(
    ui: &Ui,
    state: &GuiCommandState,
    row: &GuiBrowserSketch,
    action: GuiShellAction,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(_popup) = ui.begin_popup_context_item() else {
        return;
    };
    if ui
        .menu_item_config("Edit Sketch")
        .enabled(action.dispatchable(state))
        .build()
    {
        actions.push(action);
    }
    sketch_visibility_menu(ui, state, row, actions);
}
fn sketch_visibility_menu(
    ui: &Ui,
    state: &GuiCommandState,
    row: &GuiBrowserSketch,
    actions: &mut Vec<GuiShellAction>,
) {
    let visibility = GuiShellAction::SetSketchVisibility {
        document: state.document.unwrap_or(0),
        sketch: row.id,
        visible: !row.visible,
    };
    if ui
        .menu_item_config("Visibility")
        .selected(row.visible)
        .enabled(visibility.dispatchable(state))
        .build()
    {
        actions.push(visibility);
    }
}
fn sketch_caption(row: &GuiBrowserSketch) -> String {
    match &row.inspection {
        Ok(summary) => format!(
            "{}  [DOF {}{}]",
            row.name,
            summary.dof,
            if summary.healthy { "" } else { ", sick" }
        ),
        Err(_) => format!("{}  [unavailable]", row.name),
    }
}
fn sketch_tooltip(row: &GuiBrowserSketch) -> String {
    match &row.inspection {
        Ok(summary) => format!(
            "{} plane, {} entities. Double-click to edit.",
            summary.plane, summary.entity_count
        ),
        Err(cause) => format!("{}: {cause}", row.name),
    }
}

#[cfg(test)]
#[path = "../tests/browser/unit.rs"]
mod tests;

fn model_browser_width(viewport_width: f32) -> f32 {
    // Keep the native view centre available for picks even when content scale narrows the layout (#3850).
    300_f32.min((viewport_width * 0.5 - 16.).max(1.))
}
