// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{
    Error,
    wire::{CaptureWindowArgs, CaptureWindowResult, METHOD_VIEWPORT_CAPTURE_WINDOW},
};

/// Commands that request a later completed frame without changing model history.
pub const WINDOW_CAPTURE_METHODS: [&str; 1] = [METHOD_VIEWPORT_CAPTURE_WINDOW];

/// Owner-thread boundary; the desktop owns admission and completed-frame publication.
pub trait WindowCaptureAccess {
    /// Queue the next completed whole-window image; example: `access.request_window_capture(&args)?`.
    fn request_window_capture(
        &mut self,
        args: &CaptureWindowArgs,
    ) -> Result<CaptureWindowResult, String>;
}

pub(crate) fn apply(
    access: &mut Option<&mut dyn WindowCaptureAccess>,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let args = crate::decode(METHOD_VIEWPORT_CAPTURE_WINDOW, request)?;
    let owner = access.as_deref_mut().ok_or_else(|| {
        Error(format!(
            "window capture {args:?}: expected an injected live desktop capture owner"
        ))
    })?;
    crate::encode(&owner.request_window_capture(&args).map_err(Error)?)
}
