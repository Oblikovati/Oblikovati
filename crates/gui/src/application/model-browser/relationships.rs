// SPDX-License-Identifier: GPL-2.0-only
//! The Relationships dialog: what a browser row is built from, and what is built from it.
use crate::browser::ORIGIN_ROWS;
use crate::{GuiCommandState, GuiModelBrowser, GuiShellAction};
use imgui::{Condition, Ui};
use oblikovati_model::{document_sketch::DocumentSketchId, feature::FeatureId};

const TITLE: &str = "Relationships";
const SIZE: [f32; 2] = [420., 380.];
const CENTER_POINT: &str = "Center Point";
const LIST_HEIGHT: f32 = 100.;
const GO_BUTTON: [f32; 2] = [28., 0.];
const NO_ROWS: &str = "None";

/// A browser row a relationship can name.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum RelatedRow {
    /// A row of Origin, by its place among them.
    Origin(usize),
    Sketch(DocumentSketchId),
    Feature(FeatureId),
}
fn origin_row_named(name: &str) -> Option<RelatedRow> {
    ORIGIN_ROWS
        .iter()
        .position(|row| *row == name)
        .map(RelatedRow::Origin)
}
/// The origin plane a sketch stands on; its row reads `XY Plane` where the sketch says `XY`.
pub(crate) fn plane_of_sketch(
    browser: &GuiModelBrowser,
    sketch: DocumentSketchId,
) -> Option<RelatedRow> {
    let row = browser.sketches.iter().find(|row| row.id == sketch)?;
    let plane = &row.inspection.as_ref().ok()?.plane;
    origin_row_named(&format!("{plane} Plane"))
}
fn sketch_parents(browser: &GuiModelBrowser, sketch: DocumentSketchId) -> Vec<RelatedRow> {
    // Every sketch starts from the projected origin point of its plane.
    let center = origin_row_named(CENTER_POINT);
    plane_of_sketch(browser, sketch)
        .into_iter()
        .chain(center)
        .collect()
}
fn sketch_children(browser: &GuiModelBrowser, sketch: DocumentSketchId) -> Vec<RelatedRow> {
    browser
        .sketches
        .iter()
        .filter(|row| row.id == sketch)
        .filter_map(|row| row.consumed_by.map(RelatedRow::Feature))
        .collect()
}
fn feature_parents(browser: &GuiModelBrowser, feature: FeatureId) -> Vec<RelatedRow> {
    let source = browser
        .sketches
        .iter()
        .find(|row| row.consumed_by == Some(feature));
    source.map_or_else(Vec::new, |row| {
        plane_of_sketch(browser, row.id)
            .into_iter()
            .chain([RelatedRow::Sketch(row.id)])
            .collect()
    })
}
/// A plane's sketches, and whatever each of them went on to make; the origin itself has no parent.
fn origin_children(browser: &GuiModelBrowser, row: usize) -> Vec<RelatedRow> {
    let stands_on = |sketch: DocumentSketchId| {
        ORIGIN_ROWS[row] == CENTER_POINT
            || plane_of_sketch(browser, sketch) == Some(RelatedRow::Origin(row))
    };
    let mut children = Vec::new();
    for sketch in browser
        .sketches
        .iter()
        .filter(|sketch| stands_on(sketch.id))
    {
        children.push(RelatedRow::Sketch(sketch.id));
        children.extend(sketch.consumed_by.map(RelatedRow::Feature));
    }
    children
}
/// The rows a row is built from, then the rows built from it.
/// Example: `relations_of(&browser, RelatedRow::Sketch(id))`.
pub(crate) fn relations_of(
    browser: &GuiModelBrowser,
    row: RelatedRow,
) -> (Vec<RelatedRow>, Vec<RelatedRow>) {
    match row {
        RelatedRow::Origin(place) => (Vec::new(), origin_children(browser, place)),
        RelatedRow::Sketch(sketch) => (
            sketch_parents(browser, sketch),
            sketch_children(browser, sketch),
        ),
        RelatedRow::Feature(feature) => (feature_parents(browser, feature), Vec::new()),
    }
}
pub(crate) fn row_name(browser: &GuiModelBrowser, row: RelatedRow) -> String {
    match row {
        RelatedRow::Origin(place) => ORIGIN_ROWS
            .get(place)
            .copied()
            .unwrap_or_default()
            .to_owned(),
        RelatedRow::Sketch(sketch) => browser
            .sketches
            .iter()
            .find(|row| row.id == sketch)
            .map_or_else(String::new, |row| row.name.clone()),
        RelatedRow::Feature(feature) => browser
            .features
            .iter()
            .find(|row| row.id == feature)
            .map_or_else(String::new, |row| row.name.clone()),
    }
}
/// What the dialog asks the browser to do this frame.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum RowRequest {
    Choose(RelatedRow),
    Show(RelatedRow),
}
#[derive(Debug, Default)]
pub(crate) struct RelationshipsDialog {
    shown: Option<RelatedRow>,
    chosen: Option<RelatedRow>,
    /// The origin row the browser should show as chosen, taken once.
    chosen_origin: Option<usize>,
}
impl RelationshipsDialog {
    /// Open on one row; example: from its menu's Relationships.
    pub(crate) fn request(&mut self, row: RelatedRow) {
        self.shown = Some(row);
        self.chosen = None;
    }
    pub(crate) fn take_chosen_origin(&mut self) -> Option<usize> {
        self.chosen_origin.take()
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        browser: &GuiModelBrowser,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(shown) = self.shown else {
            return;
        };
        let mut request = None;
        let mut done = false;
        ui.window(TITLE)
            .size(SIZE, Condition::Appearing)
            // Beside the model, since what it chooses shows behind it.
            .position(
                [ui.io().display_size[0] - SIZE[0] - 10., 170.],
                Condition::Appearing,
            )
            .collapsible(false)
            .build(|| {
                ui.text(format!("Selected: {}", row_name(browser, shown)));
                let (parents, children) = relations_of(browser, shown);
                self.list(ui, browser, "Parents", &parents, &mut request);
                self.list(ui, browser, "Children", &children, &mut request);
                ui.separator();
                done = crate::widget_abi::button(ui, "Done", [80., 0.]);
            });
        self.follow(request, state, actions);
        if done {
            self.shown = None;
        }
    }
    fn list(
        &self,
        ui: &Ui,
        browser: &GuiModelBrowser,
        heading: &str,
        rows: &[RelatedRow],
        request: &mut Option<RowRequest>,
    ) {
        ui.text(heading);
        ui.child_window(format!("##relations-{heading}"))
            .size([0., LIST_HEIGHT])
            .border(true)
            .build(|| {
                if rows.is_empty() {
                    ui.text_disabled(NO_ROWS);
                }
                for row in rows {
                    let _row = ui.push_id(format!("{row:?}"));
                    if crate::widget_abi::button(ui, ">", GO_BUTTON) {
                        *request = Some(RowRequest::Show(*row));
                    }
                    ui.same_line();
                    let name = row_name(browser, *row);
                    if ui
                        .selectable_config(&name)
                        .selected(self.chosen == Some(*row))
                        .build()
                    {
                        *request = Some(RowRequest::Choose(*row));
                    }
                }
            });
    }
    fn follow(
        &mut self,
        request: Option<RowRequest>,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        match request {
            Some(RowRequest::Show(row)) => {
                self.shown = Some(row);
                self.chosen = None;
            }
            Some(RowRequest::Choose(row)) => {
                self.chosen = Some(row);
                self.choose_in_browser(row, state, actions);
            }
            None => {}
        }
    }
    fn choose_in_browser(
        &mut self,
        row: RelatedRow,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let document = state.document.unwrap_or(0);
        match row {
            RelatedRow::Origin(place) => {
                self.chosen_origin = Some(place);
                actions.push(GuiShellAction::BrowserChoice {
                    document,
                    choice: crate::GuiBrowserChoice::Nothing,
                });
            }
            RelatedRow::Sketch(sketch) => actions.push(GuiShellAction::SelectBrowserSketch {
                document,
                sketch,
                toggle: false,
            }),
            RelatedRow::Feature(feature) => {
                actions.push(GuiShellAction::SelectBrowserFeature { document, feature });
            }
        }
    }
}

#[cfg(test)]
#[path = "../../../tests/application/model-browser/relationships.rs"]
mod tests;
