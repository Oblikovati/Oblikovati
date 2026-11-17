// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use windows_sys::Win32::Graphics::Gdi::*;

pub(super) struct WindowFrame {
    info: BITMAPINFO,
    bgra: Vec<u8>,
}

impl WindowFrame {
    pub(super) fn from_rgba(extent: [u32; 2], rgba: &[u8]) -> Result<Self, String> {
        let [width, height] = valid_extent(extent)?;
        if (width as usize)
            .checked_mul(height as usize)
            .and_then(|area| area.checked_mul(4))
            != Some(rgba.len())
        {
            return Err(format!(
                "viewport frame {} bytes: expected RGBA for {extent:?}",
                rgba.len()
            ));
        }
        let mut bgra = rgba.to_vec();
        for pixel in bgra.chunks_exact_mut(4) {
            pixel.swap(0, 2);
        }
        let mut info = BITMAPINFO::default();
        info.bmiHeader.biSize = size_of::<BITMAPINFOHEADER>() as u32;
        info.bmiHeader.biWidth = width;
        info.bmiHeader.biHeight = -height;
        info.bmiHeader.biPlanes = 1;
        info.bmiHeader.biBitCount = 32;
        Ok(Self { info, bgra })
    }
}

pub(super) fn paint(hwnd: HWND, frame: Option<&WindowFrame>) {
    let mut paint = PAINTSTRUCT::default();
    // BeginPaint validates the update region. The DIB buffer lives through EndPaint.
    unsafe {
        let dc = BeginPaint(hwnd, &mut paint);
        if let Some(frame) = frame {
            let mut client = RECT::default();
            GetClientRect(hwnd, &mut client);
            StretchDIBits(
                dc,
                0,
                0,
                client.right,
                client.bottom,
                0,
                0,
                frame.info.bmiHeader.biWidth,
                -frame.info.bmiHeader.biHeight,
                frame.bgra.as_ptr().cast(),
                &frame.info,
                DIB_RGB_COLORS,
                SRCCOPY,
            );
        } else {
            FillRect(dc, &paint.rcPaint, GetStockObject(BLACK_BRUSH));
        }
        EndPaint(hwnd, &paint);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn dib_preserves_top_down_rows_and_converts_rgba_to_bgra() {
        let frame = WindowFrame::from_rgba([1, 2], &[12, 34, 56, 255, 10, 20, 30, 255]).unwrap();
        assert_eq!(frame.info.bmiHeader.biHeight, -2);
        assert_eq!(frame.bgra, [56, 34, 12, 255, 30, 20, 10, 255]);
        assert!(WindowFrame::from_rgba([1, 2], &[0; 4]).is_err());
    }
}
