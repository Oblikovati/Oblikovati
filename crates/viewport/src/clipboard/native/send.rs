// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::collections::BTreeMap;
#[derive(Clone)]
enum SelectionBytes {
    Utf8(Rc<str>),
    Latin1(Rc<[u8]>),
}
impl SelectionBytes {
    fn incremental_length(&self) -> Result<u32, String> {
        u32::try_from(self.bytes().len()).map_err(|_| {
            format!(
                "clipboard length {}: expected a 32-bit INCR lower bound",
                self.bytes().len()
            )
        })
    }
    fn bytes(&self) -> &[u8] {
        match self {
            Self::Utf8(text) => text.as_bytes(),
            Self::Latin1(bytes) => bytes,
        }
    }
}
struct OutboundText {
    target: Atom,
    snapshot: SelectionBytes,
    offset: usize,
    deadline: Instant,
}
struct RequestorMask {
    previous: EventMask,
    transfers: usize,
}
#[derive(Default)]
pub(super) struct OutboundTransfers {
    texts: BTreeMap<(Window, Atom), OutboundText>,
    masks: BTreeMap<Window, RequestorMask>,
}
impl OutboundTransfers {
    pub(super) fn len(&self) -> usize {
        self.texts.len()
    }
    pub(super) fn contains(&self, key: (Window, Atom)) -> bool {
        self.texts.contains_key(&key)
    }
    fn expired(&self, now: Instant) -> Vec<(Window, Atom)> {
        self.texts
            .iter()
            .filter_map(|(key, text)| (text.deadline <= now).then_some(*key))
            .collect()
    }
    pub(super) fn deadline(&self) -> Option<Instant> {
        self.texts.values().map(|text| text.deadline).min()
    }
}
impl SelectionEndpoint {
    pub(super) fn write_selection_text(
        &self,
        window: Window,
        property: Atom,
        target: Atom,
        snapshot: Rc<str>,
    ) -> Result<bool, String> {
        let payload = encode_selection(target, self.atoms.utf8, snapshot)?;
        let capacity = self.chunk_capacity()?;
        if payload.bytes().len() <= capacity {
            return self.write_bytes(window, property, target, payload.bytes());
        }
        if self.outbound.borrow().len() >= self.options.transfer_limit {
            return Ok(false);
        }
        self.begin_incremental(window, property, target, payload)
    }
    fn chunk_capacity(&self) -> Result<usize, String> {
        // X11 ChangeProperty has six 32-bit header words. Stay within the core request size.
        let capacity = usize::from(self.connection.setup().maximum_request_length)
            .checked_sub(6)
            .and_then(|words| words.checked_mul(4))
            .filter(|bytes| *bytes > 0)
            .ok_or_else(|| {
                format!(
                    "clipboard server request words {}: expected room for property payload",
                    self.connection.setup().maximum_request_length
                )
            })?;
        Ok(capacity.min(self.options.chunk_limit))
    }
    fn begin_incremental(
        &self,
        window: Window,
        property: Atom,
        target: Atom,
        snapshot: SelectionBytes,
    ) -> Result<bool, String> {
        let key = (window, property);
        if self.outbound.borrow().texts.contains_key(&key) {
            return Ok(false);
        }
        let length = snapshot.incremental_length()?;
        let text = OutboundText {
            target,
            snapshot,
            offset: 0,
            deadline: self.operation_deadline()?,
        };
        self.retain_requestor_mask(window)?;
        if let Err(cause) = self.write_words(window, property, self.atoms.incr, &[length]) {
            let cleanup = self.release_requestor_mask(window);
            return Err(combine_send_failure(cause, cleanup));
        }
        self.outbound.borrow_mut().texts.insert(key, text);
        Ok(true)
    }
    fn retain_requestor_mask(&self, window: Window) -> Result<(), String> {
        if let Some(mask) = self.outbound.borrow_mut().masks.get_mut(&window) {
            mask.transfers += 1;
            return Ok(());
        }
        let previous = self.requestor_event_mask(window)?;
        self.set_requestor_mask(
            window,
            previous | EventMask::PROPERTY_CHANGE | EventMask::STRUCTURE_NOTIFY,
        )?;
        self.outbound.borrow_mut().masks.insert(
            window,
            RequestorMask {
                previous,
                transfers: 1,
            },
        );
        Ok(())
    }
    fn requestor_event_mask(&self, window: Window) -> Result<EventMask, String> {
        self.connection
            .get_window_attributes(window)
            .map_err(|cause| clipboard_failure("requestor mask", cause))?
            .reply()
            .map(|reply| reply.your_event_mask)
            .map_err(|cause| clipboard_failure("requestor mask reply", cause))
    }
    fn set_requestor_mask(&self, window: Window, mask: EventMask) -> Result<(), String> {
        self.connection
            .change_window_attributes(window, &ChangeWindowAttributesAux::new().event_mask(mask))
            .map_err(|cause| clipboard_failure("requestor event mask", cause))?
            .check()
            .map_err(|cause| clipboard_failure("requestor event mask reply", cause))
    }
    fn release_requestor_mask(&self, window: Window) -> Result<(), String> {
        let mut outbound = self.outbound.borrow_mut();
        let Some(mask) = outbound.masks.get_mut(&window) else {
            return Ok(());
        };
        mask.transfers -= 1;
        if mask.transfers != 0 {
            return Ok(());
        }
        let previous = mask.previous;
        outbound.masks.remove(&window);
        drop(outbound);
        self.set_requestor_mask(window, previous)
    }
    pub(super) fn send_next_chunk(&self, property: PropertyNotifyEvent) -> Result<(), String> {
        let key = (property.window, property.atom);
        let Some(text) = self.outbound.borrow_mut().texts.remove(&key) else {
            return Ok(());
        };
        if Instant::now() >= text.deadline {
            let cleanup = self.release_requestor_mask(property.window);
            return Err(combine_send_failure(
                format!("clipboard transfer {key:?}: expected completion before deadline"),
                cleanup,
            ));
        }
        self.publish_next_chunk(key, text)
    }
    fn publish_next_chunk(
        &self,
        key: (Window, Atom),
        mut text: OutboundText,
    ) -> Result<(), String> {
        let capacity = self.chunk_capacity()?;
        let end = text
            .offset
            .saturating_add(capacity)
            .min(text.snapshot.bytes().len());
        let bytes = &text.snapshot.bytes()[text.offset..end];
        self.write_bytes(key.0, key.1, text.target, bytes)
            .map_err(|cause| combine_send_failure(cause, self.release_requestor_mask(key.0)))?;
        if bytes.is_empty() {
            return self.release_requestor_mask(key.0);
        }
        text.offset = end;
        self.outbound.borrow_mut().texts.insert(key, text);
        Ok(())
    }
    pub(super) fn expire_transfers(&self) -> Result<(), String> {
        let now = Instant::now();
        let keys = self.outbound.borrow().expired(now);
        let mut issues = Vec::new();
        for key in keys {
            self.outbound.borrow_mut().texts.remove(&key);
            let cause =
                format!("clipboard transfer {key:?}: expected acknowledgement before deadline");
            issues.push(combine_send_failure(
                cause,
                self.release_requestor_mask(key.0),
            ));
        }
        if issues.is_empty() {
            return Ok(());
        }
        Err(issues.join("; "))
    }
    pub(super) fn forget_requestor(&self, window: Window) {
        let mut outbound = self.outbound.borrow_mut();
        outbound
            .texts
            .retain(|(requestor, _), _| *requestor != window);
        outbound.masks.remove(&window);
    }
}
fn combine_send_failure(cause: String, cleanup: Result<(), String>) -> String {
    match cleanup {
        Ok(()) => cause,
        Err(cleanup) => format!("{cause}; requestor cleanup: {cleanup}"),
    }
}

fn encode_selection(target: Atom, utf8: Atom, snapshot: Rc<str>) -> Result<SelectionBytes, String> {
    if target == utf8 {
        return Ok(SelectionBytes::Utf8(snapshot));
    }
    let bytes = snapshot
        .chars()
        .map(|character| u8::try_from(u32::from(character)))
        .collect::<Result<Vec<_>, _>>()
        .map_err(|cause| {
            format!(
                "clipboard target {target}, Latin1 conversion {cause}: expected representable text"
            )
        })?;
    Ok(SelectionBytes::Latin1(bytes.into()))
}
