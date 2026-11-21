// SPDX-License-Identifier: GPL-2.0-only
//! Refuse sketch policies that the host does not yet implement.
use oblikovati_model::sketch_settings::SketchSettings;

/// Reject custom policies until their behavior is implemented (#4165, #4164).
/// Example: `require_supported_sketch_policies(&settings)?;`.
pub fn require_supported_sketch_policies(settings: &SketchSettings) -> Result<(), String> {
    if settings.custom_inference_types {
        return Err("unsupported_custom_inference_types: true; expected false until inference type selection is supported".into());
    }
    if settings.custom_relax_types {
        return Err("unsupported_custom_relax_types: true; expected false until relax type selection is supported".into());
    }
    Ok(())
}
