// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use rustycad_tkernel::{ncollection::Vec2, quantity::NamedColor};
use rustycad_tkservice::aspect::{
    AspectTypeOfDisplayText, AspectTypeOfStyleText, AspectTypeOfTriedronPosition,
};
use rustycad_tkservice::graphic3d::*;
#[derive(Default)]
pub(super) struct StatusTextOverlay {
    text: String,
    structure: Option<StatusTextStructure>,
}
struct StatusTextStructure(Handle<Graphic3dStructure>);
impl Drop for StatusTextStructure {
    fn drop(&mut self) {
        self.0.remove();
    }
}
impl StatusTextOverlay {
    pub(super) fn submit(&mut self, viewer: &V3dViewer, text: &str) -> Result<bool, String> {
        validate_status_text(text)?;
        if self.text == text {
            return Ok(false);
        }
        let replacement = if text.is_empty() {
            None
        } else {
            Some(StatusTextStructure::new(viewer, text)?)
        };
        self.structure = replacement;
        self.text = text.into();
        Ok(true)
    }
    pub(super) fn clear(&mut self) {
        self.structure = None;
        self.text.clear();
    }
}
impl StatusTextStructure {
    fn new(viewer: &V3dViewer, label: &str) -> Result<Self, String> {
        let structure = Self(
            Graphic3dStructure::new(&viewer.structure_manager(), &Handle::default())
                .map_err(|cause| cause.to_string())?,
        );
        configure_status_structure(&structure.0)?;
        add_status_label(&structure.0, label)?;
        structure.0.display();
        Ok(structure)
    }
}
fn validate_status_text(text: &str) -> Result<(), String> {
    if text.contains('\0') {
        return Err(format!(
            "viewport status {text:?}: expected UTF-8 text without NUL"
        ));
    }
    Ok(())
}
fn configure_status_structure(structure: &Graphic3dStructure) -> Result<(), String> {
    let persistence = Graphic3dTransformPers::with_corner(
        Graphic3dTransModeFlags::TMF_2D,
        AspectTypeOfTriedronPosition::LEFT_UPPER,
        Vec2::new(12, 24),
    )
    .map_err(|cause| cause.to_string())?;
    structure.set_transform_persistence(&Handle::new(persistence).cast());
    // UI text must never participate in the model's Fit/Home bounds or depth buffer.
    structure.set_infinite_state(true);
    structure.set_z_layer(GRAPHIC3D_ZLAYER_ID_TOP_OSD);
    Ok(())
}
fn add_status_label(structure: &Graphic3dStructure, label: &str) -> Result<(), String> {
    let group = structure.new_group().map_err(|cause| cause.to_string())?;
    let style = Handle::new(Graphic3dAspectText3d::new(
        Color::named(NamedColor::WHITE),
        Some("DejaVu Sans"),
        1.,
        0.,
        AspectTypeOfStyleText::Normal,
        AspectTypeOfDisplayText::Normal,
    ));
    group.set_group_primitives_aspect(&style.cast());
    let text = Handle::new(Graphic3dText::new(16.));
    text.set_text_str(label);
    text.set_position(&Point3::new(0., 0., 0.));
    group.add_text(&text, false);
    Ok(())
}
