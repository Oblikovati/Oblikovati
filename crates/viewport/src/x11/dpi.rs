// SPDX-License-Identifier: GPL-2.0-only
//! User-configured display density from X resources or XSETTINGS.
use martensite_tkservice::aspect::AspectDisplayConnection;
use std::{cell::Cell, ptr, slice};
use x11_dl::xlib;

mod xsettings;

const BASE_DPI: f32 = 96.;
const MAX_RESOURCE_WORDS: std::ffi::c_long = 256 * 1024;

pub(super) struct DisplayScale {
    root: xlib::Window,
    manager: xlib::Atom,
    settings_selection: xlib::Atom,
    settings_property: xlib::Atom,
    settings_owner: Cell<xlib::Window>,
    scale: Cell<f32>,
}

impl DisplayScale {
    pub(super) fn watch(api: &xlib::Xlib, connection: &AspectDisplayConnection) -> Self {
        let display = connection.get_display_aspect() as *mut xlib::Display;
        // RESOURCE_MANAGER is the display-wide database on screen zero.
        // SAFETY: the native window retains this display until it is destroyed.
        let root = unsafe { (api.XRootWindow)(display, 0) };
        // SAFETY: the display remains live while the native window owns this watcher.
        let atom = |name: &std::ffi::CStr| unsafe { (api.XInternAtom)(display, name.as_ptr(), 0) };
        let state = Self {
            root,
            manager: atom(c"MANAGER"),
            settings_selection: atom(c"_XSETTINGS_S0"),
            settings_property: atom(c"_XSETTINGS_SETTINGS"),
            settings_owner: Cell::new(0),
            scale: Cell::new(1.),
        };
        watch_window(api, display, root);
        state.refresh(api, display);
        state
    }

    pub(super) fn get(&self) -> f32 {
        self.scale.get()
    }

    pub(super) fn handles(&self, event: &xlib::XEvent, display: *mut xlib::Display) -> bool {
        // SAFETY: each arm reads only the member identified by the event's type.
        unsafe {
            match event.get_type() {
                xlib::PropertyNotify => {
                    let property = event.property;
                    property.display == display
                        && ((property.window == self.root
                            && property.atom == xlib::XA_RESOURCE_MANAGER)
                            || (property.window == self.settings_owner.get()
                                && property.atom == self.settings_property))
                }
                xlib::ClientMessage => {
                    let message = event.client_message;
                    message.display == display
                        && message.window == self.root
                        && message.message_type == self.manager
                        && message.format == 32
                        && message.data.get_long(1) as xlib::Atom == self.settings_selection
                }
                xlib::DestroyNotify => {
                    let destroyed = event.destroy_window;
                    destroyed.display == display && destroyed.window == self.settings_owner.get()
                }
                _ => false,
            }
        }
    }

    pub(super) fn refresh(&self, api: &xlib::Xlib, display: *mut xlib::Display) -> bool {
        let resource = read_property(
            api,
            display,
            self.root,
            xlib::XA_RESOURCE_MANAGER,
            xlib::XA_STRING,
            |bytes| resource_dpi(&String::from_utf8_lossy(bytes)),
        );
        let dpi = resource.or_else(|| self.settings_dpi(api, display));
        let density = dpi.map_or(1., |dpi| (dpi / BASE_DPI).clamp(0.5, 4.));
        self.scale.replace(density) != density
    }

    fn settings_dpi(&self, api: &xlib::Xlib, display: *mut xlib::Display) -> Option<f32> {
        // Keep the manager window stable while finding, watching, and reading it.
        // SAFETY: this connection is live and the grab is released before returning.
        unsafe { (api.XGrabServer)(display) };
        // SAFETY: the selection atom was interned on the live display.
        let owner = unsafe { (api.XGetSelectionOwner)(display, self.settings_selection) };
        let previous = self.settings_owner.replace(owner);
        let mut dpi = None;
        if owner != 0 {
            if owner != previous {
                watch_window(api, display, owner);
            }
            dpi = read_property(
                api,
                display,
                owner,
                self.settings_property,
                self.settings_property,
                xsettings::dpi,
            );
        }
        // SAFETY: release this connection's server grab and send the release promptly.
        unsafe {
            (api.XUngrabServer)(display);
            (api.XFlush)(display);
        }
        dpi
    }
}

fn watch_window(api: &xlib::Xlib, display: *mut xlib::Display, window: xlib::Window) {
    let mut attributes = std::mem::MaybeUninit::<xlib::XWindowAttributes>::uninit();
    // SAFETY: the root is permanent and the settings owner is watched under a server grab.
    unsafe {
        if (api.XGetWindowAttributes)(display, window, attributes.as_mut_ptr()) != 0 {
            let mask = attributes.assume_init().your_event_mask
                | xlib::PropertyChangeMask
                | xlib::StructureNotifyMask;
            (api.XSelectInput)(display, window, mask);
        }
    }
}

fn read_property<T>(
    api: &xlib::Xlib,
    display: *mut xlib::Display,
    window: xlib::Window,
    property: xlib::Atom,
    expected_type: xlib::Atom,
    read: impl FnOnce(&[u8]) -> Option<T>,
) -> Option<T> {
    let (mut kind, mut format, mut count, mut remaining) = (0, 0, 0, 0);
    let mut data = ptr::null_mut();
    // SAFETY: output slots are writable and the caller keeps the display and window live.
    let status = unsafe {
        (api.XGetWindowProperty)(
            display,
            window,
            property,
            0,
            MAX_RESOURCE_WORDS,
            0,
            expected_type,
            &mut kind,
            &mut format,
            &mut count,
            &mut remaining,
            &mut data,
        )
    };
    if data.is_null() {
        return None;
    }
    let result = if status == i32::from(xlib::Success)
        && kind == expected_type
        && format == 8
        && remaining == 0
    {
        // SAFETY: Xlib returned count bytes for this 8-bit property.
        let bytes = unsafe { slice::from_raw_parts(data, count as usize) };
        read(bytes)
    } else {
        None
    };
    // SAFETY: XGetWindowProperty allocated this non-null result.
    unsafe { (api.XFree)(data.cast()) };
    result
}

fn resource_dpi(resource: &str) -> Option<f32> {
    resource
        .lines()
        .filter_map(|row| row.split_once(':'))
        .rfind(|(key, _)| key.trim() == "Xft.dpi")
        .and_then(|(_, value)| value.trim().trim_end_matches('\0').parse::<f32>().ok())
        .filter(|dpi| dpi.is_finite() && *dpi > 0.)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn uses_the_desktop_dpi_and_defaults_on_absent_or_invalid_values() {
        assert_eq!(resource_dpi("Xft.dpi:\t144\n"), Some(144.));
        assert_eq!(resource_dpi("Xft.dpi: 96\nXft.dpi: 192\0"), Some(192.));
        assert_eq!(resource_dpi("Xft.dpi: 120.5"), Some(120.5));
        for resource in [
            "",
            "Xft.dpi: 0",
            "Xft.dpi: -96",
            "Xft.dpi: inf",
            "Xft.dpi: NaN",
        ] {
            assert_eq!(resource_dpi(resource), None);
        }
    }
}
