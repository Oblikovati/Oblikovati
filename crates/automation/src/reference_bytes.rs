// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::Error;
pub(crate) fn hex_bytes(text: &str) -> Result<Vec<u8>, Error> {
    if text.is_empty()
        || !text.len().is_multiple_of(2)
        || !text.bytes().all(|byte| byte.is_ascii_hexdigit())
    {
        return Err(Error(format!(
            "reference key {text:?}: expected nonempty hexadecimal byte pairs"
        )));
    }
    (0..text.len())
        .step_by(2)
        .map(|index| {
            u8::from_str_radix(&text[index..index + 2], 16).map_err(|cause| {
                Error(format!(
                    "reference key {text:?}: expected hexadecimal bytes: {cause}"
                ))
            })
        })
        .collect()
}
