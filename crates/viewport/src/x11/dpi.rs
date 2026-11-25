// SPDX-License-Identifier: GPL-2.0-only
//! User-configured display density from the X11 resource database.
use martensite_tkservice::aspect::AspectDisplayConnection;
use std::{cell::Cell, ptr, slice};
use x11_dl::xlib;

const BASE_DPI: f32 = 96.;
const MAX_RESOURCE_WORDS: std::ffi::c_long = 256 * 1024;

pub(super) struct DisplayScale {
    root: xlib::Window,
    scale: Cell<f32>,
}

impl DisplayScale {
    pub(super) fn watch(api: &xlib::Xlib, connection: &AspectDisplayConnection) -> Self {
        let display = connection.get_display_aspect() as *mut xlib::Display;
        // RESOURCE_MANAGER is the display-wide database on screen zero.
        // SAFETY: the native window retains this display until it is destroyed.
        let root = unsafe { (api.XRootWindow)(display, 0) };
        let state = Self {
            root,
            scale: Cell::new(1.),
        };
        let mut attributes = std::mem::MaybeUninit::<xlib::XWindowAttributes>::uninit();
        // SAFETY: Xlib owns the connection and initializes attributes on success.
        unsafe {
            if (api.XGetWindowAttributes)(display, root, attributes.as_mut_ptr()) != 0 {
                let mask = attributes.assume_init().your_event_mask | xlib::PropertyChangeMask;
                (api.XSelectInput)(display, root, mask);
            }
        }
        state.refresh(api, display);
        state
    }

    pub(super) fn get(&self) -> f32 {
        self.scale.get()
    }

    pub(super) fn handles(&self, event: &xlib::XEvent, display: *mut xlib::Display) -> bool {
        if event.get_type() != xlib::PropertyNotify {
            return false;
        }
        // SAFETY: PropertyNotify identifies the active member of this union.
        let property = unsafe { event.property };
        property.display == display
            && property.window == self.root
            && property.atom == xlib::XA_RESOURCE_MANAGER
    }

    pub(super) fn refresh(&self, api: &xlib::Xlib, display: *mut xlib::Display) -> bool {
        let density = read_scale(api, display, self.root);
        self.scale.replace(density) != density
    }
}

fn read_scale(api: &xlib::Xlib, display: *mut xlib::Display, root: xlib::Window) -> f32 {
    let (mut kind, mut format, mut count, mut remaining) = (0, 0, 0, 0);
    let mut data = ptr::null_mut();
    // SAFETY: output slots are writable and the caller retains the display and root window.
    let status = unsafe {
        (api.XGetWindowProperty)(
            display,
            root,
            xlib::XA_RESOURCE_MANAGER,
            0,
            MAX_RESOURCE_WORDS,
            0,
            xlib::XA_STRING,
            &mut kind,
            &mut format,
            &mut count,
            &mut remaining,
            &mut data,
        )
    };
    if data.is_null() {
        return 1.;
    }
    let scale = if status == i32::from(xlib::Success)
        && kind == xlib::XA_STRING
        && format == 8
        && remaining == 0
    {
        // SAFETY: Xlib returned count bytes for this 8-bit property.
        let resource = unsafe { slice::from_raw_parts(data, count as usize) };
        resource_scale(&String::from_utf8_lossy(resource))
    } else {
        1.
    };
    // SAFETY: XGetWindowProperty allocated this non-null result.
    unsafe { (api.XFree)(data.cast()) };
    scale
}

fn resource_scale(resource: &str) -> f32 {
    resource
        .lines()
        .filter_map(|row| row.split_once(':'))
        .rfind(|(key, _)| key.trim() == "Xft.dpi")
        .and_then(|(_, value)| value.trim().trim_end_matches('\0').parse::<f32>().ok())
        .filter(|dpi| dpi.is_finite() && *dpi > 0.)
        // Keep the scale inside the font and icon raster budgets at maximum content scale.
        .map_or(1., |dpi| (dpi / BASE_DPI).clamp(0.5, 4.))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn uses_the_desktop_dpi_and_defaults_on_absent_or_invalid_values() {
        assert_eq!(resource_scale("Xft.dpi:\t144\n"), 1.5);
        assert_eq!(resource_scale("Xft.dpi: 96\nXft.dpi: 192\0"), 2.);
        assert_eq!(resource_scale("Xft.dpi: 120.5"), 120.5 / 96.);
        assert_eq!(resource_scale("Xft.dpi: 1000000"), 4.);
        for resource in [
            "",
            "Xft.dpi: 0",
            "Xft.dpi: -96",
            "Xft.dpi: inf",
            "Xft.dpi: NaN",
        ] {
            assert_eq!(resource_scale(resource), 1.);
        }
    }
}
