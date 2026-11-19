// SPDX-License-Identifier: GPL-2.0-only
//! View state belongs to the open document, outside authored history (Go #1105).
use crate::{DocumentSession, TransactionObserver};
use std::collections::BTreeSet;

#[derive(Clone, Default)]
pub(super) struct BodyVisibilityState {
    hidden: BTreeSet<String>,
    sketch_isolated: bool,
    revision: u64,
}
impl BodyVisibilityState {
    fn effective_visibility(&self, key: &str) -> bool {
        !self.sketch_isolated && !self.hidden.contains(key)
    }
    pub(crate) fn set_sketch_isolated(&mut self, isolated: bool) -> Result<bool, String> {
        if self.sketch_isolated == isolated {
            return Ok(false);
        }
        let revision = next_visibility_revision(self.revision, "sketch isolation")?;
        self.sketch_isolated = isolated;
        self.revision = revision;
        Ok(true)
    }
    /// What the scene shows of a body changed for another reason than its visibility.
    pub(super) fn note_display_change(&mut self) {
        self.revision = self.revision.wrapping_add(1);
    }
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
    pub(super) fn replace_hidden(
        &mut self,
        hidden: &BTreeSet<String>,
        available: &BTreeSet<String>,
    ) -> Result<bool, String> {
        let Some(revision) = self.prepare_hidden(hidden, available)? else {
            return Ok(false);
        };
        self.hidden.clone_from(hidden);
        self.revision = revision;
        Ok(true)
    }
    pub(super) fn prepare_hidden(
        &self,
        hidden: &BTreeSet<String>,
        available: &BTreeSet<String>,
    ) -> Result<Option<u64>, String> {
        if let Some(key) = hidden.iter().find(|key| !available.contains(*key)) {
            return Err(format!(
                "body visibility key {key:?}: expected a current resolved body"
            ));
        }
        if &self.hidden == hidden {
            return Ok(None);
        }
        let revision = next_visibility_revision(self.revision, "part view")?;
        Ok(Some(revision))
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
        self.body_visibility.effective_visibility(key)
    }
    /// Read the transient isolation mode; example: `session.sketch_isolated()`.
    pub fn sketch_isolated(&self) -> bool {
        self.body_visibility.sketch_isolated
    }
    /// Inspect display changes independently of modelling; example: `session.body_visibility_revision()`.
    pub fn body_visibility_revision(&self) -> u64 {
        self.body_visibility.revision
    }
    /// Inspect hidden body keys for view capture; example: `session.hidden_body_keys()`.
    pub fn hidden_body_keys(&self) -> &BTreeSet<String> {
        &self.body_visibility.hidden
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
    #[test]
    fn replacing_hidden_bodies_is_atomic_and_advances_one_revision() {
        let mut state = BodyVisibilityState::default();
        let available = BTreeSet::from(["aa".into(), "bb".into()]);
        let selected = BTreeSet::from(["aa".into()]);
        assert!(state.replace_hidden(&selected, &available).unwrap());
        assert!(!state.replace_hidden(&selected, &available).unwrap());
        assert!(
            state
                .replace_hidden(&BTreeSet::from(["cc".into()]), &available)
                .is_err()
        );
        assert_eq!(state.hidden, selected);
        assert_eq!(state.revision, 1);
        state.revision = u64::MAX;
        assert!(state.replace_hidden(&BTreeSet::new(), &available).is_err());
        assert_eq!(state.hidden, selected);
    }
    #[test]
    fn sketch_isolation_hides_display_without_changing_the_prior_hidden_set() {
        let mut state = BodyVisibilityState::default();
        state.set("already-hidden", false).unwrap();
        let prior = state.hidden.clone();
        let before = state.revision;
        assert!(state.set_sketch_isolated(true).unwrap());
        assert!(!state.effective_visibility("visible"));
        assert!(!state.effective_visibility("already-hidden"));
        assert_eq!(state.hidden, prior);
        assert!(!state.set_sketch_isolated(true).unwrap());
        assert!(state.set_sketch_isolated(false).unwrap());
        assert!(state.effective_visibility("visible"));
        assert!(!state.effective_visibility("already-hidden"));
        assert_eq!(state.revision, before + 2);
    }
    #[test]
    fn exhausted_revision_does_not_change_sketch_isolation() {
        let mut state = BodyVisibilityState {
            revision: u64::MAX,
            ..Default::default()
        };
        assert!(state.set_sketch_isolated(true).is_err());
        assert!(!state.sketch_isolated);
        assert!(state.effective_visibility("body"));
    }
}
