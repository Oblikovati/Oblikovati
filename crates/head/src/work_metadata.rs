// SPDX-License-Identifier: GPL-2.0-only
//! Owner services for document metadata edits.
macro_rules! metadata_work_methods {
    () => {
        /// Apply a parameter editor request on the document owner; example: `work.edit_parameter_from_gui(GuiParameterMutation::Add, &request)?`.
        fn edit_parameter_from_gui(
            &mut self,
            mutation: oblikovati_gui::GuiParameterMutation,
            request: &oblikovati_api::wire::ParameterSetArgs,
        ) -> Result<bool, String> {
            Err(format!(
                "GUI parameter {mutation:?} {request:?}: expected a document parameter owner"
            ))
        }
        /// Set metadata on the requested document through its owner.
        /// Example: `work.set_document_property_from_gui(&request)?`.
        fn set_document_property_from_gui(
            &mut self,
            request: &oblikovati_api::wire::SetPropertyArgs,
        ) -> Result<bool, String> {
            Err(format!(
                "property {request:?}: expected a document property owner"
            ))
        }
    };
}
