// SPDX-License-Identifier: GPL-2.0-only
use super::{
    DrawingAnnotationDefinition, DrawingDocument, DrawingError, DrawingTablePresentation,
    annotation::unique_name, table::grid_table,
};
use serde::{Deserialize, Serialize};

/// An authored row of a drawing's revision history.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct DrawingRevisionRow {
    pub revision: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub date: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub description: String,
}

/// The triangle and centred text of a revision tag in sheet millimetres.
pub struct DrawingRevisionTagPresentation<'a> {
    pub text: &'a str,
    pub text_center_mm: [f64; 2],
    pub segments_mm: [[[f64; 2]; 2]; 3],
}

impl DrawingDocument {
    /// Record authored revision history, e.g. `drawing.add_revision_table("Revisions", [20., 200.], rows)?`.
    pub fn add_revision_table(
        &mut self,
        requested_name: &str,
        [x_mm, y_mm]: [f64; 2],
        revisions: Vec<DrawingRevisionRow>,
    ) -> Result<DrawingAnnotationDefinition, DrawingError> {
        let sheet = self.active_sheet();
        let name = unique_name(requested_name, &self.sheets()[sheet].annotations);
        let table = DrawingAnnotationDefinition::RevisionTable {
            name,
            x_mm,
            y_mm,
            revisions,
            characteristic: None,
            material_removal: None,
        };
        self.add_annotation_definition(sheet, table.clone())?;
        Ok(table)
    }

    /// Flag a changed region of the sheet, e.g. `drawing.add_revision_tag("Change", [150., 100.], "B")?`.
    pub fn add_revision_tag(
        &mut self,
        requested_name: &str,
        [x_mm, y_mm]: [f64; 2],
        revision: &str,
    ) -> Result<DrawingAnnotationDefinition, DrawingError> {
        let sheet = self.active_sheet();
        let name = unique_name(requested_name, &self.sheets()[sheet].annotations);
        let tag = DrawingAnnotationDefinition::RevisionTag {
            name,
            x_mm,
            y_mm,
            revision: revision.into(),
            characteristic: None,
            material_removal: None,
        };
        self.add_annotation_definition(sheet, tag.clone())?;
        Ok(tag)
    }
}

impl DrawingAnnotationDefinition {
    /// Lay out the marker around the authored centre, e.g. `annotation.revision_tag_presentation()`.
    pub fn revision_tag_presentation(&self) -> Option<DrawingRevisionTagPresentation<'_>> {
        let Self::RevisionTag {
            x_mm: x,
            y_mm: y,
            revision,
            ..
        } = self
        else {
            return None;
        };
        let tip = [*x, y + 6.];
        let left = [x - 6., y - 6.];
        let right = [x + 6., y - 6.];
        Some(DrawingRevisionTagPresentation {
            text: revision,
            text_center_mm: [*x, y - 2.],
            segments_mm: [[tip, left], [left, right], [right, tip]],
        })
    }
}

pub(super) fn revision_table(
    origin: [f64; 2],
    rows: &[DrawingRevisionRow],
) -> DrawingTablePresentation {
    let headings = ["REV".into(), "DATE".into(), "DESCRIPTION".into()];
    let cells: Vec<_> = rows
        .iter()
        .map(|row| {
            [
                row.revision.clone(),
                row.date.clone(),
                row.description.clone(),
            ]
        })
        .collect();
    grid_table(
        origin,
        &[14., 28., 60.],
        &headings,
        cells.iter().map(|row| row.as_slice()),
    )
}

pub(super) fn validate_rows(rows: &[DrawingRevisionRow]) -> Result<(), DrawingError> {
    if rows.is_empty() {
        return Err(DrawingError(
            "revision table rows []: expected at least one revision row".into(),
        ));
    }
    for row in rows {
        for text in [&row.revision, &row.date, &row.description] {
            if text.contains('\0') {
                return Err(DrawingError(format!(
                    "revision table cell {text:?}: expected text without NUL"
                )));
            }
        }
    }
    Ok(())
}
