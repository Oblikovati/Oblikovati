// SPDX-License-Identifier: GPL-2.0-only
macro_rules! work_extrusion {
    () => {
        /// Apply an Extrude dialog request through the document command owner.
        /// Example: `work.extrude_from_gui(document, sketch, 0, "10 mm", Default::default(), None)?`.
        fn extrude_from_gui(
            &mut self,
            document: u64,
            sketch: oblikovati_model::document_sketch::DocumentSketchId,
            profile: i64,
            distance: &str,
            _direction: oblikovati_gui::GuiExtrudeDirection,
            _second_distance: Option<&str>,
        ) -> Result<bool, String> {
            Err(format!(
                "extrude document {document}, sketch {sketch:?}, profile {profile}, distance {distance:?}: expected a certified extrusion owner"
            ))
        }

        /// Classify an extrusion profile on the active sketch.
        /// Example: `work.pick_extrude_profile(selection, ray)?` before previewing the draft.
        fn pick_extrude_profile(
            &mut self,
            _: oblikovati_gui::GuiExtrudeProfilePick,
            _: [[f64; 3]; 2],
        ) -> Result<bool, String> {
            Ok(false)
        }

        /// Commit an extrusion draft after its certified preview.
        /// Example: `work.commit_extrusion_draft(&draft, None, false)?`.
        fn commit_extrusion_draft(
            &mut self, draft: &oblikovati_gui::GuiExtrudePreview, _body_name: Option<&str>, _keep_sketch: bool,
        ) -> Result<bool, String> {
            Err(format!("extrusion {:?}: expected a certified draft owner", draft.reference_extent))
        }
    };
}
pub(crate) use work_extrusion;
