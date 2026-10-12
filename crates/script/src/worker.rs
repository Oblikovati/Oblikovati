// SPDX-License-Identifier: GPL-2.0-only
use crate::protocol::{ParentFrame, WorkerFrame, read_frame, write_frame};
use std::io::{self, BufReader};
/// Serve exactly one isolated worker request; example: call from the private executable entry.
/// This entry must never execute in the document-owning parent process.
pub fn worker_stdio() -> Result<(), String> {
    let mut incoming = BufReader::new(io::stdin());
    let Some(ParentFrame::Start { program }) = read_frame(&mut incoming)? else {
        return Err("worker startup: expected one Start frame".into());
    };
    program.limits.validate(&program.source)?;
    execute_program(program, incoming)
}
#[cfg(target_os = "linux")]
fn execute_program(
    program: crate::ScriptProgram,
    incoming: BufReader<io::Stdin>,
) -> Result<(), String> {
    resource_limits(&program.limits)?;
    let outcome = crate::vm::execute(&program, incoming, io::stdout());
    write_frame(&mut io::stdout(), &WorkerFrame::Finished { outcome })
}
#[cfg(not(target_os = "linux"))]
fn execute_program(_: crate::ScriptProgram, _: BufReader<io::Stdin>) -> Result<(), String> {
    Err("Lua worker platform: expected Linux; other platforms are deferred".into())
}
#[cfg(target_os = "linux")]
fn resource_limits(limits: &crate::ScriptLimits) -> Result<(), String> {
    use rustix::process::Resource;
    // VM scratch and loaded process code also need a bounded address-space allowance outside the Lua heap.
    let address = limits
        .memory_bytes
        .checked_add(128 << 20)
        .ok_or("script memory: expected representable process allowance")?;
    let cpu = limits.wall_millis.div_ceil(1000).saturating_add(1);
    for (resource, maximum) in [
        (Resource::Core, 0),
        (Resource::As, address),
        (Resource::Stack, 8 << 20),
        (Resource::Cpu, cpu),
    ] {
        apply_resource_limit(resource, maximum)?;
    }
    Ok(())
}
#[cfg(target_os = "linux")]
fn apply_resource_limit(resource: rustix::process::Resource, maximum: u64) -> Result<(), String> {
    rustix::process::setrlimit(resource, rustix::process::Rlimit { current: Some(maximum), maximum: Some(maximum) })
        .map_err(|error| format!("worker resource {resource:?} limit {maximum}: expected applicable bounded limit: {error}"))
}
