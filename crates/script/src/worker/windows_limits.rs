// SPDX-License-Identifier: GPL-2.0-only
//! Admit the private worker to a Windows job before constructing the Lua VM.
use crate::ScriptLimits;
use std::{
    os::windows::io::{AsRawHandle, FromRawHandle, OwnedHandle},
    ptr,
};
use windows_sys::Win32::System::{
    JobObjects::{
        AssignProcessToJobObject, CreateJobObjectW, JOB_OBJECT_LIMIT_ACTIVE_PROCESS,
        JOB_OBJECT_LIMIT_PROCESS_MEMORY, JOB_OBJECT_LIMIT_PROCESS_TIME,
        JOBOBJECT_EXTENDED_LIMIT_INFORMATION, JobObjectExtendedLimitInformation,
        SetInformationJobObject,
    },
    Threading::GetCurrentProcess,
};

pub(super) fn contain_worker(limits: &ScriptLimits) -> Result<OwnedHandle, String> {
    let policy = job_policy(limits)?;
    // The unnamed job handle is owned only by this worker and is not inherited.
    let raw = unsafe { CreateJobObjectW(ptr::null(), ptr::null()) };
    if raw.is_null() {
        return Err(job_error("create"));
    }
    let job = unsafe { OwnedHandle::from_raw_handle(raw) };
    apply_policy(&job, &policy)?;
    // The private worker is the sole member. Its parent remains outside this job.
    if unsafe { AssignProcessToJobObject(job.as_raw_handle(), GetCurrentProcess()) } == 0 {
        return Err(job_error("assign"));
    }
    Ok(job)
}

fn job_policy(limits: &ScriptLimits) -> Result<JOBOBJECT_EXTENDED_LIMIT_INFORMATION, String> {
    let memory = limits
        .memory_bytes
        .checked_add(128 << 20)
        .and_then(|bytes| usize::try_from(bytes).ok())
        .ok_or("script memory: expected representable process allowance")?;
    // Job CPU time uses 100 ns units. The parent separately enforces the wall deadline.
    let cpu = limits
        .wall_millis
        .div_ceil(1000)
        .checked_add(1)
        .and_then(|seconds| seconds.checked_mul(10_000_000))
        .and_then(|ticks| i64::try_from(ticks).ok())
        .ok_or("script CPU time: expected representable process allowance")?;
    let mut policy = JOBOBJECT_EXTENDED_LIMIT_INFORMATION::default();
    let basic = &mut policy.BasicLimitInformation;
    basic.LimitFlags = JOB_OBJECT_LIMIT_ACTIVE_PROCESS
        | JOB_OBJECT_LIMIT_PROCESS_MEMORY
        | JOB_OBJECT_LIMIT_PROCESS_TIME;
    basic.ActiveProcessLimit = 1;
    basic.PerProcessUserTimeLimit = cpu;
    policy.ProcessMemoryLimit = memory;
    Ok(policy)
}

fn apply_policy(
    job: &OwnedHandle,
    policy: &JOBOBJECT_EXTENDED_LIMIT_INFORMATION,
) -> Result<(), String> {
    // The information class determines the matching structure and byte count.
    let applied = unsafe {
        SetInformationJobObject(
            job.as_raw_handle(),
            JobObjectExtendedLimitInformation,
            ptr::from_ref(policy).cast(),
            size_of_val(policy) as u32,
        )
    };
    if applied == 0 {
        return Err(job_error("configure"));
    }
    Ok(())
}

fn job_error(operation: &str) -> String {
    format!(
        "script worker Windows job {operation}: {}",
        std::io::Error::last_os_error()
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn job_caps_worker_memory_cpu_and_process_count() {
        let limits = ScriptLimits::gui();
        let policy = job_policy(&limits).unwrap();
        assert_eq!(policy.ProcessMemoryLimit, 256 << 20);
        assert_eq!(
            policy.BasicLimitInformation.PerProcessUserTimeLimit,
            60_000_000
        );
        assert_eq!(policy.BasicLimitInformation.ActiveProcessLimit, 1);
        assert!(
            job_policy(&ScriptLimits {
                memory_bytes: u64::MAX,
                ..limits.clone()
            })
            .is_err()
        );
        assert!(
            job_policy(&ScriptLimits {
                wall_millis: u64::MAX,
                ..limits
            })
            .is_err()
        );
    }
}
