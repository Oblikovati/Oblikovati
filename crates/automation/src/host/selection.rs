// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::selection_events::{
    EventRelay, EventSink, SelectionClick, SelectionDispatch, dispatch_selection_click,
};
use oblikovati_session::selection::{SelectionSet, SketchPick};
impl<O: TransactionObserver, W: WorkspaceObserver> SessionHost<'_, O, W> {
    /// Apply a checked desktop click and publish its selection change through the injected relay.
    /// The receipt separates accepted picks from delivery failure; example: `host.selection_click(&click, &mut picks, &mut relay)?;`.
    pub fn selection_click<S: EventSink>(
        &self,
        click: &SelectionClick,
        picks: &mut SelectionSet<SketchPick>,
        relay: &mut EventRelay<S>,
    ) -> Result<SelectionDispatch, Error> {
        let mut workspace = self
            .workspace
            .try_borrow_mut()
            .map_err(|_| busy(oblikovati_api::wire::EVENT_SELECTION_CHANGED))?;
        let commands = workspace
            .commands(click.document)
            .map_err(|error| Error(error.to_string()))?;
        dispatch_selection_click(&commands, click, picks, relay)
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver> SessionHost<'_, O, W> {
    /// Delete through normal history, then return retirement and independent notification status.
    /// Example: `host.selection_delete(&input, &mut picks, &mut relay)?;`.
    pub fn selection_delete<S: EventSink>(
        &self,
        input: &crate::selection_events::SelectionDelete,
        picks: &mut SelectionSet<SketchPick>,
        relay: &mut EventRelay<S>,
    ) -> Result<crate::selection_events::SelectionDeletionDispatch, Error> {
        let mut workspace = self
            .workspace
            .try_borrow_mut()
            .map_err(|_| busy(oblikovati_api::wire::EVENT_SELECTION_CHANGED))?;
        let mut commands = workspace
            .commands(input.document)
            .map_err(|error| Error(error.to_string()))?;
        crate::selection_events::dispatch_selection_delete(&mut commands, input, picks, relay)
    }
}
