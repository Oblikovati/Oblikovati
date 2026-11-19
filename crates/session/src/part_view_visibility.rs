// SPDX-License-Identifier: GPL-2.0-only
//! Body visibility is retained for unlocked part views and transient for locked views (#4118).
use crate::{DocumentSession, TransactionObserver};
use oblikovati_api::contract::PartDesignView as _;
use std::collections::BTreeSet;

impl<O: TransactionObserver> DocumentSession<O> {
    /// Change resolved bodies in the active view; example: `session.set_part_view_body_visibility(&keys, &available, false)?`.
    pub fn set_part_view_body_visibility(
        &mut self,
        keys: &[String],
        available: &BTreeSet<String>,
        visible: bool,
    ) -> Result<bool, String> {
        if self.group.is_some() {
            return Err(
                "part view visibility with open transaction: expected no open group".into(),
            );
        }
        let selected = selected_body_keys(keys, available)?;
        let views = self
            .snapshot()
            .part_design_views
            .as_ref()
            .ok_or("part view visibility: expected part views")?;
        let active = views
            .active()
            .ok_or("part view visibility: expected an active view")?;
        let id = active.id();
        let locked = active.locked();
        let stored_change = selected
            .iter()
            .any(|key| active.hidden_body_keys().contains(key) == visible);
        let mut next = self.hidden_body_keys().clone();
        for key in &selected {
            if visible {
                next.remove(key);
            } else {
                next.insert(key.clone());
            }
        }
        let previous = self.body_visibility.clone();
        let display = oblikovati_model::history::PartViewDisplayTransition {
            before: oblikovati_model::history::PartViewDisplayState {
                hidden_body_keys: self.hidden_body_keys().clone(),
                camera: None,
            },
            after: oblikovati_model::history::PartViewDisplayState {
                hidden_body_keys: next.clone(),
                camera: None,
            },
        };
        let display_change = self.body_visibility.replace_hidden(&next, available)?;
        if locked || !stored_change {
            return Ok(display_change);
        }
        let result =
            self.run_part_view_edit("Change Part View Visibility", display, |snapshot, _| {
                snapshot
                    .part_design_views
                    .as_mut()
                    .ok_or("part view visibility: expected retained part views".to_string())?
                    .set_body_visibility(id, keys, available, visible)
            });
        if let Err(cause) = result {
            self.body_visibility = previous;
            return Err(cause.to_string());
        }
        Ok(true)
    }
}
fn selected_body_keys(
    keys: &[String],
    available: &BTreeSet<String>,
) -> Result<BTreeSet<String>, String> {
    if keys.is_empty() {
        return Ok(available.clone());
    }
    let selected: BTreeSet<_> = keys.iter().cloned().collect();
    if selected.len() != keys.len() || !selected.is_subset(available) {
        return Err(format!(
            "body keys {keys:?}: expected distinct current resolved bodies"
        ));
    }
    Ok(selected)
}
