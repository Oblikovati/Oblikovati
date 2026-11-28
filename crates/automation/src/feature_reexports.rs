// SPDX-License-Identifier: GPL-2.0-only
//! Public part-feature automation helper re-exports.
pub use super::boundary_patch::{
    BoundaryPatchInputs, PreparedBoundaryPatch, build_boundary_patch_definition,
    prepare_boundary_patch_definition, replace_boundary_patch,
};
pub use super::chamfer::{PreparedChamfer, prepare_chamfer};
pub use super::chamfer_v2::{
    ChamferV2Inputs, build_chamfer_v2_definition, prepare_chamfer_v2, replace_chamfer,
};
pub use super::coil::{CoilInputs, PreparedCoil, build_coil_definition, prepare_coil_definition};
pub use super::combine::{
    CombineInputs, PreparedCombine, build_combine_definition, prepare_combine_definition,
    replace_combine,
};
pub use super::decal::{
    DecalInputs, PreparedDecal, build_decal_definition, prepare_decal_definition, replace_decal,
};
pub use super::delete_face::{
    DeleteFaceFeatureInputs, PreparedDeleteFace, build_delete_face_definition,
    prepare_delete_face_definition, replace_delete_face,
};
pub use super::derive::{
    DeriveInputs, PreparedDerivedPart, build_derived_part_definition,
    prepare_derived_part_definition, replace_derived_part,
};
pub use super::direct_edit::{
    DirectEditFeatureInputs, PreparedDirectEdit, build_direct_edit_definition,
    prepare_direct_edit_definition, replace_direct_edit,
};
pub use super::draft::{
    DraftFeatureInputs, PreparedDraft, build_draft_definition, prepare_draft_definition,
    replace_draft,
};
pub use super::emboss::{
    EmbossInputs, PreparedEmboss, build_emboss_definition, prepare_emboss_definition,
};
pub use super::extend_surface::{
    ExtendSurfaceInputs, PreparedExtendSurface, build_extend_surface_definition,
    prepare_extend_surface_definition, replace_extend_surface,
};
pub use super::extrusion::{
    CapturedExtrudeCandidate, ExtrudePrefixReplay, ExtrudeRegionAccess, ExtrudeReplayPrefix,
    PreparedExtrusion, commit_prepared_extrusion, evaluated_extrude_distance, prepare_extrusion,
    prepare_owned_extrusion, prepare_owned_extrusion_profiles, validate_extrude_distance,
};
pub use super::fillet::{
    CertifiedFilletAccess, CertifiedFilletValidation, FilletAccess, FilletValidation,
    PreparedFillet, prepare_fillet,
};
pub use super::fillet_v2::{
    FilletCornerSetbackInput, FilletEdgeSetInput, FilletV2Inputs, FilletVariablePointInput,
    build_fillet_v2_definition, evaluate_positive_cm, prepare_fillet_v2, replace_fillet,
};
pub use super::finish::{
    FinishFeatureInputs, PreparedFinish, build_finish_definition, prepare_finish_definition,
    replace_finish,
};
