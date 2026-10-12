// SPDX-License-Identifier: GPL-2.0-only
use crate::{ViewportEvent, types::valid_extent};
use rustycad_tkernel::standard::Handle;
use rustycad_tkservice::{
    aspect::{AspectDisplayConnection, AspectWindow, AspectXAtom},
    xw::XwWindow,
};
use std::{
    rc::Rc,
    time::{Duration, Instant},
};
use x11_dl::xlib;
struct NativeImage<'a> {
    pointer: *mut xlib::XImage,
    api: &'a xlib::Xlib,
}
impl Drop for NativeImage<'_> {
    fn drop(&mut self) {
        unsafe {
            (self.api.XDestroyImage)(self.pointer);
        }
    }
}
pub(crate) struct NativeX11 {
    api: xlib::Xlib,
}
impl NativeX11 {
    pub fn open() -> Result<Rc<Self>, String> {
        let api = xlib::Xlib::open()
            .map_err(|error| format!("X11 runtime: expected loadable Xlib: {error}"))?;
        // Runtime creation precedes all application X11 connections and add-in activation.
        if unsafe { (api.XInitThreads)() } == 0 {
            return Err("XInitThreads returned 0: expected thread-safe Xlib initialization".into());
        }
        Ok(Rc::new(Self { api }))
    }
}
pub(crate) struct NativeWindow {
    pub window: Handle<XwWindow>,
    display: Handle<AspectDisplayConnection>,
    x11: Rc<NativeX11>,
}
impl NativeWindow {
    pub fn create(x11: Rc<NativeX11>, title: &str, extent: [u32; 2]) -> Result<Self, String> {
        let [width, height] = valid_extent(extent)?;
        validate_title(title)?;
        let display =
            Handle::new(AspectDisplayConnection::new().map_err(|error| error.to_string())?);
        let window = Handle::new(
            XwWindow::create(&display, title, 100, 100, width, height)
                .map_err(|error| error.to_string())?,
        );
        let result = Self {
            window,
            display,
            x11,
        };
        result.configure_events()?;
        result.mapped(true)?;
        result.window.do_resize();
        Ok(result)
    }
    fn pointer(&self) -> *mut xlib::Display {
        self.display.get_display_aspect() as *mut xlib::Display
    }
    fn configure_events(&self) -> Result<(), String> {
        let mut atom = self
            .display
            .get_atom(AspectXAtom::DeleteWindow)
            .map_err(|error| error.to_string())?;
        // The connection/window handles stay owned until view removal; no raw handle escapes.
        unsafe {
            (self.x11.api.XSetWMProtocols)(
                self.pointer(),
                self.window.native_handle(),
                &mut atom,
                1,
            );
        }
        self.select_events();
        Ok(())
    }
    fn select_events(&self) {
        unsafe {
            (self.x11.api.XSelectInput)(
                self.pointer(),
                self.window.native_handle(),
                xlib::ExposureMask | xlib::StructureNotifyMask | xlib::FocusChangeMask,
            );
        }
    }
    pub fn sync(&self) {
        unsafe {
            (self.x11.api.XSync)(self.pointer(), 0);
        }
    }
    pub fn resize(&self, extent: [u32; 2]) -> Result<(), String> {
        valid_extent(extent)?;
        unsafe {
            (self.x11.api.XResizeWindow)(
                self.pointer(),
                self.window.native_handle(),
                extent[0],
                extent[1],
            );
        }
        self.sync();
        self.window.do_resize();
        Ok(())
    }
    pub fn mapped(&self, mapped: bool) -> Result<(), String> {
        if mapped {
            self.window.map();
        } else {
            unsafe {
                (self.x11.api.XUnmapWindow)(self.pointer(), self.window.native_handle());
            }
        }
        self.sync();
        self.await_mapping(mapped)
    }
    fn await_mapping(&self, mapped: bool) -> Result<(), String> {
        let deadline = Instant::now() + Duration::from_secs(3);
        while self.window.is_mapped() != mapped {
            if Instant::now() >= deadline {
                return Err(format!(
                    "native window {} mapping: expected mapped={mapped} within three seconds",
                    self.window.native_handle()
                ));
            }
            std::thread::sleep(Duration::from_millis(5));
            self.sync();
        }
        Ok(())
    }
    pub fn poll(&self, limit: usize) -> Vec<ViewportEvent> {
        let mut events = Vec::new();
        for _ in 0..limit {
            if unsafe { (self.x11.api.XPending)(self.pointer()) } == 0 {
                break;
            }
            let mut event = std::mem::MaybeUninit::<xlib::XEvent>::uninit();
            unsafe {
                (self.x11.api.XNextEvent)(self.pointer(), event.as_mut_ptr());
            }
            if let Some(event) = self.translate(unsafe { event.assume_init() }) {
                events.push(event);
            }
        }
        events
    }
    fn translate(&self, event: xlib::XEvent) -> Option<ViewportEvent> {
        if unsafe { event.any.window } != self.window.native_handle() {
            return None;
        }
        match event.get_type() {
            xlib::Expose => Some(ViewportEvent::Expose),
            xlib::FocusIn | xlib::FocusOut => {
                Some(ViewportEvent::Focus(event.get_type() == xlib::FocusIn))
            }
            xlib::ConfigureNotify => {
                self.window.do_resize();
                let (width, height) = self.window.size();
                Some(ViewportEvent::Resized([width as u32, height as u32]))
            }
            xlib::ClientMessage => self.close_event(event),
            _ => None,
        }
    }
    fn close_event(&self, event: xlib::XEvent) -> Option<ViewportEvent> {
        let atom = self.display.get_atom(AspectXAtom::DeleteWindow).ok()?;
        let message = unsafe { event.client_message };
        let protocols =
            unsafe { (self.x11.api.XInternAtom)(self.pointer(), c"WM_PROTOCOLS".as_ptr(), 0) };
        (message.format == 32
            && message.message_type == protocols
            && message.data.get_long(0) as u64 == atom)
            .then_some(ViewportEvent::CloseRequested)
    }
    pub fn pixels(&self) -> Result<Vec<u8>, String> {
        if !self.window.is_mapped() {
            return Err("native pixel capture: expected a mapped viewport".into());
        }
        self.sync();
        self.window.do_resize();
        let (width, height) = self.window.size();
        let image = self.capture_image(width, height)?;
        self.image_pixels(&image, width, height)
    }
    fn capture_image(&self, width: i32, height: i32) -> Result<NativeImage<'_>, String> {
        let image = self.acquire_image(width, height);
        if image.is_null() {
            return Err(format!(
                "native pixel capture {width}x{height}: expected XGetImage pixels"
            ));
        }
        let owned = NativeImage {
            pointer: image,
            api: &self.x11.api,
        };
        validate_masks(image_masks(image))?;
        Ok(owned)
    }
    fn acquire_image(&self, width: i32, height: i32) -> *mut xlib::XImage {
        unsafe {
            (self.x11.api.XGetImage)(
                self.pointer(),
                self.window.native_handle(),
                0,
                0,
                width as u32,
                height as u32,
                !0,
                xlib::ZPixmap,
            )
        }
    }
    fn image_pixels(
        &self,
        image: &NativeImage<'_>,
        width: i32,
        height: i32,
    ) -> Result<Vec<u8>, String> {
        let mut rgba = rgba_buffer(width, height)?;
        for y in 0..height {
            for x in 0..width {
                rgba.extend(self.pixel(image.pointer, x, y));
            }
        }
        Ok(rgba)
    }
    fn pixel(&self, image: *mut xlib::XImage, x: i32, y: i32) -> [u8; 4] {
        let value = unsafe { (self.x11.api.XGetPixel)(image, x, y) };
        let masks = image_masks(image);
        let [red, green, blue] = masks.map(|mask| color_channel(value, mask));
        [red, green, blue, 255]
    }
}
fn color_channel(value: u64, mask: u64) -> u8 {
    if mask == 0 {
        return 0;
    }
    let shift = mask.trailing_zeros();
    let maximum = mask >> shift;
    (u128::from((value & mask) >> shift) * 255 / u128::from(maximum)) as u8
}

fn rgba_buffer(width: i32, height: i32) -> Result<Vec<u8>, String> {
    let extent = [width as u32, height as u32];
    valid_extent(extent)?;
    let capacity = (width as usize)
        .checked_mul(height as usize)
        .and_then(|count| count.checked_mul(4))
        .ok_or_else(|| {
            format!("native capture {extent:?}: expected representable RGBA byte count")
        })?;
    let mut rgba = Vec::new();
    rgba.try_reserve_exact(capacity).map_err(|error| {
        format!("native capture {extent:?}: expected allocatable RGBA bytes: {error}")
    })?;
    Ok(rgba)
}

fn image_masks(image: *mut xlib::XImage) -> [u64; 3] {
    unsafe { [(*image).red_mask, (*image).green_mask, (*image).blue_mask] }
}
fn validate_masks(masks: [u64; 3]) -> Result<(), String> {
    let invalid = masks.iter().any(|mask| {
        let bits = mask >> mask.trailing_zeros().min(63);
        *mask == 0 || bits & bits.wrapping_add(1) != 0
    });
    let [red, green, blue] = masks;
    if invalid || red & green != 0 || red & blue != 0 || green & blue != 0 {
        return Err(format!(
            "native image RGB masks {masks:?}: expected nonzero contiguous disjoint channels"
        ));
    }
    Ok(())
}

fn validate_title(title: &str) -> Result<(), String> {
    if title.is_empty() || title.contains('\0') {
        return Err(format!(
            "viewport title {title:?}: expected nonempty NUL-free text"
        ));
    }
    Ok(())
}

#[cfg(test)]
#[path = "../tests/x11_events.rs"]
mod tests;
