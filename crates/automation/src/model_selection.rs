// SPDX-License-Identifier: GPL-2.0-only
use crate::{Error, TransactionObserver, WorkspaceObserver, WorkspaceSession};
use oblikovati_api::wire::SelectionResult;
use oblikovati_session::selection::{Selectable, SelectionKind};

pub(crate) fn read<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
) -> Result<SelectionResult, Error> {
    let Some(document) = workspace.active_document() else {
        return selection_response(0, &[]);
    };
    let session = workspace
        .session(document)
        .map_err(|cause| Error(cause.to_string()))?;
    selection_response(
        session.selected_browser_sketches().len(),
        &session.selected_editor_picks(),
    )
}
fn selection_response(
    browser: usize,
    editor: &[oblikovati_session::selection::SketchPick],
) -> Result<SelectionResult, Error> {
    let (count, count_i64) = checked_selection_count(browser, editor.len())?;
    // Go app/selection.go at aa8297d1 uses SelectSketch=8; empty references stay index-aligned.
    let mut kinds = vec![SelectionKind::Sketch as i64; browser];
    kinds.extend(editor.iter().map(|pick| pick.selection_kind() as i64));
    Ok(SelectionResult {
        count: count_i64,
        kinds: Some(kinds),
        refs: Some(vec![String::new(); count]),
    })
}
fn checked_selection_count(browser: usize, editor: usize) -> Result<(usize, i64), Error> {
    let count = browser.checked_add(editor).ok_or_else(|| {
        Error(format!(
            "selection counts {browser}+{editor}: expected a representable usize count"
        ))
    })?;
    let count_i64 = i64::try_from(count)
        .map_err(|_| Error(format!("selection count {count}: expected an i64 count")))?;
    Ok((count, count_i64))
}

#[cfg(all(test, target_pointer_width = "64"))]
mod tests {
    #[test]
    fn overflowing_selection_count_declines_before_allocating_arrays() {
        let cause = super::selection_response(usize::MAX, &[]).unwrap_err();
        assert!(cause.to_string().contains(&usize::MAX.to_string()));
        assert!(cause.to_string().contains("expected an i64 count"));
        let sum = super::checked_selection_count(usize::MAX, 1).unwrap_err();
        assert!(
            sum.to_string()
                .contains("expected a representable usize count")
        );
    }
}
