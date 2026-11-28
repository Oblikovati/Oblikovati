// SPDX-License-Identifier: GPL-2.0-only
//! Portable monotonic and wall-clock time across native desktop and `wasm32-unknown-unknown`.
#[cfg(not(target_arch = "wasm32"))]
pub(crate) use std::time::Instant;
#[cfg(target_arch = "wasm32")]
pub(crate) use web_time::Instant;

/// Return current wall-clock time as `std::time::SystemTime` without panicking on `wasm32`.
/// Example: `let now = crate::clock::system_time_now();`.
#[cfg(not(target_arch = "wasm32"))]
pub(crate) fn system_time_now() -> std::time::SystemTime {
    std::time::SystemTime::now()
}

/// Return current wall-clock time as `std::time::SystemTime` via `web_time` on `wasm32`.
/// Example: `let now = crate::clock::system_time_now();`.
#[cfg(target_arch = "wasm32")]
pub(crate) fn system_time_now() -> std::time::SystemTime {
    let since_epoch = web_time::SystemTime::now()
        .duration_since(web_time::UNIX_EPOCH)
        .unwrap_or_default();
    std::time::UNIX_EPOCH + since_epoch
}
