// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::ptr::NonNull;
use std::sync::atomic::{AtomicBool, Ordering};
#[repr(C)]
struct NativeInputStyles {
    count_styles: std::ffi::c_ushort,
    supported_styles: *mut std::ffi::c_ulong,
}

pub(super) struct NativeInputMethod {
    api: Rc<NativeX11>,
    _display: Handle<AspectDisplayConnection>,
    method: NonNull<xlib::_XIM>,
    context: Option<NonNull<xlib::_XIC>>,
    alive: Box<AtomicBool>,
}

impl NativeInputMethod {
    pub(super) fn open(
        api: Rc<NativeX11>,
        display: Handle<AspectDisplayConnection>,
        window: u64,
    ) -> Result<Self, String> {
        let owner = display.get_display_aspect() as *mut xlib::Display;
        let method = open_native_input_method(&api.api, owner, window)?;
        let mut input = Self {
            api,
            _display: display,
            method,
            context: None,
            alive: Box::new(AtomicBool::new(true)),
        };
        input.register_retirement()?;
        input.create_context(window)?;
        Ok(input)
    }
    fn register_retirement(&self) -> Result<(), String> {
        let mut callback = xlib::XIMCallback {
            client_data: (&*self.alive as *const AtomicBool).cast_mut().cast(),
            callback: Some(input_method_retired),
        };
        register_native_retirement(self, &mut callback)
    }
    fn create_context(&mut self, window: u64) -> Result<(), String> {
        let style = (xlib::XIMPreeditNothing | xlib::XIMStatusNothing) as std::ffi::c_ulong;
        self.require_input_style(style)?;
        self.context = Some(create_native_context(self, window, style)?);
        Ok(())
    }
    fn require_input_style(&self, requested: std::ffi::c_ulong) -> Result<(), String> {
        let styles = native_input_styles(self)?;
        // The native query admitted an owned nonnull list; its lifetime stays within this call.
        unsafe { supported_style(&*styles.pointer, requested) }
    }
    pub(super) fn focus(&self, focused: bool) -> Result<(), String> {
        self.require_live_context()?;
        let context = self.context.expect("live context").as_ptr();
        unsafe {
            if focused {
                (self.api.api.XSetICFocus)(context);
            } else {
                (self.api.api.XUnsetICFocus)(context);
            }
        }
        Ok(())
    }
    pub(super) fn committed_text(&self, event: &mut xlib::XKeyEvent) -> Result<String, String> {
        self.require_live_context()?;
        let owner = self._display.get_display_aspect() as *mut xlib::Display;
        require_owned_commit(event, owner)?;
        lookup_committed_text(self, event)
    }
    fn require_live_context(&self) -> Result<(), String> {
        if !self.alive.load(Ordering::Acquire) || self.context.is_none() {
            return Err(format!(
                "XIM context alive={}: expected a live owned input method and context",
                self.alive.load(Ordering::Acquire)
            ));
        }
        Ok(())
    }
}
struct OwnedInputStyles<'a> {
    api: &'a xlib::Xlib,
    pointer: *mut NativeInputStyles,
}
impl Drop for OwnedInputStyles<'_> {
    fn drop(&mut self) {
        if !self.pointer.is_null() {
            unsafe { (self.api.XFree)(self.pointer.cast()) };
        }
    }
}
fn register_native_retirement(
    input: &NativeInputMethod,
    callback: &mut xlib::XIMCallback,
) -> Result<(), String> {
    let rejected = unsafe {
        (input.api.api.XSetIMValues)(
            input.method.as_ptr(),
            c"destroyCallback".as_ptr(),
            callback,
            std::ptr::null::<std::ffi::c_char>(),
        )
    };
    if !rejected.is_null() {
        return Err(format!(
            "XIM callback attribute {:?}: expected owned retirement notification",
            unsafe { CStr::from_ptr(rejected) }
        ));
    }
    Ok(())
}
fn create_native_context(
    input: &NativeInputMethod,
    window: u64,
    style: std::ffi::c_ulong,
) -> Result<NonNull<xlib::_XIC>, String> {
    let context = unsafe {
        (input.api.api.XCreateIC)(
            input.method.as_ptr(),
            c"inputStyle".as_ptr(),
            style,
            c"clientWindow".as_ptr(),
            window as xlib::Window,
            c"focusWindow".as_ptr(),
            window as xlib::Window,
            std::ptr::null::<std::ffi::c_char>(),
        )
    };
    NonNull::new(context).ok_or_else(|| {
        format!("XCreateIC window {window}, style {style}: expected a live input context")
    })
}
fn native_input_styles(input: &NativeInputMethod) -> Result<OwnedInputStyles<'_>, String> {
    let mut styles = OwnedInputStyles {
        api: &input.api.api,
        pointer: std::ptr::null_mut(),
    };
    let rejected = unsafe {
        (input.api.api.XGetIMValues)(
            input.method.as_ptr(),
            c"queryInputStyle".as_ptr(),
            &mut styles.pointer,
            std::ptr::null::<std::ffi::c_char>(),
        )
    };
    if !rejected.is_null() || styles.pointer.is_null() {
        return Err(format!(
            "XIM style query {:?}: expected an owned supported-style list",
            styles.pointer
        ));
    }
    Ok(styles)
}
unsafe extern "C" fn input_method_retired(_: xlib::XIM, owner: xlib::XPointer, _: xlib::XPointer) {
    if let Some(alive) = unsafe { owner.cast::<AtomicBool>().as_ref() } {
        alive.store(false, Ordering::Release);
    }
}
unsafe fn supported_style(
    styles: &NativeInputStyles,
    requested: std::ffi::c_ulong,
) -> Result<(), String> {
    if styles.count_styles == 0 || styles.supported_styles.is_null() {
        return Err(format!(
            "XIM styles count {}: expected a nonempty owned style list",
            styles.count_styles
        ));
    }
    let offered = unsafe {
        std::slice::from_raw_parts(styles.supported_styles, usize::from(styles.count_styles))
    };
    if offered.contains(&requested) {
        return Ok(());
    }
    Err(format!(
        "XIM styles {offered:?}: expected root preedit/status style {requested}"
    ))
}
impl Drop for NativeInputMethod {
    fn drop(&mut self) {
        // XIM destroy notification retires contexts; touching them afterward is invalid.
        if !self.alive.swap(false, Ordering::AcqRel) {
            return;
        }
        unsafe {
            if let Some(context) = self.context.take() {
                (self.api.api.XDestroyIC)(context.as_ptr());
            }
            (self.api.api.XCloseIM)(self.method.as_ptr());
        }
    }
}
struct Utf8LookupReply {
    count: i32,
    status: i32,
}
trait CommittedTextLookup {
    fn lookup(&self, event: &mut xlib::XKeyEvent, buffer: &mut [u8]) -> Utf8LookupReply;
}
impl CommittedTextLookup for NativeInputMethod {
    fn lookup(&self, event: &mut xlib::XKeyEvent, buffer: &mut [u8]) -> Utf8LookupReply {
        let mut status = xlib::XLookupNone;
        let count = unsafe {
            (self.api.api.Xutf8LookupString)(
                self.context.expect("live context").as_ptr(),
                event,
                buffer.as_mut_ptr().cast(),
                buffer.len() as i32,
                std::ptr::null_mut(),
                &mut status,
            )
        };
        Utf8LookupReply { count, status }
    }
}
fn lookup_committed_text(
    lookup: &impl CommittedTextLookup,
    event: &mut xlib::XKeyEvent,
) -> Result<String, String> {
    let mut buffer = vec![0; 64];
    let mut reply = lookup.lookup(event, &mut buffer);
    if reply.status == xlib::XBufferOverflow {
        buffer = required_commit_buffer(reply.count)?;
        reply = lookup.lookup(event, &mut buffer);
    }
    decode_committed_reply(&buffer, reply)
}
fn required_commit_buffer(required: i32) -> Result<Vec<u8>, String> {
    let count = usize::try_from(required)
        .ok()
        .filter(|count| *count > 0)
        .ok_or_else(|| {
            format!("XIM required UTF-8 length {required}: expected positive buffer capacity")
        })?;
    let mut buffer = Vec::new();
    buffer.try_reserve_exact(count).map_err(|cause| {
        format!("XIM UTF-8 capacity {count}: expected an allocatable commit buffer: {cause}")
    })?;
    buffer.resize(count, 0);
    Ok(buffer)
}
fn decode_committed_reply(buffer: &[u8], reply: Utf8LookupReply) -> Result<String, String> {
    if matches!(reply.status, xlib::XLookupNone | xlib::XLookupKeySym) {
        return empty_commit(reply);
    }
    if !matches!(reply.status, xlib::XLookupChars | xlib::XLookupBoth) {
        return Err(format!(
            "XIM UTF-8 status {}, length {}: expected a completed character lookup",
            reply.status, reply.count
        ));
    }
    let count = commit_byte_count(reply.count, buffer.len())?;
    let committed = std::str::from_utf8(&buffer[..count]).map_err(|cause| {
        format!(
            "XIM commit {:?}: expected valid UTF-8: {cause}",
            &buffer[..count]
        )
    })?;
    Ok(committed
        .chars()
        .filter(|character| !character.is_control())
        .collect())
}
fn open_native_input_method(
    api: &xlib::Xlib,
    owner: *mut xlib::Display,
    window: u64,
) -> Result<NonNull<xlib::_XIM>, String> {
    let method = unsafe {
        (api.XOpenIM)(
            owner,
            std::ptr::null_mut(),
            std::ptr::null_mut(),
            std::ptr::null_mut(),
        )
    };
    NonNull::new(method)
        .ok_or_else(|| format!("XOpenIM for window {window}, XMODIFIERS={:?}: expected an available configured input method", std::env::var_os("XMODIFIERS")))
}
fn require_owned_commit(event: &xlib::XKeyEvent, owner: *mut xlib::Display) -> Result<(), String> {
    if event.type_ != xlib::KeyPress || owner.is_null() || event.display != owner {
        return Err(format!(
            "XIM key type {}, display {:?}: expected KeyPress on the owned display {:?}",
            event.type_, event.display, owner
        ));
    }
    Ok(())
}
fn empty_commit(reply: Utf8LookupReply) -> Result<String, String> {
    if reply.count != 0 {
        return Err(format!(
            "XIM no-character status {}, length {}: expected zero committed bytes",
            reply.status, reply.count
        ));
    }
    Ok(String::new())
}
fn commit_byte_count(count: i32, capacity: usize) -> Result<usize, String> {
    let count = usize::try_from(count)
        .ok()
        .filter(|count| *count <= capacity)
        .ok_or_else(|| {
            format!(
                "XIM UTF-8 length {}, capacity {}: expected an in-buffer nonnegative length",
                count, capacity
            )
        })?;
    Ok(count)
}
pub(super) fn configure_input_locale(api: &xlib::Xlib) -> Result<(), String> {
    // LC_CTYPE is configured before native renderer threads; modelling numbers retain their locale.
    configure_character_locale()?;
    if unsafe { (api.XSupportsLocale)() } == 0
        || unsafe { (api.XSetLocaleModifiers)(c"".as_ptr()) }.is_null()
    {
        return Err(format!(
            "native XIM locale {}, XMODIFIERS={:?}: expected supported locale and configured input-method modifiers",
            input_locale_environment(),
            std::env::var_os("XMODIFIERS")
        ));
    }
    Ok(())
}
fn configure_character_locale() -> Result<(), String> {
    if unsafe { libc::setlocale(libc::LC_CTYPE, c"".as_ptr()) }.is_null() {
        return Err(format!(
            "native input locale {}: expected the configured LC_CTYPE locale to be available",
            input_locale_environment()
        ));
    }
    Ok(())
}
fn input_locale_environment() -> String {
    ["LC_ALL", "LC_CTYPE", "LANG"]
        .into_iter()
        .map(|name| format!("{name}={:?}", std::env::var_os(name)))
        .collect::<Vec<_>>()
        .join(", ")
}
#[cfg(test)]
#[path = "../../tests/x11_text.rs"]
mod tests;
