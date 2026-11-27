// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
#![allow(unsafe_code)]
use std::{ffi::CStr, sync::Arc, time::Duration};
use x11_dl::xlib;

struct FixtureXConnection {
    api: Arc<xlib::Xlib>,
    display: *mut xlib::Display,
}
impl FixtureXConnection {
    fn open(api: Arc<xlib::Xlib>) -> Self {
        // The fixture retains the library through the connection's close operation.
        let display = unsafe { (api.XOpenDisplay)(std::ptr::null()) };
        assert!(
            !display.is_null(),
            "X11 fixture display {:?}: expected a live connection",
            std::env::var_os("DISPLAY")
        );
        Self { api, display }
    }
    fn atom(&self, name: &CStr, existing_only: bool) -> xlib::Atom {
        // InternAtom copies the name and waits for the server's reply before returning.
        unsafe { (self.api.XInternAtom)(self.display, name.as_ptr(), i32::from(existing_only)) }
    }
}
impl Drop for FixtureXConnection {
    fn drop(&mut self) {
        // No pointer escapes this fixture; close before its library owner is released.
        unsafe { (self.api.XCloseDisplay)(self.display) };
    }
}

#[test]
fn native_display_fixture_retains_server_state_between_clients() {
    let api = Arc::new(xlib::Xlib::open().unwrap());
    // This binary has no earlier Xlib clients; initialize threads before the first open.
    assert_ne!(unsafe { (api.XInitThreads)() }, 0);
    let name =
        std::ffi::CString::new(format!("OBK_DISPLAY_LIFETIME_{}", std::process::id())).unwrap();
    let atom = publish_fixture_atom(api.clone(), &name);
    // Give a reset-enabled server time to discard state after its last client closes.
    std::thread::sleep(Duration::from_millis(150));
    let second = FixtureXConnection::open(api);
    assert_eq!(
        second.atom(&name, true),
        atom,
        "X11 fixture reset between clients: expected retained atom {atom}; use Xvfb -noreset"
    );
}

fn publish_fixture_atom(api: Arc<xlib::Xlib>, name: &CStr) -> xlib::Atom {
    let first = FixtureXConnection::open(api);
    let atom = first.atom(name, false);
    assert_ne!(
        atom, 0,
        "X11 fixture atom {atom}: expected a newly interned atom"
    );
    drop(first);
    atom
}
