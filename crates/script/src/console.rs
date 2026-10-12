// SPDX-License-Identifier: GPL-2.0-only
//! Owner-thread Script Console state over isolated execution; it captures no model.
use crate::{IsolatedLuaEngine, ScriptExecution, ScriptLimits, ScriptProgram, ScriptReport};
use oblikovati_api::client::Caller;
use std::{
    cell::Cell,
    marker::PhantomData,
    rc::Rc,
    time::{Duration, Instant},
};
mod activity;
mod output;
use activity::ActivityCaller;
pub use activity::ConsolePoll;
use output::ConsoleOutput;
/// Check compiles source only; Run executes its declared host capabilities.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ConsoleMode {
    Run,
    Check,
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ConsoleOutcome {
    pub result: Result<ScriptReport, String>,
    pub elapsed: Duration,
}
/// Copied state for one UI frame; changing it cannot mutate the live console.
#[derive(Clone, Debug)]
pub struct ConsoleSnapshot {
    pub output: Vec<String>,
    pub running: bool,
    pub checking: bool,
    pub last_run: Option<ConsoleOutcome>,
    pub last_check: Option<ConsoleOutcome>,
    pub dropped_output: u64,
}
struct ConsoleJob {
    execution: ScriptExecution,
    mode: ConsoleMode,
    started: Instant,
}
/// One owner's console controller. The non-Send marker keeps frame/model dispatch on that owner.
/// ```compile_fail
/// use oblikovati_script::console::ScriptConsoleController;
/// fn move_controller(controller: ScriptConsoleController) {
///     std::thread::spawn(move || drop(controller));
/// }
/// ```
pub struct ScriptConsoleController {
    engine: IsolatedLuaEngine,
    limits: ScriptLimits,
    methods: Vec<String>,
    job: Option<ConsoleJob>,
    output: ConsoleOutput,
    last_run: Option<ConsoleOutcome>,
    last_check: Option<ConsoleOutcome>,
    owner: PhantomData<Rc<()>>,
    changed: Cell<bool>,
}
impl ScriptConsoleController {
    /// Inject the selected worker and host inventory; example: `ScriptConsoleController::new(engine, limits, methods)`.
    pub fn new(engine: IsolatedLuaEngine, limits: ScriptLimits, mut methods: Vec<String>) -> Self {
        methods.sort();
        methods.dedup();
        Self {
            engine,
            limits,
            methods,
            job: None,
            output: ConsoleOutput::default(),
            last_run: None,
            last_check: None,
            owner: PhantomData,
            changed: Cell::new(false),
        }
    }
    /// Begin a run without executing Lua on the owner; example: `console.run(source)?;`.
    pub fn run(&mut self, source: &str) -> Result<(), String> {
        self.start(source, ConsoleMode::Run)
    }
    /// Compile only, sharing the runtime grammar and bounds; example: `console.check(source)?;`.
    pub fn check(&mut self, source: &str) -> Result<(), String> {
        self.start(source, ConsoleMode::Check)
    }
    fn start(&mut self, source: &str, mode: ConsoleMode) -> Result<(), String> {
        if self.job.is_some() {
            return Err(format!(
                "console {mode:?}: expected idle; another run or check is active"
            ));
        }
        let program = self.program(source, mode);
        let execution = self.engine.start(program)?;
        self.changed.set(true);
        self.job = Some(ConsoleJob {
            execution,
            mode,
            started: Instant::now(),
        });
        Ok(())
    }
    fn program(&self, source: &str, mode: ConsoleMode) -> ScriptProgram {
        ScriptProgram {
            source: source.into(),
            methods: self.methods.clone(),
            limits: self.limits.clone(),
            parse_only: mode == ConsoleMode::Check,
        }
    }
    /// Process at most one stream event per native frame; example: `console.poll(&host)`.
    /// Returns the completed mode once; ordinary run/check errors are recorded in the copied snapshot.
    pub fn poll<C: Caller>(&mut self, host: &C) -> Option<ConsoleMode> {
        self.poll_activity(host).completed
    }
    /// Process one event and report UI invalidation without copying retained output.
    /// Example: `console.poll_activity(&host)` lets an idle native loop avoid unnecessary redraws.
    pub fn poll_activity<C: Caller>(&mut self, host: &C) -> ConsolePoll {
        let caller = ActivityCaller {
            host,
            called: Cell::new(false),
        };
        let completed = self.poll_job(&caller);
        let output_changed = self.output.take_changed();
        let changed = self.changed.replace(false)
            || caller.called.get()
            || output_changed
            || completed.is_some();
        ConsolePoll {
            changed,
            active: self.is_active(),
            completed,
        }
    }
    fn poll_job<C: Caller>(&mut self, host: &C) -> Option<ConsoleMode> {
        let job = self.job.as_mut()?;
        let outcome = match job.execution.poll(host, &mut self.output) {
            Ok(None) => return None,
            Ok(Some(report)) => Ok(report),
            Err(error) => Err(error),
        };
        self.finish(outcome)
    }
    fn finish(&mut self, result: Result<ScriptReport, String>) -> Option<ConsoleMode> {
        let job = self.job.take()?;
        let outcome = ConsoleOutcome {
            result,
            elapsed: job.started.elapsed(),
        };
        match job.mode {
            ConsoleMode::Run => self.last_run = Some(outcome),
            ConsoleMode::Check => self.last_check = Some(outcome),
        }
        Some(job.mode)
    }
    /// Stop an active run/check; idle Stop is harmless. Example: `console.stop()`.
    pub fn stop(&self) {
        if let Some(job) = &self.job {
            self.changed.set(true);
            job.execution.stop_handle().cancel();
        }
    }
    /// Inspect execution without cloning the console snapshot; example: `console.is_active()` before close.
    pub fn is_active(&self) -> bool {
        self.job.is_some()
    }
    /// Cancel and retire the isolated child before returning; committed edits remain owned by the parent.
    /// Example: `console.shutdown()` before retiring the native window.
    pub fn shutdown(&mut self) {
        if self.job.is_none() {
            return;
        }
        self.stop();
        self.finish(Err(
            "console shutdown: execution cancelled; committed edits are retained".into(),
        ));
    }
    /// Clear retained output while preserving execution and outcome state; example: `console.clear_output()`.
    pub fn clear_output(&mut self) {
        self.output.clear();
    }
    /// Copy one immutable frame view; example: `let view = console.snapshot();`.
    pub fn snapshot(&self) -> ConsoleSnapshot {
        ConsoleSnapshot {
            output: self.output.lines(),
            running: self
                .job
                .as_ref()
                .is_some_and(|job| job.mode == ConsoleMode::Run),
            checking: self
                .job
                .as_ref()
                .is_some_and(|job| job.mode == ConsoleMode::Check),
            last_run: self.last_run.clone(),
            last_check: self.last_check.clone(),
            dropped_output: self.output.dropped(),
        }
    }
}
