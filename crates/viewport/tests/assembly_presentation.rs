// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{ViewportScene, ViewportTriangle};
use std::collections::BTreeMap;

#[derive(Debug, Default)]
struct FakeScene {
    triangles: Vec<ViewportTriangle>,
    errors: Vec<String>,
}
#[derive(Default)]
struct FakeAssemblySource {
    nodes: BTreeMap<u64, Vec<AssemblyOccurrenceRecipe>>,
}
fn occurrence(name: &str, target: &str, matrix: [f64; 16]) -> AssemblyOccurrenceRecipe {
    AssemblyOccurrenceRecipe {
        name: name.into(),
        component: target.into(),
        transform: matrix.to_vec(),
        ..Default::default()
    }
}
fn transform(scale: f64, offset: f64) -> [f64; 16] {
    [
        scale, 0., 0., offset, 0., 1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1.,
    ]
}
impl AssemblyPresentationSource for FakeAssemblySource {
    type Output = FakeScene;
    fn empty(&mut self) -> FakeScene {
        FakeScene::default()
    }
    fn occurrences(&self, owner: u64) -> Result<(u64, Vec<AssemblyOccurrenceRecipe>), String> {
        Ok((
            owner + 100,
            self.nodes
                .get(&owner)
                .ok_or_else(|| format!("missing assembly {owner}"))?
                .clone(),
        ))
    }
    fn component(&self, _: u64, occurrence: &AssemblyOccurrenceRecipe) -> Result<u64, String> {
        occurrence
            .component
            .parse()
            .map_err(|_| format!("unresolved {}", occurrence.component))
    }
    fn is_assembly(&self, component: u64) -> Result<bool, String> {
        Ok(self.nodes.contains_key(&component))
    }
    fn part(&mut self, component: u64) -> Result<FakeScene, String> {
        let triangle = ViewportTriangle {
            face_index: None,
            positions: [[component as f32, 0., 0.]; 3],
            normals: None,
            color: [0.2, 0.3, 0.4, 1.],
        };
        Ok(FakeScene {
            triangles: vec![triangle],
            errors: vec![],
        })
    }
    fn place(
        &self,
        scene: &mut FakeScene,
        occurrence: &AssemblyOccurrenceRecipe,
    ) -> Result<(), String> {
        scene.triangles = ViewportScene {
            background: [0.; 3],
            triangles: scene.triangles.clone(),
        }
        .placed(&occurrence.placement()?)?
        .triangles;
        Ok(())
    }
    fn select(&self, scene: &mut FakeScene) {
        for triangle in &mut scene.triangles {
            triangle.color = [1., 0., 0., 1.];
        }
    }
    fn append(&self, parent: &mut FakeScene, child: FakeScene) {
        parent.triangles.extend(child.triangles);
        parent.errors.extend(child.errors);
    }
    fn fail(&self, parent: &mut FakeScene, message: String) {
        parent.errors.push(message);
    }
}
#[test]
fn nested_placements_preserve_order_visibility_and_selection_boundaries() {
    let mut source = FakeAssemblySource::default();
    let mut suppressed = occurrence("hidden", "9", transform(1., 0.));
    suppressed.suppressed = true;
    let mut virtual_item = occurrence("virtual", "8", transform(1., 0.));
    virtual_item.virtual_ = true;
    source.nodes.insert(
        1,
        vec![
            occurrence("nested", "2", transform(2., 5.)),
            suppressed,
            virtual_item,
            occurrence("direct", "4", transform(1., 1.)),
        ],
    );
    source
        .nodes
        .insert(2, vec![occurrence("leaf", "3", transform(3., 7.))]);
    let selection = [(1, 101, 0)];
    let result = present_assembly(&mut source, 1, Some(&selection)).unwrap();
    assert_eq!(result.triangles.len(), 2);
    assert_eq!(result.triangles[0].positions[0], [37., 0., 0.]);
    assert_eq!(result.triangles[1].positions[0], [5., 0., 0.]);
    assert_eq!(result.triangles[0].color, [1., 0., 0., 1.]);
    assert_eq!(result.triangles[1].color, [0.2, 0.3, 0.4, 1.]);
    assert!(result.errors.is_empty());
    let unselected = present_assembly(&mut source, 1, None).unwrap();
    assert_eq!(unselected.triangles[0].color, [0.2, 0.3, 0.4, 1.]);
    assert_eq!(
        unselected.triangles[0].positions,
        result.triangles[0].positions
    );
}
#[test]
fn failures_are_local_ordered_and_do_not_hide_valid_siblings() {
    let mut source = FakeAssemblySource::default();
    source.nodes.insert(
        1,
        vec![
            occurrence("lost", "absent", transform(1., 0.)),
            occurrence("good", "7", transform(1., 0.)),
            occurrence("lost2", "bad", transform(1., 0.)),
        ],
    );
    let result = present_assembly(&mut source, 1, None).unwrap();
    assert_eq!(result.triangles.len(), 1);
    assert_eq!(
        result.errors,
        [
            "occurrence \"lost\": unresolved absent",
            "occurrence \"lost2\": unresolved bad"
        ]
    );
    assert!(
        present_assembly(&mut source, 999, None)
            .unwrap_err()
            .contains("missing assembly 999")
    );
}

#[test]
fn nested_placements_round_at_each_original_boundary() {
    let mut source = FakeAssemblySource::default();
    source
        .nodes
        .insert(1, vec![occurrence("nested", "2", transform(1.1, 0.))]);
    source
        .nodes
        .insert(2, vec![occurrence("leaf", "3", transform(1.1, 0.))]);
    let scene = present_assembly(&mut source, 1, None).unwrap();
    let inner = (3.0_f64 * 1.1) as f32;
    let expected = (f64::from(inner) * 1.1) as f32;
    let combined = (3.0_f64 * (1.1 * 1.1)) as f32;
    assert_ne!(expected.to_bits(), combined.to_bits());
    assert_eq!(
        scene.triangles[0].positions[0][0].to_bits(),
        expected.to_bits()
    );
}

#[test]
fn nested_failure_remains_local_and_keeps_later_parent_siblings() {
    let mut source = FakeAssemblySource::default();
    source.nodes.insert(
        1,
        vec![
            occurrence("nested", "2", transform(1., 0.)),
            occurrence("other", "4", transform(1., 0.)),
        ],
    );
    source.nodes.insert(
        2,
        vec![
            occurrence("lost", "bad", transform(1., 0.)),
            occurrence("good", "3", transform(1., 0.)),
        ],
    );
    let result = present_assembly(&mut source, 1, None).unwrap();
    assert_eq!(
        result
            .triangles
            .iter()
            .map(|triangle| triangle.positions[0][0])
            .collect::<Vec<_>>(),
        [3., 4.]
    );
    assert_eq!(result.errors, ["occurrence \"lost\": unresolved bad"]);
}

#[test]
fn invalid_placement_is_local_and_keeps_the_other_occurrence() {
    let mut source = FakeAssemblySource::default();
    let mut invalid = transform(1., 0.);
    invalid[15] = 0.;
    source.nodes.insert(
        1,
        vec![
            occurrence("bad placement", "3", invalid),
            occurrence("valid", "4", transform(1., 2.)),
        ],
    );
    let result = present_assembly(&mut source, 1, None).unwrap();
    assert_eq!(result.triangles.len(), 1);
    assert_eq!(result.triangles[0].positions[0], [6., 0., 0.]);
    assert!(result.errors[0].contains("occurrence \"bad placement\""));
    assert!(result.errors[0].contains("finite row-major affine matrix"));
}
#[test]
fn recursive_reference_stops_at_the_shared_depth_limit() {
    let mut source = FakeAssemblySource::default();
    source
        .nodes
        .insert(1, vec![occurrence("cycle", "1", transform(1., 0.))]);
    let result = present_assembly(&mut source, 1, None).unwrap();
    assert!(result.triangles.is_empty());
    assert_eq!(result.errors.len(), 1);
    assert!(result.errors[0].contains("expected at most 128 nested levels"));
}
