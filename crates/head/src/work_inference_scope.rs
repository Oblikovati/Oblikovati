// SPDX-License-Identifier: GPL-2.0-only
//! Desktop scope actions for an active sketch.
macro_rules! inference_scope_work_methods {
    () => {
        /// Change the scope draft selection; example: `work.select_inference_geometry_from_gui(None)?`.
        fn select_inference_geometry_from_gui(
            &mut self,
            entities: Option<&[u64]>,
        ) -> Result<bool, String> {
            Err(format!(
                "inference geometry {entities:?}: expected an active sketch owner"
            ))
        }
        /// Apply a scope through the session host; example: `work.set_inference_scope_from_gui(&scope)?`.
        fn set_inference_scope_from_gui(
            &mut self,
            scope: &oblikovati_api::wire::SketchInferenceScopeView,
        ) -> Result<bool, String> {
            Err(format!(
                "inference scope {scope:?}: expected an active sketch owner"
            ))
        }
    };
}
pub(crate) use inference_scope_work_methods;
