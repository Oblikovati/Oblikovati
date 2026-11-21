// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::os::windows::ffi::OsStrExt;
use windows_sys::Win32::{
    Foundation::{ERROR_PROC_NOT_FOUND, GetLastError},
    System::LibraryLoader::{
        GetProcAddress, LOAD_LIBRARY_SEARCH_DEFAULT_DIRS, LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR,
        LoadLibraryExW,
    },
};

impl ResidentLibrary {
    pub(super) unsafe fn open(path: &Path) -> Result<Self, Error> {
        let absolute = std::path::absolute(path).map_err(|cause| {
            Error(format!(
                "add-in path {path:?}: expected an absolute path; {cause}"
            ))
        })?;
        let mut name: Vec<u16> = absolute.as_os_str().encode_wide().collect();
        if name.contains(&0) {
            return Err(Error(format!(
                "add-in path {path:?}: expected no embedded NUL"
            )));
        }
        name.push(0);
        // The caller trusts the DLL and its constructors. Dependency lookup excludes the current directory.
        let pointer = unsafe {
            LoadLibraryExW(
                name.as_ptr(),
                std::ptr::null_mut(),
                LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR | LOAD_LIBRARY_SEARCH_DEFAULT_DIRS,
            )
        };
        NonNull::new(pointer)
            .map(|handle| Self { handle })
            .ok_or_else(|| {
                Error(format!(
                    "add-in path {path:?}: expected a loadable DLL; {}",
                    std::io::Error::last_os_error()
                ))
            })
    }

    pub(super) unsafe fn symbol<T: Copy>(&self, name: &CStr) -> Result<Option<T>, Error> {
        // Export names are C strings and this module remains resident for the process lifetime.
        let address = unsafe { GetProcAddress(self.handle.as_ptr(), name.as_ptr().cast()) };
        let Some(address) = address else {
            return if unsafe { GetLastError() } == ERROR_PROC_NOT_FOUND {
                Ok(None)
            } else {
                Err(Error(format!(
                    "add-in symbol {name:?}: {}",
                    std::io::Error::last_os_error()
                )))
            };
        };
        if size_of::<T>() != size_of_val(&address) {
            return Err(Error(format!(
                "add-in symbol {name:?}: expected a pointer-sized ABI function"
            )));
        }
        // Each caller supplies the published C prototype of the trusted export.
        Ok(Some(unsafe { std::mem::transmute_copy(&address) }))
    }
}
