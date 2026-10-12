// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{contract::AddInRegistry, types::LOAD_ON_STARTUP};
use std::{collections::VecDeque, path::PathBuf};
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum StartupPhase {
    Loading,
    Activating,
    Ready,
}
pub(super) struct NativeStartupPlan {
    remaining: VecDeque<PathBuf>,
    current: PathBuf,
    phase: StartupPhase,
    pub notices: VecDeque<crate::DesktopCommandNotice>,
}
impl NativeStartupPlan {
    pub fn new(paths: Vec<PathBuf>) -> Self {
        Self {
            remaining: paths.into(),
            current: PathBuf::new(),
            phase: StartupPhase::Ready,
            notices: VecDeque::new(),
        }
    }
    pub fn ready(&self) -> bool {
        self.phase == StartupPhase::Ready
    }
    pub fn next<W: DispatchWakeup + 'static>(
        &mut self,
        runtime: &mut NativeAddInRuntime<W>,
    ) -> Result<(), String> {
        let Some(path) = self.remaining.pop_front() else {
            self.phase = StartupPhase::Ready;
            return Ok(());
        };
        self.current = path;
        self.phase = StartupPhase::Loading;
        unsafe { runtime.start_load(&self.current) }.map_err(|cause| cause.to_string())
    }
    pub fn completed<W: DispatchWakeup + 'static>(
        &mut self,
        runtime: &mut NativeAddInRuntime<W>,
        outcome: Result<NativeCompletion, String>,
    ) -> Result<(), String> {
        let completion = match outcome {
            Ok(completion) => completion,
            Err(cause) => {
                self.notices
                    .push_back(crate::DesktopCommandNotice::AddInStartupFailed {
                        library: self.current.clone(),
                        cause,
                    });
                return self.next(runtime);
            }
        };
        match (self.phase, completion) {
            (StartupPhase::Loading, NativeCompletion::Loaded(id)) => self.loaded(runtime, id),
            (StartupPhase::Activating, NativeCompletion::Finished) => self.next(runtime),
            (phase, completion) => Err(format!(
                "native startup {phase:?}, {completion:?}: expected load then activation"
            )),
        }
    }
    fn loaded<W: DispatchWakeup + 'static>(
        &mut self,
        runtime: &mut NativeAddInRuntime<W>,
        id: String,
    ) -> Result<(), String> {
        if runtime.load_behavior(id.clone()) != LOAD_ON_STARTUP {
            return self.next(runtime);
        }
        self.phase = StartupPhase::Activating;
        unsafe { runtime.start_activate(&id) }.map_err(|cause| cause.to_string())
    }
}
