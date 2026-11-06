// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiClipboard, GuiFrameEngine, clipboard::test_provider::FakeClipboardProvider};
use std::{ffi::CStr, rc::Rc};
#[test]
fn native_clipboard_callbacks_copy_unicode_and_refuse_nul_without_aborting() {
    let provider = Rc::new(FakeClipboardProvider::default());
    let clipboard = GuiClipboard::new(provider.clone(), 32).unwrap();
    let owner = GuiFrameEngine::with_clipboard(clipboard.clone()).unwrap();
    unsafe { imgui::sys::igSetClipboardText(c"é🙂".as_ptr()) };
    let pointer = unsafe { imgui::sys::igGetClipboardText() };
    assert!(!pointer.is_null());
    assert_eq!(unsafe { CStr::from_ptr(pointer) }.to_str().unwrap(), "é🙂");
    *provider.text.borrow_mut() = Some("a\0b".into());
    assert!(unsafe { imgui::sys::igGetClipboardText() }.is_null());
    assert!(clipboard.issue().unwrap().contains("byte 1"));
    drop(owner);
}
