// SPDX-License-Identifier: GPL-2.0-only
//! Bounded Lua 5.1 workers; documents and host capabilities stay in the parent.
mod limits;
mod process;
mod protocol;
#[cfg(target_os = "linux")]
mod vm;
mod worker;
pub use limits::{ScriptLimits, ScriptProgram, ScriptReport};
pub use process::{IsolatedLuaEngine, ScriptExecution, ScriptStop};
pub use worker::worker_stdio;
/// Private executable entry; this is not part of the Lua or public wire contract.
pub const WORKER_ARGUMENT: &str = "--internal-lua-worker";

pub mod console;

pub mod editor;

pub mod syntax;
mod vocabulary;
