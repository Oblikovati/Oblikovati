// SPDX-License-Identifier: GPL-2.0-only
//! #3851: consume checked GUI target observations without hiding renderer faults.
use super::{Viewport, ViewportOptions, next_generation};
use martensite_render_core::{
    Graphic3dRenderDriver, RenderBehaviorConfiguration, RenderCoordinateConvention, RenderError,
    RenderGuiFreshPacket, RenderGuiSurfaceChange, RenderGuiSurfaceOutcome, RenderGuiSurfacePolicy,
    RenderLineWidthPolicy, RenderPipelinePolicies, RenderUnlitFaceColorPolicy,
};
use martensite_tkernel::standard::Handle;
#[cfg(any(target_os = "linux", target_os = "windows"))]
use martensite_vulkan::VulkanOptions;
use martensite_webgpu::{WebGpuOptions, WebGpuRenderDriver};

fn viewport_pipeline_policies() -> RenderPipelinePolicies {
    RenderPipelinePolicies {
        gui_surface: RenderGuiSurfacePolicy::CheckedV2,
        unlit_faces: RenderUnlitFaceColorPolicy::SideColorsV2,
        ..Default::default()
    }
}

/// Select checked native Vulkan GUI publication; example: `native_gui_driver(options)?`.
#[cfg(any(target_os = "linux", target_os = "windows"))]
pub(super) fn native_gui_driver(
    options: ViewportOptions,
) -> Result<Handle<Graphic3dRenderDriver>, String> {
    Graphic3dRenderDriver::with_pipeline_configuration(
        VulkanOptions {
            validation: options.validation,
            device_index: options.physical_device,
        },
        viewport_pipeline_policies(),
    )
    .map_err(|error| error.to_string())
}

/// Select checked WebGPU GUI publication asynchronously; example: `webgpu_gui_driver_async(options).await?`.
pub(super) async fn webgpu_gui_driver_async(
    options: ViewportOptions,
) -> Result<Handle<Graphic3dRenderDriver>, String> {
    WebGpuRenderDriver::with_policies_async(
        WebGpuOptions {
            validation: options.validation,
            ..WebGpuOptions::default()
        },
        viewport_pipeline_policies(),
        RenderCoordinateConvention::default(),
        RenderLineWidthPolicy::default(),
        RenderBehaviorConfiguration::default(),
    )
    .await
    .map_err(|error| error.to_string())
}

/// Select checked WebGPU GUI publication synchronously on native targets.
#[cfg(not(target_arch = "wasm32"))]
pub(super) fn webgpu_gui_driver_blocking(
    options: ViewportOptions,
) -> Result<Handle<Graphic3dRenderDriver>, String> {
    martensite_webgpu::realm::block_on(webgpu_gui_driver_async(options))
}
impl Viewport {
    /// Keep real faults distinct from a fresh-frame request; example: `view.checked_gui_result(draw, "redraw")?`.
    pub(super) fn checked_gui_result<T>(
        &mut self,
        result: Result<T, RenderError>,
        stage: &str,
    ) -> Result<Option<T>, String> {
        let error = match result {
            Ok(value) => return Ok(Some(value)),
            Err(error) => error,
        };
        if let Some(RenderGuiSurfaceOutcome::SurfaceChanged(change)) = error.gui_surface_outcome() {
            self.observe_checked_gui_surface(change)?;
            return Ok(None);
        }
        Err(format!("viewport {} {stage}: {error}", self.id))
    }
    fn observe_checked_gui_surface(
        &mut self,
        change: &RenderGuiSurfaceChange,
    ) -> Result<(), String> {
        self.completed = None;
        let previous = self.refresh_checked_gui_surface(change)?;
        self.gui_surface = match change.fresh_packet {
            RenderGuiFreshPacket::Required => self.gui_surface.or(Some(previous)),
            RenderGuiFreshPacket::AlreadyCurrent => Some(self.surface),
            RenderGuiFreshPacket::Absent => None,
        };
        Ok(())
    }
    fn refresh_checked_gui_surface(
        &mut self,
        change: &RenderGuiSurfaceChange,
    ) -> Result<u64, String> {
        let previous = self.surface;
        let unobserved = self.gui_surface.is_none_or(|gui| gui == previous);
        self.refresh_extent()?;
        // Same-size target changes need one new generation; an already stale packet must not count it twice.
        let invalidated =
            change.target_revision_changed || change.fresh_packet == RenderGuiFreshPacket::Required;
        if self.surface == previous && unobserved && invalidated {
            self.surface = next_generation(self.surface)?;
        }
        Ok(previous)
    }
}
