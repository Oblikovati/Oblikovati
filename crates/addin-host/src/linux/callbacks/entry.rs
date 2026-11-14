// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::{
    ffi::{c_char, c_int},
    panic::{AssertUnwindSafe, catch_unwind},
};
struct CopiedRequest {
    method: String,
    arguments: Vec<u8>,
}
pub(super) unsafe extern "C" fn host_call(
    method: *const c_char,
    arguments: *const u8,
    count: c_int,
    reply: *mut *mut u8,
    reply_length: *mut c_int,
) -> c_int {
    if reply.is_null() || reply_length.is_null() {
        return ffi::OBK_ERR;
    }
    // SAFETY: trusted callers provide writable aligned output slots; initialize them before any fallible work.
    unsafe {
        *reply = std::ptr::null_mut();
        *reply_length = 0;
    }
    let caught = catch_unwind(AssertUnwindSafe(|| unsafe {
        invoke(method, arguments, count)
    }));
    let outcome = match caught {
        Ok(result) => result,
        Err(_) => {
            retire_endpoint();
            Err(Error(
                "host dispatch panicked: expected a normal result; callback admission retired"
                    .into(),
            ))
        }
    };
    unsafe { buffers::write_reply(outcome, reply, reply_length) }
}
unsafe fn invoke(
    method: *const c_char,
    arguments: *const u8,
    count: c_int,
) -> Result<Vec<u8>, Error> {
    let request = unsafe { copy_request(method, arguments, count)? };
    let lease = lease_endpoint()?;
    lease.0.transport.call(&request.method, &request.arguments)
}
unsafe fn copy_request(
    method: *const c_char,
    arguments: *const u8,
    count: c_int,
) -> Result<CopiedRequest, Error> {
    validate_shape(method, arguments, count)?;
    // SAFETY: null/length guards precede references; NUL termination and readable extents remain C caller obligations.
    let bytes = unsafe { CStr::from_ptr(method) };
    let method = bytes
        .to_str()
        .map_err(|error| {
            Error(format!(
                "host method bytes {:?}: expected UTF-8; {error}",
                bytes.to_bytes()
            ))
        })?
        .to_owned();
    let arguments = if count == 0 {
        Vec::new()
    } else {
        unsafe { std::slice::from_raw_parts(arguments, count as usize) }.to_vec()
    };
    Ok(CopiedRequest { method, arguments })
}
fn validate_shape(method: *const c_char, arguments: *const u8, count: c_int) -> Result<(), Error> {
    if method.is_null() || count < 0 || (arguments.is_null() && count > 0) {
        return Err(Error(format!(
            "host request method {method:?}, payload {arguments:?}, C length {count}: expected nonnull method, nonnegative length and live nonempty payload"
        )));
    }
    Ok(())
}
