// SPDX-License-Identifier: GPL-2.0-only
//! Model-state table work on the desktop owner (#4119).
macro_rules! model_state_table_work_methods {
    () => {
        fn verify_model_state_table_from_gui(
            &mut self,
            document: u64,
            request: &oblikovati_api::wire::SetModelStateTableArgs,
        ) -> Result<bool, String> {
            self.require_browser_document(document)?;
            let result = oblikovati_api::client::Client::new(
                oblikovati_automation::FileQueryHost::new(&self.files),
            )
            .model_states()
            .verify_table(request)
            .map_err(|cause| cause.to_string())?;
            self.model_state_verification = Some((document, request.clone(), result));
            Ok(true)
        }
        fn set_model_state_table_from_gui(
            &mut self,
            document: u64,
            request_id: u64,
            request: &oblikovati_api::wire::SetModelStateTableArgs,
        ) -> Result<bool, String> {
            let result = self.require_browser_document(document).and_then(|()| {
                oblikovati_api::client::Client::new(self.document_host())
                    .model_states()
                    .set_table(request)
                    .map(|_| ())
                    .map_err(|cause| cause.to_string())
            });
            self.model_state_table_ack = Some((document, request_id, result));
            Ok(true)
        }
        fn set_document_property_from_gui(
            &mut self,
            request: &oblikovati_api::wire::SetPropertyArgs,
        ) -> Result<bool, String> {
            self.set_typed_document_property(request)
        }
    };
}
