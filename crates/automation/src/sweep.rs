// SPDX-License-Identifier: GPL-2.0-only
//! Resolve the Sweep command and the features.add API through certified Part replay.
use oblikovati_api::{
    Error,
    wire::{AddFeatureArgs, featureargs::Sweep},
};
use oblikovati_martensite::{PrismResolutionPolicy, certified_sketch_regions_with_datums};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId, sketch_prism::SketchDirectedCurve, sketch_sweep::SketchSweepDefinition,
        work_geometry::WorkGeometryEvaluation,
    },
    history::{DocumentSnapshot, RecipeReservations},
    sketch::{OwnedSketchGeometry, Sketch2D, SketchEntityId},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    request: AddFeatureArgs,
) -> Result<Vec<u8>, Error> {
    let input =
        serde_json::from_value::<Sweep>(request.args.0.unwrap_or_default()).map_err(|cause| {
            Error(format!(
                "sweep arguments: expected the canonical Sweep object: {cause}"
            ))
        })?;
    let document = crate::part_owner::active_part(files)?;
    let owner =
        replay.ok_or_else(|| Error("sweep: expected a certified part replay owner".into()))?;
    let policy = owner
        .resolution_policy()
        .ok_or_else(|| Error("sweep: expected a certified resolution policy".into()))?;
    owner.ensure_part(files, document).map_err(Error)?;
    let datums = owner.datum_evaluation(files, document).map_err(Error)?;
    let target = (input.operation == "join")
        .then(|| sweep_join_target(files, owner, document, None))
        .transpose()?;
    let workspace = files.workspace();
    let guid = workspace
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let snapshot = workspace.session(document).map_err(api_error)?.snapshot();
    let prepared = prepare_sweep_for_target(snapshot, &datums, policy, &input, target)?;
    let prefix = owner.candidate_prefix(files, document).map_err(Error)?;
    let ((reply, candidate), transition) = crate::part_feature_transaction::create(
        files,
        document,
        guid,
        policy,
        prefix,
        "Sweep",
        |snapshot, ids| prepared.create(snapshot, ids),
    )?;
    crate::fillet::publish_blend_candidate(files, Some(owner), document, candidate, &transition)?;
    Ok(reply)
}

/// Find the sole other live solid for a Sweep Join.
/// Example: `sweep_join_target(files, replay, document, None)?` before creating a Sweep.
pub fn sweep_join_target<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    replay: &dyn crate::PartUpdateAccess<O, W, F>,
    document: u64,
    except: Option<FeatureId>,
) -> Result<Vec<u8>, Error> {
    let evaluated = replay.current_part(files, document).map_err(Error)?;
    let mut bodies = evaluated
        .health()
        .iter()
        .filter(|row| Some(row.feature) != except)
        .filter_map(|row| evaluated.body(row.feature));
    let (Some(body), None) = (bodies.next(), bodies.next()) else {
        return Err(Error(
            "sweep Join: expected one eligible target body".into(),
        ));
    };
    let workspace = files.workspace();
    let session = workspace.session(document).map_err(api_error)?;
    let guid = workspace
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    body.body_reference_key(session.history(), guid)
        .map(<[u8]>::to_vec)
        .map_err(Error)
}

/// A Sweep whose transient input indices have been resolved to authored sources.
pub struct PreparedSweep {
    name: String,
    definition: SketchSweepDefinition,
}
impl PreparedSweep {
    /// Append the recipe to a private candidate. Example: `prepared.create(snapshot, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_sketch_sweep(&self.name, self.definition, ids)
            .map_err(api_error)
    }
}

/// Discover the only visible, unambiguous profile and independent straight path.
/// Example: `automatic_sweep(snapshot, &datums, policy)` for a new Sweep draft.
pub fn automatic_sweep(
    snapshot: &DocumentSnapshot,
    datums: &WorkGeometryEvaluation,
    policy: PrismResolutionPolicy,
) -> Option<Sweep> {
    let mut profiles = Vec::new();
    let mut paths = Vec::new();
    for (index, sketch) in snapshot
        .sketches
        .sketches()
        .iter()
        .enumerate()
        .filter(|(_, sketch)| sketch.visible())
    {
        if straight_path(sketch.geometry()).is_some() {
            paths.push(i64::try_from(index).ok()?);
        } else if let Ok(regions) =
            certified_sketch_regions_with_datums(snapshot, sketch.id(), policy, datums)
        {
            for region in 0..regions.len() {
                profiles.push((i64::try_from(index).ok()?, i64::try_from(region).ok()?));
            }
        }
    }
    let ([(sketch_index, profile_index)], [path_sketch_index]) =
        (profiles.as_slice(), paths.as_slice())
    else {
        return None;
    };
    Some(Sweep {
        sketch_index: *sketch_index,
        profile_index: *profile_index,
        path_sketch_index: *path_sketch_index,
        path_index: 0,
        ..Default::default()
    })
}

/// Translate the current indexed selections into a durable Sweep recipe.
/// Example: `prepare_sweep(snapshot, &datums, policy, &input)?`.
pub fn prepare_sweep(
    snapshot: &DocumentSnapshot,
    datums: &WorkGeometryEvaluation,
    policy: PrismResolutionPolicy,
    input: &Sweep,
) -> Result<PreparedSweep, Error> {
    prepare_sweep_for_target(snapshot, datums, policy, input, None)
}

/// Prepare the typed Sweep and bind its optional Join target.
/// Example: `prepare_sweep_for_target(snapshot, &datums, policy, &input, target)?`.
pub fn prepare_sweep_for_target(
    snapshot: &DocumentSnapshot,
    datums: &WorkGeometryEvaluation,
    policy: PrismResolutionPolicy,
    input: &Sweep,
    target: Option<Vec<u8>>,
) -> Result<PreparedSweep, Error> {
    validate_options(input)?;
    let profile_sketch = sketch_index(snapshot, input.sketch_index)?;
    let path_sketch = sketch_index(snapshot, input.path_sketch_index)?;
    let regions = certified_sketch_regions_with_datums(snapshot, profile_sketch, policy, datums)
        .map_err(api_error)?;
    let profile = usize::try_from(input.profile_index)
        .ok()
        .and_then(|index| regions.get(index))
        .ok_or_else(|| {
            Error(format!(
                "sweep profile index {}: expected one of {} closed regions",
                input.profile_index,
                regions.len()
            ))
        })?;
    let path = snapshot.sketches.get(path_sketch).map_err(api_error)?;
    let curve = straight_path(path.geometry()).filter(|_| input.path_index == 0)
        .ok_or_else(|| Error(format!("sweep path index {}: expected index 0 on a sketch containing one non-construction line", input.path_index)))?;
    let definition = SketchSweepDefinition::new(
        profile_sketch,
        profile.selection(),
        path_sketch,
        vec![SketchDirectedCurve {
            curve,
            reversed: false,
        }],
    )
    .map_err(api_error)?;
    let definition = if input.orientation == "parallelToOriginalProfile" {
        definition.with_fixed_orientation()
    } else if input.taper.trim().is_empty() {
        definition
    } else {
        prepare_sweep_taper(snapshot, definition, &input.taper)?
    };
    let definition = if input.twist.trim().is_empty() {
        definition
    } else {
        prepare_sweep_twist(snapshot, definition, &input.twist)?
    };
    let definition = match (input.operation.as_str(), target) {
        ("join", Some(key)) => definition.with_join_target(key).map_err(api_error)?,
        ("join", None) => return Err(Error("sweep Join: expected a named target body".into())),
        (_, _) => definition,
    };
    definition.resolve(snapshot, datums).map_err(api_error)?;
    Ok(PreparedSweep {
        name: crate::fillet::next_name(snapshot, "Sweep")?,
        definition,
    })
}

/// Resolve a taper expression for a Sweep draft. Example: `prepare_sweep_taper(snapshot, definition, "5 deg")?`.
pub fn prepare_sweep_taper(
    snapshot: &DocumentSnapshot,
    definition: SketchSweepDefinition,
    expression: &str,
) -> Result<SketchSweepDefinition, Error> {
    let taper = crate::move_body::angle(snapshot, expression, "sweep taper")?;
    definition.with_taper(taper).map_err(api_error)
}

/// Resolve the Twist field against the angular unit and parameter environment.
/// Example: `prepare_sweep_twist(snapshot, definition, "90 deg")?`.
pub fn prepare_sweep_twist(
    snapshot: &DocumentSnapshot,
    definition: SketchSweepDefinition,
    expression: &str,
) -> Result<SketchSweepDefinition, Error> {
    let angle = crate::move_body::angle(snapshot, expression, "sweep twist")
        .map_err(|cause| Error(format!("sweep twist {expression:?}: {cause}")))?;
    definition.with_twist(angle).map_err(api_error)
}

fn validate_options(input: &Sweep) -> Result<(), Error> {
    if input.orientation == "parallelToOriginalProfile" && !input.taper.trim().is_empty() {
        return Err(Error(
            "sweep taper: Fixed orientation uses the original profile".into(),
        ));
    }
    if input.orientation == "parallelToOriginalProfile" && !input.twist.trim().is_empty() {
        return Err(Error(
            "sweep twist: Fixed orientation uses the original profile".into(),
        ));
    }
    if !matches!(input.definition_type.as_str(), "" | "path")
        || !matches!(input.operation.as_str(), "" | "new" | "join")
        || !matches!(
            input.orientation.as_str(),
            "" | "normalToPath" | "parallelToOriginalProfile"
        )
        || input
            .align_vector
            .as_ref()
            .is_some_and(|items| !items.is_empty())
        || input
            .twist_stations
            .as_ref()
            .is_some_and(|items| !items.is_empty())
        || input
            .path_points
            .as_ref()
            .is_some_and(|items| !items.is_empty())
        || input.rail_sketch_index != 0
        || input.rail_index != 0
        || !input.scaling.is_empty()
        || !input.guide_face_key.is_empty()
        || input.tool_body_index.is_some()
    {
        return Err(Error("sweep options: expected New Solid or Join with Follow Path or Fixed orientation, without a guide or station overrides".into()));
    }
    Ok(())
}

fn straight_path(geometry: &Sketch2D) -> Option<SketchEntityId> {
    let index = geometry.read_index();
    let mut curves = index.entity_ids().iter().copied().filter_map(|id| {
        if index.is_construction(id).ok()? {
            return None;
        }
        let entity = index.entity(id).ok()?;
        (!matches!(entity, OwnedSketchGeometry::Point(_))).then_some((id, entity))
    });
    let (id, OwnedSketchGeometry::Line(_)) = curves.next()? else {
        return None;
    };
    curves.next().is_none().then_some(id)
}

fn sketch_index(snapshot: &DocumentSnapshot, index: i64) -> Result<DocumentSketchId, Error> {
    usize::try_from(index)
        .ok()
        .and_then(|index| snapshot.sketches.sketches().get(index))
        .map(|sketch| sketch.id())
        .ok_or_else(|| {
            Error(format!(
                "sweep sketch index {index}: expected an index in 0..{}",
                snapshot.sketches.sketches().len()
            ))
        })
}

fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}
