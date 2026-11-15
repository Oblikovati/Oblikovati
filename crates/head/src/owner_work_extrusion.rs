// SPDX-License-Identifier: GPL-2.0-only
//! Extrusion methods on the desktop work boundary.
macro_rules! desktop_extrusion_work {
    () => {
        fn extrude_from_gui(
            &mut self,
            document: u64,
            sketch: oblikovati_model::document_sketch::DocumentSketchId,
            profile: i64,
            distance: &str,
        ) -> Result<bool, String> {
            self.create_extrusion_from_gui(document, sketch, profile, distance)
        }
        fn extrusion_preview_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiExtrudePreview>,
        ) -> Result<bool, String> {
            let Some(preview) = request else {
                return Ok(self.clear_extrusion_preview());
            };
            if let Some(generation) = preview.generation {
                let current = self
                    .files
                    .workspace()
                    .session(preview.document)
                    .map_err(|cause| cause.to_string())?
                    .generation();
                if current != generation {
                    return Ok(false);
                }
            }
            self.set_extrusion_preview(
                preview.document,
                preview.sketch,
                preview.profile,
                &preview.distance,
            )
        }
        fn validate_extrusion_distance(
            &mut self,
            document: u64,
            distance: &str,
        ) -> Result<bool, String> {
            self.validate_extrusion_distance_from_gui(document, distance)
        }
        fn preview_extrusion_edit(
            &mut self,
            request: &oblikovati_gui::GuiExtrudeEditRequest,
        ) -> Result<bool, String> {
            if !self.feature_edit_is_current(request)? {
                return Ok(false);
            }
            self.preview_edited_extrusion(request.document, request.feature.0, &request.distance)
        }
        fn commit_extrusion_edit(
            &mut self,
            request: &oblikovati_gui::GuiExtrudeEditRequest,
        ) -> Result<bool, String> {
            self.commit_edited_extrusion(request)
        }
    };
}
