// SPDX-License-Identifier: GPL-2.0-only
use super::{text::NativeInputMethod, xlib};
use std::cell::RefCell;
pub(super) struct NativeTextChannel {
    method: Option<NativeInputMethod>,
    issue: RefCell<Option<String>>,
}
impl NativeTextChannel {
    pub(super) fn admit(admission: Result<NativeInputMethod, String>) -> Self {
        let (method, issue) = match admission {
            Ok(method) => (Some(method), None),
            Err(issue) => (None, Some(issue)),
        };
        Self {
            method,
            issue: RefCell::new(issue),
        }
    }
    pub(super) fn issue(&self) -> Option<String> {
        self.issue.borrow().clone()
    }
    pub(super) fn focus(&self, focused: bool) {
        if self.issue.borrow().is_some() {
            return;
        }
        let Some(method) = &self.method else {
            return;
        };
        if let Err(cause) = method.focus(focused) {
            self.decline(cause);
        }
    }
    pub(super) fn committed_text(&self, event: &mut xlib::XKeyEvent) -> String {
        if self.issue.borrow().is_some() {
            return String::new();
        }
        let Some(method) = &self.method else {
            return String::new();
        };
        match method.committed_text(event) {
            Ok(commit) => commit,
            Err(cause) => {
                self.decline(cause);
                String::new()
            }
        }
    }
    fn decline(&self, cause: String) {
        // Quarantine this context. Keep physical input and report the failure through the owner.
        let release = self
            .method
            .as_ref()
            .and_then(|method| method.focus(false).err());
        let issue = match release.filter(|release| *release != cause) {
            Some(release) => format!("{cause}; input focus retirement: {release}"),
            None => cause,
        };
        *self.issue.borrow_mut() = Some(issue);
    }
}
#[cfg(test)]
#[path = "../../tests/x11_text_channel.rs"]
mod tests;
