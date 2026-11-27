// SPDX-License-Identifier: GPL-2.0-only
//! Resolve and lay out a parts list from a current, host-owned BOM.
use super::{
    DrawingAnnotationDefinition, DrawingError, DrawingTablePresentation, table::grid_table,
};

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
/// The table geometry produced from a parts list.
pub type DrawingPartsListPresentation = DrawingTablePresentation;

impl DrawingAnnotationDefinition {
    /// Resolve and lay out a parts list; example: `annotation.evaluate_parts_list("assembly.oba", source)?`.
    pub fn evaluate_parts_list(
        &self,
        reference: &str,
        source: &dyn DrawingPartsListSource,
    ) -> Result<DrawingPartsListPresentation, DrawingError> {
        self.validate()?;
        if !matches!(self, Self::PartsList { .. }) {
            return Err(DrawingError(format!(
                "annotation {:?}: expected a parts list",
                self.name()
            )));
        }
        let rows = resolve_rows(reference, source)?;
        self.table_from_rows(&rows)
    }
    fn table_from_rows(
        &self,
        rows: &[DrawingPartsListRow],
    ) -> Result<DrawingPartsListPresentation, DrawingError> {
        self.validate()?;
        let Self::PartsList { x_mm, y_mm, .. } = self else {
            return Err(DrawingError(format!(
                "annotation {:?}: expected a parts list",
                self.name()
            )));
        };
        Ok(parts_list_geometry([*x_mm, *y_mm], rows))
    }
}
fn resolve_rows(
    reference: &str,
    source: &dyn DrawingPartsListSource,
) -> Result<Vec<DrawingPartsListRow>, DrawingError> {
    super::validation::require_label("parts list model reference", reference)?;
    source.parts_list_rows(reference).map_err(|cause| {
        DrawingError(format!(
            "parts list model {reference:?}: expected a current BOM: {cause}"
        ))
    })
}
fn parts_list_geometry(
    origin: [f64; 2],
    rows: &[DrawingPartsListRow],
) -> DrawingPartsListPresentation {
    let cells: Vec<[String; 4]> = rows.iter().map(row_cells).collect();
    grid_table(
        origin,
        &COLUMNS.map(|(_, width)| width),
        &COLUMNS.map(|(heading, _)| heading.into()),
        cells.iter().map(|row| row.as_slice()),
    )
}

fn row_cells(row: &DrawingPartsListRow) -> [String; 4] {
    [
        row.item.to_string(),
        row.part_number.clone(),
        row.description.clone(),
        row.quantity.to_string(),
    ]
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
        let mut parts_lists = target
            .annotations
            .iter()
            .filter(|annotation| annotation.requires_model())
            .peekable();
        if parts_lists.peek().is_none() {
            return Ok(Vec::new());
        }
        let rows = resolve_rows(self.model_reference(), source)?;
        parts_lists
            .map(|annotation| {
                let table = annotation.table_from_rows(&rows)?;
                Ok((annotation.name().to_string(), table))
            })
            .collect()
    }
}
