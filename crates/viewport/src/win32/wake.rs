// SPDX-License-Identifier: GPL-2.0-only
use std::{
    os::windows::io::{AsRawHandle, FromRawHandle, OwnedHandle},
    sync::Arc,
    time::Duration,
};
use windows_sys::Win32::{
    Foundation::{WAIT_FAILED, WAIT_OBJECT_0, WAIT_TIMEOUT},
    System::Threading::{CreateEventW, INFINITE, SetEvent},
    UI::WindowsAndMessaging::{MWMO_INPUTAVAILABLE, MsgWaitForMultipleObjectsEx, QS_ALLINPUT},
};

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct ViewportReady {
    pub native_events: bool,
    pub worker_wake: bool,
}

/// A coalesced, nonblocking signal owned independently of the desktop window.
#[derive(Clone)]
pub struct ViewportWake(Arc<OwnedHandle>);

impl ViewportWake {
    /// Create an initially empty worker signal with `ViewportWake::new()?`.
    pub fn new() -> Result<Self, String> {
        // The auto-reset event belongs to the returned OwnedHandle.
        let raw = unsafe { CreateEventW(std::ptr::null(), 0, 0, std::ptr::null()) };
        if raw.is_null() {
            return Err(format!(
                "viewport wake event: {}",
                std::io::Error::last_os_error()
            ));
        }
        Ok(Self(Arc::new(unsafe { OwnedHandle::from_raw_handle(raw) })))
    }

    /// Wake the owner thread with `wake.wake()?`.
    pub fn wake(&self) -> Result<(), String> {
        if unsafe { SetEvent(self.0.as_raw_handle()) } == 0 {
            return Err(format!(
                "viewport wake signal: {}",
                std::io::Error::last_os_error()
            ));
        }
        Ok(())
    }

    pub(crate) fn wait(&self, timeout: Option<Duration>) -> Result<ViewportReady, String> {
        let millis = timeout.map_or(INFINITE, |duration| {
            duration
                .as_nanos()
                .div_ceil(1_000_000)
                .min(u128::from(INFINITE - 1)) as u32
        });
        let handle = self.0.as_raw_handle();
        let status = unsafe {
            MsgWaitForMultipleObjectsEx(1, &handle, millis, QS_ALLINPUT, MWMO_INPUTAVAILABLE)
        };
        match status {
            WAIT_OBJECT_0 => Ok(ViewportReady {
                worker_wake: true,
                native_events: false,
            }),
            s if s == WAIT_OBJECT_0 + 1 => Ok(ViewportReady {
                worker_wake: false,
                native_events: true,
            }),
            WAIT_TIMEOUT => Ok(ViewportReady::default()),
            WAIT_FAILED => Err(format!(
                "viewport event wait: {}",
                std::io::Error::last_os_error()
            )),
            _ => Err(format!(
                "viewport event wait {status}: expected event, input or timeout"
            )),
        }
    }
}
