// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::ffi::c_int;
pub(super) fn c_length(length: usize) -> Result<c_int, Error> {
    c_int::try_from(length).map_err(|_| {
        Error(format!(
            "C payload length {length}: expected 0..={}",
            c_int::MAX
        ))
    })
}
pub(super) fn payload_pointer(bytes: &[u8]) -> *const u8 {
    if bytes.is_empty() {
        return std::ptr::null();
    }
    bytes.as_ptr()
}
struct AddInReply {
    pointer: *mut u8,
    length: c_int,
    release: ffi::ObkFree,
}
impl Drop for AddInReply {
    fn drop(&mut self) {
        if self.pointer.is_null() {
            return;
        }
        // SAFETY: trusted automation returns add-in-owned storage; it is released once by the matching export.
        unsafe { (self.release)(self.pointer) };
    }
}
impl AddInReply {
    unsafe fn copy(&self) -> Result<Vec<u8>, Error> {
        if self.length < 0 || (self.pointer.is_null() && self.length != 0) {
            return Err(Error(format!(
                "automation reply pointer {:?}, C length {}: expected nonnegative length and live storage for nonempty reply",
                self.pointer, self.length
            )));
        }
        if self.length == 0 {
            return Ok(Vec::new());
        }
        // SAFETY: signed length and null checks precede reference creation; remaining extent is the trusted ABI obligation.
        Ok(unsafe { std::slice::from_raw_parts(self.pointer, self.length as usize) }.to_vec())
    }
}
pub(super) unsafe fn invoke(
    export: ffi::ObkAddInAutomation,
    free: ffi::ObkFree,
    method: &CStr,
    arguments: &[u8],
    length: c_int,
) -> Result<Vec<u8>, Error> {
    let mut reply = AddInReply {
        pointer: std::ptr::null_mut(),
        length: 0,
        release: free,
    };
    // SAFETY: imported prototypes and live inputs; output slots are aligned writable local storage.
    let status = unsafe {
        export(
            method.as_ptr(),
            payload_pointer(arguments),
            length,
            &mut reply.pointer,
            &mut reply.length,
        )
    };
    let copied = unsafe { reply.copy()? };
    if status == ffi::OBK_OK {
        return Ok(copied);
    }
    let error_text = String::from_utf8(copied).map_err(|error| {
        Error(format!(
            "automation {method:?} status {status} error bytes {:?}: expected UTF-8",
            error.as_bytes()
        ))
    })?;
    Err(Error(format!(
        "automation {method:?} returned {status}: {error_text}; expected OBK_OK ({})",
        ffi::OBK_OK
    )))
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn lengths_and_empty_pointer_follow_c_int_shape() {
        assert_eq!(c_length(0).unwrap(), 0);
        assert_eq!(c_length(c_int::MAX as usize).unwrap(), c_int::MAX);
        assert!(c_length(c_int::MAX as usize + 1).is_err());
        assert!(payload_pointer(&[]).is_null());
        let payload = [1, 2];
        assert_eq!(payload_pointer(&payload), payload.as_ptr());
    }
}
