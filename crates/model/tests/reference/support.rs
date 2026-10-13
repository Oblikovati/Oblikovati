// SPDX-License-Identifier: GPL-2.0-only
pub fn hex_bytes(source: &str) -> Vec<u8> {
    source
        .as_bytes()
        .as_chunks::<2>()
        .0
        .iter()
        .map(|pair| u8::from_str_radix(std::str::from_utf8(pair).unwrap(), 16).unwrap())
        .collect()
}
