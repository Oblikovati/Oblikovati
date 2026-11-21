// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{
    Error,
    wire::{EVENT_SELECTION_CHANGED, SelectionChangedEvent},
};
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_session::{
    TransactionObserver,
    selection::{SelectionClickMode, SelectionSet, SketchPick},
    workspace::SessionCommands,
};

/// Outbound event transport; selection is already accepted when this notification is sent.
pub trait EventSink {
    /// Deliver owned-contract JSON or name the delivery failure; example: enqueue `event_json`.
    fn send_event(&mut self, event_json: &[u8]) -> Result<(), String>;
}
/// A typed relay over an injected transport, independent of native callbacks and model mutation.
pub struct EventRelay<S> {
    sink: S,
}
impl<S: EventSink> EventRelay<S> {
    /// Attach the owner's transport; example: `EventRelay::new(outbound_queue)`.
    pub fn new(sink: S) -> Self {
        Self { sink }
    }
    /// Publish the API's existing selection event, including empty selections.
    /// Example: `relay.selection_changed(0)?;` retries notification without replaying a click.
    pub fn selection_changed(&mut self, count: usize) -> Result<(), Error> {
        let count = i64::try_from(count).map_err(|_| {
            Error(format!(
                "selection count {count}: expected an API count in 0..={}",
                i64::MAX
            ))
        })?;
        let event = SelectionChangedEvent {
            type_: EVENT_SELECTION_CHANGED.into(),
            count,
        };
        let encoded = crate::encode(&event)?;
        self.sink.send_event(&encoded).map_err(|cause| Error(format!(
            "selection event {event:?} delivery failed: {cause}; expected an accepted outbound notification"
        )))
    }
    /// Recover the injected transport at shutdown; example: `let queue = relay.into_sink();`.
    pub fn into_sink(self) -> S {
        self.sink
    }
}
/// Resolved desktop input; this is not an API wire request or a pixel hit-test result.
#[derive(Clone, Copy, Debug)]
pub struct SelectionClick {
    pub document: u64,
    pub sketch: DocumentSketchId,
    pub picked: Option<SketchPick>,
    pub mode: SelectionClickMode,
    pub tool_active: bool,
}
/// Keep an accepted selection change separate from failure to deliver its notification.
#[derive(Debug)]
pub struct SelectionDispatch {
    pub count: Option<usize>,
    pub delivery_error: Option<Error>,
}
pub(crate) fn dispatch_selection_click<O: TransactionObserver, S: EventSink>(
    commands: &SessionCommands<'_, O>,
    click: &SelectionClick,
    picks: &mut SelectionSet<SketchPick>,
    relay: &mut EventRelay<S>,
) -> Result<SelectionDispatch, Error> {
    let count = commands
        .apply_sketch_click(
            click.sketch,
            click.picked,
            click.mode,
            click.tool_active,
            picks,
        )
        .map_err(|error| Error(error.to_string()))?;
    let delivery_error = count.and_then(|count| relay.selection_changed(count).err());
    Ok(SelectionDispatch {
        count,
        delivery_error,
    })
}

/// Editor Delete intent, scoped to an owned document; this is not a wire request.
#[derive(Clone, Copy, Debug)]
pub struct SelectionDelete {
    pub document: u64,
    pub active_sketch: Option<DocumentSketchId>,
    pub tool_active: bool,
}
/// Accepted geometry retirement and any independent notification delivery failure.
#[derive(Debug)]
pub struct SelectionDeletionDispatch {
    pub retired: oblikovati_model::sketch::SketchEntityDeletion,
    pub delivery_error: Option<Error>,
}
pub(crate) fn dispatch_selection_delete<O: TransactionObserver, S: EventSink>(
    commands: &mut SessionCommands<'_, O>,
    input: &SelectionDelete,
    picks: &mut SelectionSet<SketchPick>,
    relay: &mut EventRelay<S>,
) -> Result<SelectionDeletionDispatch, Error> {
    let retired = commands
        .delete_selected_sketch_picks(input.active_sketch, input.tool_active, picks)
        .map_err(|error| Error(error.to_string()))?;
    let changed = !retired.entities.is_empty() || !retired.constraints.is_empty();
    let delivery_error = changed.then(|| relay.selection_changed(0).err()).flatten();
    Ok(SelectionDeletionDispatch {
        retired,
        delivery_error,
    })
}
