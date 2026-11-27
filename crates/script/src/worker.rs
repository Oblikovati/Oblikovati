// SPDX-License-Identifier: GPL-2.0-only
use crate::protocol::{ParentFrame, read_frame};
#[cfg(any(target_os = "linux", target_os = "windows"))]
use crate::protocol::{WorkerFrame, write_frame};
use std::io::{self, BufReader};
#[cfg(target_os = "windows")]
#[allow(unsafe_code)]
mod windows_limits;
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
#[cfg(any(target_os = "linux", target_os = "windows"))]
fn execute_program(
    program: crate::ScriptProgram,
    incoming: BufReader<io::Stdin>,
) -> Result<(), String> {
    #[cfg(target_os = "linux")]
    let containment = resource_limits(&program.limits);
    #[cfg(target_os = "windows")]
    let containment = windows_limits::contain_worker(&program.limits);
    // Keep the worker admitted until the terminal frame is written. Admission
    // errors use the protocol because the parent only drains worker stderr.
    let outcome = match &containment {
        Ok(_) => crate::vm::execute(&program, incoming, io::stdout()),
        Err(error) => Err(error.clone()),
    };
    write_frame(&mut io::stdout(), &WorkerFrame::Finished { outcome })
}

#[cfg(not(any(target_os = "linux", target_os = "windows")))]
fn execute_program(_: crate::ScriptProgram, _: BufReader<io::Stdin>) -> Result<(), String> {
    Err("Lua worker platform: expected Linux or Windows".into())
}
#[cfg(target_os = "linux")]
fn resource_limits(limits: &crate::ScriptLimits) -> Result<(), String> {
    use rustix::process::Resource;
    let address = address_space_limit(mapped_address_space_bytes(), limits)?;
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
fn address_space_limit(mapped_bytes: u64, limits: &crate::ScriptLimits) -> Result<u64, String> {
    // VM scratch and already-mapped process code/libraries sit outside the Lua heap allowance.
    mapped_bytes
        .checked_add(128 << 20)
        .and_then(|base| base.checked_add(limits.memory_bytes))
        .ok_or_else(|| "script memory: expected representable process allowance".into())
}
#[cfg(target_os = "linux")]
fn mapped_address_space_bytes() -> u64 {
    let Ok(status) = std::fs::read_to_string("/proc/self/status") else {
        return 0;
    };
    status
        .lines()
        .find_map(|line| {
            let value = line.strip_prefix("VmSize:")?.trim();
            let kib = value.strip_suffix("kB")?.trim().parse::<u64>().ok()?;
            kib.checked_mul(1024)
        })
        .unwrap_or(0)
}
#[cfg(target_os = "linux")]
fn apply_resource_limit(resource: rustix::process::Resource, maximum: u64) -> Result<(), String> {
    rustix::process::setrlimit(
        resource,
        rustix::process::Rlimit {
            current: Some(maximum),
            maximum: Some(maximum),
        },
    )
    .map_err(|error| {
        format!("worker resource {resource:?} limit {maximum}: expected applicable bounded limit: {error}")
    })
}

#[cfg(all(test, target_os = "linux"))]
mod tests {
    use super::*;

    #[test]
    fn address_space_limit_preserves_mapped_image_and_refuses_overflow() {
        let limits = crate::ScriptLimits::gui();
        let mapped = mapped_address_space_bytes();
        assert!(mapped > 0, "expected Linux /proc/self/status VmSize");
        assert_eq!(
            address_space_limit(mapped, &limits).unwrap(),
            mapped + (128 << 20) + limits.memory_bytes
        );
        assert!(
            address_space_limit(
                mapped,
                &crate::ScriptLimits {
                    memory_bytes: u64::MAX,
                    ..limits
                }
            )
            .is_err()
        );
    }
}
