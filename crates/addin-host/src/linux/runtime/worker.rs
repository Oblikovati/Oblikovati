// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{contract::AddInRegistry, types::AddInLoadBehavior};
use std::path::PathBuf;
#[derive(Debug)]
pub(super) enum NativeAction {
    Load(PathBuf),
    Activate(String),
    Deactivate(String),
    Policy(String, AddInLoadBehavior),
    Notify(String, Vec<u8>),
    Automation(String, String, Vec<u8>),
    Shutdown,
}
pub(super) struct NativeRequest {
    pub action: NativeAction,
    pub completed: Sender<NativeReply>,
}
pub(super) struct NativeReply {
    pub outcome: Result<NativeCompletion, Error>,
    pub catalog: Vec<AddInInfo>,
    pub states: Vec<(String, AddInState)>,
    pub counts: HostBufferCounts,
}
pub(super) fn run<W: DispatchWakeup>(
    incoming: Receiver<NativeRequest>,
    wakeup: RuntimeSignal<W>,
    binding: HostCallbackBinding,
) {
    let mut registry = NativeAddInRegistry::new(binding);
    for request in incoming {
        let outcome = apply(&mut registry, request.action);
        let finished = outcome == Ok(NativeCompletion::Shutdown);
        let reply = checkpoint(&registry, outcome);
        let _ = request.completed.send(reply);
        let _ = wakeup.wake();
        if finished {
            break;
        }
    }
}
fn checkpoint(
    registry: &NativeAddInRegistry,
    outcome: Result<NativeCompletion, Error>,
) -> NativeReply {
    let ids = registry.registered().unwrap_or_default();
    let catalog = ids
        .iter()
        .map(|id| registry.describe(id).expect("registered identity"))
        .collect();
    let states = ids
        .into_iter()
        .map(|id| {
            let state = registry.lifecycle_state(&id).expect("registered lifecycle");
            (id, state)
        })
        .collect();
    NativeReply {
        outcome,
        catalog,
        states,
        counts: registry.buffer_counts(),
    }
}
fn apply(
    registry: &mut NativeAddInRegistry,
    action: NativeAction,
) -> Result<NativeCompletion, Error> {
    // SAFETY: runtime admission trusts each library's ABI and fixed worker affinity; callbacks stay owned.
    unsafe {
        match action {
            NativeAction::Load(path) => registry.load(&path).map(NativeCompletion::Loaded),
            NativeAction::Activate(id) => registry.activate(&id).map(finished),
            NativeAction::Deactivate(id) => registry.deactivate(&id).map(finished),
            NativeAction::Policy(id, policy) => {
                registry.set_load_behavior(&id, policy).map(finished)
            }
            NativeAction::Notify(id, bytes) => registry.notify(&id, &bytes).map(finished),
            NativeAction::Automation(id, method, bytes) => registry
                .call_automation(&id, &method, &bytes)
                .map(NativeCompletion::Automation),
            NativeAction::Shutdown => stop(registry),
        }
    }
}
fn finished(_: ()) -> NativeCompletion {
    NativeCompletion::Finished
}
unsafe fn stop(registry: &mut NativeAddInRegistry) -> Result<NativeCompletion, Error> {
    // SAFETY: same worker, joined add-in contract, and caller's owner loop still drains callbacks.
    unsafe { registry.shutdown() }
        .map(|()| NativeCompletion::Shutdown)
        .map_err(|failures| {
            Error(format!(
                "native shutdown: expected all cleanups to succeed; {}",
                failures
                    .into_iter()
                    .map(|error| error.0)
                    .collect::<Vec<_>>()
                    .join("; ")
            ))
        })
}
