// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_session::files::FileWorkspaceError;
use std::path::PathBuf;

type SourceBindings = Vec<(u64, Option<PathBuf>)>;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn lifecycle_command(
        &mut self,
        method: &str,
        request: &[u8],
    ) -> Result<Vec<u8>, Error> {
        let before = self.source_bindings()?;
        let response = self.command(method, request);
        if before == self.source_bindings()? {
            return response;
        }
        // Close-all and batch save can change ownership before a later item fails.
        let refresh = self
            .files
            .refresh_all_derived()
            .map_err(|error| error.to_string())
            .and_then(command_owner::check_refresh_report);
        combine_lifecycle_result(method, response, refresh)
    }

    fn source_bindings(&self) -> Result<SourceBindings, Error> {
        self.files
            .workspace()
            .documents()
            .iter()
            .map(|&id| {
                let path = match self.files.path(id) {
                    Ok(path) => Some(path.to_owned()),
                    Err(FileWorkspaceError::UnboundDocument(_)) => None,
                    Err(error) => return Err(Error(error.to_string())),
                };
                Ok((id, path))
            })
            .collect()
    }
}

fn combine_lifecycle_result(
    method: &str,
    response: Result<Vec<u8>, Error>,
    refresh: Result<(), String>,
) -> Result<Vec<u8>, Error> {
    let Err(reason) = refresh else {
        return response;
    };
    let command = match response {
        Ok(_) => "command completed".into(),
        Err(error) => format!("command failed after changing file ownership: {error}"),
    };
    Err(Error(format!(
        "{method}: {command}; derived refresh is incomplete: {reason}"
    )))
}
