// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::ffi::CString;
pub(super) struct NativeCallbackCaller {
    call: ffi::ObkHostCall,
    free: ffi::ObkHostFree,
}
impl NativeCallbackCaller {
    pub(super) fn new(callbacks: (ffi::ObkHostCall, ffi::ObkHostFree)) -> Self {
        Self {
            call: callbacks.0,
            free: callbacks.1,
        }
    }
}
impl Caller for NativeCallbackCaller {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        let method = CString::new(method).map_err(|error| Error(error.to_string()))?;
        let count = i32::try_from(request.len()).map_err(|error| Error(error.to_string()))?;
        let mut pointer = std::ptr::null_mut();
        let mut length = 0;
        // SAFETY: constructor receives only the registered host's functions; all input/output storage remains live.
        let status = unsafe { self.invoke(&method, request, count, &mut pointer, &mut length) };
        assert!(!pointer.is_null() && length >= 0);
        let reply = unsafe { std::slice::from_raw_parts(pointer, length as usize) }.to_vec();
        unsafe { (self.free)(pointer) };
        if status == ffi::OBK_OK {
            return Ok(reply);
        }
        Err(Error(String::from_utf8(reply).unwrap()))
    }
}

impl NativeCallbackCaller {
    unsafe fn invoke(
        &self,
        method: &std::ffi::CStr,
        request: &[u8],
        count: i32,
        pointer: &mut *mut u8,
        length: &mut i32,
    ) -> i32 {
        // SAFETY: caller keeps every referenced input and output alive through the canonical C callback.
        unsafe { (self.call)(method.as_ptr(), request.as_ptr(), count, pointer, length) }
    }
}
