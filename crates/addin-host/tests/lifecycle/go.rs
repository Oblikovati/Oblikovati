// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::{collections::BTreeMap, ffi::CStr, path::PathBuf, sync::Mutex};
struct EchoHostBuffers {
    live: BTreeMap<usize, Box<[u8]>>,
    freed: usize,
}
static ECHO_HOST: Mutex<EchoHostBuffers> = Mutex::new(EchoHostBuffers {
    live: BTreeMap::new(),
    freed: 0,
});
unsafe extern "C" fn echo_call(
    method: *const c_char,
    request: *const u8,
    length: i32,
    reply: *mut *mut u8,
    reply_length: *mut i32,
) -> i32 {
    let caught = std::panic::catch_unwind(|| unsafe {
        echo_reply(method, request, length, reply, reply_length)
    });
    caught.unwrap_or(ffi::OBK_ERR)
}
unsafe fn echo_reply(
    method: *const c_char,
    request: *const u8,
    length: i32,
    reply: *mut *mut u8,
    reply_length: *mut i32,
) -> i32 {
    if method.is_null() || reply.is_null() || reply_length.is_null() || length < 0 {
        return ffi::OBK_ERR;
    }
    if request.is_null() && length != 0 {
        return ffi::OBK_ERR;
    }
    // SAFETY: trusted Go echo fixture supplies readable method/request and writable output storage.
    if unsafe { CStr::from_ptr(method) }.to_bytes() != b"echo" {
        return ffi::OBK_ERR;
    }
    let bytes = if length == 0 {
        Vec::new()
    } else {
        unsafe { std::slice::from_raw_parts(request, length as usize) }.to_vec()
    };
    let Ok(mut buffers) = ECHO_HOST.lock() else {
        return ffi::OBK_ERR;
    };
    let mut bytes = bytes.into_boxed_slice();
    let pointer = bytes.as_mut_ptr();
    buffers.live.insert(pointer as usize, bytes);
    unsafe {
        *reply = pointer;
        *reply_length = length;
    }
    ffi::OBK_OK
}
unsafe extern "C" fn echo_free(pointer: *mut u8) {
    let Ok(mut buffers) = ECHO_HOST.lock() else {
        return;
    };
    if buffers.live.remove(&(pointer as usize)).is_some() {
        buffers.freed += 1;
    }
}
#[test]
fn real_go_activation_reads_and_frees_host_response_then_automation_returns_addin_owned_bytes() {
    let compiler = FixtureCompiler::new();
    let reference = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("../../../Oblikovati/head/internal/addinhost/testdata/echoaddin/main.go");
    let path = compiler.go_library(&reference);
    let mut addin = open(&path);
    let before = ECHO_HOST.lock().unwrap().freed;
    // SAFETY: the fixed Go fixture calls only echo; static callbacks and buffer store remain live through cleanup.
    unsafe { addin.activate(echo_call, echo_free) }.unwrap();
    assert_eq!(addin.state(), AddInState::Active);
    assert_host_free(before);
    assert_eq!(
        automation(&mut addin, "echo", b"ping").unwrap(),
        br#"{"method":"echo","payload":"ping"}"#
    );
    // SAFETY: Go fixture synchronously consumes notification bytes and retains no pointer.
    unsafe { addin.notify(b"event") }.unwrap();
    deactivation(&mut addin).unwrap();
    assert_eq!(addin.state(), AddInState::Inactive);
}
fn assert_host_free(before: usize) {
    let buffers = ECHO_HOST.lock().unwrap();
    assert_eq!(buffers.freed, before + 1);
    assert!(buffers.live.is_empty());
}
const _: ffi::ObkHostCall = echo_call;
const _: ffi::ObkHostFree = echo_free;
