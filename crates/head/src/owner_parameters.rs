// SPDX-License-Identifier: GPL-2.0-only
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
    pub(super) fn gui_parameter_rows(
        &self,
    ) -> Result<Vec<oblikovati_api::wire::ParameterInfo>, String> {
        let query = oblikovati_automation::SessionQueryHost::new(self.files.workspace());
        oblikovati_api::client::Client::new(query)
            .parameters()
            .list()
            .map(|reply| reply.parameters.unwrap_or_default())
            .map_err(|cause| cause.to_string())
    }
}
