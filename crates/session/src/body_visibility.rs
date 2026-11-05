// SPDX-License-Identifier: GPL-2.0-only
//! View state belongs to the open document, outside authored history (Go #1105).
use crate::{DocumentSession, TransactionObserver};
use std::collections::BTreeSet;

#[derive(Default)]
pub(super) struct BodyVisibilityState {
    hidden: BTreeSet<String>,
    revision: u64,
}
impl BodyVisibilityState {
    pub(super) fn set(&mut self, key: &str, visible: bool) -> Result<bool, String> {
        if key.is_empty() {
            return Err(format!(
                "body visibility key {key:?}: expected a nonempty resolved reference"
            ));
        }
        if self.hidden.contains(key) != visible {
            return Ok(false);
        }
        let revision = next_visibility_revision(self.revision, key)?;
        if visible {
            self.hidden.remove(key);
        } else {
            self.hidden.insert(key.into());
        }
        self.revision = revision;
        Ok(true)
    }
}
fn next_visibility_revision(revision: u64, key: &str) -> Result<u64, String> {
    revision.checked_add(1).ok_or_else(|| {
        format!(
            "body visibility revision {revision} for {key:?}: expected space for one display change"
        )
    })
}
impl<O: TransactionObserver> DocumentSession<O> {
    /// Read document-local view state; example: `session.body_visible(resolved_key)`.
    pub fn body_visible(&self, key: &str) -> bool {
        !self.body_visibility.hidden.contains(key)
    }
    /// Inspect display changes independently of modelling; example: `session.body_visibility_revision()`.
    pub fn body_visibility_revision(&self) -> u64 {
        self.body_visibility.revision
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn visibility_is_idempotent_and_refuses_empty_keys() {
        let mut state = BodyVisibilityState::default();
        assert!(
            state
                .set("", false)
                .unwrap_err()
                .contains("nonempty resolved reference")
        );
        assert!(!state.set("key", true).unwrap());
        assert!(state.set("key", false).unwrap());
        assert!(!state.set("key", false).unwrap());
        assert!(state.set("key", true).unwrap());
        assert_eq!(state.revision, 2);
        assert!(state.hidden.is_empty());
    }
    #[test]
    fn exhausted_visibility_revision_preserves_existing_flags() {
        let mut state = BodyVisibilityState {
            revision: u64::MAX,
            ..Default::default()
        };
        let error = state.set("key", false).unwrap_err();
        assert!(error.contains(&u64::MAX.to_string()) && error.contains("key"));
        assert_eq!(state.revision, u64::MAX);
        assert!(state.hidden.is_empty());
        assert!(!state.set("key", true).unwrap());
    }
}
