// SPDX-License-Identifier: GPL-2.0-only
//! Browser `wasm-bindgen` entry points driving the full Oblikovati desktop UI on WebGPU.
#![cfg(target_arch = "wasm32")]

use oblikovati_head::web_host::{WebDesktopHost, WebStepOutcome};
use oblikovati_viewport::{
    ViewportEvent, ViewportOptions, ViewportRuntime, WebKeyEvent, WebModifiers, WebPointerEvent,
    WebPointerKind, WebResizeEvent, WebWheelEvent, translate_web_key, translate_web_pointer,
    translate_web_resize, translate_web_wheel,
    web::{WebWorkerPool, init_web_worker_pool_from_url, web_worker_pool_size},
};
use serde::Serialize;
use wasm_bindgen::prelude::*;

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct WebStatusSnapshot {
    device_name: String,
    extent: [u32; 2],
    active_workers: usize,
    open_documents: usize,
}

/// Orchestrator-worker owner for the full Oblikovati desktop UI running on WebAssembly + WebGPU.
#[wasm_bindgen]
pub struct WebCadRuntime {
    host: WebDesktopHost,
    pool: Option<WebWorkerPool>,
    dpr: f64,
}

#[wasm_bindgen]
impl WebCadRuntime {
    /// Initialize the compute worker pool, WebGPU driver, and full Oblikovati desktop UI.
    /// Example (JS): `const cad = await WebCadRuntime.create(canvas, dpr, 4, glueUrl, false);`.
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
        let dpr = if device_pixel_ratio.is_finite() && device_pixel_ratio > 0.0 {
            device_pixel_ratio
        } else {
            1.0
        };
        let options = ViewportOptions {
            validation,
            physical_device: None,
        };
        let runtime = ViewportRuntime::new_webgpu_async(options)
            .await
            .map_err(|error| JsValue::from_str(&error))?;
        let view = runtime
            .open_offscreen_canvas("#viewport", canvas, dpr)
            .map_err(|error| JsValue::from_str(&error))?;
        let host = WebDesktopHost::new(runtime, view, validation)
            .map_err(|error| JsValue::from_str(&error))?;
        Ok(Self { host, pool, dpr })
    }

    /// Advance the desktop UI and owner loop by one tick; returns next wake delay in ms or `-1`.
    pub fn tick(&mut self) -> Result<i32, JsValue> {
        let outcome = self
            .host
            .step()
            .map_err(|error| JsValue::from_str(&error))?;
        Ok(encode_wake_ms(outcome))
    }

    /// Forward a browser pointer event (`kind`: 0=down, 1=up, 2=move); returns next wake delay ms.
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
    ) -> Result<i32, JsValue> {
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
        if let Some(translated) = translate_web_pointer(event, self.dpr) {
            self.host.push_event(translated);
        }
        self.tick()
    }

    /// Forward a browser wheel event; returns next wake delay in ms or `-1`.
    pub fn wheel_event(
        &mut self,
        css_x: f64,
        css_y: f64,
        delta_y: f64,
        delta_mode: u32,
    ) -> Result<i32, JsValue> {
        let event = WebWheelEvent {
            css_x,
            css_y,
            delta_y,
            delta_mode,
        };
        if let Some(translated) = translate_web_wheel(event, self.dpr) {
            self.host.push_event(translated);
        }
        self.tick()
    }

    /// Forward a browser keyboard event (both physical key and text); returns next wake delay ms.
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
    ) -> Result<i32, JsValue> {
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
        for translated in translate_web_key(&event) {
            self.host.push_event(translated);
        }
        self.tick()
    }

    /// Forward canvas focus or blur state to the desktop UI.
    pub fn focus_event(&mut self, focused: bool) -> Result<i32, JsValue> {
        self.host.push_event(ViewportEvent::Focus(focused));
        self.tick()
    }

    /// Resize the WebGPU canvas from CSS dimensions and device pixel ratio.
    pub fn resize(
        &mut self,
        css_width: f64,
        css_height: f64,
        device_pixel_ratio: f64,
    ) -> Result<String, JsValue> {
        if device_pixel_ratio.is_finite() && device_pixel_ratio > 0.0 {
            self.dpr = device_pixel_ratio;
        }
        let extent = translate_web_resize(WebResizeEvent {
            css_width,
            css_height,
            device_pixel_ratio: self.dpr,
        })
        .map_err(|error| JsValue::from_str(&error))?;
        self.host
            .resize(extent)
            .map_err(|error| JsValue::from_str(&error))?;
        let wake_after_ms = self.tick()?;
        Ok(format!(
            "{{\"width\":{},\"height\":{},\"wakeAfterMs\":{}}}",
            extent[0], extent[1], wake_after_ms
        ))
    }

    /// Open an uploaded `.opd` document from raw bytes into the desktop UI.
    pub fn open_document_bytes(&mut self, name: &str, bytes: &[u8]) -> Result<i32, JsValue> {
        self.host
            .open_uploaded_document(name, bytes)
            .map_err(|error| JsValue::from_str(&error))?;
        self.tick()
    }

    /// Asynchronously read back the composed RGBA8 framebuffer for E2E verification.
    pub async fn capture_rgba_async(&mut self) -> Result<js_sys::Uint8Array, JsValue> {
        let _ = self
            .host
            .step()
            .map_err(|error| JsValue::from_str(&error))?;
        let capture = self
            .host
            .capture_async()
            .await
            .map_err(|error| JsValue::from_str(&error))?;
        Ok(js_sys::Uint8Array::from(capture.rgba.as_slice()))
    }

    /// Return JSON runtime diagnostics (device name, framebuffer extent, active workers).
    pub fn status_json(&mut self) -> Result<String, JsValue> {
        let extent = self
            .host
            .drawable_extent()
            .map_err(|error| JsValue::from_str(&error))?;
        let open_documents = self
            .host
            .open_document_count()
            .map_err(|error| JsValue::from_str(&error))?;
        let snapshot = WebStatusSnapshot {
            device_name: self.host.device_name(),
            extent,
            active_workers: web_worker_pool_size(),
            open_documents,
        };
        serde_json::to_string(&snapshot).map_err(|error| JsValue::from_str(&error.to_string()))
    }

    /// Return accumulated WebGPU validation errors as a JSON string array.
    pub fn validation_errors_json(&self) -> Result<String, JsValue> {
        let errors = self
            .host
            .validation_errors()
            .map_err(|error| JsValue::from_str(&error))?;
        serde_json::to_string(&errors).map_err(|error| JsValue::from_str(&error.to_string()))
    }

    /// Return the host Oblikovati API major/minor version as JSON.
    pub fn api_version_json(&self) -> String {
        let (major, minor) = self.host.api_version();
        format!("{{\"major\":{major},\"minor\":{minor}}}")
    }

    /// Validate a WebAssembly add-in's compiled API version against the host contract.
    pub fn check_addin_compatibility(&self, major: i32, minor: i32) -> Result<(), JsValue> {
        WebDesktopHost::check_addin_compatibility(major, minor)
            .map_err(|error| JsValue::from_str(&error))
    }

    /// Execute a public Oblikovati API wire call on behalf of a loaded WebAssembly add-in.
    pub fn call_api(
        &mut self,
        method: &str,
        request: &[u8],
    ) -> Result<js_sys::Uint8Array, JsValue> {
        let reply = self
            .host
            .call_api(method, request)
            .map_err(|error| JsValue::from_str(&error))?;
        Ok(js_sys::Uint8Array::from(reply.as_slice()))
    }

    /// Cleanly retire the desktop host and terminate background WASM workers.
    pub fn close(&mut self) {
        let _ = self.host.finish();
        if let Some(pool) = self.pool.take() {
            pool.terminate();
        }
    }
}

fn encode_wake_ms(outcome: WebStepOutcome) -> i32 {
    match outcome.wake_after_ms {
        Some(ms) => i32::try_from(ms).unwrap_or(i32::MAX),
        None => -1,
    }
}
