// SPDX-License-Identifier: GPL-2.0-only
use super::{DrawingError, DrawingSheetRecipe, spelling::border_label};
use oblikovati_api::types::{
    ALPHABETICAL_BORDER_LABEL, BorderLabelMode, NO_BORDER_LABEL, NUMERIC_BORDER_LABEL,
};

impl DrawingSheetRecipe {
    pub(super) fn set_zoned_border(
        &mut self,
        [columns, rows]: [i64; 2],
        [column_label, row_label]: [&str; 2],
    ) -> Result<(), DrawingError> {
        if columns <= 0 || rows <= 0 {
            return Err(DrawingError(format!(
                "border zones {columns}×{rows}: expected positive column and row counts"
            )));
        }
        let horizontal = parse_label(column_label)?;
        let vertical = parse_label(row_label)?;
        self.border = true;
        self.border_h_zones = columns;
        self.border_v_zones = rows;
        self.border_h_label = horizontal;
        self.border_v_label = vertical;
        Ok(())
    }

    pub(super) fn validate_border(&self) -> Result<(), DrawingError> {
        let [columns, rows] = [self.border_h_zones, self.border_v_zones];
        let labels = [self.border_h_label, self.border_v_label];
        let plain = columns == 0 && rows == 0 && labels == [ALPHABETICAL_BORDER_LABEL; 2];
        let zoned = self.border
            && columns > 0
            && rows > 0
            && labels
                .iter()
                .all(|label| border_label::name(label).is_some());
        if plain || zoned {
            return Ok(());
        }
        Err(DrawingError(format!(
            "sheet {:?} border zones {columns}×{rows} and labels {labels:?}: expected a plain border or a bordered sheet with positive counts and canonical labels",
            self.name
        )))
    }
}

pub(super) fn parse_label(value: &str) -> Result<BorderLabelMode, DrawingError> {
    border_label::parse(value).ok_or_else(|| {
        DrawingError(format!(
            "border label mode {value:?}: expected alphabetical, numeric, or none"
        ))
    })
}

/// Label a zero-based zone, for example `border_zone_label(26, ALPHABETICAL_BORDER_LABEL)` gives AA.
pub fn border_zone_label(index: u64, mode: BorderLabelMode) -> String {
    match mode {
        NUMERIC_BORDER_LABEL => (u128::from(index) + 1).to_string(),
        ALPHABETICAL_BORDER_LABEL => alphabetical_label(index),
        NO_BORDER_LABEL => String::new(),
        _ => String::new(),
    }
}

fn alphabetical_label(mut index: u64) -> String {
    let mut letters = Vec::new();
    loop {
        letters.push(char::from(b'A' + (index % 26) as u8));
        if index < 26 {
            return letters.into_iter().rev().collect();
        }
        index = index / 26 - 1;
    }
}
