// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) struct ProbeExports {
    pub(super) id: ffi::ObkAddInId,
    pub(super) manifest: ffi::ObkAddInManifest,
    major: Option<ffi::ObkAddInApiMajor>,
    minor: Option<ffi::ObkAddInApiMinor>,
    pub(super) has_automation: bool,
}
impl ProbeExports {
    pub(super) unsafe fn resolve(library: &ResidentLibrary) -> Result<Self, Error> {
        // SAFETY: all symbol types are imported canonical ABI prototypes; the trusted mapping stays resident.
        unsafe {
            require::<ffi::ObkAddInActivate>(library, c"ObkAddInActivate")?;
            require::<ffi::ObkAddInDeactivate>(library, c"ObkAddInDeactivate")?;
            require::<ffi::ObkAddInNotify>(library, c"ObkAddInNotify")?;
            require::<ffi::ObkFree>(library, c"ObkFree")?;
            Ok(Self {
                id: require(library, c"ObkAddInId")?,
                manifest: require(library, c"ObkAddInManifest")?,
                major: library.symbol(c"ObkAddInApiMajor")?,
                minor: library.symbol(c"ObkAddInApiMinor")?,
                has_automation: library
                    .symbol::<ffi::ObkAddInAutomation>(c"ObkAddInAutomation")?
                    .is_some(),
            })
        }
    }
    pub(super) unsafe fn version(&self) -> Option<(i32, i32)> {
        let (major, minor) = (self.major?, self.minor?);
        // SAFETY: resolved exact C prototypes, valid for the resident mapping's lifetime.
        Some(unsafe { (major(), minor()) })
    }
}
unsafe fn require<T: Copy>(library: &ResidentLibrary, name: &CStr) -> Result<T, Error> {
    // SAFETY: private callers use exact ffi prototypes and a resident trusted mapping.
    unsafe { library.symbol(name)? }.ok_or_else(|| {
        Error(format!(
            "add-in symbol {name:?}: expected a required published ABI export"
        ))
    })
}
pub(super) unsafe fn copy_string(export: ffi::ObkAddInId, name: &str) -> Result<String, Error> {
    // SAFETY: trusted export has the canonical string prototype; validity is the probe caller's obligation.
    let pointer = unsafe { export() };
    if pointer.is_null() {
        return Err(Error(format!(
            "add-in export {name} returned null: expected a valid NUL-terminated UTF-8 string"
        )));
    }
    let bytes = unsafe { CStr::from_ptr(pointer) };
    bytes.to_str().map(str::to_owned).map_err(|error| {
        Error(format!(
            "add-in export {name} bytes {:?}: expected UTF-8; {error}",
            bytes.to_bytes()
        ))
    })
}
