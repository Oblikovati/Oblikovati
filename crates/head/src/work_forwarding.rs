// SPDX-License-Identifier: GPL-2.0-only
// Keep authoring state on the same document owner when scripts or add-ins wrap frame work.
macro_rules! forward_owner_view_work {
    ($owner:ident) => {
        fn set_orientation_lock_from_gui(&mut self, locked: bool) -> Result<bool, String> {
            self.$owner.set_orientation_lock_from_gui(locked)
        }
        fn orientation_selection_pivot(&mut self) -> Result<Option<[f64; 3]>, String> {
            self.$owner.orientation_selection_pivot()
        }
        fn look_at_selection_plane(
            &self,
        ) -> Result<Option<oblikovati_model::sketch::SketchPlane>, String> {
            self.$owner.look_at_selection_plane()
        }
        fn set_selection_mode_from_gui(
            &mut self,
            mode: oblikovati_gui::GuiSelectionMode,
        ) -> Result<bool, String> {
            self.$owner.set_selection_mode_from_gui(mode)
        }
        fn set_selection_enabled_from_gui(
            &mut self,
            kind: oblikovati_gui::GuiSelectionKind,
            enabled: bool,
        ) -> Result<bool, String> {
            self.$owner.set_selection_enabled_from_gui(kind, enabled)
        }
        fn move_selection_kind_from_gui(&mut self, from: usize, to: usize) -> Result<bool, String> {
            self.$owner.move_selection_kind_from_gui(from, to)
        }
        fn reset_selection_filter_from_gui(&mut self) -> Result<bool, String> {
            self.$owner.reset_selection_filter_from_gui()
        }
        fn jump_history_from_gui(&mut self, document: u64, position: i64) -> Result<bool, String> {
            self.$owner.jump_history_from_gui(document, position)
        }
        fn update_part_from_gui(&mut self) -> Result<bool, String> {
            self.$owner.update_part_from_gui()
        }
        fn set_document_units_from_gui(
            &mut self,
            document: u64,
            request: &oblikovati_api::wire::SetDocumentUnitsArgs,
        ) -> Result<bool, String> {
            self.$owner.set_document_units_from_gui(document, request)
        }
        fn base_view_preview_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiBaseViewRequest>,
        ) -> Result<bool, String> {
            self.$owner.base_view_preview_from_gui(request)
        }
        fn base_view_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiBaseViewRequest,
        ) -> Result<bool, String> {
            self.$owner.base_view_from_gui(request)
        }
        fn place_component_from_gui(
            &mut self,
            assembly: u64,
            request: &oblikovati_api::wire::PlaceOccurrenceArgs,
        ) -> Result<bool, String> {
            self.$owner.place_component_from_gui(assembly, request)
        }
        fn place_component_file_from_gui(
            &mut self,
            assembly: u64,
            path: &str,
            name: &str,
        ) -> Result<bool, String> {
            self.$owner
                .place_component_file_from_gui(assembly, path, name)
        }
        fn preview_component_from_gui(
            &mut self,
            request: Option<(u64, u64)>,
        ) -> Result<bool, String> {
            self.$owner.preview_component_from_gui(request)
        }
        fn preview_component_file_from_gui(
            &mut self,
            assembly: u64,
            path: &str,
        ) -> Result<bool, String> {
            self.$owner.preview_component_file_from_gui(assembly, path)
        }
        fn create_offset_plane_from_gui(
            &mut self,
            document: u64,
            base: oblikovati_gui::GuiOffsetPlaneBase,
            offset: &str,
        ) -> Result<bool, String> {
            self.$owner
                .create_offset_plane_from_gui(document, base, offset)
        }
        fn validate_offset_plane_from_gui(
            &mut self,
            document: u64,
            offset: &str,
        ) -> Result<bool, String> {
            self.$owner.validate_offset_plane_from_gui(document, offset)
        }
        fn preview_fillet_edit(
            &mut self,
            request: &oblikovati_gui::GuiFilletEditRequest,
        ) -> Result<bool, String> {
            self.$owner.preview_fillet_edit(request)
        }
        fn commit_fillet_edit(
            &mut self,
            request: &oblikovati_gui::GuiFilletEditRequest,
        ) -> Result<bool, String> {
            self.$owner.commit_fillet_edit(request)
        }
        fn preview_chamfer_edit(
            &mut self,
            request: &oblikovati_gui::GuiFilletEditRequest,
        ) -> Result<bool, String> {
            self.$owner.preview_chamfer_edit(request)
        }
        fn commit_chamfer_edit(
            &mut self,
            request: &oblikovati_gui::GuiFilletEditRequest,
        ) -> Result<bool, String> {
            self.$owner.commit_chamfer_edit(request)
        }
        fn preview_move_body_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiMoveBodyRequest>,
        ) -> Result<bool, String> {
            self.$owner.preview_move_body_from_gui(request)
        }
        fn commit_move_body_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiMoveBodyRequest,
        ) -> Result<bool, String> {
            self.$owner.commit_move_body_from_gui(request)
        }
        fn preview_shell_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiShellFeatureRequest>,
        ) -> Result<bool, String> {
            self.$owner.preview_shell_from_gui(request)
        }
        fn commit_shell_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiShellFeatureRequest,
        ) -> Result<bool, String> {
            self.$owner.commit_shell_from_gui(request)
        }
        fn preview_hole_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiHoleRequest>,
        ) -> Result<bool, String> {
            self.$owner.preview_hole_from_gui(request)
        }
        fn commit_hole_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiHoleRequest,
        ) -> Result<bool, String> {
            self.$owner.commit_hole_from_gui(request)
        }
        fn preview_fillet_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiFilletRequest>,
        ) -> Result<bool, String> {
            self.$owner.preview_fillet_from_gui(request)
        }
        fn commit_fillet_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiFilletRequest,
        ) -> Result<bool, String> {
            self.$owner.commit_fillet_from_gui(request)
        }
        fn preview_chamfer_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiFilletRequest>,
        ) -> Result<bool, String> {
            self.$owner.preview_chamfer_from_gui(request)
        }
        fn commit_chamfer_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiFilletRequest,
        ) -> Result<bool, String> {
            self.$owner.commit_chamfer_from_gui(request)
        }
        fn extrude_from_gui(
            &mut self,
            document: u64,
            sketch: oblikovati_model::document_sketch::DocumentSketchId,
            profile: i64,
            distance: &str,
        ) -> Result<bool, String> {
            self.$owner
                .extrude_from_gui(document, sketch, profile, distance)
        }
        fn extrusion_preview_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiExtrudePreview>,
        ) -> Result<bool, String> {
            self.$owner.extrusion_preview_from_gui(request)
        }
        fn validate_extrusion_distance(
            &mut self,
            document: u64,
            distance: &str,
        ) -> Result<bool, String> {
            self.$owner.validate_extrusion_distance(document, distance)
        }
        fn preview_extrusion_edit(
            &mut self,
            request: &oblikovati_gui::GuiExtrudeEditRequest,
        ) -> Result<bool, String> {
            self.$owner.preview_extrusion_edit(request)
        }
        fn commit_extrusion_edit(
            &mut self,
            request: &oblikovati_gui::GuiExtrudeEditRequest,
        ) -> Result<bool, String> {
            self.$owner.commit_extrusion_edit(request)
        }
        fn activate_document_from_gui(
            &mut self,
            request: &oblikovati_api::wire::ActivateDocumentArgs,
        ) -> Result<bool, String> {
            self.$owner.activate_document_from_gui(request)
        }

        fn close_document_from_gui(
            &mut self,
            request: &oblikovati_api::wire::CloseDocumentArgs,
        ) -> Result<bool, String> {
            self.$owner.close_document_from_gui(request)
        }
        fn open_document_from_gui(
            &mut self,
            request: &oblikovati_api::wire::OpenDocumentArgs,
        ) -> Result<bool, String> {
            self.$owner.open_document_from_gui(request)
        }
        fn load_full_from_gui(&mut self, id: u64) -> Result<bool, String> {
            self.$owner.load_full_from_gui(id)
        }
        fn open_deferred_drawing_from_gui(&mut self, path: &str) -> Result<bool, String> {
            self.$owner.open_deferred_drawing_from_gui(path)
        }
        // #3811: live browser actions must reach the same owner through every wrapper.
        fn select_browser_block_from_gui(
            &mut self,
            document: u64,
            sketch: oblikovati_model::document_sketch::DocumentSketchId,
            instance: oblikovati_model::sketch::SketchEntityId,
            toggle: bool,
        ) -> Result<bool, String> {
            self.$owner
                .select_browser_block_from_gui(document, sketch, instance, toggle)
        }
        fn select_browser_feature_from_gui(
            &mut self,
            document: u64,
            feature: oblikovati_model::feature::FeatureId,
        ) -> Result<bool, String> {
            self.$owner
                .select_browser_feature_from_gui(document, feature)
        }
        fn update_body_properties_from_gui(
            &mut self,
            document: u64,
            body: i64,
        ) -> Result<bool, String> {
            self.$owner.update_body_properties_from_gui(document, body)
        }
        fn assign_part_asset_from_gui(
            &mut self,
            document: u64,
            kind: oblikovati_gui::GuiPartAssetKind,
            id: &str,
        ) -> Result<bool, String> {
            self.$owner.assign_part_asset_from_gui(document, kind, id)
        }
        fn set_body_appearance_from_gui(
            &mut self,
            document: u64,
            key: &str,
            appearance: &str,
        ) -> Result<bool, String> {
            self.$owner
                .set_body_appearance_from_gui(document, key, appearance)
        }
        fn edit_feature_from_gui(
            &mut self,
            document: u64,
            edit: &oblikovati_gui::GuiFeatureEdit,
        ) -> Result<bool, String> {
            self.$owner.edit_feature_from_gui(document, edit)
        }
        fn browser_choice_from_gui(
            &mut self,
            document: u64,
            choice: &oblikovati_gui::GuiBrowserChoice,
        ) -> Result<bool, String> {
            self.$owner.browser_choice_from_gui(document, choice)
        }
        fn keep_document_picture_from_gui(
            &mut self,
            document: u64,
            picture: Option<&oblikovati_gui::GuiPreviewImage>,
        ) -> Result<bool, String> {
            self.$owner
                .keep_document_picture_from_gui(document, picture)
        }
        fn inspect_file_from_gui(&mut self, path: &str) -> Result<bool, String> {
            self.$owner.inspect_file_from_gui(path)
        }
        fn navigation_prompt_from_gui(&mut self, prompt: Option<&str>) -> Result<bool, String> {
            self.$owner.navigation_prompt_from_gui(prompt)
        }
        fn face_edges_on_line(
            &mut self,
            line: [[f64; 3]; 2],
        ) -> Result<Vec<[[f64; 3]; 2]>, String> {
            self.$owner.face_edges_on_line(line)
        }
        fn click_model_from_gui(
            &mut self,
            document: u64,
            line: [[f64; 3]; 2],
        ) -> Result<bool, String> {
            self.$owner.click_model_from_gui(document, line)
        }
        fn click_model_with_window(
            &mut self,
            document: u64,
            line: [[f64; 3]; 2],
            window: f64,
        ) -> Result<bool, String> {
            self.$owner.click_model_with_window(document, line, window)
        }
        fn click_model_with_modifiers(
            &mut self,
            document: u64,
            line: [[f64; 3]; 2],
            window: Option<f64>,
            toggle: bool,
        ) -> Result<bool, String> {
            self.$owner
                .click_model_with_modifiers(document, line, window, toggle)
        }
        fn context_click_model(
            &mut self,
            document: u64,
            line: [[f64; 3]; 2],
            window: Option<f64>,
        ) -> Result<bool, String> {
            self.$owner.context_click_model(document, line, window)
        }
        fn selected_browser_outline(&mut self) -> Vec<[[f64; 3]; 2]> {
            self.$owner.selected_browser_outline()
        }
        fn selected_edge_outlines(&mut self) -> Vec<Vec<[[f64; 3]; 2]>> {
            self.$owner.selected_edge_outlines()
        }
        fn observe_window_capture_extent(&mut self, extent: Option<[u32; 2]>) {
            self.$owner.observe_window_capture_extent(extent);
        }
        fn take_window_capture_requests(&mut self) -> Vec<$crate::WindowCaptureJob> {
            self.$owner.take_window_capture_requests()
        }
        // #3796: an add-in layer must retain the same UI authoring operations as the script layer.
        fn create_part_document(&mut self) -> Result<bool, String> {
            self.$owner.create_part_document()
        }
        fn create_document_template(
            &mut self,
            template: oblikovati_gui::GuiDocumentTemplate,
        ) -> Result<bool, String> {
            self.$owner.create_document_template(template)
        }
        fn select_block_geometry_from_gui(&mut self, active: bool) -> Result<bool, String> {
            self.$owner.select_block_geometry_from_gui(active)
        }
        fn datum_planes(&mut self) -> Result<$crate::HeadDatumPlaneUpdate, String> {
            self.$owner.datum_planes()
        }
        fn take_scene(&mut self) -> Result<Option<$crate::HeadSceneUpdate>, String> {
            self.$owner.take_scene()
        }
    };
}
pub(crate) use forward_owner_view_work;
