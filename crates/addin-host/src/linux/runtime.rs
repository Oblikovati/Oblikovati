// SPDX-License-Identifier: GPL-2.0-only
use super::{AddInState, HostBufferCounts, HostCallbackBinding, NativeAddInRegistry};
use oblikovati_api::{Error, client::Caller, wire::*};
use oblikovati_automation::dispatch::{
    DispatchWakeup, OwnerDispatch, QueuedCaller, owner_dispatch,
};
use std::{
    path::Path,
    sync::{
        Arc,
        mpsc::{self, Receiver, Sender, SyncSender},
    },
    thread::{self, JoinHandle},
    time::Duration,
};
mod controls;
mod queries;
mod worker;
use queries::CatalogCaller;
use worker::{NativeAction, NativeReply, NativeRequest};
/// One completed native control; these are host implementation results, not new wire DTOs.
#[derive(Debug, PartialEq)]
pub enum NativeCompletion {
    Loaded(String),
    Finished,
    Automation(Vec<u8>),
    Shutdown,
}
struct RuntimeSignal<W> {
    signal: Sender<()>,
    external: Arc<W>,
}
impl<W> Clone for RuntimeSignal<W> {
    fn clone(&self) -> Self {
        Self {
            signal: self.signal.clone(),
            external: Arc::clone(&self.external),
        }
    }
}
impl<W: DispatchWakeup> DispatchWakeup for RuntimeSignal<W> {
    fn wake(&self) -> Result<(), String> {
        self.signal.send(()).map_err(|error| error.to_string())?;
        self.external.wake()
    }
}
/// Persistent foreign-call thread plus a bounded owner queue; it owns no document or UI state.
/// Poll it from native frames or wait on the CLI owner. Drop closes admission, not foreign resources.
pub struct NativeAddInRuntime<W: DispatchWakeup> {
    owner: OwnerDispatch<RuntimeSignal<W>>,
    caller: QueuedCaller<RuntimeSignal<W>>,
    awake: Receiver<()>,
    requests: SyncSender<NativeRequest>,
    worker: Option<JoinHandle<()>>,
    pending: Option<Receiver<NativeReply>>,
    catalog: Vec<AddInInfo>,
    states: Vec<(String, AddInState)>,
    counts: HostBufferCounts,
}
impl<W: DispatchWakeup + 'static> NativeAddInRuntime<W> {
    /// Bind once and start the native thread; example: `NativeAddInRuntime::new(16, timeout, wake)?;`.
    pub fn new(capacity: usize, timeout: Duration, external: W) -> Result<Self, Error> {
        let (wakeup, awake) = runtime_signal(external);
        let (owner, queued) = owner_dispatch(capacity, timeout, wakeup.clone());
        let binding = HostCallbackBinding::install(Arc::new(queued.clone()))?;
        let (requests, incoming) = mpsc::sync_channel(1);
        let worker = thread::spawn(move || worker::run(incoming, wakeup, binding));
        Ok(Self {
            owner,
            caller: queued,
            awake,
            requests,
            worker: Some(worker),
            pending: None,
            catalog: Vec::new(),
            states: Vec::new(),
            counts: HostBufferCounts::default(),
        })
    }
    /// Clone a model-free worker transport; example: `Client::new(runtime.caller())` on a worker.
    /// It must still use the owner drain loop; owner-thread synchronous calls are refused.
    pub fn caller(&self) -> impl Caller + Clone + Send + Sync + 'static + use<W> {
        self.caller.clone()
    }
    /// Drain one frame and inspect native completion; example: `runtime.poll(32, &host)?;`.
    /// Model ownership stays with the borrowed host; accepted requests retain queue timeout semantics.
    pub fn poll<C: Caller>(
        &mut self,
        budget: usize,
        host: &C,
    ) -> Result<Option<Result<NativeCompletion, Error>>, Error> {
        self.owner.drain(
            budget,
            &CatalogCaller {
                catalog: &self.catalog,
                model: host,
            },
        )?;
        let Some(pending) = &self.pending else {
            return Ok(None);
        };
        match pending.try_recv() {
            Ok(reply) => self.complete(reply).map(Some),
            Err(mpsc::TryRecvError::Empty) => Ok(None),
            Err(mpsc::TryRecvError::Disconnected) => self.worker_lost(),
        }
    }
    /// Wait while draining owner calls; example: `runtime.wait(&host)?;` after start_activate.
    /// There is no foreign timeout/cancellation claim; a hung in-process library cannot be killed safely.
    pub fn wait<C: Caller>(&mut self, host: &C) -> Result<NativeCompletion, Error> {
        if self.pending.is_none() {
            return Err(Error(
                "native control absent: expected a pending operation".into(),
            ));
        }
        loop {
            if let Some(outcome) = self.poll(32, host)? {
                return outcome;
            }
            let _ = self.awake.recv_timeout(Duration::from_millis(20));
        }
    }
    fn begin(&mut self, action: NativeAction) -> Result<(), Error> {
        if self.pending.is_some() || self.worker.is_none() {
            return Err(Error(format!(
                "native control {action:?}: expected an idle live worker"
            )));
        }
        let (completed, reply) = mpsc::channel();
        self.requests
            .try_send(NativeRequest { action, completed })
            .map_err(|error| {
                Error(format!(
                    "native control admission: expected a live empty worker queue: {error}"
                ))
            })?;
        self.pending = Some(reply);
        Ok(())
    }
    fn complete(&mut self, reply: NativeReply) -> Result<Result<NativeCompletion, Error>, Error> {
        self.pending = None;
        self.catalog = reply.catalog;
        self.states = reply.states;
        self.counts = reply.counts;
        if reply.outcome == Ok(NativeCompletion::Shutdown) {
            self.join_worker()?;
        }
        Ok(reply.outcome)
    }
    fn join_worker(&mut self) -> Result<(), Error> {
        self.owner.close();
        if let Some(worker) = self.worker.take() {
            worker
                .join()
                .map_err(|_| Error("native worker panicked: expected joined shutdown".into()))?;
        }
        Ok(())
    }
    fn worker_lost(&mut self) -> Result<Option<Result<NativeCompletion, Error>>, Error> {
        self.pending = None;
        self.owner.close();
        self.worker.take();
        Err(Error(
            "native control worker disconnected: expected a completion; owner admission stopped"
                .into(),
        ))
    }
}

fn runtime_signal<W>(external: W) -> (RuntimeSignal<W>, Receiver<()>) {
    let (signal, awake) = mpsc::channel();
    (
        RuntimeSignal {
            signal,
            external: Arc::new(external),
        },
        awake,
    )
}
