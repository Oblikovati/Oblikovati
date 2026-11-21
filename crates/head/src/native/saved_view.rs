// SPDX-License-Identifier: GPL-2.0-only
pub(super) type SketchCameraState = (
    crate::HeadSketchEdit,
    Option<oblikovati_viewport::ViewportCamera>,
);
pub(super) type SavedWindowView = (
    oblikovati_viewport::ViewportCamera,
    Option<SketchCameraState>,
);
