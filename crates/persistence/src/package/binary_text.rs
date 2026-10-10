// SPDX-License-Identifier: GPL-2.0-only

use super::PackageCodecError;
use base64::{
    Engine, alphabet, engine::GeneralPurpose, engine::general_purpose::GeneralPurposeConfig,
};

pub(super) fn decode_base64(owner: &str, encoded: &str) -> Result<Vec<u8>, PackageCodecError> {
    // Go StdEncoding ignores CR/LF and permits nonzero unused padding bits.
    let compact: Vec<u8> = encoded
        .bytes()
        .filter(|byte| !matches!(byte, b'\r' | b'\n'))
        .collect();
    let config = GeneralPurposeConfig::new().with_decode_allow_trailing_bits(true);
    GeneralPurpose::new(&alphabet::STANDARD, config)
        .decode(compact)
        .map_err(|error| {
            PackageCodecError(format!(
                "{owner} value {encoded:?}: expected padded standard base64: {error}"
            ))
        })
}
