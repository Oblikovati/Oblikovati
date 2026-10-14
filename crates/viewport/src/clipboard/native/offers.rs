// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::collections::BTreeSet;
struct OfferedWords {
    kind: Atom,
    values: Vec<u32>,
}
impl SelectionEndpoint {
    pub(super) fn write_selection_property(
        &self,
        request: SelectionRequestEvent,
        property: Atom,
    ) -> Result<bool, String> {
        if request.owner != self.window || request.selection != self.atoms.clipboard {
            return Ok(false);
        }
        if self
            .outbound
            .borrow()
            .contains((request.requestor, property))
        {
            return Ok(false);
        }
        let Some(text) = self.snapshot.borrow().clone() else {
            return Ok(false);
        };
        if request.target == self.atoms.multiple {
            return self.write_multiple(request, property, text);
        }
        self.write_offered_target(request.requestor, property, request.target, text)
    }
    fn offered_targets(&self, text: &str) -> Vec<Atom> {
        let mut targets = vec![
            self.atoms.targets,
            self.atoms.timestamp,
            self.atoms.multiple,
            self.atoms.utf8,
            self.atoms.save_targets,
            self.atoms.target_sizes,
        ];
        if text
            .chars()
            .all(|character| u8::try_from(u32::from(character)).is_ok())
        {
            targets.push(AtomEnum::STRING.into());
        }
        targets
    }
    fn write_offered_target(
        &self,
        window: Window,
        property: Atom,
        target: Atom,
        text: Rc<str>,
    ) -> Result<bool, String> {
        if let Some(words) = self.offered_metadata(target, &text) {
            return self.write_words(window, property, words.kind, &words.values);
        }
        let is_text = target == self.atoms.utf8
            || target == u32::from(AtomEnum::STRING)
                && self.offered_targets(&text).contains(&target);
        if !is_text {
            return Ok(false);
        }
        self.write_selection_text(window, property, target, text)
    }
    fn offered_metadata(&self, target: Atom, text: &str) -> Option<OfferedWords> {
        let (kind, values) = match target {
            target if target == self.atoms.save_targets => (self.atoms.null, Vec::new()),
            target if target == self.atoms.timestamp => {
                (AtomEnum::INTEGER.into(), vec![self.acquired.get()])
            }
            target if target == self.atoms.targets => {
                (AtomEnum::ATOM.into(), self.offered_targets(text))
            }
            target if target == self.atoms.target_sizes => {
                (AtomEnum::ATOM.into(), self.offered_target_sizes(text))
            }
            _ => return None,
        };
        Some(OfferedWords { kind, values })
    }
    fn offered_target_sizes(&self, text: &str) -> Vec<u32> {
        let targets = self.offered_targets(text);
        targets
            .iter()
            .flat_map(|target| {
                let bytes = self.offered_target_size(*target, text, targets.len());
                [*target, bytes]
            })
            .collect()
    }
    fn offered_target_size(&self, target: Atom, text: &str, count: usize) -> u32 {
        if target == self.atoms.save_targets {
            return u32::MAX; // SAVE_TARGETS is a side-effect marker; the specification encodes -1.
        }
        let bytes = match target {
            target if target == self.atoms.targets => count * 4,
            target if target == self.atoms.timestamp => 4,
            target if target == self.atoms.target_sizes => count * 8,
            target if target == self.atoms.utf8 => text.len(),
            target if target == u32::from(AtomEnum::STRING) => text.chars().count(),
            _ => 0,
        };
        i32::try_from(bytes).map(|bytes| bytes as u32).unwrap_or(0)
    }
    fn write_multiple(
        &self,
        request: SelectionRequestEvent,
        property: Atom,
        text: Rc<str>,
    ) -> Result<bool, String> {
        let mut pairs = self.read_multiple_pairs(request.requestor, property)?;
        let mut destinations = BTreeSet::new();
        for pair in pairs.as_chunks_mut::<2>().0 {
            if !self.accept_multiple_destination(
                request.requestor,
                pair,
                property,
                &mut destinations,
            ) || !self.write_offered_target(request.requestor, pair[1], pair[0], text.clone())?
            {
                pair[1] = 0;
            }
        }
        self.write_words(request.requestor, property, self.atoms.atom_pair, &pairs)
    }
    fn accept_multiple_destination(
        &self,
        window: Window,
        pair: &[u32],
        list_property: Atom,
        used: &mut BTreeSet<Atom>,
    ) -> bool {
        pair[0] != self.atoms.multiple
            && pair[1] != 0
            && pair[1] != list_property
            && used.insert(pair[1])
            && !self.outbound.borrow().contains((window, pair[1]))
    }
    fn read_multiple_pairs(&self, window: Window, property: Atom) -> Result<Vec<u32>, String> {
        let words = self
            .options
            .target_limit
            .checked_mul(2)
            .and_then(|count| u32::try_from(count).ok())
            .ok_or_else(|| {
                format!(
                    "clipboard target limit {}: expected representable MULTIPLE pair count",
                    self.options.target_limit
                )
            })?;
        let reply =
            self.read_native_property(false, window, property, self.atoms.atom_pair, words)?;
        decode_multiple_pairs(reply, self.atoms.atom_pair)
    }
}
fn decode_multiple_pairs(reply: GetPropertyReply, expected: Atom) -> Result<Vec<u32>, String> {
    let pairs: Vec<u32> = reply.value32().into_iter().flatten().collect();
    if reply.type_ != expected
        || reply.format != 32
        || reply.bytes_after != 0
        || !pairs.len().is_multiple_of(2)
    {
        return Err(format!(
            "clipboard MULTIPLE type {}, format {}, remaining {}, words {}: expected bounded ATOM_PAIR/32 pairs",
            reply.type_,
            reply.format,
            reply.bytes_after,
            pairs.len()
        ));
    }
    Ok(pairs)
}
