// SPDX-License-Identifier: GPL-2.0-only
use super::*;
struct NameReader<'a> {
    remaining: &'a [u8],
}
pub(super) fn payload(envelope: ReferenceEnvelope<'_>) -> Result<SemanticReferenceName, String> {
    let mut reader = NameReader {
        remaining: envelope.payload,
    };
    let guid = reader
        .take(16)?
        .try_into()
        .expect("sixteen validated UUID bytes");
    let document = DocumentGuid(uuid::Uuid::from_bytes(guid));
    let operation = reader.tag()?;
    let role = reader.tag()?;
    let sources = reader.sources()?;
    if !reader.remaining.is_empty() {
        return Err(format!(
            "semantic payload trailing bytes {}: expected none",
            reader.remaining.len()
        ));
    }
    SemanticReferenceName::new(
        document,
        envelope.context,
        envelope.kind,
        operation,
        role,
        &sources,
    )
}
impl<'a> NameReader<'a> {
    fn take(&mut self, count: usize) -> Result<&'a [u8], String> {
        if self.remaining.len() < count {
            return Err(format!(
                "semantic payload remaining {} bytes: expected at least {count}",
                self.remaining.len()
            ));
        }
        let (value, remaining) = self.remaining.split_at(count);
        self.remaining = remaining;
        Ok(value)
    }
    fn count(&mut self) -> Result<usize, String> {
        let bytes = self.take(2)?.try_into().expect("two validated count bytes");
        Ok(u16::from_le_bytes(bytes) as usize)
    }
    fn tag(&mut self) -> Result<&'a str, String> {
        let count = self.count()?;
        if !(1..=64).contains(&count) {
            return Err(format!("semantic tag length {count}: expected 1..64 bytes"));
        }
        std::str::from_utf8(self.take(count)?)
            .map_err(|cause| format!("semantic tag: expected ASCII text: {cause}"))
    }
    fn sources(&mut self) -> Result<Vec<u64>, String> {
        let count = self.count()?;
        if count > 64 {
            return Err(format!(
                "semantic sources count {count}: expected at most 64"
            ));
        }
        let mut sources = Vec::with_capacity(count);
        for _ in 0..count {
            let bytes = self
                .take(8)?
                .try_into()
                .expect("eight validated source bytes");
            sources.push(u64::from_le_bytes(bytes));
        }
        Ok(sources)
    }
}
