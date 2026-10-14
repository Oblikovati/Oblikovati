// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl SelectionEndpoint {
    pub(super) fn with_requestor<T>(
        &self,
        perform: impl FnOnce(Window) -> Result<T, String>,
    ) -> Result<T, String> {
        // A fresh window separates completed, late, and failed conversions without a guessed sequence.
        let requestor = create_selection_window(&self.connection, self.screen)?;
        let result = perform(requestor);
        let cleanup = self
            .connection
            .destroy_window(requestor)
            .map_err(|cause| clipboard_failure("requestor retirement", cause))
            .and_then(|cookie| {
                cookie
                    .check()
                    .map_err(|cause| clipboard_failure("requestor retirement reply", cause))
            });
        finish_requestor(result, cleanup)
    }
}
pub(super) fn finish_requestor<T>(
    result: Result<T, String>,
    cleanup: Result<(), String>,
) -> Result<T, String> {
    match (result, cleanup) {
        (Ok(value), Ok(())) => Ok(value),
        (Err(cause), Ok(())) | (Ok(_), Err(cause)) => Err(cause),
        (Err(cause), Err(cleanup)) => Err(format!("{cause}; requestor retirement: {cleanup}")),
    }
}
