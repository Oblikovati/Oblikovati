// SPDX-License-Identifier: GPL-2.0-only
//! View identities share the document's permanent recipe reservation stream (#4118).
use super::{DocumentEditHistory, DocumentSnapshot, HistoryError, RecipeReservations};
use oblikovati_api::contract::PartDesignView as _;

pub(super) fn authored_ids(snapshot: &DocumentSnapshot) -> Vec<u64> {
    snapshot
        .part_design_views
        .iter()
        .flat_map(|views| views.list())
        .filter(|view| !view.built_in())
        .map(|view| view.id())
        .collect()
}
impl DocumentSnapshot {
    pub(super) fn validate_part_design_views(&self) -> Result<(), HistoryError> {
        if let Some(views) = &self.part_design_views {
            views.validate().map_err(HistoryError::PartDesignViews)?;
            if !self.design_views.is_empty() {
                return Err(HistoryError::PartDesignViews(format!(
                    "legacy view names {:?}: expected one part view source",
                    self.design_views
                )));
            }
        }
        Ok(())
    }
    pub(super) fn validate_part_design_view_transition(
        &self,
        before: &Self,
    ) -> Result<(), HistoryError> {
        if before.part_design_views.is_some() && self.part_design_views.is_none() {
            return Err(HistoryError::PartDesignViews(
                "part views None: expected retained part views".into(),
            ));
        }
        self.validate_part_design_views()
    }
}
impl DocumentEditHistory {
    pub(super) fn validate_part_design_view_ids(
        &self,
        candidate: &DocumentSnapshot,
        before: u64,
    ) -> Result<(), HistoryError> {
        candidate.validate_part_design_views()?;
        super::sketch_ids::require_reserved(
            authored_ids(candidate),
            authored_ids(&self.current),
            before,
            self.high_water(),
            HistoryError::UnreservedIdentity,
        )
    }
}
impl RecipeReservations<'_> {
    /// Reserve an authored view ID that undo cannot reuse. Example: `ids.allocate_part_design_view()?`.
    pub fn allocate_part_design_view(&mut self) -> Result<u64, crate::identity::RecipeIdError> {
        loop {
            let id = self.allocate()?;
            if id > 5 {
                return Ok(id);
            }
        }
    }
}
