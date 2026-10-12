// SPDX-License-Identifier: GPL-2.0-only
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct HeadWorkStatus {
    pub changed: bool,
    pub pending: bool,
    pub wake_after: Option<std::time::Duration>,
}
/// Coalesced model changes from the injected document observers.
pub trait HeadInvalidation {
    /// Consume pending invalidation; example: `factory.take_changed()` after direct UI commands.
    fn take_changed(&self) -> bool;
}
/// Owner-thread services borrow model state only while the window loop polls them.
pub trait HeadWork {
    /// Drain one bounded request batch; example: `work.poll(32)?` before presentation.
    fn poll(&mut self, budget: usize) -> Result<HeadWorkStatus, String>;
    /// Check close without discarding documents; example: `work.close_allowed()?` after a native close request.
    fn close_allowed(&mut self) -> Result<bool, String>;
    /// Stop admission before window retirement; example: `work.finish()?` after the owner loop ends.
    fn finish(&mut self) -> Result<(), String>;
}
pub(crate) struct IdleHeadWork;
impl HeadWork for IdleHeadWork {
    fn poll(&mut self, _: usize) -> Result<HeadWorkStatus, String> {
        Ok(HeadWorkStatus::default())
    }
    fn close_allowed(&mut self) -> Result<bool, String> {
        Ok(true)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}
