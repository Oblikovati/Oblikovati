// SPDX-License-Identifier: GPL-2.0-only
//! Native fixture candidates for the constraint settings scenarios.
use super::super::ApplicationWorld;
use oblikovati_model::{
    document_sketch::SketchPlacement,
    feature::work_plane::WorkPlaneOrigin,
    quantity::Dimension,
    sketch::{
        PlanarConstraint, SketchGeometricConstraintCreation, SketchPointInput,
        SketchPrimitiveCreation,
    },
    units::UnitsOfMeasure,
};
use std::path::Path;

#[test]
fn constraint_settings_native_fixtures_reopen_with_expected_geometry() {
    let mut world = ApplicationWorld::new();
    for with_line in [false, true] {
        world.start_sketch_on("XY Plane");
        if with_line {
            add_horizontal_line(&mut world);
        }
        verify_geometry(&mut world, with_line);
        world.finish_sketch();
        let document = world.active_id();
        let name = if with_line {
            "constraint-settings-line.opd"
        } else {
            "planar-sketch-empty-xy.opd"
        };
        let path = world.save_part_to_file(document, name);
        verify_saved_plane(&path);
        world.close_active_tab();
        world.reopen_through_dialog(&path);
        open_sketch(&mut world);
        verify_geometry(&mut world, with_line);
        if let Some(directory) = std::env::var_os("OBK_EXPORT_CONSTRAINT_FIXTURES") {
            export_fixture(&path, &Path::new(&directory).join(name));
        }
        world.finish_sketch();
        world.close_active_tab();
    }
    for (name, with_line) in [
        ("planar-sketch-empty-xy.opd", false),
        ("constraint-settings-line.opd", true),
    ] {
        let path = Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../test-utilities/fixtures")
            .join(name);
        verify_saved_plane(&path);
        world.reopen_through_dialog(&path);
        open_sketch(&mut world);
        verify_geometry(&mut world, with_line);
        world.finish_sketch();
        world.close_active_tab();
    }
    world.close();
}

fn add_horizontal_line(world: &mut ApplicationWorld) {
    let length = UnitsOfMeasure::default()
        .from_preferred(40., Dimension::Length)
        .value;
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let mut commands = world.app().owner.document_commands(edit.document);
    let line = commands
        .create_sketch_primitive(
            edit.sketch,
            &SketchPrimitiveCreation::Line {
                endpoints: [
                    SketchPointInput::New([1., 1.]),
                    SketchPointInput::New([1. + length, 1.]),
                ],
                construction: false,
                centerline: false,
            },
        )
        .unwrap()
        .entity;
    commands
        .create_sketch_geometric_constraint(
            edit.sketch,
            &SketchGeometricConstraintCreation::Relation(PlanarConstraint::LineHorizontal { line }),
        )
        .unwrap();
    world.pump(Vec::new());
}

fn verify_geometry(world: &mut ApplicationWorld, with_line: bool) {
    let geometry = world.edited_geometry();
    assert_eq!(geometry.lines().len(), usize::from(with_line));
    assert!(geometry.circles().is_empty() && geometry.arcs().is_empty());
    assert_eq!(geometry.dimensional_constraints().count(), 0);
    if !with_line {
        assert!(geometry.constraints().is_empty());
        return;
    }
    let line = &geometry.lines()[0];
    let points = line
        .endpoints
        .map(|id| geometry.point(id).unwrap().position);
    let length = UnitsOfMeasure::default()
        .from_preferred(40., Dimension::Length)
        .value;
    assert!(
        ((points[1][0] - points[0][0]).hypot(points[1][1] - points[0][1]) - length).abs() < 1e-9
    );
    assert_eq!(geometry.constraints().len(), 1);
    assert!(
        matches!(geometry.constraints()[0].relation, PlanarConstraint::LineHorizontal { line: id } if id == line.id)
    );
}

fn verify_saved_plane(path: &Path) {
    let package =
        oblikovati_persistence::package::DocumentPackage::decode(&std::fs::read(path).unwrap())
            .unwrap();
    let restored = package.restore_sketches().unwrap();
    let [sketch] = restored.sketches.sketches() else {
        panic!("expected one sketch")
    };
    assert_eq!(sketch.name(), "Sketch1");
    assert_eq!(
        sketch.placement(),
        SketchPlacement::OriginPlane(WorkPlaneOrigin::XY)
    );
}

fn open_sketch(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    let row = world
        .frame
        .runs
        .iter()
        .find(|run| run.text.starts_with("[DOF"))
        .expect("saved sketch row")
        .center()
        .map(|value| value as i32);
    world.double_click(row);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

fn export_fixture(source: &Path, output: &Path) {
    use std::io::Write;
    let mut file = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(output)
        .unwrap();
    file.write_all(&std::fs::read(source).unwrap()).unwrap();
}
