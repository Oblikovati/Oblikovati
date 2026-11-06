// SPDX-License-Identifier: GPL-2.0-only
//! The only application module that imports the selected Lua VM.
use crate::{
    ScriptProgram, ScriptReport,
    protocol::{ParentFrame, WorkerFrame, read_frame, write_frame},
};
use mlua::{Function, Lua, LuaOptions, StdLib, Table, Value};
use std::{
    cell::{Cell, RefCell},
    io::{BufReader, Stdin, Stdout},
    rc::Rc,
};
mod bindings;
mod convert;
mod quota;
struct LuaHostPipe {
    incoming: BufReader<Stdin>,
    outgoing: Stdout,
    report: ScriptReport,
    limits: crate::ScriptLimits,
}
impl LuaHostPipe {
    fn call(&mut self, method: String, arguments: Vec<u8>) -> mlua::Result<Vec<u8>> {
        self.report.host_calls += 1;
        if self.report.host_calls > self.limits.host_calls {
            return Err(failure(format!(
                "host calls {}: expected at most {}",
                self.report.host_calls, self.limits.host_calls
            )));
        }
        write_frame(&mut self.outgoing, &WorkerFrame::Call { method, arguments })
            .map_err(failure)?;
        match read_frame(&mut self.incoming).map_err(failure)? {
            Some(ParentFrame::Reply { outcome }) => outcome.map_err(failure),
            _ => Err(failure("host reply: expected one Reply frame")),
        }
    }
    fn print(&mut self, line: String) -> mlua::Result<()> {
        self.report.output_bytes = self.report.output_bytes.saturating_add(line.len() + 1);
        if self.report.output_bytes > self.limits.output_bytes {
            return Err(failure(format!(
                "script output {} bytes: expected at most {}",
                self.report.output_bytes, self.limits.output_bytes
            )));
        }
        write_frame(&mut self.outgoing, &WorkerFrame::Print { line }).map_err(failure)
    }
}
pub(super) fn execute(
    program: &ScriptProgram,
    incoming: BufReader<Stdin>,
    outgoing: Stdout,
) -> Result<ScriptReport, String> {
    let lua = sandbox(&program.limits).map_err(|error| error.to_string())?;
    let instructions = quota::install(&lua, &program.limits).map_err(|error| error.to_string())?;
    let pipe = Rc::new(RefCell::new(LuaHostPipe {
        incoming,
        outgoing,
        report: ScriptReport::default(),
        limits: program.limits.clone(),
    }));
    if !program.parse_only {
        bindings::install(&lua, pipe.clone(), &program.methods)
            .map_err(|error| error.to_string())?;
    }
    run_source(&lua, program).map_err(|error| error.to_string())?;
    let mut report = pipe.borrow().report.clone();
    report.instructions = instructions.get();
    Ok(report)
}
fn run_source(lua: &Lua, program: &ScriptProgram) -> mlua::Result<()> {
    let chunk = lua
        .load(&program.source)
        .set_name("script")
        .set_mode(mlua::chunk::ChunkMode::Text);
    if program.parse_only {
        return chunk.into_function().map(|_| ());
    }
    chunk.exec()
}
fn sandbox(limits: &crate::ScriptLimits) -> mlua::Result<Lua> {
    let lua = Lua::new_with(
        StdLib::TABLE | StdLib::STRING | StdLib::MATH,
        LuaOptions::default(),
    )?;
    lua.set_memory_limit(usize::try_from(limits.memory_bytes).map_err(failure)?)?;
    strip_globals(&lua)?;
    Ok(lua)
}
fn strip_globals(lua: &Lua) -> mlua::Result<()> {
    for name in [
        "load",
        "loadstring",
        "loadfile",
        "dofile",
        "require",
        "collectgarbage",
        "print",
        "module",
        "newproxy",
    ] {
        lua.globals().set(name, Value::Nil)?;
    }
    Ok(())
}

fn failure(message: impl ToString) -> mlua::Error {
    mlua::Error::RuntimeError(message.to_string())
}
