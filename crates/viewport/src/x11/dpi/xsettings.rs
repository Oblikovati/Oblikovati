// SPDX-License-Identifier: GPL-2.0-only
//! Read the XSETTINGS integer DPI published by the desktop manager.
const DPI_FIXED_POINT: f32 = 1024.;

pub(super) fn dpi(mut bytes: &[u8]) -> Option<f32> {
    let little_endian = match take(&mut bytes, 4)?[0] {
        0 => true,
        1 => false,
        _ => return None,
    };
    take(&mut bytes, 4)?; // Settings serial.
    let count = integer(take(&mut bytes, 4)?, little_endian)?;
    let mut dpi = None;
    for _ in 0..count {
        let (name, value) = setting(&mut bytes, little_endian)?;
        if name == b"Xft/DPI" {
            dpi = value.filter(|value| *value > 0);
        }
    }
    dpi.map(|value| value as f32 / DPI_FIXED_POINT)
}

fn setting<'a>(bytes: &mut &'a [u8], little_endian: bool) -> Option<(&'a [u8], Option<i32>)> {
    let header = take(bytes, 4)?;
    let name_length = integer(&header[2..], little_endian)? as usize;
    let name = &take(bytes, name_length.next_multiple_of(4))?[..name_length];
    take(bytes, 4)?; // Last-change serial.
    let value = match header[0] {
        0 => Some(integer(take(bytes, 4)?, little_endian)? as i32),
        1 => {
            let length = integer(take(bytes, 4)?, little_endian)? as usize;
            take(bytes, length.checked_add(3)? & !3)?;
            None
        }
        2 => {
            take(bytes, 8)?;
            None
        }
        _ => return None,
    };
    Some((name, value))
}

fn take<'a>(bytes: &mut &'a [u8], size: usize) -> Option<&'a [u8]> {
    let (prefix, remaining) = bytes.split_at_checked(size)?;
    *bytes = remaining;
    Some(prefix)
}

fn integer(bytes: &[u8], little: bool) -> Option<u32> {
    match *bytes {
        [a, b] => Some(u32::from(if little {
            u16::from_le_bytes([a, b])
        } else {
            u16::from_be_bytes([a, b])
        })),
        [a, b, c, d] => Some(if little {
            u32::from_le_bytes([a, b, c, d])
        } else {
            u32::from_be_bytes([a, b, c, d])
        }),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::dpi;

    #[test]
    fn reads_fixed_point_dpi_after_a_string_setting() {
        let mut data = [0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0].to_vec();
        data.extend_from_slice(b"\x01\x00\x08\x00Xft/RGBA\x00\x00\x00\x00\x04\x00\x00\x00none");
        data.extend_from_slice(b"\x00\x00\x07\x00Xft/DPI\x00\x00\x00\x00\x00\x00\x40\x02\x00");
        assert_eq!(dpi(&data), Some(144.));
        assert_eq!(dpi(&data[..data.len() - 1]), None);
    }

    #[test]
    fn reads_big_endian_and_refuses_nonpositive_dpi() {
        let mut data = b"\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x07Xft/DPI\x00\x00\x00\x00\x00\x00\x02\x40\x00".to_vec();
        assert_eq!(dpi(&data), Some(144.));
        data[24..].fill(0xff);
        assert_eq!(dpi(&data), None);
    }
}
