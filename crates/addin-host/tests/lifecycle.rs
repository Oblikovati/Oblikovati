// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
#![allow(unsafe_code)]
#[path = "support/compiler.rs"]
mod compiler;
#[path = "lifecycle/go.rs"]
mod go;
#[path = "lifecycle/actions.rs"]
mod lifecycle_tests;
use compiler::FixtureCompiler;
use oblikovati_addin_host::{AddInState, LoadedAddIn, open_addin};
use oblikovati_api::{API_MAJOR, API_MINOR, ffi};
use std::{ffi::c_char, path::Path};
fn open(path: &Path) -> LoadedAddIn {
    // SAFETY: known fixtures implement the exact published ABI; mappings remain resident.
    unsafe { open_addin(path) }.unwrap()
}
unsafe extern "C" fn unused_host(
    _: *const c_char,
    _: *const u8,
    _: i32,
    _: *mut *mut u8,
    _: *mut i32,
) -> i32 {
    ffi::OBK_ERR
}
unsafe extern "C" fn unused_free(_: *mut u8) {}
fn activate(addin: &mut LoadedAddIn) -> Result<(), oblikovati_api::Error> {
    // SAFETY: fixture has no host calls; static callbacks outlive all explicit cleanup.
    unsafe { addin.activate(unused_host, unused_free) }
}
fn automation(
    addin: &mut LoadedAddIn,
    method: &str,
    bytes: &[u8],
) -> Result<Vec<u8>, oblikovati_api::Error> {
    // SAFETY: fixture allocator supplies live extents and a matching, counted ObkFree.
    unsafe { addin.automation(method, bytes) }
}
fn deactivation(addin: &mut LoadedAddIn) -> Result<(), oblikovati_api::Error> {
    // SAFETY: same-thread fixture calls with live static host callbacks.
    unsafe { addin.deactivate() }
}
fn free_count(addin: &mut LoadedAddIn) -> usize {
    String::from_utf8(automation(addin, "free_count", &[]).unwrap())
        .unwrap()
        .parse()
        .unwrap()
}

#[path = "support/resources.rs"]
mod resources;
