// SPDX-License-Identifier: GPL-2.0-only
//! Resolve and lay out a parts list from a current, host-owned BOM.
use super::{DrawingAnnotationDefinition, DrawingError};

const ROW_HEIGHT_MM: f64 = 8.;
const COLUMNS: [(&str, f64); 4] = [
    ("ITEM", 12.),
    ("PART NUMBER", 40.),
    ("DESCRIPTION", 50.),
    ("QTY", 12.),
];
/// One current parts-only BOM row supplied by the referenced model's owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct DrawingPartsListRow {
    pub item: u32,
    pub part_number: String,
    pub description: String,
    pub quantity: u32,
}
/// Resolve a model reference to current BOM rows without retaining a document handle.
pub trait DrawingPartsListSource {
    /// Read current rows; example: `source.parts_list_rows("assembly.oba")?`.
    fn parts_list_rows(&self, model_reference: &str) -> Result<Vec<DrawingPartsListRow>, String>;
}
/// A line segment in sheet millimeters.
#[derive(Clone, Debug, PartialEq)]
pub struct DrawingTableLine {
    pub start: [f64; 2],
    pub end: [f64; 2],
}
/// A table cell's text and center in sheet millimeters.
#[derive(Clone, Debug, PartialEq)]
pub struct DrawingTableLabel {
    pub text: String,
    pub center: [f64; 2],
}
/// Derived table geometry; it is rebuilt from the source and is never persisted.
#[derive(Clone, Debug, PartialEq)]
pub struct DrawingPartsListPresentation {
    pub lines: Vec<DrawingTableLine>,
    pub labels: Vec<DrawingTableLabel>,
    pub row_count: usize,
}
impl DrawingAnnotationDefinition {
    /// Resolve and lay out a parts list; example: `annotation.evaluate_parts_list("assembly.oba", source)?`.
    pub fn evaluate_parts_list(
        &self,
        reference: &str,
        source: &dyn DrawingPartsListSource,
    ) -> Result<DrawingPartsListPresentation, DrawingError> {
        self.validate()?;
        super::validation::require_label("parts list model reference", reference)?;
        let rows = source.parts_list_rows(reference).map_err(|cause| {
            DrawingError(format!(
                "parts list model {reference:?}: expected a current BOM: {cause}"
            ))
        })?;
        let Self::PartsList { x_mm, y_mm, .. } = self;
        Ok(parts_list_geometry([*x_mm, *y_mm], &rows))
    }
}
fn parts_list_geometry(
    origin: [f64; 2],
    rows: &[DrawingPartsListRow],
) -> DrawingPartsListPresentation {
    let cells: Vec<[String; 4]> = rows.iter().map(row_cells).collect();
    let mut labels = Vec::new();
    add_labels(
        &mut labels,
        origin,
        &COLUMNS.map(|(heading, _)| heading.to_string()),
    );
    for (index, row) in cells.iter().enumerate() {
        add_labels(
            &mut labels,
            [origin[0], origin[1] - (index + 1) as f64 * ROW_HEIGHT_MM],
            row,
        );
    }
    DrawingPartsListPresentation {
        lines: grid_lines(origin, rows.len()),
        labels,
        row_count: rows.len(),
    }
}
fn row_cells(row: &DrawingPartsListRow) -> [String; 4] {
    [
        row.item.to_string(),
        row.part_number.clone(),
        row.description.clone(),
        row.quantity.to_string(),
    ]
}
fn add_labels(labels: &mut Vec<DrawingTableLabel>, origin: [f64; 2], values: &[String; 4]) {
    let mut x = origin[0];
    for ((_, width), text) in COLUMNS.iter().zip(values) {
        labels.push(DrawingTableLabel {
            text: text.clone(),
            center: [x + width / 2., origin[1] - ROW_HEIGHT_MM / 2.],
        });
        x += width;
    }
}
fn grid_lines(origin: [f64; 2], row_count: usize) -> Vec<DrawingTableLine> {
    let total: f64 = COLUMNS.iter().map(|(_, width)| width).sum();
    let bottom = origin[1] - (row_count + 1) as f64 * ROW_HEIGHT_MM;
    let mut lines = Vec::new();
    for row in 0..=row_count + 1 {
        let y = origin[1] - row as f64 * ROW_HEIGHT_MM;
        lines.push(DrawingTableLine {
            start: [origin[0], y],
            end: [origin[0] + total, y],
        });
    }
    let mut x = origin[0];
    for (_, width) in COLUMNS {
        lines.push(DrawingTableLine {
            start: [x, origin[1]],
            end: [x, bottom],
        });
        x += width;
    }
    lines.push(DrawingTableLine {
        start: [x, origin[1]],
        end: [x, bottom],
    });
    lines
}

impl super::DrawingDocument {
    /// Resolve every parts list on a sheet using the drawing's model reference.
    /// Example: `drawing.evaluate_parts_lists(0, source)?` rebuilds the active sheet's tables.
    pub fn evaluate_parts_lists(
        &self,
        sheet: usize,
        source: &dyn DrawingPartsListSource,
    ) -> Result<Vec<(String, DrawingPartsListPresentation)>, DrawingError> {
        let target = self.sheets().get(sheet).ok_or_else(|| {
            DrawingError(format!(
                "sheet index {sheet}: expected an index below {}",
                self.sheets().len()
            ))
        })?;
        target
            .annotations
            .iter()
            .map(|annotation| {
                let table = annotation.evaluate_parts_list(self.model_reference(), source)?;
                Ok((annotation.name().to_string(), table))
            })
            .collect()
    }
}
