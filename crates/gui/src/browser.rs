// SPDX-License-Identifier: GPL-2.0-only
use crate::application::workspace_layout::browser_view::BrowserView;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{MouseButton, StyleColor, Ui};
use oblikovati_model::{document_sketch::DocumentSketchId, feature::FeatureId};

/// The browser tree a document kind shows; each kind names its own branches.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiBrowserKind {
    #[default]
    Part,
    SheetMetal,
    Assembly,
    Drawing,
    Presentation,
}
/// Immutable document structure; only the desktop owner can activate an edit.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiModelBrowser {
    pub name: String,
    pub kind: GuiBrowserKind,
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
/// Draw the browser tree, or the rows a search leaves; example: inside the Model window.
pub(crate) fn browser_contents(
    ui: &Ui,
    state: &GuiCommandState,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(browser) = &state.browser else {
        ui.text("(no document)");
        return;
    };
    view.header(ui, browser.kind == crate::GuiBrowserKind::Drawing);
    let _document = ui.push_id(format!("document-{:?}", state.document));
    ui.text(&browser.name);
    if let Some(query) = view.query() {
        crate::application::workspace_layout::browser_view::search_results(ui, browser, query);
        return;
    }
    let tree = BrowserTree {
        ui,
        state,
        browser,
        expansion: view.take_expansion(),
    };
    tree.draw(view, actions);
}
/// One frame's tree; `expansion` is set for the single frame after Expand All or Collapse All.
struct BrowserTree<'a> {
    ui: &'a Ui,
    state: &'a GuiCommandState,
    browser: &'a GuiModelBrowser,
    expansion: Option<bool>,
}
impl BrowserTree<'_> {
    fn draw(&self, view: &mut BrowserView, actions: &mut Vec<GuiShellAction>) {
        match self.browser.kind {
            GuiBrowserKind::Part => self.modeling("End of Part", actions),
            GuiBrowserKind::SheetMetal => self.folded_model(actions),
            GuiBrowserKind::Assembly => self.assembly(actions),
            GuiBrowserKind::Drawing => self.drawing(view),
            GuiBrowserKind::Presentation => self.ui.text("Scene1"),
        }
    }
    fn branch(&self, label: &str, default_open: bool) -> Option<imgui::TreeNodeToken<'_>> {
        let node = self.ui.tree_node_config(label).default_open(default_open);
        match self.expansion {
            Some(open) => node.opened(open, imgui::Condition::Always).push(),
            None => node.push(),
        }
    }
    fn modeling(&self, end_marker: &str, actions: &mut Vec<GuiShellAction>) {
        self.origin();
        self.parameters();
        self.sketches(actions);
        self.features(actions);
        // The marker closes a modelled document's history; a document without an origin has none.
        if self.browser.has_origin {
            self.ui.text(end_marker);
        }
    }
    fn folded_model(&self, actions: &mut Vec<GuiShellAction>) {
        if let Some(_folded) = self.branch("Folded Model", true) {
            self.modeling("End of Folded", actions);
        }
    }
    fn assembly(&self, actions: &mut Vec<GuiShellAction>) {
        self.ui.text("Representations");
        self.ui.text("Relationships");
        self.modeling("End of Assembly", actions);
    }
    fn drawing(&self, view: &mut BrowserView) {
        for (branch, (label, rows)) in DRAWING_TREE.iter().enumerate() {
            let Some(_node) = self.branch(label, true) else {
                continue;
            };
            for (row, name) in rows.iter().enumerate() {
                let selected = view.selected_row == Some((branch, row));
                let chosen = if *name == SHEET_TITLE_BLOCK {
                    self.title_block_row(selected)
                } else {
                    self.ui.selectable_config(name).selected(selected).build()
                };
                if chosen {
                    view.selected_row = Some((branch, row));
                }
            }
        }
    }
    /// A selectable row that is also a branch: its arrow opens it, its text selects it.
    fn title_block_row(&self, selected: bool) -> bool {
        let node = self
            .ui
            .tree_node_config(SHEET_TITLE_BLOCK)
            .open_on_arrow(true)
            .default_open(true)
            .selected(selected);
        let token = match self.expansion {
            Some(open) => node.opened(open, imgui::Condition::Always).push(),
            None => node.push(),
        };
        let chosen = self.ui.is_item_clicked() && !self.ui.is_item_toggled_open();
        if token.is_some() {
            self.ui.text(TITLE_BLOCK_FIELD);
        }
        chosen
    }
    fn origin(&self) {
        if !self.browser.has_origin {
            return;
        }
        if let Some(_origin) = self.branch("Origin", true) {
            for label in ORIGIN_ROWS {
                self.ui.bullet_text(label);
            }
        }
    }
    fn parameters(&self) {
        if let Some(_parameters) = self.branch("Parameters", false) {
            for row in &self.state.parameters {
                self.ui.bullet_text(format!("{} = {}", row.name, row.value));
            }
        }
    }
    fn sketches(&self, actions: &mut Vec<GuiShellAction>) {
        if let Some(_sketches) = self.branch("Sketches", true) {
            for row in self
                .browser
                .sketches
                .iter()
                .filter(|row| row.consumed_by.is_none())
            {
                sketch_row(self.ui, self.state, row, actions);
            }
        }
    }
    fn features(&self, actions: &mut Vec<GuiShellAction>) {
        if let Some(_features) = self.branch("Features", true) {
            for feature in &self.browser.features {
                feature_row(self.ui, self.state, self.browser, feature, actions);
            }
        }
    }
}
const SHEET_TITLE_BLOCK: &str = "Title Block";
/// What a title block holds, one level under its row on the sheet.
const TITLE_BLOCK_FIELD: &str = "Field Text";
/// A drawing's browser tree: the built-in template's one sheet and the resources it draws from.
pub(crate) const DRAWING_TREE: [(&str, &[&str]); 2] = [
    (
        "Drawing Resources",
        &["Sheet Formats", "Default Border", "Default Title Block"],
    ),
    ("Sheet:1", &["Default Border", "Title Block"]),
];
pub(crate) const ORIGIN_ROWS: [&str; 7] = [
    "Center Point",
    "X Axis",
    "Y Axis",
    "Z Axis",
    "XY Plane",
    "XZ Plane",
    "YZ Plane",
];
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

pub(crate) fn model_browser_width(viewport_width: f32) -> f32 {
    // Keep the native view centre available for picks even when content scale narrows the layout (#3850).
    300_f32.min((viewport_width * 0.5 - 16.).max(1.))
}
