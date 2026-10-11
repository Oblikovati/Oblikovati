// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
#![allow(unsafe_code)]
use oblikovati_addin_host::HostCallbackBinding;
use oblikovati_api::{Error, client::Caller, ffi};
use std::{ffi::CStr, sync::Arc};
#[path = "callbacks/cases.rs"]
mod cases;
#[path = "support/compiler.rs"]
mod compiler;
#[path = "callbacks/queue.rs"]
mod queue;
fn isolated_case(test: &str) -> bool {
    isolation::case("OBK_CALLBACK_CASE", test)
}
fn callback(
    call: ffi::ObkHostCall,
    free: ffi::ObkHostFree,
    method: &CStr,
    bytes: &[u8],
) -> (i32, Vec<u8>) {
    let mut pointer = std::ptr::null_mut();
    let mut length = 0;
    // SAFETY: inputs and outputs remain live, and these are the host's canonical C function pointers.
    let status = unsafe {
        call(
            method.as_ptr(),
            bytes.as_ptr(),
            bytes.len() as i32,
            &mut pointer,
            &mut length,
        )
    };
    assert!(!pointer.is_null());
    assert!(length >= 0);
    let output = unsafe { std::slice::from_raw_parts(pointer, length as usize) }.to_vec();
    unsafe { free(pointer) };
    (status, output)
}
struct EchoCaller;
impl Caller for EchoCaller {
    fn call(&self, _: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        Ok(request.to_vec())
    }
}
fn binding() -> HostCallbackBinding {
    HostCallbackBinding::install(Arc::new(EchoCaller)).unwrap()
}

#[path = "support/isolation.rs"]
mod isolation;

#[path = "support/resources.rs"]
mod resources;
