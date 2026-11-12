// SPDX-License-Identifier: GPL-2.0-only
// Keep authoring state on the same document owner when scripts or add-ins wrap frame work.
macro_rules! forward_owner_view_work {
    ($owner:ident) => {
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
        // #3811: live browser actions must reach the same owner through every wrapper.
        fn select_browser_sketch_from_gui(
            &mut self,
            document: u64,
            sketch: oblikovati_model::document_sketch::DocumentSketchId,
            toggle: bool,
        ) -> Result<bool, String> {
            self.$owner
                .select_browser_sketch_from_gui(document, sketch, toggle)
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
        fn set_sketch_line_style_from_gui(
            &mut self,
            document: u64,
            sketch: oblikovati_model::document_sketch::DocumentSketchId,
            style: &oblikovati_gui::GuiSketchLineStyle,
        ) -> Result<bool, String> {
            self.$owner
                .set_sketch_line_style_from_gui(document, sketch, style)
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
        fn selected_browser_outline(&mut self) -> Vec<[[f64; 3]; 2]> {
            self.$owner.selected_browser_outline()
        }
        fn set_sketch_visibility_from_gui(
            &mut self,
            document: u64,
            sketch: oblikovati_model::document_sketch::DocumentSketchId,
            visible: bool,
        ) -> Result<bool, String> {
            self.$owner
                .set_sketch_visibility_from_gui(document, sketch, visible)
        }
        fn edit_existing_sketch_from_gui(
            &mut self,
            document: u64,
            sketch: oblikovati_model::document_sketch::DocumentSketchId,
        ) -> Result<bool, String> {
            self.$owner.edit_existing_sketch_from_gui(document, sketch)
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
        fn edit_sketch_from_gui(&mut self, start: bool) -> Result<bool, String> {
            self.$owner.edit_sketch_from_gui(start)
        }
        fn toggle_sketch_format_from_gui(
            &mut self,
            format: oblikovati_model::sketch::SketchCurveFormat,
        ) -> Result<bool, String> {
            self.$owner.toggle_sketch_format_from_gui(format)
        }
        fn delete_sketch_selection_from_gui(&mut self) -> Result<bool, String> {
            self.$owner.delete_sketch_selection_from_gui()
        }
        fn select_block_geometry_from_gui(&mut self, active: bool) -> Result<bool, String> {
            self.$owner.select_block_geometry_from_gui(active)
        }
        fn create_sketch_block_from_gui(&mut self, name: &str) -> Result<bool, String> {
            self.$owner.create_sketch_block_from_gui(name)
        }
        fn place_sketch_tool_from_gui(
            &mut self,
            tool: oblikovati_gui::GuiSketchTool,
        ) -> Result<bool, String> {
            self.$owner.place_sketch_tool_from_gui(tool)
        }
        fn sketch_dimension_annotations(&self) -> Result<$crate::HeadDimensionAnnotations, String> {
            self.$owner.sketch_dimension_annotations()
        }
        fn dimension_tool_from_gui(
            &mut self,
            action: &oblikovati_gui::GuiDimensionAction,
        ) -> Result<bool, String> {
            self.$owner.dimension_tool_from_gui(action)
        }
        fn constrain_sketch_from_gui(
            &mut self,
            choice: oblikovati_gui::GuiSketchConstraint,
        ) -> Result<bool, String> {
            self.$owner.constrain_sketch_from_gui(choice)
        }
        fn decline_sketch_edit(&mut self, cause: String) -> Result<(), String> {
            self.$owner.decline_sketch_edit(cause)
        }
        fn sketch_edit(&self) -> Option<$crate::HeadSketchEdit> {
            self.$owner.sketch_edit()
        }
        fn sketch_preview(&self) -> Option<oblikovati_viewport::ViewportSketchPreview> {
            self.$owner.sketch_preview()
        }
        fn take_sketch_scene(&mut self) -> Result<Option<$crate::HeadSketchSceneUpdate>, String> {
            self.$owner.take_sketch_scene()
        }
        fn take_scene(&mut self) -> Result<Option<$crate::HeadSceneUpdate>, String> {
            self.$owner.take_scene()
        }
    };
}
pub(crate) use forward_owner_view_work;
