// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, RecipeReservations, SketchPrimitiveCreationError};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{
        SketchEntityId, SketchError, SketchText2D, SketchTextFieldProjection,
        bind_sketch_text_parameters, bind_text_parameter_changes,
        resolve_document_sketch_text_editor,
    },
};
use oblikovati_api::types::SketchTextStyle;
use std::borrow::Cow;

impl DocumentSnapshot {
    /// Read linked values and their editor ranges, for example `10.00` and `FILENAME`.
    /// The canvas resolves its file-name field to the actual file name.
    pub fn project_sketch_text_editor(
        &self,
        style: &SketchTextStyle,
        existing: Option<&SketchText2D>,
    ) -> Result<SketchTextFieldProjection, SketchError> {
        resolve_document_sketch_text_editor(style, existing, &self.parameters)
    }

    /// Resolve a text draft before sampling outlines, for example with `"Part6.opd"`.
    /// An edited draft can reuse the existing parameter identities.
    pub fn resolve_sketch_text_candidate<'a>(
        &self,
        style: &'a SketchTextStyle,
        existing: Option<&SketchText2D>,
        file_name: &str,
    ) -> Result<Cow<'a, SketchTextStyle>, SketchError> {
        let bindings = bind_text_parameter_changes(style, existing, &self.parameters)?;
        crate::sketch::resolve_document_sketch_text(style, &bindings, &self.parameters, file_name)
    }

    /// Add text and its defining point in one candidate, for example `snapshot.create_sketch_text(sketch, point, style, ids)?`.
    pub fn create_sketch_text(
        &mut self,
        sketch: DocumentSketchId,
        position: [f64; 2],
        style: SketchTextStyle,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchEntityId, SketchPrimitiveCreationError> {
        use SketchPrimitiveCreationError::Sketch;
        let parameter_bindings =
            bind_sketch_text_parameters(&style, &self.parameters).map_err(Sketch)?;
        let mut geometry = self
            .sketches
            .get(sketch)
            .map_err(Sketch)?
            .geometry()
            .clone();
        let id = ids
            .allocate_sketch_entity()
            .map_err(SketchPrimitiveCreationError::Identity)?;
        let anchor = ids
            .allocate_sketch_entity()
            .map_err(SketchPrimitiveCreationError::Identity)?;
        geometry
            .add_defining_point(anchor, position)
            .map_err(Sketch)?;
        geometry
            .add_text(SketchText2D {
                id,
                anchor,
                style,
                parameter_bindings,
            })
            .map_err(Sketch)?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(Sketch)?;
        Ok(id)
    }

    /// Edit a text entity while preserving its anchor, for example `snapshot.edit_sketch_text(sketch, id, style)?`.
    pub fn edit_sketch_text(
        &mut self,
        sketch: DocumentSketchId,
        id: SketchEntityId,
        style: SketchTextStyle,
    ) -> Result<(), SketchError> {
        let mut geometry = self.sketches.get(sketch)?.geometry().clone();
        let parameter_bindings =
            bind_text_parameter_changes(&style, Some(geometry.text(id)?), &self.parameters)?;
        geometry.set_bound_text_style(id, style, parameter_bindings)?;
        self.sketches.replace_geometry(sketch, geometry)
    }
}
