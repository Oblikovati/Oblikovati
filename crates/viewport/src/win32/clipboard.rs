// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::ViewportClipboard;
use std::rc::Weak;
use windows_sys::Win32::{
    Foundation::{GlobalFree, HGLOBAL},
    System::{
        DataExchange::{
            CloseClipboard, EmptyClipboard, GetClipboardData, IsClipboardFormatAvailable,
            OpenClipboard, SetClipboardData,
        },
        Memory::{GMEM_MOVEABLE, GlobalAlloc, GlobalLock, GlobalSize, GlobalUnlock},
        Ole::CF_UNICODETEXT,
    },
};

const FORMAT: u32 = CF_UNICODETEXT as u32;
const LIMIT: usize = 1024 * 1024;

/// Windows Unicode text clipboard tied to the owner window.
pub struct WindowsClipboard {
    hwnd: HWND,
    window: Weak<WindowEvents>,
}
impl WindowsClipboard {
    pub(super) fn new(hwnd: HWND, window: Weak<WindowEvents>) -> Self {
        Self { hwnd, window }
    }

    fn open(&self) -> Result<ClipboardLease, String> {
        if self.window.upgrade().is_none() {
            return Err("Windows clipboard: expected a live owner window".into());
        }
        if unsafe { OpenClipboard(self.hwnd) } == 0 {
            return Err(win_error("open Windows clipboard"));
        }
        Ok(ClipboardLease)
    }
}

impl ViewportClipboard for WindowsClipboard {
    fn read_text(&self) -> Result<Option<String>, String> {
        let _clipboard = self.open()?;
        if unsafe { IsClipboardFormatAvailable(FORMAT) } == 0 {
            return Ok(None);
        }
        let memory = unsafe { GetClipboardData(FORMAT) };
        if memory.is_null() {
            return Err(win_error("read Windows clipboard text"));
        }
        let bound = unsafe { GlobalSize(memory) }.min((LIMIT * 2 + 1) * 2) / 2;
        let lock = MemoryLock::acquire(memory)?;
        // CF_UNICODETEXT is bounded by the allocation and terminated by NUL.
        let units = unsafe { std::slice::from_raw_parts(lock.address.cast::<u16>(), bound) };
        let end = units
            .iter()
            .position(|unit| *unit == 0)
            .ok_or("Windows clipboard text: expected a terminator within the clipboard limit")?;
        let text = String::from_utf16(&units[..end])
            .map_err(|err| format!("Windows clipboard text: {err}"))?;
        let text = text.replace("\r\n", "\n");
        if text.len() > LIMIT {
            return Err(format!(
                "Windows clipboard text {} bytes: expected at most {LIMIT}",
                text.len()
            ));
        }
        Ok(Some(text))
    }

    fn write_text(&self, text: &str) -> Result<(), String> {
        if text.len() > LIMIT || text.contains('\0') {
            return Err(format!(
                "Windows clipboard text {} bytes: expected at most {LIMIT} and no embedded NUL",
                text.len()
            ));
        }
        let normalized = text.replace("\r\n", "\n").replace('\n', "\r\n");
        let units: Vec<u16> = normalized.encode_utf16().chain([0]).collect();
        let mut memory = ClipboardMemory(unsafe { GlobalAlloc(GMEM_MOVEABLE, units.len() * 2) });
        if memory.0.is_null() {
            return Err(win_error("allocate Windows clipboard text"));
        }
        {
            let lock = MemoryLock::acquire(memory.0)?;
            unsafe { ptr::copy_nonoverlapping(units.as_ptr(), lock.address.cast(), units.len()) };
        }
        let _clipboard = self.open()?;
        if unsafe { EmptyClipboard() } == 0
            || unsafe { SetClipboardData(FORMAT, memory.0) }.is_null()
        {
            return Err(win_error("publish Windows clipboard text"));
        }
        // The OS now owns the movable allocation.
        memory.0 = ptr::null_mut();
        Ok(())
    }
}

struct ClipboardLease;
impl Drop for ClipboardLease {
    fn drop(&mut self) {
        unsafe { CloseClipboard() };
    }
}
struct ClipboardMemory(HGLOBAL);
impl Drop for ClipboardMemory {
    fn drop(&mut self) {
        if !self.0.is_null() {
            unsafe { GlobalFree(self.0) };
        }
    }
}
struct MemoryLock {
    memory: HGLOBAL,
    address: *mut std::ffi::c_void,
}
impl MemoryLock {
    fn acquire(memory: HGLOBAL) -> Result<Self, String> {
        let address = unsafe { GlobalLock(memory) };
        if address.is_null() {
            return Err(win_error("lock Windows clipboard text"));
        }
        Ok(Self { memory, address })
    }
}
impl Drop for MemoryLock {
    fn drop(&mut self) {
        unsafe { GlobalUnlock(self.memory) };
    }
}
