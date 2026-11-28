// SPDX-License-Identifier: GPL-2.0-only
//! Browser `wasm-bindgen` entry points for the Orchestrator Worker realm.
#![cfg(target_arch = "wasm32")]

use crate::WebSession;
use oblikovati_viewport::{
    ViewportOptions, ViewportPresentation, WebKeyEvent, WebModifiers, WebPointerEvent,
    WebPointerKind, WebResizeEvent, WebWheelEvent,
    web::{WebWorkerPool, init_web_worker_pool_from_url, web_worker_pool_size},
};
use wasm_bindgen::prelude::*;

/// Orchestrator-worker owner for the Oblikovati WebAssembly + WebGPU session.
#[wasm_bindgen]
pub struct WebCadRuntime {
    session: WebSession,
    pool: Option<WebWorkerPool>,
}

#[wasm_bindgen]
impl WebCadRuntime {
    /// Initialize the compute worker pool, WebGPU driver, and bound `OffscreenCanvas` session.
    /// Example (JS): `const cad = await WebCadRuntime.create(canvas, dpr, 4, glueUrl, true);`.
    pub async fn create(
        canvas: web_sys::OffscreenCanvas,
        device_pixel_ratio: f64,
        max_workers: usize,
        glue_url: String,
        validation: bool,
    ) -> Result<WebCadRuntime, JsValue> {
        let pool = if max_workers > 0 {
            Some(
                init_web_worker_pool_from_url(max_workers, &glue_url)
                    .await
                    .map_err(|error| JsValue::from_str(&error))?,
            )
        } else {
            None
        };
        let options = ViewportOptions {
            validation,
            physical_device: None,
        };
        let mut session = WebSession::new_offscreen_canvas(
            options,
            "#oblikovati-canvas",
            canvas,
            device_pixel_ratio,
        )
        .await
        .map_err(|error| JsValue::from_str(&error))?;
        session
            .load_sample_part()
            .map_err(|error| JsValue::from_str(&error))?;
        let _ = session
            .render_frame()
            .map_err(|error| JsValue::from_str(&error))?;
        Ok(Self { session, pool })
    }

    /// Execute a JSON-encoded [`crate::WebCommand`] and return the updated JSON state snapshot.
    pub fn dispatch_command_json(&mut self, command_json: &str) -> Result<String, JsValue> {
        self.session
            .execute_command_json(command_json)
            .map_err(|error| JsValue::from_str(&error))
    }

    /// Load an `.opd` document from raw bytes into the in-memory browser workspace.
    pub fn load_document_bytes(&mut self, name: &str, bytes: &[u8]) -> Result<String, JsValue> {
        self.session
            .open_document_bytes(name, bytes)
            .map_err(|error| JsValue::from_str(&error))?;
        self.session
            .snapshot_json()
            .map_err(|error| JsValue::from_str(&error))
    }

    /// Export the active document as `.opd` bytes for browser download.
    pub fn export_document_bytes(&mut self) -> Result<js_sys::Uint8Array, JsValue> {
        let (_name, bytes) = self
            .session
            .export_active_document_bytes()
            .map_err(|error| JsValue::from_str(&error))?;
        Ok(js_sys::Uint8Array::from(bytes.as_slice()))
    }

    /// Forward a canvas pointer event (`kind`: 0=down, 1=up, 2=move); returns true if redraw needed.
    #[allow(clippy::too_many_arguments)]
    pub fn pointer_event(
        &mut self,
        kind: u8,
        css_x: f64,
        css_y: f64,
        button: i16,
        shift: bool,
        control: bool,
        alt: bool,
        meta: bool,
    ) -> Result<bool, JsValue> {
        let pointer_kind = match kind {
            0 => WebPointerKind::Down,
            1 => WebPointerKind::Up,
            _ => WebPointerKind::Move,
        };
        let event = WebPointerEvent {
            kind: pointer_kind,
            css_x,
            css_y,
            button,
            modifiers: WebModifiers {
                shift,
                control,
                alt,
                meta,
            },
        };
        self.session
            .handle_pointer(event)
            .map_err(|error| JsValue::from_str(&error))
    }

    /// Forward a canvas wheel event; returns true if the camera moved and a redraw is needed.
    pub fn wheel_event(
        &mut self,
        css_x: f64,
        css_y: f64,
        delta_y: f64,
        delta_mode: u32,
    ) -> Result<bool, JsValue> {
        let event = WebWheelEvent {
            css_x,
            css_y,
            delta_y,
            delta_mode,
        };
        self.session
            .handle_wheel(event)
            .map_err(|error| JsValue::from_str(&error))
    }

    /// Forward a keyboard event; returns true if the scene or camera changed.
    #[allow(clippy::too_many_arguments)]
    pub fn key_event(
        &mut self,
        code: String,
        key: String,
        pressed: bool,
        shift: bool,
        control: bool,
        alt: bool,
        meta: bool,
    ) -> Result<bool, JsValue> {
        let event = WebKeyEvent {
            code,
            key,
            pressed,
            modifiers: WebModifiers {
                shift,
                control,
                alt,
                meta,
            },
        };
        self.session
            .handle_key(&event)
            .map_err(|error| JsValue::from_str(&error))
    }

    /// Resize the WebGPU canvas from CSS dimensions and device pixel ratio.
    pub fn resize(
        &mut self,
        css_width: f64,
        css_height: f64,
        device_pixel_ratio: f64,
    ) -> Result<String, JsValue> {
        let extent = self
            .session
            .resize(WebResizeEvent {
                css_width,
                css_height,
                device_pixel_ratio,
            })
            .map_err(|error| JsValue::from_str(&error))?;
        Ok(format!(
            "{{\"width\":{},\"height\":{}}}",
            extent[0], extent[1]
        ))
    }

    /// Present the latest frame to the bound `OffscreenCanvas` without blocking readback.
    pub fn render_frame(&mut self) -> Result<bool, JsValue> {
        self.session
            .render_frame()
            .map_err(|error| JsValue::from_str(&error))
    }

    /// Present and asynchronously read back RGBA8 pixels for verification or thumbnail capture.
    pub async fn capture_rgba_async(&mut self) -> Result<js_sys::Uint8Array, JsValue> {
        match self
            .session
            .present_async()
            .await
            .map_err(|error| JsValue::from_str(&error))?
        {
            ViewportPresentation::Completed(capture) => {
                Ok(js_sys::Uint8Array::from(capture.rgba.as_slice()))
            }
            ViewportPresentation::SurfaceChanged(extent) => Err(JsValue::from_str(&format!(
                "capture surface changed to {extent:?}; retry after resize"
            ))),
        }
    }

    /// Serialize the current browser workbench snapshot to JSON.
    pub fn state_json(&self) -> Result<String, JsValue> {
        self.session
            .snapshot_json()
            .map_err(|error| JsValue::from_str(&error))
    }

    /// Return the number of active background WASM compute workers.
    pub fn active_workers(&self) -> usize {
        web_worker_pool_size()
    }

    /// Return accumulated WebGPU validation errors as a JSON string array.
    pub fn validation_errors_json(&self) -> Result<String, JsValue> {
        let errors = self
            .session
            .validation_errors()
            .map_err(|error| JsValue::from_str(&error))?;
        serde_json::to_string(&errors).map_err(|error| JsValue::from_str(&error.to_string()))
    }

    /// Close the viewport and terminate background compute workers cleanly.
    pub fn close(&mut self) {
        self.session.close();
        if let Some(pool) = self.pool.take() {
            pool.terminate();
        }
    }
}
