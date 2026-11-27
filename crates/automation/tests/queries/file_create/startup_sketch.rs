// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    application_options::{PartApplicationOptions, PartPreferenceAccess},
    document_sketch::SketchPlacement,
    feature::work_plane::WorkPlaneOrigin,
};

struct FixedPartOptions(PartApplicationOptions);
impl PartPreferenceAccess for FixedPartOptions {
    fn part_options(&self) -> PartApplicationOptions {
        self.0
    }
    fn set_part_options(&mut self, options: PartApplicationOptions) -> Result<(), String> {
        self.0 = options;
        Ok(())
    }
}
struct VetoSketchFactory;
impl DocumentObserverFactory for VetoSketchFactory {
    type Observer = QueryObserver;
    fn create(&mut self) -> Result<QueryObserver, String> {
        Ok(QueryObserver {
            veto: true,
            ..Default::default()
        })
    }
}
fn create_request(kind: &str) -> CreateDocumentArgs {
    CreateDocumentArgs {
        type_: kind.into(),
        name: "Test".into(),
        ..Default::default()
    }
}
#[test]
fn creation_uses_each_selected_plane_and_only_for_parts() {
    for (plane, project) in [
        (None, true),
        (Some(WorkPlaneOrigin::XY), true),
        (Some(WorkPlaneOrigin::YZ), false),
        (Some(WorkPlaneOrigin::XZ), true),
    ] {
        let mut files =
            DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
        let mut source = QueryGuids(0);
        let mut factory = QueryObserverFactory {
            created: 0,
            limit: 2,
        };
        let mut options = FixedPartOptions(PartApplicationOptions {
            startup_sketch_plane: plane,
            ..Default::default()
        });
        let client = Client::new(
            FileHost::new(&mut files, &mut source, "test")
                .with_observer_factory(&mut factory)
                .with_part_preferences(&mut options)
                .with_startup_projection(project),
        );
        let part = client.documents().create(&create_request("part")).unwrap();
        let assembly = client
            .documents()
            .create(&create_request("assembly"))
            .unwrap();
        drop(client);
        let part_sketches = &files
            .workspace()
            .session(part.id)
            .unwrap()
            .snapshot()
            .sketches;
        assert_eq!(part_sketches.sketches().len(), usize::from(plane.is_some()));
        if let Some(plane) = plane {
            let sketch = &part_sketches.sketches()[0];
            assert_eq!(sketch.name(), "Sketch1");
            assert_eq!(sketch.placement(), SketchPlacement::OriginPlane(plane));
            assert_eq!(sketch.point_projections().len(), usize::from(project));
        }
        assert!(
            files
                .workspace()
                .session(assembly.id)
                .unwrap()
                .snapshot()
                .sketches
                .sketches()
                .is_empty()
        );
    }
}
#[test]
fn refused_startup_sketch_discards_new_document_and_restores_previous_focus() {
    let (mut files, previous, mut source) = created_files(false);
    let mut factory = VetoSketchFactory;
    let mut options = FixedPartOptions(PartApplicationOptions {
        startup_sketch_plane: Some(WorkPlaneOrigin::XY),
        ..Default::default()
    });
    let client = Client::new(
        FileHost::new(&mut files, &mut source, "test")
            .with_observer_factory(&mut factory)
            .with_part_preferences(&mut options),
    );
    let error = client
        .documents()
        .create(&create_request("part"))
        .unwrap_err();
    assert!(error.to_string().contains("settings are locked"), "{error}");
    drop(client);
    assert_eq!(files.workspace().documents(), &[previous]);
    assert_eq!(files.workspace().active_document(), Some(previous));
}
