// SPDX-License-Identifier: GPL-2.0-only
use super::GuiParameterAck;
pub(super) struct ParameterEditRequest<Packet> {
    pub(super) last_request: u64,
    pub(super) pending: Option<Packet>,
    pub(super) issue: Option<String>,
}
impl<Packet> Default for ParameterEditRequest<Packet> {
    fn default() -> Self {
        Self {
            last_request: 0,
            pending: None,
            issue: None,
        }
    }
}
impl<Packet: PartialEq> ParameterEditRequest<Packet> {
    pub(super) fn prepare(&self) -> Result<u64, String> {
        if self.pending.is_some() {
            return Err(format!(
                "GUI parameter request {}: expected an idle editor before submission",
                self.last_request
            ));
        }
        self.last_request.checked_add(1).ok_or_else(|| {
            format!(
                "GUI parameter request {}: expected an unused request identity",
                self.last_request
            )
        })
    }
    pub(super) fn publish(&mut self, request: u64, packet: Packet) -> Result<(), String> {
        let expected = self.prepare()?;
        if request != expected {
            return Err(format!(
                "GUI parameter request {request}: expected next request {expected}"
            ));
        }
        self.last_request = request;
        self.pending = Some(packet);
        self.issue = None;
        Ok(())
    }
    pub(super) fn acknowledge(
        &mut self,
        packet: &Packet,
        result: Result<(), String>,
    ) -> GuiParameterAck {
        if self.pending.as_ref() != Some(packet) {
            return GuiParameterAck::Obsolete;
        }
        self.pending = None;
        if let Err(cause) = result {
            self.issue = Some(cause);
            return GuiParameterAck::Failed;
        }
        self.issue = None;
        GuiParameterAck::Applied
    }
    pub(super) fn retire(&mut self) {
        self.pending = None;
        self.issue = None;
    }
}
#[cfg(test)]
#[path = "../../tests/parameters/request.rs"]
mod tests;
