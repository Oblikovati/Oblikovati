// SPDX-License-Identifier: GPL-2.0-only
//! Group creation and activation into a single document edit.
use super::*;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn copy_model_state(&mut self, id: u64) -> Result<bool, String> {
        oblikovati_api::client::Client::new(self.document_host())
            .model_states()
            .copy(&oblikovati_api::wire::ModelStateNameArgs {
                id,
                name: String::new(),
            })
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    pub(super) fn rename_model_state(&mut self, id: u64, name: &str) -> Result<bool, String> {
        oblikovati_api::client::Client::new(self.document_host())
            .model_states()
            .rename(&oblikovati_api::wire::ModelStateNameArgs {
                id,
                name: name.into(),
            })
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    pub(super) fn create_active_model_state(&mut self, document: u64) -> Result<bool, String> {
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .begin("Create Model State")
            .map_err(|cause| cause.to_string())?;
        let result = self.create_and_activate_model_state();
        let mut commands = self
            .files
            .commands(document)
            .map_err(|cause| cause.to_string())?;
        if result.is_err() {
            commands.abort().map_err(|cause| cause.to_string())?;
            return result;
        }
        commands.end().map_err(|cause| cause.to_string())?;
        result
    }
    fn create_and_activate_model_state(&mut self) -> Result<bool, String> {
        let client = oblikovati_api::client::Client::new(self.document_host());
        let member = client
            .model_states()
            .create(&Default::default())
            .map_err(|cause| cause.to_string())?
            .model_state;
        client
            .model_states()
            .activate(&oblikovati_api::wire::RepRef { id: member.id })
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
}
