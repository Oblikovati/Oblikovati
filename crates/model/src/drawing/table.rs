// SPDX-License-Identifier: GPL-2.0-only
use super::{DrawingAnnotationDefinition, DrawingDocument, DrawingError, annotation::unique_name};

const ROW_HEIGHT_MM: f64 = 8.;
const CUSTOM_COLUMN_MM: f64 = 36.;

/// A line segment in sheet millimetres.
#[derive(Clone, Debug, PartialEq)]
pub struct DrawingTableLine {
    pub start: [f64; 2],
    pub end: [f64; 2],
}

/// The text and extent of one table cell in sheet millimetres.
#[derive(Clone, Debug, PartialEq)]
pub struct DrawingTableLabel {
    pub text: String,
    pub center: [f64; 2],
    pub cell_size: [f64; 2],
}

/// Derived grid and text geometry rebuilt from the table content.
#[derive(Clone, Debug, PartialEq)]
pub struct DrawingTablePresentation {
    pub lines: Vec<DrawingTableLine>,
    pub labels: Vec<DrawingTableLabel>,
    pub row_count: usize,
}

impl DrawingDocument {
    /// Insert an independent sheet table, e.g. `drawing.add_custom_table("Finish", [20., 200.], vec!["ZONE".into()], vec![])?`.
    pub fn add_custom_table(
        &mut self,
        requested_name: &str,
        [x_mm, y_mm]: [f64; 2],
        headers: Vec<String>,
        rows: Vec<Option<Vec<String>>>,
    ) -> Result<DrawingAnnotationDefinition, DrawingError> {
        let sheet = self.active_sheet();
        let name = unique_name(requested_name, &self.sheets()[sheet].annotations);
        let table = DrawingAnnotationDefinition::CustomTable {
            name,
            x_mm,
            y_mm,
            headers,
            rows,
            characteristic: None,
            material_removal: None,
        };
        self.add_annotation_definition(sheet, table.clone())?;
        Ok(table)
    }
}

impl DrawingAnnotationDefinition {
    /// Lay out user-authored table cells, e.g. `annotation.custom_table_presentation()`.
    pub fn custom_table_presentation(&self) -> Option<DrawingTablePresentation> {
        let Self::CustomTable {
            x_mm,
            y_mm,
            headers,
            rows,
            ..
        } = self
        else {
            return None;
        };
        let widths = vec![CUSTOM_COLUMN_MM; headers.len()];
        Some(grid_table(
            [*x_mm, *y_mm],
            &widths,
            headers,
            rows.iter().map(|row| row.as_deref().unwrap_or_default()),
        ))
    }
}

pub(super) fn validate_cells(
    headers: &[String],
    rows: &[Option<Vec<String>>],
) -> Result<(), DrawingError> {
    if headers.is_empty() {
        return Err(DrawingError(
            "custom table headers []: expected at least one column".into(),
        ));
    }
    for text in headers.iter().chain(rows.iter().flatten().flatten()) {
        if text.contains('\0') {
            return Err(DrawingError(format!(
                "custom table cell {text:?}: expected text without NUL"
            )));
        }
    }
    Ok(())
}

pub(super) fn grid_table<'a>(
    origin: [f64; 2],
    widths: &[f64],
    headers: &'a [String],
    rows: impl ExactSizeIterator<Item = &'a [String]>,
) -> DrawingTablePresentation {
    let row_count = rows.len();
    let mut labels = Vec::new();
    for (index, row) in std::iter::once(headers).chain(rows).enumerate() {
        let row_top = origin[1] - index as f64 * ROW_HEIGHT_MM;
        add_labels(&mut labels, [origin[0], row_top], widths, row);
    }
    DrawingTablePresentation {
        lines: grid_lines(origin, widths, row_count),
        labels,
        row_count,
    }
}

fn add_labels(
    labels: &mut Vec<DrawingTableLabel>,
    [mut x, top]: [f64; 2],
    widths: &[f64],
    values: &[String],
) {
    for (index, width) in widths.iter().copied().enumerate() {
        labels.push(DrawingTableLabel {
            text: values.get(index).cloned().unwrap_or_default(),
            center: [x + width / 2., top - ROW_HEIGHT_MM / 2.],
            cell_size: [width, ROW_HEIGHT_MM],
        });
        x += width;
    }
}

fn grid_lines([left, top]: [f64; 2], widths: &[f64], row_count: usize) -> Vec<DrawingTableLine> {
    let right = left + widths.iter().sum::<f64>();
    let bottom = top - (row_count + 1) as f64 * ROW_HEIGHT_MM;
    let mut lines: Vec<_> = (0..=row_count + 1)
        .map(|row| {
            let y = top - row as f64 * ROW_HEIGHT_MM;
            DrawingTableLine {
                start: [left, y],
                end: [right, y],
            }
        })
        .collect();
    let mut x = left;
    for width in widths.iter().copied().chain(std::iter::once(0.)) {
        lines.push(DrawingTableLine {
            start: [x, top],
            end: [x, bottom],
        });
        x += width;
    }
    lines
}
