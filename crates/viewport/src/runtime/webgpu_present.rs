// SPDX-License-Identifier: GPL-2.0-only
//! Deferred and asynchronous WebGPU presentation and readback on `Viewport`.
use super::{Viewport, ViewportCapture, ViewportPresentation};
use martensite_webgpu::WebGpuRenderViewExt;

impl Viewport {
    /// Present once without blocking for pixel readback; returns false when the surface changed.
    /// Example: `view.present_frame()?` drives interactive WebGPU or desktop redraw loops.
    pub fn present_frame(&mut self) -> Result<bool, String> {
        if !self.draw_current_frame()? {
            return Ok(false);
        }
        self.completed = Some(self.generations());
        Ok(true)
    }

    /// Present once and asynchronously read back the WebGPU surface without blocking the event loop.
    /// Example: `let presented = view.present_async().await?;`.
    pub async fn present_async(&mut self) -> Result<ViewportPresentation, String> {
        if !self.draw_current_frame()? {
            return Ok(ViewportPresentation::SurfaceChanged(self.extent));
        }
        let result = self.live()?.renderer.capture_async().await;
        let Some(frame) = self.checked_gui_result(result, "capture")? else {
            return Ok(ViewportPresentation::SurfaceChanged(self.extent));
        };
        if let Some(extent) = self.observe_draw_extent()? {
            return Ok(ViewportPresentation::SurfaceChanged(extent));
        }
        self.require_frame_extent([frame.width, frame.height])?;
        self.completed = Some(self.generations());
        Ok(ViewportPresentation::Completed(self.capture_from(frame)))
    }

    /// Asynchronously read back the latest completed WebGPU frame.
    /// Example: `let capture = viewport.capture_async().await?;`.
    pub async fn capture_async(&self) -> Result<ViewportCapture, String> {
        self.require_current_frame()?;
        let frame = self
            .live()?
            .renderer
            .capture_async()
            .await
            .map_err(|error| format!("viewport {} capture: {error}", self.id))?;
        self.require_frame_extent([frame.width, frame.height])?;
        Ok(self.capture_from(frame))
    }
}

#[cfg(all(test, not(target_arch = "wasm32")))]
mod tests {
    use crate::{
        ViewportCamera, ViewportOptions, ViewportPresentation, ViewportRuntime, ViewportScene,
        ViewportTriangle,
    };

    #[test]
    fn webgpu_runtime_presents_and_captures_async_with_zero_validation_errors() {
        let runtime = ViewportRuntime::new_webgpu(ViewportOptions {
            validation: true,
            physical_device: None,
        })
        .unwrap();
        assert!(!runtime.device_name().is_empty());

        let mut view = runtime.open_offscreen([128, 96]).unwrap();
        let scene = ViewportScene {
            background: [0.05, 0.08, 0.12],
            triangles: vec![ViewportTriangle {
                positions: [[-5.0, -5.0, 0.0], [5.0, -5.0, 0.0], [0.0, 5.0, 0.0]],
                normals: Some([[0.0, 0.0, 1.0]; 3]),
                color: [0.85, 0.55, 0.2, 1.0],
                face_index: Some(0),
            }],
        };
        view.submit(&scene).unwrap();
        view.set_camera(ViewportCamera::default()).unwrap();

        assert!(view.present_frame().unwrap());
        let cap1 = martensite_webgpu::realm::block_on(view.capture_async()).unwrap();
        assert_eq!(cap1.extent, [128, 96]);
        assert_eq!(cap1.rgba.len(), 128 * 96 * 4);

        let presented = martensite_webgpu::realm::block_on(view.present_async()).unwrap();
        let ViewportPresentation::Completed(cap2) = presented else {
            panic!("expected completed WebGPU presentation");
        };
        assert_eq!(cap2.extent, [128, 96]);
        let lit = cap2
            .rgba
            .as_chunks::<4>()
            .0
            .iter()
            .filter(|px| px[0] > 40 || px[1] > 40 || px[2] > 40)
            .count();
        assert!(
            lit > 100,
            "expected visible rendered triangle pixels, got {lit}"
        );
        assert!(runtime.validation_errors().unwrap().is_empty());
    }
}
