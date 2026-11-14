// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::{collections::BTreeMap, ffi::c_int};
#[derive(Default)]
struct BufferArena {
    live: BTreeMap<usize, Box<[u8]>>,
    refused: usize,
}
static HOST_BUFFERS: Mutex<BufferArena> = Mutex::new(BufferArena {
    live: BTreeMap::new(),
    refused: 0,
});
/// Native transport diagnostics, not a public API wire DTO.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct HostBufferCounts {
    pub outstanding: usize,
    pub refused_frees: usize,
}
pub(super) fn counts() -> HostBufferCounts {
    let arena = HOST_BUFFERS
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
    HostBufferCounts {
        outstanding: arena.live.len(),
        refused_frees: arena.refused,
    }
}
pub(super) fn allocate(bytes: Vec<u8>) -> Result<(*mut u8, c_int), Error> {
    let length = response::c_length(bytes.len())?;
    let mut storage = bytes;
    storage.push(0); // Preserve Go's always-allocated response ownership, including an empty success.
    let mut storage = storage.into_boxed_slice();
    let pointer = storage.as_mut_ptr();
    HOST_BUFFERS
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner)
        .live
        .insert(pointer as usize, storage);
    Ok((pointer, length))
}
pub(super) unsafe extern "C" fn host_free(pointer: *mut u8) {
    if pointer.is_null() {
        return;
    }
    // The pointer is only a lookup key. Unknown or duplicate frees never create a Rust reference.
    let mut arena = HOST_BUFFERS
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
    if arena.live.remove(&(pointer as usize)).is_none() {
        arena.refused = arena.refused.saturating_add(1);
    }
}
pub(super) unsafe fn write_reply(
    result: Result<Vec<u8>, Error>,
    reply: *mut *mut u8,
    length: *mut c_int,
) -> c_int {
    let allocated = match result {
        Ok(bytes) => allocate(bytes).map(|slots| (ffi::OBK_OK, slots)),
        Err(error) => allocate(error.to_string().into_bytes()).map(|slots| (ffi::OBK_ERR, slots)),
    };
    let allocated = allocated.or_else(|error| {
        allocate(error.to_string().into_bytes()).map(|slots| (ffi::OBK_ERR, slots))
    });
    let Ok((status, (pointer, count))) = allocated else {
        return ffi::OBK_ERR;
    };
    // SAFETY: entry validates nonnull output slots; their writable extent is the trusted C caller's obligation.
    unsafe {
        *reply = pointer;
        *length = count;
    }
    status
}
