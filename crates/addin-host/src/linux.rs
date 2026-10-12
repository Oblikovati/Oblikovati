// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{API_MAJOR, API_MINOR, ffi};
use std::{
    ffi::{CStr, CString, c_void},
    os::unix::ffi::OsStrExt,
    path::Path,
    ptr::NonNull,
};
mod callbacks;
mod exports;
pub use callbacks::{HostBufferCounts, HostCallbackBinding};
mod lifecycle;
mod registry;
mod response;
mod runtime;
pub use lifecycle::{AddInState, LoadedAddIn};
pub use registry::NativeAddInRegistry;
pub use runtime::{NativeAddInRuntime, NativeCatalogCaller, NativeCompletion};

struct ResidentLibrary {
    handle: NonNull<c_void>,
}
impl ResidentLibrary {
    unsafe fn open(path: &Path) -> Result<Self, Error> {
        let name = CString::new(path.as_os_str().as_bytes()).map_err(|error| {
            Error(format!(
                "add-in path {path:?}: expected no embedded NUL; {error}"
            ))
        })?;
        // SAFETY: caller trusts library constructors; CString remains valid throughout dlopen.
        let pointer = unsafe { libc::dlopen(name.as_ptr(), libc::RTLD_NOW | libc::RTLD_LOCAL) };
        NonNull::new(pointer)
            .map(|handle| Self { handle })
            .ok_or_else(|| {
                Error(format!(
                    "add-in path {path:?}: expected a loadable library; {}",
                    last_loader_error()
                ))
            })
    }
    unsafe fn symbol<T: Copy>(&self, name: &CStr) -> Result<Option<T>, Error> {
        // SAFETY: the mapping remains resident; names are static C strings and T is a canonical ffi function pointer.
        unsafe {
            libc::dlerror();
        }
        let address = unsafe { libc::dlsym(self.handle.as_ptr(), name.as_ptr()) };
        let failed = unsafe { !libc::dlerror().is_null() };
        if failed || address.is_null() {
            return Ok(None);
        }
        if std::mem::size_of::<T>() != std::mem::size_of_val(&address) {
            return Err(Error(format!(
                "symbol {name:?}: expected a pointer-sized ABI function, got {} bytes",
                std::mem::size_of::<T>()
            )));
        }
        // SAFETY: private callers select only the exact published prototype; trusted exports must implement it.
        Ok(Some(unsafe {
            std::mem::transmute_copy::<*mut c_void, T>(&address)
        }))
    }
}
// No dlclose: even a Go library's load-time constructors can start runtime threads before Activate.
fn last_loader_error() -> String {
    // SAFETY: dlerror returns null or a loader-owned NUL string, copied before another loader operation.
    let error = unsafe { libc::dlerror() };
    if error.is_null() {
        return "loader returned no diagnostic".into();
    }
    unsafe { CStr::from_ptr(error) }
        .to_string_lossy()
        .into_owned()
}
/// Probe a real library using the released ABI, without activating it. The mapping stays resident.
/// Example: `let probe = unsafe { probe_library(path) }?;`.
/// # Safety
/// The library and its constructors/exports must be trusted and obey the exact API C ABI.
/// String exports must return valid NUL-terminated storage throughout each read. Foreign faults are not contained.
pub unsafe fn probe_library(path: &Path) -> Result<AddInProbe, Error> {
    // SAFETY: delegated unchanged to the trusted opener; no lifecycle call occurs during admission.
    Ok(unsafe { open_addin(path)? }.metadata().clone())
}
/// Admit a trusted library and retain its callable ABI exports, initially inactive.
/// Example: `let mut addin = unsafe { open_addin(path) }?;`.
/// # Safety
/// The obligations of `probe_library` apply. Lifecycle callers must obey foreign thread affinity
/// and exclude other wrappers or foreign callers from the same library state during each call.
/// Repeated dlopen does not prove separate library state.
pub unsafe fn open_addin(path: &Path) -> Result<LoadedAddIn, Error> {
    // SAFETY: caller supplies a trusted ABI implementation; mappings remain resident.
    let library = unsafe { ResidentLibrary::open(path)? };
    let exports = unsafe { exports::ProbeExports::resolve(&library)? };
    let metadata = unsafe { copy_metadata(&exports)? };
    Ok(LoadedAddIn::new(exports, metadata))
}
unsafe fn copy_metadata(exports: &exports::ProbeExports) -> Result<AddInProbe, Error> {
    // SAFETY: resolved canonical exports stay resident; trusted strings are copied immediately.
    let (major, minor) = check_api_version(unsafe { exports.version() }, (API_MAJOR, API_MINOR))?;
    let id = unsafe { exports::copy_string(exports.id, "ObkAddInId")? };
    let manifest = unsafe { exports::copy_string(exports.manifest, "ObkAddInManifest")? };
    Ok(AddInProbe {
        id,
        manifest,
        major,
        minor,
        has_automation: exports.automation.is_some(),
    })
}
