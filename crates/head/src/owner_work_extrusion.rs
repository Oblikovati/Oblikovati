// SPDX-License-Identifier: GPL-2.0-only
//! Extrusion methods on the desktop work boundary.
macro_rules! desktop_extrusion_work {
    () => {
        fn preview_fillet_edit(
            &mut self,
            request: &oblikovati_gui::GuiFilletEditRequest,
        ) -> Result<bool, String> {
            self.preview_edited_fillet(request)
        }
        fn commit_fillet_edit(
            &mut self,
            request: &oblikovati_gui::GuiFilletEditRequest,
        ) -> Result<bool, String> {
            self.commit_edited_fillet(request)
        }
        fn preview_chamfer_edit(
            &mut self,
            request: &oblikovati_gui::GuiFilletEditRequest,
        ) -> Result<bool, String> {
            self.preview_edited_chamfer(request)
        }
        fn commit_chamfer_edit(
            &mut self,
            request: &oblikovati_gui::GuiFilletEditRequest,
        ) -> Result<bool, String> {
            self.commit_edited_fillet(request)
        }
        fn preview_move_body_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiMoveBodyRequest>,
        ) -> Result<bool, String> {
            let Some(request) = request else {
                return Ok(self.clear_extrusion_preview());
            };
            self.preview_move_body(request)
        }
        fn commit_move_body_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiMoveBodyRequest,
        ) -> Result<bool, String> {
            self.commit_move_body(request)
        }
        fn loft_from_gui(&mut self, action: oblikovati_gui::GuiLoftAction) -> Result<bool, String> {
            self.loft_command(action)
        }
        fn revolve_from_gui(
            &mut self,
            action: oblikovati_gui::GuiRevolveAction,
        ) -> Result<bool, String> {
            self.revolve_command(action)
        }
        fn coil_from_gui(&mut self, action: oblikovati_gui::GuiCoilAction) -> Result<bool, String> {
            self.coil_command(action)
        }
        fn emboss_from_gui(
            &mut self,
            action: oblikovati_gui::GuiEmbossAction,
        ) -> Result<bool, String> {
            self.emboss_command(action)
        }
        fn derive_from_gui(
            &mut self,
            action: oblikovati_gui::GuiDeriveAction,
        ) -> Result<bool, String> {
            self.derive_command(action)
        }
        fn rib_from_gui(&mut self, action: oblikovati_gui::GuiRibAction) -> Result<bool, String> {
            self.rib_command(action)
        }
        fn decal_from_gui(
            &mut self,
            action: oblikovati_gui::GuiDecalAction,
        ) -> Result<bool, String> {
            self.decal_command(action)
        }
        fn import_from_gui(
            &mut self,
            action: oblikovati_gui::GuiImportAction,
        ) -> Result<bool, String> {
            self.import_command(action)
        }
        fn unwrap_from_gui(
            &mut self,
            action: oblikovati_gui::GuiUnwrapAction,
        ) -> Result<bool, String> {
            self.unwrap_command(action)
        }
        fn sweep_from_gui(
            &mut self,
            action: oblikovati_gui::GuiSweepAction,
        ) -> Result<bool, String> {
            self.sweep_command(action)
        }
        fn preview_shell_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiShellFeatureRequest>,
        ) -> Result<bool, String> {
            let Some(request) = request else {
                return Ok(self.clear_extrusion_preview());
            };
            self.preview_selected_shell(request)
        }
        fn commit_shell_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiShellFeatureRequest,
        ) -> Result<bool, String> {
            self.commit_selected_shell(request)
        }
        fn hole_from_gui(&mut self, action: oblikovati_gui::GuiHoleAction) -> Result<bool, String> {
            self.hole_command(action)
        }
        fn preview_fillet_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiFilletRequest>,
        ) -> Result<bool, String> {
            let Some(request) = request else {
                return Ok(self.clear_extrusion_preview());
            };
            self.require_blend_selection(request)?;
            self.preview_selected_fillet(request.document, &request.radius)
        }
        fn commit_fillet_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiFilletRequest,
        ) -> Result<bool, String> {
            self.require_blend_selection(request)?;
            self.commit_selected_fillet(request.document, &request.radius)
        }
        fn preview_chamfer_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiFilletRequest>,
        ) -> Result<bool, String> {
            let Some(request) = request else {
                return Ok(self.clear_extrusion_preview());
            };
            self.require_blend_selection(request)?;
            self.preview_selected_chamfer(request.document, &request.radius)
        }
        fn commit_chamfer_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiFilletRequest,
        ) -> Result<bool, String> {
            self.require_blend_selection(request)?;
            self.commit_selected_chamfer(request.document, &request.radius)
        }
        fn create_offset_plane_from_gui(
            &mut self,
            document: u64,
            base: oblikovati_gui::GuiOffsetPlaneBase,
            offset: &str,
        ) -> Result<bool, String> {
            self.create_offset_plane_from_selection(document, base, offset)
        }
        fn validate_offset_plane_from_gui(
            &mut self,
            document: u64,
            offset: &str,
        ) -> Result<bool, String> {
            self.publish_offset_plane_validation(document, offset)
        }
        fn extrude_from_gui(
            &mut self,
            document: u64,
            sketch: oblikovati_model::document_sketch::DocumentSketchId,
            profile: i64,
            distance: &str,
            direction: oblikovati_gui::GuiExtrudeDirection,
            second_distance: Option<&str>,
        ) -> Result<bool, String> {
            self.create_extrusion_from_gui(
                document,
                sketch,
                profile,
                distance,
                direction,
                second_distance,
            )
        }
        fn commit_extrusion_draft(
            &mut self,
            draft: &oblikovati_gui::GuiExtrudePreview,
            body_name: Option<&str>,
            keep_sketch: bool,
        ) -> Result<bool, String> {
            self.commit_extrusion_draft(draft, body_name, keep_sketch)
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
            self.set_extrusion_preview(preview)
        }
        fn validate_extrusion_distance(
            &mut self,
            document: u64,
            distance: &str,
            second_distance: Option<&str>,
        ) -> Result<bool, String> {
            self.validate_extrusion_distance_from_gui(document, distance, second_distance)
        }
        fn preview_extrusion_edit(
            &mut self,
            request: &oblikovati_gui::GuiExtrudeEditRequest,
        ) -> Result<bool, String> {
            if !self.feature_edit_is_current(request)? {
                return Ok(false);
            }
            self.preview_edited_extrusion(
                request.document,
                request.feature.0,
                &request.distance,
                request.second_distance.as_deref(),
                request.taper.as_deref(),
            )
        }
        fn commit_extrusion_edit(
            &mut self,
            request: &oblikovati_gui::GuiExtrudeEditRequest,
        ) -> Result<bool, String> {
            self.commit_edited_extrusion(request)
        }
    };
}
