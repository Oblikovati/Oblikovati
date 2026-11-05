// SPDX-License-Identifier: GPL-2.0-only
use crate::{Error, TransactionObserver, WorkspaceObserver, WorkspaceSession};
use oblikovati_api::wire::SelectionResult;
use oblikovati_session::selection::SelectionKind;

pub(crate) fn read<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
) -> Result<SelectionResult, Error> {
    let Some(document) = workspace.active_document() else {
        return browser_selection_response(0);
    };
    let session = workspace
        .session(document)
        .map_err(|cause| Error(cause.to_string()))?;
    browser_selection_response(session.selected_browser_sketches().len())
}
fn browser_selection_response(count: usize) -> Result<SelectionResult, Error> {
    let count_i64 = i64::try_from(count)
        .map_err(|_| Error(format!("selection count {count}: expected an i64 count")))?;
    // Go app/selection.go at aa8297d1 uses SelectSketch=8; empty references stay index-aligned.
    Ok(SelectionResult {
        count: count_i64,
        kinds: Some(vec![SelectionKind::Sketch as i64; count]),
        refs: Some(vec![String::new(); count]),
    })
}

#[cfg(all(test, target_pointer_width = "64"))]
mod tests {
    #[test]
    fn overflowing_selection_count_declines_before_allocating_arrays() {
        let cause = super::browser_selection_response(usize::MAX).unwrap_err();
        assert!(cause.to_string().contains(&usize::MAX.to_string()));
        assert!(cause.to_string().contains("expected an i64 count"));
    }
}
