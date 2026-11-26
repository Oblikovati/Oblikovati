// SPDX-License-Identifier: GPL-2.0-only
use super::{DrawingError, DrawingSheetRecipe, border::parse_label, validation::validate_sheet};
use oblikovati_api::types::ALPHABETICAL_BORDER_LABEL;
use serde::{Deserialize, Serialize};

/// A named blank sheet used to create sheets with a repeatable paper and layout.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(transparent)]
pub struct DrawingSheetFormat(DrawingSheetRecipe);

impl DrawingSheetFormat {
    /// Finish a format from paper and border settings.
    /// Example: `DrawingSheetFormat::new(paper, [4, 3], ["numeric", ""], "topLeft")?`.
    pub fn new(
        mut paper: DrawingSheetRecipe,
        counts: [i64; 2],
        labels: [&str; 2],
        corner: &str,
    ) -> Result<Self, DrawingError> {
        paper.set_title_block(corner)?;
        if counts.contains(&0) {
            apply_plain_border(&mut paper, counts, labels)?;
        } else {
            paper.set_zoned_border(counts, labels)?;
        }
        let format = Self(paper);
        format.validate()?;
        Ok(format)
    }

    /// Read the registration name. Example: `format.name()`.
    pub fn name(&self) -> &str {
        &self.0.name
    }

    pub(super) fn stamp(&self, name: &str) -> DrawingSheetRecipe {
        let mut sheet = self.0.clone();
        sheet.name = name.into();
        sheet
    }

    pub(super) fn validate(&self) -> Result<(), DrawingError> {
        let sheet = &self.0;
        validate_sheet(sheet)?;
        if sheet.border
            && sheet.title_block == "Default"
            && sheet.revision.is_empty()
            && sheet.views.is_empty()
            && sheet.annotations.is_empty()
        {
            return Ok(());
        }
        Err(DrawingError(format!(
            "sheet format {:?}: expected a blank sheet with the default border and title block",
            sheet.name
        )))
    }
}

fn apply_plain_border(
    sheet: &mut DrawingSheetRecipe,
    counts: [i64; 2],
    labels: [&str; 2],
) -> Result<(), DrawingError> {
    if counts.iter().any(|count| *count < 0) {
        return Err(DrawingError(format!(
            "format border zones {counts:?}: expected nonnegative counts"
        )));
    }
    for label in labels {
        parse_label(label)?;
    }
    sheet.border = true;
    sheet.border_h_zones = 0;
    sheet.border_v_zones = 0;
    sheet.border_h_label = ALPHABETICAL_BORDER_LABEL;
    sheet.border_v_label = ALPHABETICAL_BORDER_LABEL;
    Ok(())
}
