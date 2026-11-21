// SPDX-License-Identifier: GPL-2.0-only
//! Resolve Shell openings to one exact current body (#4106).

pub(super) struct ShellBodyFaces {
    pub body: Vec<u8>,
    pub faces: Vec<Vec<u8>>,
}

pub(super) fn unique_shell_face_body(
    faces: &[Vec<u8>],
    bodies: &[ShellBodyFaces],
) -> Result<Vec<u8>, String> {
    let mut selected = None;
    for face in faces {
        let matches: Vec<_> = bodies
            .iter()
            .enumerate()
            .flat_map(|(index, body)| {
                body.faces
                    .iter()
                    .filter(move |candidate| *candidate == face)
                    .map(move |_| index)
            })
            .collect();
        let [index] = matches.as_slice() else {
            return Err(format!(
                "shell face {face:?}, matches {}: expected exactly one current face",
                matches.len()
            ));
        };
        if selected.is_some_and(|previous| previous != *index) {
            return Err(format!(
                "shell face {face:?}: expected every face on the same current body"
            ));
        }
        selected = Some(*index);
    }
    let index =
        selected.ok_or_else(|| "shell faces []: expected at least one current face".to_owned())?;
    let key = &bodies[index].body;
    let matches = bodies.iter().filter(|body| body.body == *key).count();
    if matches != 1 {
        return Err(format!(
            "shell body {key:?}, matches {matches}: expected exactly one current body"
        ));
    }
    Ok(key.clone())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn selected_faces_from_distinct_current_bodies_are_rejected() {
        let bodies = vec![
            ShellBodyFaces {
                body: vec![1],
                faces: vec![vec![10]],
            },
            ShellBodyFaces {
                body: vec![2],
                faces: vec![vec![20]],
            },
        ];
        assert_eq!(
            unique_shell_face_body(&[vec![10]], &bodies).unwrap(),
            vec![1]
        );
        assert!(unique_shell_face_body(&[vec![10], vec![20]], &bodies).is_err());
        assert!(unique_shell_face_body(&[vec![30]], &bodies).is_err());
    }

    #[test]
    fn ambiguous_or_repeated_face_keys_are_rejected() {
        let bodies = vec![ShellBodyFaces {
            body: vec![1],
            faces: vec![vec![10], vec![10]],
        }];
        assert!(unique_shell_face_body(&[vec![10]], &bodies).is_err());
        let bodies = vec![
            ShellBodyFaces {
                body: vec![1],
                faces: vec![vec![10]],
            },
            ShellBodyFaces {
                body: vec![1],
                faces: vec![vec![20]],
            },
        ];
        assert!(unique_shell_face_body(&[vec![10]], &bodies).is_err());
        assert!(unique_shell_face_body(&[vec![10], vec![20]], &bodies).is_err());
    }
}
