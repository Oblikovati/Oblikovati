// SPDX-License-Identifier: GPL-2.0-only

use super::*;

/// Link edits require source resolution and coordinated ownership of both documents.
pub const DERIVED_COMMAND_METHODS: [&str; 2] = [
    METHOD_PARAMETERS_DERIVED_TABLES_ADD,
    METHOD_PARAMETERS_DERIVED_TABLES_SET_LINKED,
];

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn derived_command(
        &mut self,
        method: &str,
        request: &[u8],
    ) -> Result<Vec<u8>, Error> {
        let target = self
            .files
            .workspace()
            .active_document()
            .ok_or_else(|| Error("derived link edit: expected an active document".into()))?;
        let table = self.apply_links(target, method, request)?;
        command_owner::CommandOwner::refresh_after_command(self.files, target)?;
        let parameters = &self
            .files
            .workspace()
            .session(target)
            .map_err(|error| Error(error.to_string()))?
            .snapshot()
            .parameters;
        let table = parameters
            .derived_table(table)
            .expect("successful link edit");
        encode(&crate::file_queries::derived::describe(
            self.files, parameters, table,
        )?)
    }

    fn apply_links(&mut self, target: u64, method: &str, request: &[u8]) -> Result<i64, Error> {
        if method == METHOD_PARAMETERS_DERIVED_TABLES_ADD {
            let request: DerivedParameterTableAddArgs = decode(method, request)?;
            return self
                .files
                .add_derived_table(
                    target,
                    &request.source_document,
                    request.linked.as_deref().unwrap_or_default(),
                )
                .map_err(|error| Error(error.to_string()));
        }
        let request: DerivedParameterTableSetLinkedArgs = decode(method, request)?;
        self.files
            .set_derived_links(
                target,
                request.id,
                request.linked.as_deref().unwrap_or_default(),
            )
            .map_err(|error| Error(error.to_string()))?;
        Ok(request.id)
    }
}
