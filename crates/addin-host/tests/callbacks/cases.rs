// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn success_empty_error_and_retired_calls_own_responses_until_matching_free() {
    if isolated_case(
        "cases::success_empty_error_and_retired_calls_own_responses_until_matching_free",
    ) {
        return;
    }
    let binding = binding();
    let (call, free) = binding.callbacks();
    verify_success_replies(call, free);
    assert_eq!(binding.in_flight(), 0);
    assert_eq!(binding.buffer_counts().outstanding, 0);
    assert!(HostCallbackBinding::install(Arc::new(EchoCaller)).is_err());
    binding.close();
    let (status, error) = callback(call, free, c"closed", b"");
    assert_eq!(status, ffi::OBK_ERR);
    assert!(String::from_utf8(error).unwrap().contains("retired"));
    assert!(HostCallbackBinding::install(Arc::new(EchoCaller)).is_err());
    assert_eq!(binding.buffer_counts().outstanding, 0);
}
#[test]
fn malformed_native_inputs_decline_before_transport_and_return_owned_errors() {
    if isolated_case(
        "cases::malformed_native_inputs_decline_before_transport_and_return_owned_errors",
    ) {
        return;
    }
    let binding = binding();
    let (call, free) = binding.callbacks();
    for (method, pointer, count) in [
        (std::ptr::null(), std::ptr::null(), 0),
        (c"bad".as_ptr(), std::ptr::null(), -1),
        (c"bad".as_ptr(), std::ptr::null(), 2),
    ] {
        verify_malformed(call, free, method, pointer, count);
    }
    verify_encoding(call, free);
    assert_eq!(binding.buffer_counts().outstanding, 0);
}
fn verify_malformed(
    call: ffi::ObkHostCall,
    free: ffi::ObkHostFree,
    method: *const std::ffi::c_char,
    pointer: *const u8,
    count: i32,
) {
    let mut response = std::ptr::null_mut();
    let mut length = 0;
    // SAFETY: invalid scalar/null combinations are refused before references; output slots remain valid.
    assert_eq!(
        unsafe { call(method, pointer, count, &mut response, &mut length) },
        ffi::OBK_ERR
    );
    assert!(length > 0);
    let error = unsafe { std::slice::from_raw_parts(response, length as usize) };
    assert!(String::from_utf8_lossy(error).contains("expected"));
    unsafe { free(response) };
}
#[test]
fn invalid_output_slots_and_unknown_frees_do_not_dereference_native_pointers() {
    if isolated_case(
        "cases::invalid_output_slots_and_unknown_frees_do_not_dereference_native_pointers",
    ) {
        return;
    }
    let binding = binding();
    let (call, free) = binding.callbacks();
    verify_invalid_output_and_free(&binding, call, free);
    assert_eq!(binding.buffer_counts().outstanding, 0);
}
struct PanickingCaller;
impl Caller for PanickingCaller {
    fn call(&self, _: &str, _: &[u8]) -> Result<Vec<u8>, Error> {
        panic!("callback dispatch fixture")
    }
}
#[test]
fn dispatch_panic_is_contained_and_callback_identity_is_permanently_retired() {
    if isolated_case(
        "cases::dispatch_panic_is_contained_and_callback_identity_is_permanently_retired",
    ) {
        return;
    }
    let binding = HostCallbackBinding::install(Arc::new(PanickingCaller)).unwrap();
    let (call, free) = binding.callbacks();
    let (status, error) = callback(call, free, c"panic", b"");
    assert_eq!(status, ffi::OBK_ERR);
    assert!(String::from_utf8(error).unwrap().contains("panicked"));
    assert_eq!(binding.in_flight(), 0);
    assert!(HostCallbackBinding::install(Arc::new(EchoCaller)).is_err());
    let (status, error) = callback(call, free, c"future", b"");
    assert_eq!(status, ffi::OBK_ERR);
    assert!(String::from_utf8(error).unwrap().contains("retired"));
}
struct RefusingCaller;
impl Caller for RefusingCaller {
    fn call(&self, method: &str, _: &[u8]) -> Result<Vec<u8>, Error> {
        Err(Error(format!("method {method}: expected fixture success")))
    }
}
#[test]
fn ordinary_transport_errors_and_late_host_free_survive_binding_drop() {
    if isolated_case("cases::ordinary_transport_errors_and_late_host_free_survive_binding_drop") {
        return;
    }
    let binding = HostCallbackBinding::install(Arc::new(RefusingCaller)).unwrap();
    let (call, free) = binding.callbacks();
    let (status, bytes) = callback(call, free, c"refused", b"arg");
    assert_eq!(status, ffi::OBK_ERR);
    assert!(String::from_utf8(bytes).unwrap().contains("refused"));
    verify_late_free(binding, call, free);
}
fn verify_late_free(binding: HostCallbackBinding, call: ffi::ObkHostCall, free: ffi::ObkHostFree) {
    let mut pointer = std::ptr::null_mut();
    let mut length = 0;
    // SAFETY: writable output slots and the registered host's own free callback.
    unsafe {
        call(
            c"owned-error".as_ptr(),
            std::ptr::null(),
            0,
            &mut pointer,
            &mut length,
        )
    };
    assert_eq!(binding.buffer_counts().outstanding, 1);
    drop(binding);
    unsafe { free(pointer) };
    assert!(HostCallbackBinding::install(Arc::new(EchoCaller)).is_err());
}

fn verify_success_replies(call: ffi::ObkHostCall, free: ffi::ObkHostFree) {
    assert_eq!(
        callback(call, free, c"echo", b"bytes"),
        (ffi::OBK_OK, b"bytes".to_vec())
    );
    assert_eq!(callback(call, free, c"empty", &[]), (ffi::OBK_OK, vec![]));
}
fn verify_encoding(call: ffi::ObkHostCall, free: ffi::ObkHostFree) {
    let method = c"\xff";
    let (status, error) = callback(call, free, method, b"");
    assert_eq!(status, ffi::OBK_ERR);
    assert!(String::from_utf8(error).unwrap().contains("expected UTF-8"));
}
fn verify_invalid_output_and_free(
    binding: &HostCallbackBinding,
    call: ffi::ObkHostCall,
    free: ffi::ObkHostFree,
) {
    // SAFETY: missing output slots decline before writes; an unknown free is only a lookup key.
    let status = unsafe {
        call(
            c"empty".as_ptr(),
            std::ptr::null(),
            0,
            std::ptr::null_mut(),
            std::ptr::null_mut(),
        )
    };
    assert_eq!(status, ffi::OBK_ERR);
    let mut unknown = 0u8;
    unsafe {
        free(&mut unknown);
        free(std::ptr::null_mut());
    }
    assert_eq!(binding.buffer_counts().refused_frees, 1);
}
