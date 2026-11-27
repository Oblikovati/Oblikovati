// SPDX-License-Identifier: GPL-2.0-only
//! Wait on all native targets without draining input from any of them.
use super::{NativeWindow, ViewTarget, Viewport};
use crate::{ViewportReady, ViewportWake};
use std::{rc::Rc, time::Duration};

impl Viewport {
    /// Wait without consuming native events. Example: `view.wait_events(&wake, None)?`.
    pub fn wait_events(
        &self,
        wake: &ViewportWake,
        timeout: Option<Duration>,
    ) -> Result<ViewportReady, String> {
        self.wait_events_for_windows(&[], wake, timeout)
    }

    /// Wait for any window in this runtime. Example: `main.wait_events_for_windows(&[&drawing], &wake, None)?`.
    pub fn wait_events_for_windows(
        &self,
        others: &[&Self],
        wake: &ViewportWake,
        timeout: Option<Duration>,
    ) -> Result<ViewportReady, String> {
        self.wait_window_sources(
            others,
            wake,
            timeout,
            #[cfg(target_os = "linux")]
            None,
        )
    }

    /// Wait for one window or clipboard event. Example: `main.wait_events_with_clipboard(&wake, &clipboard, None)?`.
    #[cfg(target_os = "linux")]
    pub fn wait_events_with_clipboard(
        &self,
        wake: &ViewportWake,
        clipboard: &crate::NativeClipboardEndpoint,
        timeout: Option<Duration>,
    ) -> Result<ViewportReady, String> {
        self.wait_events_for_windows_with_clipboard(&[], wake, clipboard, timeout)
    }

    /// Wait for any window, clipboard transfer, or worker signal.
    /// Example: `main.wait_events_for_windows_with_clipboard(&[&drawing], &wake, &clipboard, None)?`.
    #[cfg(target_os = "linux")]
    pub fn wait_events_for_windows_with_clipboard(
        &self,
        others: &[&Self],
        wake: &ViewportWake,
        clipboard: &crate::NativeClipboardEndpoint,
        timeout: Option<Duration>,
    ) -> Result<ViewportReady, String> {
        self.wait_window_sources(others, wake, timeout, Some(clipboard))
    }

    fn event_window(&self) -> Result<&NativeWindow, String> {
        self.live()?;
        let Some(ViewTarget::Native(window)) = &self.target else {
            return Err(format!(
                "viewport {}: expected a native event source",
                self.id
            ));
        };
        Ok(window)
    }

    fn wait_window_sources(
        &self,
        others: &[&Self],
        wake: &ViewportWake,
        timeout: Option<Duration>,
        #[cfg(target_os = "linux")] clipboard: Option<&crate::NativeClipboardEndpoint>,
    ) -> Result<ViewportReady, String> {
        let main = self.event_window()?;
        let additional = others
            .iter()
            .map(|view| self.related_event_window(view))
            .collect::<Result<Vec<_>, _>>()?;
        wait_for_native_sources(
            main,
            &additional,
            wake,
            timeout,
            #[cfg(target_os = "linux")]
            clipboard,
        )
    }

    fn related_event_window<'a>(&self, other: &'a Self) -> Result<&'a NativeWindow, String> {
        if !Rc::ptr_eq(&self._runtime, &other._runtime) {
            return Err(format!(
                "viewport {}: expected the runtime of viewport {}",
                other.id, self.id
            ));
        }
        other.event_window()
    }
}

#[cfg(target_os = "linux")]
pub(crate) fn wait_for_native_sources<W: crate::linux::LinuxEventSource>(
    main: &W,
    additional: &[&W],
    wake: &ViewportWake,
    timeout: Option<Duration>,
    clipboard: Option<&crate::NativeClipboardEndpoint>,
) -> Result<ViewportReady, String> {
    let windows = std::iter::once(main).chain(additional.iter().copied());
    for window in windows.clone() {
        if !window.pending()? {
            continue;
        }
        return Ok(ViewportReady {
            native_events: true,
            worker_wake: wake.consume()?,
        });
    }
    let timeout = windows
        .clone()
        .filter_map(W::next_input_timeout)
        .chain(timeout)
        .min();
    let sources = windows.map(W::descriptor).collect::<Result<Vec<_>, _>>()?;
    match clipboard {
        Some(clipboard) => clipboard.wait_with_sources(wake, &sources, timeout),
        None => wake.wait_sources(&sources, timeout),
    }
}

#[cfg(target_arch = "wasm32")]
fn wait_for_native_sources(
    main: &NativeWindow,
    additional: &[&NativeWindow],
    wake: &ViewportWake,
    _timeout: Option<Duration>,
) -> Result<ViewportReady, String> {
    let native_events =
        main.has_pending_input() || additional.iter().any(|window| window.has_pending_input());
    Ok(ViewportReady {
        native_events,
        worker_wake: wake.consume()?,
    })
}

#[cfg(target_os = "windows")]
fn wait_for_native_sources(
    main: &NativeWindow,
    additional: &[&NativeWindow],
    wake: &ViewportWake,
    timeout: Option<Duration>,
) -> Result<ViewportReady, String> {
    // Thread messages wake every Win32 window, but earlier dispatch can leave a per-window queue.
    if additional.iter().any(|window| window.has_pending_input()) {
        let mut ready = wake.wait(Some(Duration::ZERO))?;
        ready.native_events = true;
        return Ok(ready);
    }
    main.wait_events(wake, timeout)
}
