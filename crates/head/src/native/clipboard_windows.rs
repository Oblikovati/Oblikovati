// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_viewport::{ViewportClipboard, WindowsClipboard};
use std::{cell::RefCell, rc::Rc};

pub(super) const CLIPBOARD_BYTES: usize = 1024 * 1024;

pub(super) struct NativeClipboardChannel {
    clipboard: WindowsClipboard,
    issue: RefCell<Option<String>>,
}

impl NativeClipboardChannel {
    pub(super) fn open(clipboard: WindowsClipboard) -> Rc<Self> {
        Rc::new(Self {
            clipboard,
            issue: RefCell::default(),
        })
    }

    pub(super) fn issue(&self) -> Option<String> {
        self.issue.borrow().clone()
    }

    fn record<T>(&self, result: Result<T, String>) -> Result<T, String> {
        *self.issue.borrow_mut() = result.as_ref().err().cloned();
        result
    }
}

impl ViewportClipboard for NativeClipboardChannel {
    fn read_text(&self) -> Result<Option<String>, String> {
        self.record(self.clipboard.read_text())
    }

    fn write_text(&self, text: &str) -> Result<(), String> {
        self.record(self.clipboard.write_text(text))
    }
}
