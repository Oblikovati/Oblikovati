// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[derive(Default)]
pub(crate) struct FakeClipboardProvider {
    pub(crate) text: RefCell<Option<String>>,
    pub(crate) failure: RefCell<Option<String>>,
    pub(crate) writes: RefCell<Vec<String>>,
}
impl ViewportClipboard for FakeClipboardProvider {
    fn read_text(&self) -> Result<Option<String>, String> {
        if let Some(cause) = self.failure.borrow().clone() {
            return Err(cause);
        }
        Ok(self.text.borrow().clone())
    }
    fn write_text(&self, text: &str) -> Result<(), String> {
        if let Some(cause) = self.failure.borrow().clone() {
            return Err(cause);
        }
        self.writes.borrow_mut().push(text.to_owned());
        *self.text.borrow_mut() = Some(text.to_owned());
        Ok(())
    }
}
