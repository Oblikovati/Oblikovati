// SPDX-License-Identifier: GPL-2.0-only
use rustix::{
    event::{EventfdFlags, PollFd, PollFlags, Timespec, eventfd, poll},
    fd::{BorrowedFd, OwnedFd},
    io::{Errno, read, write},
};
use std::{sync::Arc, time::Duration};
/// A copied readiness result; both sources can be ready together.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct ViewportReady {
    pub native_events: bool,
    pub worker_wake: bool,
}
/// A coalesced, nonblocking worker signal. Clones contain no window, renderer, or model.
#[derive(Clone)]
pub struct ViewportWake {
    descriptor: Arc<OwnedFd>,
}
impl ViewportWake {
    /// Create the owner loop's wake signal; example: `let wake = ViewportWake::new()?;`.
    pub fn new() -> Result<Self, String> {
        let descriptor = eventfd(0, EventfdFlags::NONBLOCK | EventfdFlags::CLOEXEC)
            .map_err(|error| format!("viewport wake: expected an event descriptor: {error}"))?;
        Ok(Self {
            descriptor: Arc::new(descriptor),
        })
    }
    /// Wake the owner after work is admitted; example: `worker_wake.wake()?;`.
    pub fn wake(&self) -> Result<(), String> {
        match write(&*self.descriptor, &1_u64.to_ne_bytes()) {
            Ok(8) | Err(Errno::AGAIN) => Ok(()),
            outcome => Err(format!(
                "viewport wake {outcome:?}: expected one queued event counter"
            )),
        }
    }
    pub(crate) fn consume(&self) -> Result<bool, String> {
        let mut counter = [0_u8; 8];
        match read(&*self.descriptor, &mut counter) {
            Ok(8) => Ok(true),
            Err(Errno::AGAIN) => Ok(false),
            outcome => Err(format!(
                "viewport wake read {outcome:?}: expected a counter or empty signal"
            )),
        }
    }
    pub(crate) fn wait(
        &self,
        native: BorrowedFd<'_>,
        timeout: Option<Duration>,
    ) -> Result<ViewportReady, String> {
        let timeout = wait_timeout(timeout)?;
        let mut descriptors = [
            PollFd::new(&native, PollFlags::IN),
            PollFd::new(&*self.descriptor, PollFlags::IN),
        ];
        if !poll_once(&mut descriptors, timeout.as_ref())? {
            return Ok(ViewportReady::default());
        }
        check_descriptors(&descriptors)?;
        Ok(ViewportReady {
            native_events: descriptors[0].revents().contains(PollFlags::IN),
            worker_wake: self.consume()?,
        })
    }
}
fn wait_timeout(timeout: Option<Duration>) -> Result<Option<Timespec>, String> {
    timeout
        .map(Timespec::try_from)
        .transpose()
        .map_err(|error| {
            format!("viewport wait {timeout:?}: expected representable duration: {error}")
        })
}
fn poll_once(descriptors: &mut [PollFd<'_>], timeout: Option<&Timespec>) -> Result<bool, String> {
    match poll(descriptors, timeout) {
        Ok(_) => Ok(true),
        Err(Errno::INTR) => Ok(false),
        Err(error) => Err(format!(
            "viewport event poll: expected live descriptors: {error}"
        )),
    }
}
fn check_descriptors(descriptors: &[PollFd<'_>]) -> Result<(), String> {
    for (index, descriptor) in descriptors.iter().enumerate() {
        let flags = descriptor.revents();
        if flags.intersects(PollFlags::ERR | PollFlags::HUP | PollFlags::NVAL) {
            return Err(format!(
                "viewport wait descriptor {index} {flags:?}: expected a live event source"
            ));
        }
    }
    Ok(())
}
#[cfg(test)]
#[path = "../tests/wake.rs"]
mod tests;
