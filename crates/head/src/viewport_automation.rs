// SPDX-License-Identifier: GPL-2.0-only
use crate::HeadWindow;
use oblikovati_api::{Error, wire::*};
use oblikovati_automation::{decode_wire_arguments, encode_wire_reply};
use oblikovati_viewport::{ViewportButton, ViewportEvent, ViewportInput, ViewportKey};
use std::path::PathBuf;

/// Route the public viewport automation methods on the native owner thread.
pub(crate) fn apply(
    window: &mut dyn HeadWindow,
    method: &str,
    bytes: &[u8],
) -> Option<Result<Vec<u8>, Error>> {
    let result = match method {
        METHOD_VIEWPORT_CAPTURE => capture(window, method, bytes),
        METHOD_VIEWPORT_CAPTURE_WINDOW => capture_window(window, method, bytes),
        METHOD_VIEWPORT_CLICK => click(window, method, bytes),
        METHOD_VIEWPORT_KEY => key(window, method, bytes),
        METHOD_VIEWPORT_SCROLL => scroll(window, method, bytes),
        _ => return None,
    };
    Some(result)
}

fn capture(window: &mut dyn HeadWindow, method: &str, bytes: &[u8]) -> Result<Vec<u8>, Error> {
    let args: CaptureViewportArgs = decode_wire_arguments(method, bytes)?;
    encode_capture(window, capture_path(&args.path, "viewport")?)
}

fn capture_window(
    window: &mut dyn HeadWindow,
    method: &str,
    bytes: &[u8],
) -> Result<Vec<u8>, Error> {
    let args: CaptureWindowArgs = decode_wire_arguments(method, bytes)?;
    encode_capture(window, capture_path(&args.path, "window")?)
}

fn encode_capture(window: &mut dyn HeadWindow, path: PathBuf) -> Result<Vec<u8>, Error> {
    let capture = window.capture_window().map_err(Error)?;
    crate::capture::write_rgba_png(&path, &capture).map_err(Error)?;
    encode_wire_reply(&CaptureWindowResult {
        path: path.to_string_lossy().into_owned(),
        width: i64::from(capture.extent[0]),
        height: i64::from(capture.extent[1]),
    })
}

fn click(window: &mut dyn HeadWindow, method: &str, bytes: &[u8]) -> Result<Vec<u8>, Error> {
    let args: ClickViewportArgs = decode_wire_arguments(method, bytes)?;
    let position = pixels(args.x, args.y)?;
    let button = button(&args.button)?;
    let modifiers = modifiers(args.shift, args.ctrl, args.alt);
    for pressed in [true, false] {
        window
            .inject_viewport_input(ViewportEvent::Input(ViewportInput::Button {
                button,
                pressed,
                position,
                modifiers,
            }))
            .map_err(Error)?;
    }
    encode_wire_reply(&ClickViewportResult {
        x: args.x,
        y: args.y,
        ..Default::default()
    })
}

fn key(window: &mut dyn HeadWindow, method: &str, bytes: &[u8]) -> Result<Vec<u8>, Error> {
    let args: PressKeyArgs = decode_wire_arguments(method, bytes)?;
    let key = match args.key.as_str() {
        "Escape" => ViewportKey::Cancel,
        "Home" => ViewportKey::Home,
        "Ctrl+S" => ViewportKey::Save,
        "Ctrl+Z" => ViewportKey::Undo,
        "Ctrl+Y" => ViewportKey::Redo,
        value => {
            return Err(Error(format!(
                "viewport key {value:?}: expected Escape, Home, Ctrl+S, Ctrl+Z, or Ctrl+Y"
            )));
        }
    };
    window
        .inject_viewport_input(ViewportEvent::Input(ViewportInput::Key {
            key,
            pressed: true,
        }))
        .map_err(Error)?;
    encode_wire_reply(&PressKeyResult::default())
}

fn scroll(window: &mut dyn HeadWindow, method: &str, bytes: &[u8]) -> Result<Vec<u8>, Error> {
    let args: ScrollViewportArgs = decode_wire_arguments(method, bytes)?;
    let notches = rounded_i32(args.dy, "dy")?;
    window
        .inject_viewport_input(ViewportEvent::Input(ViewportInput::Wheel {
            notches,
            position: pixels(args.x, args.y)?,
        }))
        .map_err(Error)?;
    encode_wire_reply(&ScrollViewportResult::default())
}

fn capture_path(value: &str, kind: &str) -> Result<PathBuf, Error> {
    if value.is_empty() {
        return Ok(std::env::temp_dir().join(format!("oblikovati-{kind}.png")));
    }
    if value.ends_with(".png") {
        return Ok(PathBuf::from(value));
    }
    Err(Error(format!(
        "viewport capture path {value:?}: expected a .png path"
    )))
}

fn pixels(x: f64, y: f64) -> Result<[i32; 2], Error> {
    Ok([rounded_i32(x, "x")?, rounded_i32(y, "y")?])
}

fn rounded_i32(value: f64, label: &str) -> Result<i32, Error> {
    if value.is_finite() && value.round() >= i32::MIN as f64 && value.round() <= i32::MAX as f64 {
        return Ok(value.round() as i32);
    }
    Err(Error(format!(
        "viewport {label} {value:?}: expected a finite 32-bit pixel value"
    )))
}

fn button(value: &str) -> Result<ViewportButton, Error> {
    match value {
        "" | "left" => Ok(ViewportButton::Left),
        "middle" => Ok(ViewportButton::Middle),
        "right" => Ok(ViewportButton::Right),
        _ => Err(Error(format!(
            "viewport button {value:?}: expected left, middle, or right"
        ))),
    }
}

fn modifiers(shift: bool, control: bool, alt: bool) -> oblikovati_viewport::ViewportModifiers {
    oblikovati_viewport::ViewportModifiers {
        shift,
        control,
        alt,
        super_key: false,
    }
}
