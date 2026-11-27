// SPDX-License-Identifier: GPL-2.0-only
use super::{DrawingAnnotationDefinition, DrawingDocument, DrawingError, annotation::unique_name};

const BOX_MM: f64 = 8.;
const STEM_MM: f64 = 4.;
const TRIANGLE_MM: f64 = 5.;

/// The letter, frame, stem and solid triangle in sheet millimetres.
pub struct DrawingDatumFeaturePresentation<'a> {
    pub letter: &'a str,
    pub text_center_mm: [f64; 2],
    pub frame_and_stem_mm: [[[f64; 2]; 2]; 5],
    pub triangle_mm: [[f64; 2]; 3],
}

impl DrawingDocument {
    /// Place a datum symbol at its top-left mark, e.g. `drawing.add_datum_feature("Reference", [30., 80.], "A")?`.
    pub fn add_datum_feature(
        &mut self,
        requested_name: &str,
        [x_mm, y_mm]: [f64; 2],
        letter: &str,
    ) -> Result<DrawingAnnotationDefinition, DrawingError> {
        let sheet = self.active_sheet();
        let name = unique_name(requested_name, &self.sheets()[sheet].annotations);
        let symbol = DrawingAnnotationDefinition::DatumFeature {
            name,
            x_mm,
            y_mm,
            letter: letter.into(),
            characteristic: None,
            material_removal: None,
        };
        self.add_annotation_definition(sheet, symbol.clone())?;
        Ok(symbol)
    }
}

impl DrawingAnnotationDefinition {
    /// Compute the sheet symbol, e.g. `annotation.datum_feature_presentation()`.
    pub fn datum_feature_presentation(&self) -> Option<DrawingDatumFeaturePresentation<'_>> {
        let Self::DatumFeature {
            x_mm: x,
            y_mm: y,
            letter,
            ..
        } = self
        else {
            return None;
        };
        let [x, y] = [*x, *y];
        let center_x = x + BOX_MM / 2.;
        let bottom = y - BOX_MM;
        let stem_end = bottom - STEM_MM;
        let right = x + BOX_MM;
        let base = stem_end - TRIANGLE_MM;
        Some(DrawingDatumFeaturePresentation {
            letter,
            text_center_mm: [center_x, y - BOX_MM / 2.],
            frame_and_stem_mm: [
                [[x, y], [right, y]],
                [[right, y], [right, bottom]],
                [[right, bottom], [x, bottom]],
                [[x, bottom], [x, y]],
                [[center_x, bottom], [center_x, stem_end]],
            ],
            triangle_mm: [
                [center_x, stem_end],
                [center_x + TRIANGLE_MM / 2., base],
                [center_x - TRIANGLE_MM / 2., base],
            ],
        })
    }
}

impl DrawingDatumFeaturePresentation<'_> {
    /// Return the legacy curve representation count, e.g. `symbol.curve_count()`.
    pub fn curve_count(&self) -> usize {
        // The wire count includes the triangle outline and its two interior fill strokes.
        self.frame_and_stem_mm.len() + self.triangle_mm.len() + 2
    }
}
