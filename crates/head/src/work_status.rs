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
