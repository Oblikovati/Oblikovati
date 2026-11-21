// SPDX-License-Identifier: GPL-2.0-only
use super::{DrawingError, DrawingSheetRecipe};
use oblikovati_api::types::*;
const STANDARD_MM: &[(SheetSize, [f64; 2])] = &[
    (SHEET_SIZE_A0, [841., 1189.]),
    (SHEET_SIZE_A1, [594., 841.]),
    (SHEET_SIZE_A2, [420., 594.]),
    (SHEET_SIZE_A3, [297., 420.]),
    (SHEET_SIZE_A4, [210., 297.]),
    (SHEET_SIZE_ANSI_A, [215.9, 279.4]),
    (SHEET_SIZE_ANSI_B, [279.4, 431.8]),
    (SHEET_SIZE_ANSI_C, [431.8, 558.8]),
    (SHEET_SIZE_ANSI_D, [558.8, 863.6]),
    (SHEET_SIZE_ANSI_E, [863.6, 1117.6]),
];
impl DrawingSheetRecipe {
    /// The Go model/drawing/content.go default: one bordered A3 landscape sheet.
    /// Example: `DrawingSheetRecipe::a3_landscape("Sheet:1")`.
    pub fn a3_landscape(name: &str) -> Self {
        Self {
            name: name.into(),
            size: SHEET_SIZE_A3,
            orientation: SHEET_LANDSCAPE,
            width_mm: 0.,
            height_mm: 0.,
            border: true,
            title_block: "Default".into(),
            title_block_at: BOTTOM_RIGHT_TITLE_BLOCK,
            revision: String::new(),
            views: Vec::new(),
            annotations: Vec::new(),
        }
    }
    /// Return laid-out millimeter dimensions; custom dimensions are already orientation-applied.
    /// Example: `sheet.dimensions_mm()? == [420.,297.]` for A3 landscape.
    pub fn dimensions_mm(&self) -> Result<[f64; 2], DrawingError> {
        self.require_sheet_values()?;
        if self.size == SHEET_SIZE_CUSTOM {
            return self.custom_dimensions();
        }
        let mut size = self.standard_dimensions()?;
        if self.orientation == SHEET_LANDSCAPE {
            size.swap(0, 1);
        }
        Ok(size)
    }
    fn require_sheet_values(&self) -> Result<(), DrawingError> {
        if ![SHEET_PORTRAIT, SHEET_LANDSCAPE].contains(&self.orientation) {
            return Err(DrawingError(format!(
                "sheet {:?} orientation {:?}: expected portrait or landscape",
                self.name, self.orientation
            )));
        }
        let dimensions = [self.width_mm, self.height_mm];
        if dimensions
            .iter()
            .any(|side| !side.is_finite() || *side < 0.)
        {
            return Err(DrawingError(format!(
                "sheet {:?} stored dimensions {dimensions:?}: expected nonnegative finite millimeters",
                self.name
            )));
        }
        Ok(())
    }
    fn standard_dimensions(&self) -> Result<[f64; 2], DrawingError> {
        let size = STANDARD_MM
            .iter()
            .find(|(kind, _)| *kind == self.size)
            .map(|(_, size)| *size)
            .ok_or_else(|| {
                DrawingError(format!(
                    "sheet {:?} size {:?}: expected a canonical standard or custom sheet",
                    self.name, self.size
                ))
            })?;
        Ok(size)
    }
    fn custom_dimensions(&self) -> Result<[f64; 2], DrawingError> {
        let dimensions = [self.width_mm, self.height_mm];
        if dimensions
            .iter()
            .any(|side| !side.is_finite() || *side <= 0.)
        {
            return Err(DrawingError(format!(
                "sheet {:?} custom dimensions {dimensions:?}: expected positive finite millimeters",
                self.name
            )));
        }
        Ok(dimensions)
    }
}
