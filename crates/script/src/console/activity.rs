// SPDX-License-Identifier: GPL-2.0-only
use super::ConsoleMode;
use oblikovati_api::{Error, client::Caller};
use std::cell::Cell;
/// One bounded owner poll; active execution and newly changed UI state are separate.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct ConsolePoll {
    pub changed: bool,
    pub active: bool,
    pub completed: Option<ConsoleMode>,
}
pub(super) struct ActivityCaller<'a, C> {
    pub host: &'a C,
    pub called: Cell<bool>,
}
impl<C: Caller> Caller for ActivityCaller<'_, C> {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        self.called.set(true);
        self.host.call(method, request)
    }
}
