// SPDX-License-Identifier: GPL-2.0-only
use super::{
    DrawingAnnotationDefinition, DrawingDocument, DrawingError, DrawingTableLine,
    DrawingTablePresentation,
    annotation::unique_name,
    spelling,
    symbols::{characteristic_glyph, characteristic_text},
    table::grid_table,
    validation::require_label,
};
use oblikovati_api::types::GeometricCharacteristic;

/// Parse a canonical characteristic, e.g. `parse_geometric_characteristic("position")?`.
pub fn parse_geometric_characteristic(
    value: &str,
) -> Result<GeometricCharacteristic, DrawingError> {
    spelling::geometric_characteristic::parse(value).ok_or_else(|| {
        DrawingError(format!(
            "geometric characteristic {value:?}: expected a canonical API spelling"
        ))
    })
}

impl DrawingDocument {
    /// Place a control frame, e.g. `drawing.add_feature_control_frame("Hole", [40., 50.], characteristic, "0.2", vec!["A".into()])?`.
    pub fn add_feature_control_frame(
        &mut self,
        requested_name: &str,
        [x_mm, y_mm]: [f64; 2],
        characteristic: GeometricCharacteristic,
        tolerance: &str,
        datums: Vec<String>,
    ) -> Result<DrawingAnnotationDefinition, DrawingError> {
        let sheet = self.active_sheet();
        let name = unique_name(requested_name, &self.sheets()[sheet].annotations);
        let frame = DrawingAnnotationDefinition::FeatureControlFrame {
            name,
            x_mm,
            y_mm,
            characteristic,
            tolerance: tolerance.into(),
            datums,
            material_removal: None,
        };
        self.add_annotation_definition(sheet, frame.clone())?;
        Ok(frame)
    }
}

pub(super) fn frame_presentation(
    [x, y]: [f64; 2],
    characteristic: GeometricCharacteristic,
    tolerance: &str,
    datums: &[String],
) -> DrawingTablePresentation {
    let glyph = characteristic_glyph(characteristic, [x + 4., y - 4.], 2.2);
    let symbol = if glyph.is_empty() {
        characteristic_text(characteristic)
    } else {
        ""
    };
    let mut labels = vec![symbol.into(), tolerance.into()];
    labels.extend_from_slice(datums);
    let mut widths = vec![text_width(symbol).max(8.), text_width(tolerance).max(8.)];
    widths.extend(
        datums
            .iter()
            .map(|datum| 6. + datum.chars().count().saturating_sub(1) as f64 * 2.4),
    );
    let mut frame = grid_table([x, y], &widths, &labels, std::iter::empty());
    frame.lines.extend(
        glyph
            .into_iter()
            .map(|[start, end]| DrawingTableLine { start, end }),
    );
    frame
}

fn text_width(text: &str) -> f64 {
    text.chars().count() as f64 * 2.4 + 4.
}

pub(super) fn validate_frame(
    characteristic: GeometricCharacteristic,
    tolerance: &str,
    datums: &[String],
) -> Result<(), DrawingError> {
    if spelling::geometric_characteristic::name(&characteristic).is_none() {
        return Err(DrawingError(format!(
            "geometric characteristic {characteristic:?}: expected a canonical API value"
        )));
    }
    require_label("geometric tolerance", tolerance)?;
    if let Some(datum) = datums.iter().find(|datum| datum.contains('\0')) {
        return Err(DrawingError(format!(
            "datum reference {datum:?}: expected text without NUL"
        )));
    }
    Ok(())
}
