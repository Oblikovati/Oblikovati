// SPDX-License-Identifier: GPL-2.0-only
//! Check Wayland presentation through the Martensite release consumed by the desktop.
use martensite_tkernel::standard::Handle;
use martensite_tkservice::{
    aspect::{AspectDisplayBackend, AspectDisplayConnection, AspectWindow},
    wl::WlWindow,
};
use martensite_tkv3d::v3d::V3dViewer;
use martensite_vulkan::{
    Graphic3dRenderDriver, Graphic3dRenderView, RenderGuiSurfacePolicy, VulkanOptions,
};

#[test]
#[ignore = "requires a Wayland compositor and Vulkan validation layers"]
fn pinned_martensite_presents_a_native_wayland_frame() {
    let window = mapped_wayland_window();
    let driver = Graphic3dRenderDriver::with_gui_surface_policy(
        VulkanOptions {
            validation: true,
            device_index: None,
        },
        RenderGuiSurfacePolicy::CheckedV2,
    )
    .unwrap();
    let viewer = V3dViewer::new(&driver.cast()).unwrap();
    let view = viewer.create_view().unwrap();
    view.set_immediate_update(false);
    view.set_window(&window.cast(), 0).unwrap();
    let renderer: Handle<Graphic3dRenderView> = view.get().unwrap().view().cast();
    renderer.try_redraw().unwrap();
    let frame = renderer.capture().unwrap();
    assert_eq!((frame.width, frame.height), (160, 120));
    assert_eq!(frame.rgba.len(), 160 * 120 * 4);
    view.remove();
    let errors = driver.validation_errors().unwrap();
    assert!(errors.is_empty(), "Vulkan validation errors: {errors:?}");
}

fn mapped_wayland_window() -> Handle<WlWindow> {
    let display = Handle::new(AspectDisplayConnection::new().unwrap());
    assert_eq!(display.backend(), AspectDisplayBackend::Wayland);
    let window = Handle::new(
        WlWindow::create(&display, "Oblikovati Wayland presentation test", 160, 120).unwrap(),
    );
    window.map();
    window
}
