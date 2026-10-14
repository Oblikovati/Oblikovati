// SPDX-License-Identifier: GPL-2.0-only
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum TextEncoding {
    Utf8,
    Latin1,
}
#[derive(Debug, PartialEq, Eq)]
pub(super) enum ReceiveStep {
    Pending,
    Complete(String),
}
pub(super) struct IncrementalText {
    encoding: TextEncoding,
    minimum: usize,
    limit: usize,
    received: Vec<u8>,
    finished: bool,
}
impl IncrementalText {
    pub(super) fn begin(
        encoding: TextEncoding,
        minimum: usize,
        limit: usize,
    ) -> Result<Self, String> {
        require_byte_budget(minimum, limit)?;
        Ok(Self {
            encoding,
            minimum,
            limit,
            received: Vec::new(),
            finished: false,
        })
    }
    pub(super) fn accept(
        &mut self,
        encoding: TextEncoding,
        chunk: &[u8],
    ) -> Result<ReceiveStep, String> {
        let outcome = self.accept_chunk(encoding, chunk);
        if outcome.is_err() {
            self.finished = true;
        }
        outcome
    }
    fn accept_chunk(
        &mut self,
        encoding: TextEncoding,
        chunk: &[u8],
    ) -> Result<ReceiveStep, String> {
        if self.finished || encoding != self.encoding {
            return Err(format!(
                "clipboard finished={}, encoding={encoding:?}: expected active {:?} transfer",
                self.finished, self.encoding
            ));
        }
        if chunk.is_empty() {
            return self.complete();
        }
        self.require_chunk_capacity(chunk.len())?;
        self.received.extend_from_slice(chunk);
        Ok(ReceiveStep::Pending)
    }
    fn require_chunk_capacity(&self, chunk_length: usize) -> Result<(), String> {
        let total = self
            .received
            .len()
            .checked_add(chunk_length)
            .ok_or_else(|| {
                format!("clipboard chunk length {chunk_length}: expected representable total")
            })?;
        require_byte_budget(total, self.limit)
    }
    fn complete(&mut self) -> Result<ReceiveStep, String> {
        self.finished = true;
        if self.received.len() < self.minimum {
            return Err(format!(
                "clipboard length {}: expected at least {} advertised bytes",
                self.received.len(),
                self.minimum
            ));
        }
        decode_text(self.encoding, &self.received, self.limit).map(ReceiveStep::Complete)
    }
}
fn require_byte_budget(length: usize, limit: usize) -> Result<(), String> {
    if length > limit {
        return Err(format!(
            "clipboard length {length}: expected at most {limit} bytes"
        ));
    }
    Ok(())
}
pub(super) fn decode_text(
    encoding: TextEncoding,
    bytes: &[u8],
    limit: usize,
) -> Result<String, String> {
    require_byte_budget(bytes.len(), limit)?;
    if let Some(offset) = bytes.iter().position(|byte| *byte == 0) {
        return Err(format!(
            "clipboard NUL at byte {offset}: expected C-string-safe text"
        ));
    }
    match encoding {
        TextEncoding::Utf8 => std::str::from_utf8(bytes)
            .map(str::to_owned)
            .map_err(|cause| format!("clipboard UTF-8 {cause}: expected complete valid text")),
        TextEncoding::Latin1 => decode_latin1(bytes, limit),
    }
}

fn decode_latin1(bytes: &[u8], limit: usize) -> Result<String, String> {
    let decoded = bytes
        .iter()
        .try_fold(0_usize, |length, byte| {
            length.checked_add(char::from(*byte).len_utf8())
        })
        .ok_or_else(|| {
            format!(
                "clipboard Latin1 length {}: expected representable UTF-8 length",
                bytes.len()
            )
        })?;
    require_byte_budget(decoded, limit)?;
    Ok(bytes.iter().map(|byte| char::from(*byte)).collect())
}
#[cfg(test)]
#[path = "../../../tests/clipboard/text.rs"]
mod tests;
