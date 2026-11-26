// SPDX-License-Identifier: GPL-2.0-only
use super::spelling::{sheet_orientation, sheet_size, title_block_location};
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
            border_h_zones: 0,
            border_v_zones: 0,
            border_h_label: ALPHABETICAL_BORDER_LABEL,
            border_v_label: ALPHABETICAL_BORDER_LABEL,
            title_block: "Default".into(),
            title_block_at: BOTTOM_RIGHT_TITLE_BLOCK,
            revision: String::new(),
            views: Vec::new(),
            annotations: Vec::new(),
        }
    }
    /// Prepare a bordered paper format from canonical names.
    /// Example: `DrawingSheetRecipe::with_paper("Sheet:2", "a4", "portrait", [0.; 2])?`.
    pub fn with_paper(
        name: &str,
        size: &str,
        orientation: &str,
        dimensions: [f64; 2],
    ) -> Result<Self, DrawingError> {
        let mut sheet = Self::a3_landscape(name);
        sheet.size =
            sheet_size::parse(if size.is_empty() { "custom" } else { size }).ok_or_else(|| {
                DrawingError(format!(
                    "sheet size {size:?}: expected a standard size or custom"
                ))
            })?;
        sheet.orientation = sheet_orientation::parse(if orientation.is_empty() {
            "portrait"
        } else {
            orientation
        })
        .ok_or_else(|| {
            DrawingError(format!(
                "sheet orientation {orientation:?}: expected portrait or landscape"
            ))
        })?;
        if sheet.size == SHEET_SIZE_CUSTOM {
            [sheet.width_mm, sheet.height_mm] = dimensions;
        }
        sheet.dimensions_mm()?;
        Ok(sheet)
    }
    /// Report authored format names, with an empty corner for a sheet without a title block.
    /// Example: `DrawingSheetRecipe::a3_landscape("Sheet:1").paper_spelling()?`.
    pub fn paper_spelling(&self) -> Result<[&'static str; 3], DrawingError> {
        let values = [
            sheet_size::name(&self.size),
            sheet_orientation::name(&self.orientation),
            if self.title_block.is_empty() {
                Some("")
            } else {
                title_block_location::name(&self.title_block_at)
            },
        ];
        match values {
            [Some(size), Some(orientation), Some(corner)] => Ok([size, orientation, corner]),
            _ => Err(DrawingError(format!(
                "sheet {:?} format {:?}/{:?}/{:?}: expected canonical paper format names",
                self.name, self.size, self.orientation, self.title_block_at
            ))),
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
