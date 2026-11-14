// SPDX-License-Identifier: GPL-2.0-only

use super::PackageCodecError;
use base64::{
    Engine, alphabet,
    engine::general_purpose::GeneralPurposeConfig,
    engine::{DecodePaddingMode, GeneralPurpose},
};

pub(super) fn decode_base64(owner: &str, encoded: &str) -> Result<Vec<u8>, PackageCodecError> {
    decode_go_base64(
        owner,
        encoded,
        &alphabet::STANDARD,
        DecodePaddingMode::RequireCanonical,
        "padded standard",
    )
}

pub(super) fn decode_raw_url_base64(
    owner: &str,
    encoded: &str,
) -> Result<Vec<u8>, PackageCodecError> {
    decode_go_base64(
        owner,
        encoded,
        &alphabet::URL_SAFE,
        DecodePaddingMode::RequireNone,
        "unpadded URL-safe",
    )
}

fn decode_go_base64(
    owner: &str,
    encoded: &str,
    alphabet: &alphabet::Alphabet,
    padding: DecodePaddingMode,
    expected: &str,
) -> Result<Vec<u8>, PackageCodecError> {
    // Go's decoders ignore CR/LF and permit nonzero unused trailing bits.
    let compact: Vec<u8> = encoded
        .bytes()
        .filter(|byte| !matches!(byte, b'\r' | b'\n'))
        .collect();
    let config = GeneralPurposeConfig::new()
        .with_decode_allow_trailing_bits(true)
        .with_decode_padding_mode(padding);
    GeneralPurpose::new(alphabet, config)
        .decode(compact)
        .map_err(|error| {
            PackageCodecError(format!(
                "{owner} value {encoded:?}: expected {expected} base64: {error}"
            ))
        })
}
