// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_automation::dispatch::DispatchWakeup;
use std::sync::mpsc::Sender;
/// A named wake transport for deterministic owner/worker admission handshakes.
pub struct WakeSignal(pub Sender<()>);
impl DispatchWakeup for WakeSignal {
    fn wake(&self) -> Result<(), String> {
        self.0.send(()).map_err(|error| error.to_string())
    }
}
