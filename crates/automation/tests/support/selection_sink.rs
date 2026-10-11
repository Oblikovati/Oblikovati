// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_automation::EventSink;
#[derive(Default)]
pub struct RecordingEventSink {
    pub payloads: Vec<Vec<u8>>,
    pub attempts: usize,
    pub remaining_refusals: usize,
}
impl EventSink for RecordingEventSink {
    fn send_event(&mut self, event_json: &[u8]) -> Result<(), String> {
        self.attempts += 1;
        if self.remaining_refusals > 0 {
            self.remaining_refusals -= 1;
            return Err("outbound queue is closed".into());
        }
        self.payloads.push(event_json.to_vec());
        Ok(())
    }
}
