// SPDX-License-Identifier: GPL-2.0-only
use super::*;
struct FakeImageServer {
    setup: Setup,
}
impl FakeImageServer {
    fn new(depth: u8, bits: u8, padding: u8, order: ImageOrder, masks: [u32; 3]) -> Self {
        let visual = Visualtype {
            visual_id: 42,
            class: VisualClass::TRUE_COLOR,
            red_mask: masks[0],
            green_mask: masks[1],
            blue_mask: masks[2],
            ..Default::default()
        };
        let root = Screen {
            allowed_depths: vec![Depth {
                depth,
                visuals: vec![visual],
            }],
            ..Default::default()
        };
        Self {
            setup: Setup {
                pixmap_formats: vec![Format {
                    depth,
                    bits_per_pixel: bits,
                    scanline_pad: padding,
                }],
                roots: vec![root],
                image_byte_order: order,
                ..Default::default()
            },
        }
    }
    fn layout(&self, width: u16) -> Result<NativeImageLayout, String> {
        let image = GetImageReply {
            depth: self.setup.pixmap_formats[0].depth,
            visual: 42,
            sequence: 0,
            data: Vec::new(),
        };
        NativeImageLayout::from_setup(&self.setup, &image, width)
    }
}
#[test]
fn byte_order_padding_and_unused_high_bits_preserve_exact_rgb() {
    for (order, pixel) in [
        (ImageOrder::LSB_FIRST, [0x33, 0x22, 0x11, 0xee]),
        (ImageOrder::MSB_FIRST, [0x11, 0x22, 0x33, 0xee]),
    ] {
        for bits in [24, 32] {
            let encoded = if bits == 32 && order == ImageOrder::MSB_FIRST {
                [0xee, 0x11, 0x22, 0x33]
            } else {
                pixel
            };
            let server = FakeImageServer::new(24, bits, 32, order, [0xff0000, 0xff00, 0xff]);
            assert_eq!(
                server.layout(1).unwrap().rgba(&encoded, [1, 1]).unwrap(),
                [0x11, 0x22, 0x33, 255]
            );
        }
    }
}
#[test]
fn row_padding_and_reply_padding_are_separate_from_pixel_values() {
    let server = FakeImageServer::new(24, 24, 32, ImageOrder::LSB_FIRST, [0xff0000, 0xff00, 0xff]);
    let encoded = [0x33, 0x22, 0x11, 0xff, 0x66, 0x55, 0x44, 0xee];
    assert_eq!(
        server.layout(1).unwrap().rgba(&encoded, [1, 2]).unwrap(),
        [0x11, 0x22, 0x33, 255, 0x44, 0x55, 0x66, 255]
    );
    let server = FakeImageServer::new(24, 24, 8, ImageOrder::LSB_FIRST, [0xff0000, 0xff00, 0xff]);
    assert_eq!(
        server
            .layout(1)
            .unwrap()
            .rgba(&encoded[..4], [1, 1])
            .unwrap(),
        [0x11, 0x22, 0x33, 255]
    );
}
#[test]
fn smaller_truecolor_pixels_use_the_same_integer_channel_scaling() {
    let server = FakeImageServer::new(8, 8, 32, ImageOrder::LSB_FIRST, [0xe0, 0x1c, 3]);
    assert_eq!(
        server
            .layout(1)
            .unwrap()
            .rgba(&[0xe3, 0xff, 0xee, 0xdd], [1, 1])
            .unwrap(),
        [255, 0, 255, 255]
    );
    for (order, bytes) in [
        (ImageOrder::LSB_FIRST, [0x00, 0xf8, 0xaa, 0xbb]),
        (ImageOrder::MSB_FIRST, [0xf8, 0x00, 0xaa, 0xbb]),
    ] {
        let server = FakeImageServer::new(16, 16, 32, order, [0xf800, 0x7e0, 0x1f]);
        assert_eq!(
            server.layout(1).unwrap().rgba(&bytes, [1, 1]).unwrap(),
            [255, 0, 0, 255]
        );
    }
}
#[test]
fn truncated_extra_and_mismatched_images_are_declined_before_row_access() {
    let server = FakeImageServer::new(24, 32, 32, ImageOrder::LSB_FIRST, [0xff0000, 0xff00, 0xff]);
    let layout = server.layout(1).unwrap();
    assert!(
        layout
            .rgba(&[0; 3], [1, 1])
            .unwrap_err()
            .contains("expected 4")
    );
    assert!(layout.rgba(&[0; 8], [1, 1]).is_err());
    assert!(layout.rgba(&[0; 4], [2, 1]).is_err());
    assert!(layout.rgba(&[], [1, 0]).is_err());
    assert!(expected_image_bytes(usize::MAX, 2).is_err());
    assert!(expected_image_bytes(usize::MAX, 1).is_err());
}
#[test]
fn malformed_storage_masks_and_byte_order_are_named_declines() {
    for (depth, bits, padding) in [(24, 4, 32), (24, 32, 0), (33, 32, 32)] {
        let server = FakeImageServer::new(
            depth,
            bits,
            padding,
            ImageOrder::LSB_FIRST,
            [0xff0000, 0xff00, 0xff],
        );
        assert!(server.layout(1).is_err());
    }
    let server = FakeImageServer::new(24, 32, 32, ImageOrder::from(2), [0xff0000, 0xff00, 0xff]);
    assert!(server.layout(1).err().unwrap().contains("image order"));
    let server = FakeImageServer::new(8, 8, 32, ImageOrder::LSB_FIRST, [0xff0000, 0xff00, 0xff]);
    assert!(server.layout(1).err().unwrap().contains("8-bit"));
    assert!(server.layout(0).is_err());
}
#[test]
fn unknown_depth_visual_and_non_truecolor_metadata_are_declined() {
    let mut server =
        FakeImageServer::new(24, 32, 32, ImageOrder::LSB_FIRST, [0xff0000, 0xff00, 0xff]);
    server.setup.roots[0].allowed_depths[0].visuals[0].class = VisualClass::PSEUDO_COLOR;
    assert!(server.layout(1).err().unwrap().contains("TrueColor"));
    server.setup.roots[0].allowed_depths.clear();
    assert!(server.layout(1).err().unwrap().contains("visual"));
    let image = GetImageReply {
        depth: 17,
        visual: 42,
        sequence: 0,
        data: Vec::new(),
    };
    assert!(
        NativeImageLayout::from_setup(&server.setup, &image, 1)
            .err()
            .unwrap()
            .contains("depth 17")
    );
}
