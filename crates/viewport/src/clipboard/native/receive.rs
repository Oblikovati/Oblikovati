// SPDX-License-Identifier: GPL-2.0-only
use super::text::{IncrementalText, ReceiveStep, TextEncoding, decode_text};
use super::*;
impl SelectionEndpoint {
    pub(super) fn read_current_text(&self) -> Result<Option<String>, String> {
        let owner = self.selection_owner(self.atoms.clipboard)?;
        if owner == 0 {
            return Ok(None);
        }
        if owner == self.window {
            return Ok(self.snapshot.borrow().as_ref().map(|text| text.to_string()));
        }
        self.read_foreign_text().map(Some)
    }
    fn read_foreign_text(&self) -> Result<String, String> {
        self.with_requestor(|requestor| self.read_on_window(requestor))
    }
    fn read_on_window(&self, requestor: Window) -> Result<String, String> {
        let deadline = self.operation_deadline()?;
        let targets = self.convert_property(requestor, self.atoms.targets, deadline)?;
        let encoding = classify_text_target(&targets, self.atoms.utf8)?;
        let target = match encoding {
            TextEncoding::Utf8 => self.atoms.utf8,
            TextEncoding::Latin1 => AtomEnum::STRING.into(),
        };
        let header = self.convert_property(requestor, target, deadline)?;
        if header.type_ == self.atoms.incr {
            return self.read_incremental(requestor, target, encoding, header, deadline);
        }
        decode_text_property(encoding, target, &header, self.options.byte_limit)
    }
    fn convert_property(
        &self,
        requestor: Window,
        target: Atom,
        deadline: Instant,
    ) -> Result<GetPropertyReply, String> {
        self.request_conversion(
            requestor,
            self.atoms.clipboard,
            target,
            self.atoms.receive,
            x11rb::CURRENT_TIME,
        )?;
        let notification = self.await_conversion(requestor, target, deadline)?;
        if notification.property != self.atoms.receive {
            return Err(format!(
                "clipboard target {target}, property {}: expected receive property {}",
                notification.property, self.atoms.receive
            ));
        }
        let limit = if target == self.atoms.targets {
            self.options.target_limit * 4
        } else {
            self.options.byte_limit.max(4)
        };
        self.read_property(requestor, limit)
    }
    fn await_conversion(
        &self,
        requestor: Window,
        target: Atom,
        deadline: Instant,
    ) -> Result<SelectionNotifyEvent, String> {
        loop {
            require_live_deadline(deadline)?;
            if let Some(event) = self.next_event()? {
                if let Event::SelectionNotify(notification) = &event
                    && notification.requestor == requestor
                    && notification.selection == self.atoms.clipboard
                    && notification.target == target
                {
                    return Ok(*notification);
                }
                self.service_event(event)?;
                continue;
            }
            properties::wait_selection(&self.connection, deadline)?;
        }
    }
    fn read_property(&self, requestor: Window, limit: usize) -> Result<GetPropertyReply, String> {
        let words = limit.div_ceil(4);
        let length = u32::try_from(words).map_err(|_| {
            format!("clipboard property budget {words}: expected a 32-bit word count")
        })?;
        let reply = self.read_native_property(
            true,
            requestor,
            self.atoms.receive,
            AtomEnum::ANY.into(),
            length,
        )?;
        if reply.bytes_after != 0 {
            return Err(format!(
                "clipboard remaining bytes {}: expected a complete property within {} bytes",
                reply.bytes_after, limit
            ));
        }
        Ok(reply)
    }
    fn read_incremental(
        &self,
        requestor: Window,
        target: Atom,
        encoding: TextEncoding,
        header: GetPropertyReply,
        deadline: Instant,
    ) -> Result<String, String> {
        let minimum = incremental_lower_bound(&header)?;
        let mut receiver = IncrementalText::begin(encoding, minimum, self.options.byte_limit)?;
        loop {
            self.await_chunk(requestor, deadline)?;
            let property = self.read_property(requestor, self.options.byte_limit.max(4))?;
            require_text_property(target, &property)?;
            if let ReceiveStep::Complete(text) = receiver.accept(encoding, &property.value)? {
                return Ok(text);
            }
        }
    }
    fn await_chunk(&self, requestor: Window, deadline: Instant) -> Result<(), String> {
        loop {
            require_live_deadline(deadline)?;
            if let Some(event) = self.next_event()? {
                if let Event::PropertyNotify(property) = &event
                    && property.window == requestor
                    && property.atom == self.atoms.receive
                    && property.state == Property::NEW_VALUE
                {
                    return Ok(());
                }
                self.service_event(event)?;
                continue;
            }
            properties::wait_selection(&self.connection, deadline)?;
        }
    }
}
pub(super) fn require_live_deadline(deadline: Instant) -> Result<(), String> {
    if Instant::now() >= deadline {
        return Err(
            "clipboard transfer deadline expired: expected a responsive selection peer".into(),
        );
    }
    Ok(())
}
fn classify_text_target(property: &GetPropertyReply, utf8: Atom) -> Result<TextEncoding, String> {
    if property.type_ != u32::from(AtomEnum::ATOM) || property.format != 32 {
        return Err(format!(
            "clipboard TARGETS type {}, format {}: expected ATOM/32",
            property.type_, property.format
        ));
    }
    let targets: Vec<u32> = property
        .value32()
        .ok_or("clipboard TARGETS: expected complete atom words")?
        .collect();
    if targets.contains(&utf8) {
        return Ok(TextEncoding::Utf8);
    }
    if targets.contains(&AtomEnum::STRING.into()) {
        return Ok(TextEncoding::Latin1);
    }
    Err(format!(
        "clipboard targets {targets:?}: expected UTF8_STRING or STRING text"
    ))
}
fn require_text_property(target: Atom, property: &GetPropertyReply) -> Result<(), String> {
    if property.type_ != target || property.format != 8 {
        return Err(format!(
            "clipboard type {}, format {}: expected target {target} with format 8",
            property.type_, property.format
        ));
    }
    Ok(())
}
fn decode_text_property(
    encoding: TextEncoding,
    target: Atom,
    property: &GetPropertyReply,
    limit: usize,
) -> Result<String, String> {
    require_text_property(target, property)?;
    decode_text(encoding, &property.value, limit)
}
fn incremental_lower_bound(header: &GetPropertyReply) -> Result<usize, String> {
    let words: Vec<u32> = header.value32().into_iter().flatten().collect();
    if header.format != 32 || words.len() != 1 {
        return Err(format!(
            "clipboard INCR format {}, words {words:?}: expected one 32-bit lower bound",
            header.format
        ));
    }
    Ok(words[0] as usize)
}

#[cfg(test)]
#[path = "../../../tests/clipboard/receive.rs"]
mod tests;
