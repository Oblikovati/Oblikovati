// SPDX-License-Identifier: GPL-2.0-only
// Keep sketch commands on the same owner through service wrappers.
macro_rules! forward_owner_sketch_work {
    ($owner:ident) => {
        fn set_sketch_isolation_from_gui(&mut self, isolated: bool) -> Result<bool, String> {
            self.$owner.set_sketch_isolation_from_gui(isolated)
        }
        fn select_browser_sketch_from_gui(
            &mut self,
            document: u64,
            sketch: oblikovati_model::document_sketch::DocumentSketchId,
            toggle: bool,
        ) -> Result<bool, String> {
            self.$owner
                .select_browser_sketch_from_gui(document, sketch, toggle)
        }
        fn place_sketch_block_from_gui(&mut self, name: &str) -> Result<bool, String> {
            self.$owner.place_sketch_block_from_gui(name)
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
        fn start_sketch_on_face_from_gui(
            &mut self,
            document: u64,
            line: [[f64; 3]; 2],
        ) -> Result<bool, String> {
            self.$owner.start_sketch_on_face_from_gui(document, line)
        }
        fn start_sketch_on_selected_face_from_gui(
            &mut self,
            document: u64,
            key: &str,
        ) -> Result<bool, String> {
            self.$owner
                .start_sketch_on_selected_face_from_gui(document, key)
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
        fn edit_sketch_from_gui(&mut self, start: bool) -> Result<bool, String> {
            self.$owner.edit_sketch_from_gui(start)
        }
        fn toggle_sketch_format_from_gui(
            &mut self,
            format: oblikovati_model::sketch::SketchCurveFormat,
        ) -> Result<bool, String> {
            self.$owner.toggle_sketch_format_from_gui(format)
        }
        fn delete_sketch_relation_from_gui(&mut self, id: u64) -> Result<bool, String> {
            self.$owner.delete_sketch_relation_from_gui(id)
        }
        fn dismiss_sketch_context(&mut self, token: u64) -> Result<bool, String> {
            self.$owner.dismiss_sketch_context(token)
        }
        fn select_sketch_context_candidate(
            &mut self,
            token: u64,
            entity: oblikovati_model::sketch::SketchEntityId,
        ) -> Result<bool, String> {
            self.$owner.select_sketch_context_candidate(token, entity)
        }
        fn preview_sketch_pattern_from_gui(
            &mut self,
            request: Option<oblikovati_model::history::SketchRectangularPatternRequest>,
        ) -> Result<bool, String> {
            self.$owner.preview_sketch_pattern_from_gui(request)
        }
        fn create_sketch_pattern_from_gui(
            &mut self,
            request: oblikovati_model::history::SketchRectangularPatternRequest,
        ) -> Result<bool, String> {
            self.$owner.create_sketch_pattern_from_gui(request)
        }
        fn preview_sketch_pattern_edit_from_gui(
            &mut self,
            pattern: oblikovati_model::sketch::SketchEntityId,
            request: oblikovati_model::history::SketchRectangularPatternRequest,
        ) -> Result<bool, String> {
            self.$owner
                .preview_sketch_pattern_edit_from_gui(pattern, request)
        }
        fn edit_sketch_pattern_from_gui(
            &mut self,
            pattern: oblikovati_model::sketch::SketchEntityId,
            request: oblikovati_model::history::SketchRectangularPatternRequest,
        ) -> Result<bool, String> {
            self.$owner.edit_sketch_pattern_from_gui(pattern, request)
        }
        fn sketch_move_from_gui(
            &mut self,
            action: oblikovati_gui::GuiSketchMoveAction,
        ) -> Result<bool, String> {
            self.$owner.sketch_move_from_gui(action)
        }
        fn delete_sketch_selection_from_gui(&mut self) -> Result<bool, String> {
            self.$owner.delete_sketch_selection_from_gui()
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
    };
}
pub(crate) use forward_owner_sketch_work;
