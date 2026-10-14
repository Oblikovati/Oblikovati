// SPDX-License-Identifier: GPL-2.0-only
use crate::{types::valid_extent, x11::NativeWindow};
use rustycad_tkernel::standard::Handle;
use rustycad_tkservice::aspect::{AspectNeutralWindow, AspectWindow, AspectWindowMembers};
pub(crate) enum ViewTarget {
    Native(Box<NativeWindow>),
    Offscreen(Handle<AspectNeutralWindow>),
}
impl ViewTarget {
    pub fn offscreen(extent: [u32; 2]) -> Result<Self, String> {
        let [width, height] = valid_extent(extent)?;
        let window = Handle::new(AspectNeutralWindow::default());
        window.set_virtual(true);
        window.set_size(width, height);
        Ok(Self::Offscreen(window))
    }
    pub fn window(&self) -> Handle<dyn AspectWindow> {
        match self {
            Self::Native(native) => native.window.cast(),
            Self::Offscreen(window) => window.cast(),
        }
    }
    pub fn extent(&self) -> [u32; 2] {
        let window = self.window();
        if let Self::Native(_) = self {
            window.do_resize();
        }
        let (width, height) = window.size();
        [width as u32, height as u32]
    }
    pub fn resize(&self, extent: [u32; 2]) -> Result<[u32; 2], String> {
        let [width, height] = valid_extent(extent)?;
        match self {
            Self::Native(native) => native.resize(extent)?,
            Self::Offscreen(window) => {
                window.set_size(width, height);
            }
        }
        Ok(self.extent())
    }
}

impl ViewTarget {
    pub fn require_presentation(&self) -> Result<(), String> {
        if let Self::Native(native) = self
            && !native.window.is_mapped()
        {
            return Err("native viewport redraw: expected a mapped drawable; hidden view has no presentation".into());
        }
        Ok(())
    }
}
