// SPDX-License-Identifier: GPL-2.0-only
//! Select one Linux native window backend per desktop runtime.
use crate::{ViewportEvent, ViewportWindowAction, wayland, x11};
use martensite_tkernel::standard::Handle;
use martensite_tkservice::aspect::AspectWindow;
use std::{os::fd::BorrowedFd, rc::Rc, time::Duration};
use wayland_client::Connection;

pub(crate) enum NativeLinux {
    Wayland(Connection),
    X11(Rc<x11::NativeX11>),
}

impl NativeLinux {
    pub(crate) fn open() -> Result<Self, String> {
        let session_advertises_wayland = ["WAYLAND_DISPLAY", "WAYLAND_SOCKET"]
            .into_iter()
            .any(|name| std::env::var_os(name).is_some_and(|value| !value.is_empty()));
        if session_advertises_wayland && let Ok(connection) = Connection::connect_to_env() {
            return Ok(Self::Wayland(connection));
        }
        x11::NativeX11::open().map(Self::X11)
    }

    pub(crate) fn create(
        &self,
        title: &str,
        extent: [u32; 2],
        system_frame: bool,
    ) -> Result<NativeWindow, String> {
        match self {
            Self::Wayland(connection) => wayland::NativeWaylandWindow::create(
                connection.clone(),
                title,
                extent,
                system_frame,
            )
            .map(Box::new)
            .map(NativeWindow::Wayland),
            Self::X11(x11) => x11::NativeWindow::create(x11.clone(), title, extent, system_frame)
                .map(Box::new)
                .map(NativeWindow::X11),
        }
    }
}

pub(crate) enum NativeWindow {
    Wayland(Box<wayland::NativeWaylandWindow>),
    X11(Box<x11::NativeWindow>),
}

macro_rules! window_method {
    ($name:ident($($arg:ident: $kind:ty),*) -> $result:ty) => {
        pub(crate) fn $name(&self, $($arg: $kind),*) -> $result {
            match self {
                Self::Wayland(window) => window.$name($($arg),*),
                Self::X11(window) => window.$name($($arg),*),
            }
        }
    };
}

impl NativeWindow {
    window_method!(backing_scale() -> f32);
    window_method!(is_mapped() -> bool);
    window_method!(mapped(mapped: bool) -> Result<(), String>);
    window_method!(poll(limit: usize) -> Result<Vec<ViewportEvent>, String>);
    window_method!(resize(extent: [u32; 2]) -> Result<(), String>);
    window_method!(request_accessible_focus() -> Result<(), String>);
    window_method!(set_title(title: &str) -> Result<(), String>);
    window_method!(text_input_issue() -> Option<String>);
    window_method!(title() -> Result<String, String>);
    window_method!(window_action(action: ViewportWindowAction) -> Result<(), String>);
    window_method!(window_maximized() -> Result<bool, String>);
    window_method!(window_screen_bounds() -> Result<[i32; 4], String>);

    pub(crate) fn window(&self) -> Handle<dyn AspectWindow> {
        match self {
            Self::Wayland(native) => native.window.cast(),
            Self::X11(native) => native.window.cast(),
        }
    }

    pub(crate) fn pixels(&self) -> Result<Vec<u8>, String> {
        match self {
            Self::Wayland(_) => {
                Err("Wayland: compositor-owned pixels cannot be read as an X11 drawable".into())
            }
            Self::X11(native) => native.pixels(),
        }
    }
}

pub(crate) trait LinuxEventSource {
    fn pending(&self) -> Result<bool, String>;
    fn descriptor(&self) -> Result<BorrowedFd<'_>, String>;
    fn next_input_timeout(&self) -> Option<Duration> {
        None
    }
}

impl LinuxEventSource for x11::NativeWindow {
    fn pending(&self) -> Result<bool, String> {
        Ok(self.has_pending_input())
    }
    fn descriptor(&self) -> Result<BorrowedFd<'_>, String> {
        self.connection_descriptor()
    }
}

impl LinuxEventSource for NativeWindow {
    fn pending(&self) -> Result<bool, String> {
        match self {
            Self::Wayland(native) => native.has_pending_input(),
            Self::X11(native) => native.pending(),
        }
    }
    fn descriptor(&self) -> Result<BorrowedFd<'_>, String> {
        match self {
            Self::Wayland(native) => Ok(native.connection_descriptor()),
            Self::X11(native) => native.descriptor(),
        }
    }

    fn next_input_timeout(&self) -> Option<Duration> {
        match self {
            Self::Wayland(native) => native.next_input_timeout(),
            Self::X11(_) => None,
        }
    }
}
