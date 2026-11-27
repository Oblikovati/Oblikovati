// SPDX-License-Identifier: GPL-2.0-only
use std::time::Duration;
#[derive(Clone, Copy, Debug)]
pub struct DesktopDispatchOptions {
    pub capacity: usize,
    pub timeout: Duration,
}
impl Default for DesktopDispatchOptions {
    fn default() -> Self {
        // The fixed Go head uses dispatch.New(64) and hostCallTimeout = 10 seconds.
        Self {
            capacity: 64,
            timeout: Duration::from_secs(10),
        }
    }
}
