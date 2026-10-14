// SPDX-License-Identifier: GPL-2.0-only
use crate::{ViewportEvent, types::valid_extent};
#[path = "x11/input.rs"]
mod input;
mod keyboard;
mod keyboard_map;
mod layout;
mod text;
mod text_channel;
use rustycad_tkernel::standard::Handle;
use rustycad_tkservice::{
    aspect::{AspectDisplayConnection, AspectWindow, AspectXAtom},
    xw::XwWindow,
};
use std::cell::RefCell;
use std::collections::VecDeque;
use std::{
    ffi::{CStr, CString},
    rc::Rc,
    sync::{Arc, OnceLock},
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
    api: Arc<xlib::Xlib>,
    input_locale: Result<(), String>,
}
#[derive(Clone)]
struct InitializedXlib {
    api: Arc<xlib::Xlib>,
    input_locale: Result<(), String>,
}
impl NativeX11 {
    pub fn open() -> Result<Rc<Self>, String> {
        // Runtime creation precedes all application X11 connections and add-in activation.
        // Keep the initialized library loaded: XInitThreads state belongs to that library.
        static API: OnceLock<Result<InitializedXlib, String>> = OnceLock::new();
        let initialized = API.get_or_init(load_threaded_xlib).clone()?;
        Ok(Rc::new(Self {
            api: initialized.api,
            input_locale: initialized.input_locale,
        }))
    }
}
fn load_threaded_xlib() -> Result<InitializedXlib, String> {
    let api = xlib::Xlib::open()
        .map_err(|cause| format!("X11 runtime: expected loadable Xlib: {cause}"))?;
    // Xlib's thread contract: https://www.x.org/releases/current/doc/libX11/libX11/libX11.html
    require_threaded_xlib(unsafe { (api.XInitThreads)() })?;
    let input_locale = text::configure_input_locale(&api);
    Ok(InitializedXlib {
        api: Arc::new(api),
        input_locale,
    })
}
fn require_threaded_xlib(status: i32) -> Result<(), String> {
    if status == 0 {
        return Err(format!(
            "XInitThreads returned {status}: expected thread-safe Xlib initialization"
        ));
    }
    Ok(())
}
pub(crate) struct NativeWindow {
    keyboard: keyboard::NativeKeyboard,
    copied_input: RefCell<VecDeque<ViewportEvent>>,
    pub window: Handle<XwWindow>,
    display: Handle<AspectDisplayConnection>,
    x11: Rc<NativeX11>,
}
impl NativeWindow {
    pub fn create(x11: Rc<NativeX11>, title: &str, extent: [u32; 2]) -> Result<Self, String> {
        valid_extent(extent)?;
        validate_title(title)?;
        let display =
            Handle::new(AspectDisplayConnection::new().map_err(|error| error.to_string())?);
        Self::create_with_display(x11, display, title, extent)
    }
    pub(super) fn create_with_display(
        x11: Rc<NativeX11>,
        display: Handle<AspectDisplayConnection>,
        title: &str,
        extent: [u32; 2],
    ) -> Result<Self, String> {
        let [width, height] = valid_extent(extent)?;
        validate_title(title)?;
        let window = Handle::new(
            XwWindow::create(&display, title, 100, 100, width, height)
                .map_err(|error| error.to_string())?,
        );
        let result = Self {
            keyboard: keyboard::NativeKeyboard::open(
                x11.clone(),
                display.clone(),
                window.native_handle(),
            )?,
            copied_input: RefCell::new(VecDeque::new()),
            window,
            display,
            x11,
        };
        result.activate_window()?;
        Ok(result)
    }
    fn activate_window(&self) -> Result<(), String> {
        self.configure_events()?;
        self.mapped(true)?;
        self.window.do_resize();
        Ok(())
    }
    pub(super) fn text_input_issue(&self) -> Option<String> {
        self.keyboard.text_input_issue()
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
                xlib::ExposureMask
                    | xlib::StructureNotifyMask
                    | xlib::FocusChangeMask
                    | xlib::ButtonPressMask
                    | xlib::ButtonReleaseMask
                    | xlib::PointerMotionMask
                    | xlib::KeyPressMask
                    | xlib::KeyReleaseMask,
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
    pub fn wait_events(
        &self,
        wake: &crate::ViewportWake,
        timeout: Option<Duration>,
    ) -> Result<crate::ViewportReady, String> {
        if !self.copied_input.borrow().is_empty()
            || unsafe { (self.x11.api.XPending)(self.pointer()) } != 0
        {
            return Ok(crate::ViewportReady {
                native_events: true,
                worker_wake: wake.consume()?,
            });
        }
        let descriptor = unsafe { (self.x11.api.XConnectionNumber)(self.pointer()) };
        if descriptor < 0 {
            return Err(format!(
                "X11 descriptor {descriptor}: expected a live connection"
            ));
        }
        // The borrowed connection stays owned by this native target throughout the wait.
        let native = unsafe { rustix::fd::BorrowedFd::borrow_raw(descriptor) };
        wake.wait(native, timeout)
    }
    pub fn poll(&self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        let mut events = Vec::new();
        for _ in 0..limit {
            if let Some(event) = self.copied_input.borrow_mut().pop_front() {
                events.push(event);
                continue;
            }
            if unsafe { (self.x11.api.XPending)(self.pointer()) } == 0 {
                break;
            }
            let mut event = std::mem::MaybeUninit::<xlib::XEvent>::uninit();
            unsafe {
                (self.x11.api.XNextEvent)(self.pointer(), event.as_mut_ptr());
            }
            self.copied_input
                .borrow_mut()
                .extend(self.copy_event(unsafe { event.assume_init() })?);
        }
        Ok(events)
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
            xlib::ConfigureNotify => Some(self.resize_event()),
            xlib::MapNotify | xlib::UnmapNotify => {
                Some(ViewportEvent::Mapped(event.get_type() == xlib::MapNotify))
            }
            xlib::ClientMessage => self.close_event(event),
            _ => input::translate_native_pointer(self.pointer(), event).map(ViewportEvent::Input),
        }
    }
    fn resize_event(&self) -> ViewportEvent {
        self.window.do_resize();
        let (width, height) = self.window.size();
        ViewportEvent::Resized([width as u32, height as u32])
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
#[path = "../tests/private_x11.rs"]
mod private_server;
#[cfg(test)]
#[path = "../tests/x11_events.rs"]
mod tests;

impl NativeWindow {
    pub fn set_title(&self, title: &str) -> Result<(), String> {
        validate_title(title)?;
        let title = CString::new(title).map_err(|cause| {
            format!("viewport title {title:?}: expected NUL-free text: {cause}")
        })?;
        unsafe {
            (self.x11.api.XStoreName)(
                self.pointer(),
                self.window.native_handle() as xlib::Window,
                title.as_ptr(),
            );
            (self.x11.api.XFlush)(self.pointer());
        }
        Ok(())
    }
    pub fn title(&self) -> Result<String, String> {
        let pointer = self.title_pointer()?;
        let bytes = unsafe { CStr::from_ptr(pointer) }.to_bytes().to_vec();
        unsafe {
            (self.x11.api.XFree)(pointer.cast());
        }
        String::from_utf8(bytes).map_err(|cause| {
            format!(
                "native title bytes {:?}: expected UTF-8 text: {cause}",
                cause.as_bytes()
            )
        })
    }
    fn title_pointer(&self) -> Result<*mut std::ffi::c_char, String> {
        let mut pointer = std::ptr::null_mut();
        let found = unsafe {
            (self.x11.api.XFetchName)(
                self.pointer(),
                self.window.native_handle() as xlib::Window,
                &mut pointer,
            )
        };
        if found == 0 || pointer.is_null() {
            return Err("native title is absent: expected a stored window name".into());
        }
        Ok(pointer)
    }
}
