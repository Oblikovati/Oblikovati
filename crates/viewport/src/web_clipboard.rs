// SPDX-License-Identifier: GPL-2.0-only
//! Browser-compatible text clipboard buffer implementing `ViewportClipboard`.
use crate::ViewportClipboard;
use std::cell::RefCell;

/// Owner-thread in-memory text clipboard for WebAssembly hosts.
///
/// ```
/// use oblikovati_viewport::{ViewportClipboard, WebClipboard};
/// let clipboard = WebClipboard::default();
/// clipboard.write_text("width = 10 mm").unwrap();
/// assert_eq!(clipboard.read_text().unwrap().as_deref(), Some("width = 10 mm"));
/// ```
#[derive(Default)]
pub struct WebClipboard {
    text: RefCell<Option<String>>,
}

impl ViewportClipboard for WebClipboard {
    fn read_text(&self) -> Result<Option<String>, String> {
        Ok(self.text.borrow().clone())
    }

    fn write_text(&self, text: &str) -> Result<(), String> {
        *self.text.borrow_mut() = Some(text.to_owned());
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn web_clipboard_round_trips_owned_text() {
        let clipboard = WebClipboard::default();
        assert_eq!(clipboard.read_text().unwrap(), None);
        clipboard.write_text("42 mm").unwrap();
        assert_eq!(clipboard.read_text().unwrap().as_deref(), Some("42 mm"));
    }
}
