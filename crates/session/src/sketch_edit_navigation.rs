// SPDX-License-Identifier: GPL-2.0-only
//! Transient edit-mode consequences of document history navigation.
use crate::{DocumentSession, SessionError, TransactionObserver};
use oblikovati_api::{types::TRANSACTION_POINT_CURRENT, wire::EVENT_TRANSACTION_COMMITTED};
use oblikovati_model::document_sketch::DocumentSketchId;

/// Editor identities on a navigation step; example: undoing Finish opens `to` from `from`.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct SketchEditNavigation {
    pub from: Option<DocumentSketchId>,
    pub to: Option<DocumentSketchId>,
}
impl SketchEditNavigation {
    /// Resolve only a compatible editor identity; example: `navigation.target(current, focused)`.
    pub fn target(
        self,
        current: Option<DocumentSketchId>,
        focused: Option<DocumentSketchId>,
    ) -> Option<Option<DocumentSketchId>> {
        let source = current.or(focused);
        (source == self.from || source == self.to).then_some(self.to)
    }
}
impl<O: TransactionObserver> DocumentSession<O> {
    /// Inspect an unapplied editor navigation; example: `session.pending_sketch_navigation()` for a read-only query.
    pub fn pending_sketch_navigation(&self) -> Option<SketchEditNavigation> {
        self.pending_sketch_navigation
    }
    pub(crate) fn tag_sketch_creation(
        &mut self,
        sketch: DocumentSketchId,
    ) -> Result<(), SessionError> {
        self.history.tag_sketch_creation(sketch)?;
        Ok(())
    }
    pub(crate) fn finish_sketch_history(
        &mut self,
        sketch: DocumentSketchId,
    ) -> Result<(), SessionError> {
        self.finish_sketch_history_using(sketch, None)
    }
    pub(crate) fn finish_sketch_history_with_datums(
        &mut self,
        sketch: DocumentSketchId,
        datums: &crate::part_evaluation::CurrentPartDatums,
    ) -> Result<(), SessionError> {
        let current = datums
            .current(self)
            .map_err(SessionError::PartEvaluation)?
            .clone();
        self.finish_sketch_history_using(sketch, Some(&current))
    }
    fn finish_sketch_history_using(
        &mut self,
        sketch: DocumentSketchId,
        datums: Option<&oblikovati_model::feature::work_geometry::WorkGeometryEvaluation>,
    ) -> Result<(), SessionError> {
        match datums {
            Some(datums) => self
                .history
                .validate_sketch_finish_with_datums(sketch, datums)?,
            None => self.history.validate_sketch_finish(sketch)?,
        }
        let event = self.payload(
            EVENT_TRANSACTION_COMMITTED,
            "Finish Sketch",
            TRANSACTION_POINT_CURRENT,
        );
        self.notify_before(&event)?;
        match datums {
            Some(datums) => self
                .history
                .record_sketch_finish_with_datums(sketch, datums)?,
            None => self.history.record_sketch_finish(sketch)?,
        }
        self.notify_after(&event);
        Ok(())
    }
    pub(crate) fn remember_sketch_navigation(&mut self, navigation: Option<SketchEditNavigation>) {
        if let Some(mut navigation) = navigation {
            if let Some(previous) = self.pending_sketch_navigation {
                navigation.from = previous.from;
            }
            self.pending_sketch_navigation = Some(navigation);
        }
    }
    pub(crate) fn take_sketch_navigation(&mut self) -> Option<SketchEditNavigation> {
        self.pending_sketch_navigation.take()
    }
    pub(crate) fn clear_sketch_navigation(&mut self) {
        self.pending_sketch_navigation = None;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn navigation_refuses_an_unrelated_edit_and_accepts_its_focused_source() {
        let sketch = DocumentSketchId(1);
        let other = DocumentSketchId(2);
        let navigation = SketchEditNavigation {
            from: Some(sketch),
            to: None,
        };
        assert_eq!(navigation.target(Some(other), Some(sketch)), None);
        assert_eq!(navigation.target(None, Some(sketch)), Some(None));
        assert_eq!(navigation.target(None, None), Some(None));
    }
}
