// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    ScriptProgram, ScriptReport,
    protocol::{ParentFrame, WorkerFrame, write_frame},
};
use oblikovati_api::client::Caller;
use std::{
    ffi::OsString,
    io::Write,
    path::{Path, PathBuf},
    process::{Child, ChildStdin, Command, Stdio},
    sync::{
        Arc, Mutex,
        atomic::{AtomicBool, Ordering},
        mpsc::{self, Receiver},
    },
    thread::{self, JoinHandle},
    time::{Duration, Instant},
};
mod events;
mod inbox;
mod readers;
mod watch;
#[derive(Clone, Default)]
pub struct ScriptStop(Arc<AtomicBool>);
impl ScriptStop {
    /// Stop interpreter work and new host calls; example: `execution.stop_handle().cancel()`.
    pub fn cancel(&self) {
        self.0.store(true, Ordering::Release);
    }
}
/// Own the executable location; native Lua is never initialized by this parent-side interface.
pub struct IsolatedLuaEngine {
    executable: PathBuf,
    arguments: Vec<OsString>,
    active: Arc<AtomicBool>,
}
impl IsolatedLuaEngine {
    /// Select a trusted private worker entry; example: `IsolatedLuaEngine::new(executable, args)`.
    pub fn new(executable: &Path, arguments: Vec<OsString>) -> Self {
        Self {
            executable: executable.into(),
            arguments,
            active: Arc::new(AtomicBool::new(false)),
        }
    }
    /// Start one isolated interpreter; example: `engine.start(program)?;` then poll on the model owner.
    pub fn start(&self, program: ScriptProgram) -> Result<ScriptExecution, String> {
        program.limits.validate(&program.source)?;
        let lease = RunLease::acquire(self.active.clone())?;
        let child = Command::new(&self.executable)
            .args(&self.arguments)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .spawn()
            .map_err(|error| {
                format!(
                    "script worker {:?}: expected executable child: {error}",
                    self.executable
                )
            })?;
        ScriptExecution::attach(child, program, lease)
    }
}
struct WorkerProcess {
    child: Arc<Mutex<Child>>,
    input: Arc<Mutex<ChildStdin>>,
    readers: Vec<JoinHandle<()>>,
    monitor: Option<JoinHandle<()>>,
    finished: Arc<AtomicBool>,
}
pub struct ScriptExecution {
    worker: WorkerProcess,
    incoming: Receiver<Result<WorkerFrame, String>>,
    stop: ScriptStop,
    deadline: Instant,
    limits: crate::ScriptLimits,
    output_bytes: usize,
    host_calls: u64,
    completed: bool,
    lease: Option<RunLease>,
}
impl ScriptExecution {
    fn attach(child: Child, program: ScriptProgram, lease: RunLease) -> Result<Self, String> {
        let stop = ScriptStop::default();
        let deadline = Instant::now()
            .checked_add(Duration::from_millis(program.limits.wall_millis))
            .ok_or("script wall limit: expected a representable deadline")?;
        let (worker, incoming, events) = WorkerProcess::attach(child, deadline, stop.clone())?;
        let mut execution = Self {
            worker,
            incoming,
            stop,
            deadline,
            limits: program.limits.clone(),
            output_bytes: 0,
            host_calls: 0,
            completed: false,
            lease: Some(lease),
        };
        execution.start_upload(program, events);
        Ok(execution)
    }
    fn start_upload(
        &mut self,
        program: ScriptProgram,
        events: mpsc::SyncSender<Result<WorkerFrame, String>>,
    ) {
        let input = self.worker.input.clone();
        self.worker.readers.push(thread::spawn(move || {
            let result = input
                .lock()
                .map_err(|_| "worker startup input poisoned: expected writable pipe".to_owned())
                .and_then(|mut input| write_frame(&mut *input, &ParentFrame::Start { program }));
            if let Err(error) = result {
                let _ = events.send(Err(error));
            }
        }));
    }
    /// Copy the cancellation handle; example: send `execution.stop_handle()` to the Stop button.
    pub fn stop_handle(&self) -> ScriptStop {
        self.stop.clone()
    }
    /// Process one owner-side event without moving the model; example: `execution.poll(&host, output)?;`.
    pub fn poll<C: Caller>(
        &mut self,
        host: &C,
        output: &mut dyn Write,
    ) -> Result<Option<ScriptReport>, String> {
        if self.completed {
            return Err("script execution complete: expected a live run".into());
        }
        if let Err(error) = self.require_live() {
            return self.fail(error);
        }
        self.receive(false, host, output)
    }

    /// Service the CLI owner until completion; example: `execution.wait(&host, output)?;`.
    pub fn wait<C: Caller>(
        &mut self,
        host: &C,
        output: &mut dyn Write,
    ) -> Result<ScriptReport, String> {
        loop {
            if self.completed {
                return Err("script execution complete: expected a live run".into());
            }
            if let Err(error) = self.require_live() {
                return self.fail(error);
            }
            if let Some(report) = self.receive(true, host, output)? {
                return Ok(report);
            }
        }
    }

    fn require_live(&self) -> Result<(), String> {
        if self.stop.0.load(Ordering::Acquire) {
            return Err(
                "script cancelled: expected a live run; committed edits are retained".into(),
            );
        }
        if Instant::now() >= self.deadline {
            return Err(format!(
                "script wall limit {} ms exceeded: expected bounded worker completion",
                self.limits.wall_millis
            ));
        }
        Ok(())
    }
    fn fail<T>(&mut self, failure: String) -> Result<T, String> {
        self.finish();
        Err(failure)
    }
    fn finish(&mut self) {
        self.completed = true;
        self.worker.finished.store(true, Ordering::Release);
        watch::terminate(&self.worker.child);
        // Drop the receiver before joins so bounded stream producers cannot remain blocked.
        let (_, empty) = mpsc::channel();
        self.incoming = empty;
        for reader in self.worker.readers.drain(..) {
            let _ = reader.join();
        }
        if let Some(monitor) = self.worker.monitor.take() {
            let _ = monitor.join();
        }
        self.lease.take();
    }
}
impl Drop for ScriptExecution {
    fn drop(&mut self) {
        if !self.completed {
            self.finish();
        }
    }
}

struct RunLease(Arc<AtomicBool>);
impl RunLease {
    fn acquire(active: Arc<AtomicBool>) -> Result<Self, String> {
        active
            .compare_exchange(false, true, Ordering::AcqRel, Ordering::Acquire)
            .map_err(|_| "script engine busy: expected one execution per session".to_owned())?;
        Ok(Self(active))
    }
}
impl Drop for RunLease {
    fn drop(&mut self) {
        self.0.store(false, Ordering::Release);
    }
}

type WorkerInbox = Receiver<Result<WorkerFrame, String>>;
type WorkerEvents = mpsc::SyncSender<Result<WorkerFrame, String>>;
impl WorkerProcess {
    fn attach(
        mut child: Child,
        deadline: Instant,
        stop: ScriptStop,
    ) -> Result<(Self, WorkerInbox, WorkerEvents), String> {
        let (events, incoming) = mpsc::sync_channel(4);
        let input = Arc::new(Mutex::new(
            child
                .stdin
                .take()
                .ok_or("worker stdin: expected piped input")?,
        ));
        let readers = readers::start(&mut child, events.clone())?;
        let worker = Self::new(child, input, readers, deadline, stop);
        Ok((worker, incoming, events))
    }
    fn new(
        child: Child,
        input: Arc<Mutex<ChildStdin>>,
        readers: Vec<JoinHandle<()>>,
        deadline: Instant,
        stop: ScriptStop,
    ) -> Self {
        let child = Arc::new(Mutex::new(child));
        let finished = Arc::new(AtomicBool::new(false));
        let monitor = watch::start(child.clone(), finished.clone(), stop, deadline);
        Self {
            child,
            input,
            readers,
            monitor: Some(monitor),
            finished,
        }
    }
}
