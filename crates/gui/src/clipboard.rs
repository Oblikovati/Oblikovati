// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_viewport::ViewportClipboard;
use std::{cell::RefCell, rc::Rc};
/// A GUI callback owns its platform dependency and a shared operation-health record.
/// Example: `GuiClipboard::new(provider, 1024 * 1024)?` validates before C callbacks.
#[derive(Clone)]
pub struct GuiClipboard {
    provider: Rc<dyn ViewportClipboard>,
    byte_limit: usize,
    issue: Rc<RefCell<Option<String>>>,
}
impl GuiClipboard {
    /// Inject bounded clipboard I/O on the GUI thread; example: `GuiClipboard::new(provider, 4096)?`.
    pub fn new(provider: Rc<dyn ViewportClipboard>, byte_limit: usize) -> Result<Self, String> {
        if byte_limit == 0 {
            return Err(format!(
                "clipboard byte limit {byte_limit}: expected a positive transfer budget"
            ));
        }
        Ok(Self {
            provider,
            byte_limit,
            issue: Rc::new(RefCell::new(None)),
        })
    }
    /// Inspect the last failed operation; example: `clipboard.issue()` feeds the health panel.
    pub fn issue(&self) -> Option<String> {
        self.issue.borrow().clone()
    }
    fn read_checked(&self) -> Result<Option<String>, String> {
        let text = self.provider.read_text()?;
        if let Some(text) = &text {
            require_callback_text(text, self.byte_limit)?;
        }
        Ok(text)
    }
    fn write_checked(&self, text: &str) -> Result<(), String> {
        require_callback_text(text, self.byte_limit)?;
        self.provider.write_text(text)
    }
    fn record_operation<T>(&self, result: Result<T, String>) -> Option<T> {
        match result {
            Ok(value) => {
                self.issue.borrow_mut().take();
                Some(value)
            }
            Err(cause) => {
                *self.issue.borrow_mut() = Some(cause);
                None
            }
        }
    }
}
impl imgui::ClipboardBackend for GuiClipboard {
    fn get(&mut self) -> Option<String> {
        self.record_operation(self.read_checked()).flatten()
    }
    fn set(&mut self, value: &str) {
        self.record_operation(self.write_checked(value));
    }
}
fn require_callback_text(text: &str, byte_limit: usize) -> Result<(), String> {
    // imgui-rs 0.12.0 aborts when its C callback constructs CString from a NUL-bearing value.
    if let Some(offset) = text.as_bytes().iter().position(|byte| *byte == 0) {
        return Err(format!(
            "clipboard NUL at byte {offset}: expected C-string-safe text"
        ));
    }
    if text.len() > byte_limit {
        return Err(format!(
            "clipboard length {}: expected at most {byte_limit} bytes",
            text.len()
        ));
    }
    Ok(())
}
#[cfg(test)]
#[path = "../tests/clipboard/unit.rs"]
mod tests;

#[cfg(test)]
#[path = "../tests/clipboard/fake.rs"]
pub(crate) mod test_provider;
