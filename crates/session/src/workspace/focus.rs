// SPDX-License-Identifier: GPL-2.0-only
use super::{WorkspaceObserver, WorkspaceSession};
use crate::TransactionObserver;
/// An uninterrupted workspace focus identity, independent of model history.
#[derive(Clone, Debug)]
pub struct WorkspaceFocusIdentity(std::sync::Arc<()>);
impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    /// Remember focus without a wrapping counter; example: `let focus = workspace.focus_identity();`.
    pub fn focus_identity(&self) -> WorkspaceFocusIdentity {
        WorkspaceFocusIdentity(self.focus_identity.clone())
    }
    /// Test whether focus stayed on the same document; example: `workspace.retains_focus(&focus)`.
    pub fn retains_focus(&self, identity: &WorkspaceFocusIdentity) -> bool {
        std::sync::Arc::ptr_eq(&self.focus_identity, &identity.0)
    }
}
