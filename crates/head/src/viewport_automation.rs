// SPDX-License-Identifier: GPL-2.0-only
use crate::HeadWindow;
use oblikovati_api::{Error, wire::*};
use oblikovati_automation::{decode_wire_arguments, encode_wire_reply};
use oblikovati_viewport::{
    ViewportButton, ViewportEvent, ViewportInput, ViewportKey, ViewportModifiers,
    ViewportPhysicalKey,
};
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
    let captured = captured_image(window, capture_path(&args.path, "viewport")?)?;
    encode_wire_reply(&CaptureViewportResult {
        path: captured.path,
        width: captured.width,
        height: captured.height,
    })
}

fn capture_window(
    window: &mut dyn HeadWindow,
    method: &str,
    bytes: &[u8],
) -> Result<Vec<u8>, Error> {
    let args: CaptureWindowArgs = decode_wire_arguments(method, bytes)?;
    let captured = captured_image(window, capture_path(&args.path, "window")?)?;
    encode_wire_reply(&CaptureWindowResult {
        path: captured.path,
        width: captured.width,
        height: captured.height,
    })
}

struct CapturedImage {
    path: String,
    width: i64,
    height: i64,
}

fn captured_image(window: &mut dyn HeadWindow, path: PathBuf) -> Result<CapturedImage, Error> {
    let capture = window.capture_window().map_err(Error)?;
    crate::capture::write_rgba_png(&path, &capture).map_err(Error)?;
    Ok(CapturedImage {
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
    let key = physical_key(&args.key)?;
    let modifiers = key_modifiers(&args);
    let command = command_key(&args.key, modifiers);
    inject_key(window, key, command, modifiers)?;
    inject_text(window, &args.key, modifiers)?;
    encode_wire_reply(&PressKeyResult::default())
}

fn inject_key(
    window: &mut dyn HeadWindow,
    key: ViewportPhysicalKey,
    command: Option<ViewportKey>,
    modifiers: ViewportModifiers,
) -> Result<(), Error> {
    for pressed in [true, false] {
        window
            .inject_viewport_input(ViewportEvent::Input(ViewportInput::PhysicalKey {
                key,
                layout_key: key,
                command,
                pressed,
                modifiers,
            }))
            .map_err(Error)?;
    }
    Ok(())
}

fn inject_text(
    window: &mut dyn HeadWindow,
    value: &str,
    modifiers: ViewportModifiers,
) -> Result<(), Error> {
    if modifiers.control || modifiers.alt || value.chars().count() != 1 {
        return Ok(());
    }
    let Some(character) = value
        .chars()
        .next()
        .filter(|character| !character.is_control())
    else {
        return Ok(());
    };
    window
        .inject_viewport_input(ViewportEvent::Input(ViewportInput::Text(character)))
        .map_err(Error)
}

fn physical_key(value: &str) -> Result<ViewportPhysicalKey, Error> {
    let key = navigation_key(value)
        .or_else(|| editing_key(value))
        .or_else(|| command_letter_key(value));
    key.ok_or_else(|| {
        Error(format!(
            "viewport key {value:?}: expected a supported physical key or Ctrl+S, Ctrl+Y, Ctrl+Z"
        ))
    })
}

fn navigation_key(value: &str) -> Option<ViewportPhysicalKey> {
    Some(match value {
        "Tab" => ViewportPhysicalKey::Tab,
        "Left" => ViewportPhysicalKey::LeftArrow,
        "Right" => ViewportPhysicalKey::RightArrow,
        "Up" => ViewportPhysicalKey::UpArrow,
        "Down" => ViewportPhysicalKey::DownArrow,
        "PageUp" => ViewportPhysicalKey::PageUp,
        "PageDown" => ViewportPhysicalKey::PageDown,
        "Home" => ViewportPhysicalKey::Home,
        "End" => ViewportPhysicalKey::End,
        "Insert" => ViewportPhysicalKey::Insert,
        _ => return None,
    })
}

fn editing_key(value: &str) -> Option<ViewportPhysicalKey> {
    match value {
        "Delete" => Some(ViewportPhysicalKey::Delete),
        "Backspace" => Some(ViewportPhysicalKey::Backspace),
        "Space" => Some(ViewportPhysicalKey::Space),
        "Enter" => Some(ViewportPhysicalKey::Enter),
        "Escape" => Some(ViewportPhysicalKey::Escape),
        _ => None,
    }
}

fn command_letter_key(value: &str) -> Option<ViewportPhysicalKey> {
    match value {
        "A" | "a" => Some(ViewportPhysicalKey::A),
        "S" | "s" | "Ctrl+S" => Some(ViewportPhysicalKey::S),
        "Y" | "y" | "Ctrl+Y" => Some(ViewportPhysicalKey::Y),
        "Z" | "z" | "Ctrl+Z" => Some(ViewportPhysicalKey::Z),
        _ => None,
    }
}

fn key_modifiers(args: &PressKeyArgs) -> ViewportModifiers {
    let control = args.ctrl || args.key.starts_with("Ctrl+");
    ViewportModifiers {
        shift: args.shift,
        control,
        alt: args.alt,
        super_key: false,
    }
}

fn command_key(value: &str, modifiers: ViewportModifiers) -> Option<ViewportKey> {
    match (value, modifiers.control) {
        ("Escape", _) => Some(ViewportKey::Cancel),
        ("Home", _) => Some(ViewportKey::Home),
        ("s" | "S" | "Ctrl+S", true) => Some(ViewportKey::Save),
        ("z" | "Z" | "Ctrl+Z", true) if modifiers.shift => Some(ViewportKey::Redo),
        ("y" | "Y" | "Ctrl+Y", true) => Some(ViewportKey::Redo),
        ("z" | "Z" | "Ctrl+Z", true) => Some(ViewportKey::Undo),
        _ => None,
    }
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
