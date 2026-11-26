// SPDX-License-Identifier: GPL-2.0-only
use super::{
    DrawingAnnotationDefinition, DrawingDocument, DrawingError, annotation::unique_name, spelling,
};
use oblikovati_api::types::{
    MATERIAL_REMOVAL_PROHIBITED, MATERIAL_REMOVAL_REQUIRED, MaterialRemoval,
};
use std::f64::consts::TAU;

/// The roughness label and symbol outline in sheet millimetres.
pub struct DrawingSurfaceTexturePresentation<'a> {
    pub roughness: &'a str,
    pub text_center_mm: [f64; 2],
    pub segments_mm: Vec<[[f64; 2]; 2]>,
}

/// Read canonical material-removal text. Example: `parse_material_removal("required")?`.
pub fn parse_material_removal(value: &str) -> Result<MaterialRemoval, DrawingError> {
    spelling::material_removal::parse(value).ok_or_else(|| {
        DrawingError(format!(
            "material removal {value:?}: expected any, required, or prohibited"
        ))
    })
}

impl DrawingDocument {
    /// Place a finish symbol at its contact point, e.g. `drawing.add_surface_texture("Finish", [40., 50.], "Ra 1.6", removal)?`.
    pub fn add_surface_texture(
        &mut self,
        requested_name: &str,
        [x_mm, y_mm]: [f64; 2],
        roughness: &str,
        removal: MaterialRemoval,
    ) -> Result<DrawingAnnotationDefinition, DrawingError> {
        let sheet = self.active_sheet();
        let name = unique_name(requested_name, &self.sheets()[sheet].annotations);
        let symbol = DrawingAnnotationDefinition::SurfaceTexture {
            name,
            x_mm,
            y_mm,
            roughness: roughness.into(),
            characteristic: None,
            removal,
        };
        self.add_annotation_definition(sheet, symbol.clone())?;
        Ok(symbol)
    }
}

impl DrawingAnnotationDefinition {
    /// Compute the finish symbol, e.g. `annotation.surface_texture_presentation()`.
    pub fn surface_texture_presentation(&self) -> Option<DrawingSurfaceTexturePresentation<'_>> {
        let Self::SurfaceTexture {
            x_mm,
            y_mm,
            roughness,
            removal,
            ..
        } = self
        else {
            return None;
        };
        let [x, y] = [*x_mm, *y_mm];
        let left = [x - 3., y + 4.];
        let right = [x + 5., y + 8.8];
        let end = [right[0] + 12., right[1]];
        let mut segments = vec![[[x, y], left], [[x, y], right], [right, end]];
        match *removal {
            MATERIAL_REMOVAL_REQUIRED => segments.push([left, right]),
            MATERIAL_REMOVAL_PROHIBITED => segments.extend(vertex_circle([x, y])),
            _ => {}
        }
        Some(DrawingSurfaceTexturePresentation {
            roughness,
            text_center_mm: [right[0] + 6., right[1] + 2.4],
            segments_mm: segments,
        })
    }
}

fn vertex_circle([x, y]: [f64; 2]) -> impl Iterator<Item = [[f64; 2]; 2]> {
    const SEGMENTS: u32 = 24;
    const RADIUS_MM: f64 = 1.8;
    let point = move |index: u32| {
        let angle = TAU * f64::from(index % SEGMENTS) / f64::from(SEGMENTS);
        [
            x + RADIUS_MM * angle.cos(),
            y + RADIUS_MM * (1. + angle.sin()),
        ]
    };
    (0..SEGMENTS).map(move |index| [point(index), point(index + 1)])
}

pub(super) fn validate_texture(
    roughness: &str,
    removal: MaterialRemoval,
) -> Result<(), DrawingError> {
    if spelling::material_removal::name(&removal).is_none() {
        return Err(DrawingError(format!(
            "material removal {removal:?}: expected a canonical value"
        )));
    }
    if roughness.contains('\0') {
        return Err(DrawingError(format!(
            "surface roughness {roughness:?}: expected text without NUL"
        )));
    }
    Ok(())
}
