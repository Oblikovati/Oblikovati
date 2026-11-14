// SPDX-License-Identifier: GPL-2.0-only
//! Explicit units for scalar editor text, preserving the authored expression structure.
use super::token::{Token, TokenKind, tokenize};
use crate::{quantity::Dimension, units::UnitsOfMeasure};

pub(super) fn scalar_display(
    source: &str,
    units: &UnitsOfMeasure,
    bare_unit: &str,
    promote_first: bool,
) -> String {
    let Ok(tokens) = tokenize(source) else {
        return source.into();
    };
    let mut parts = Vec::new();
    let mut index = 0;
    let mut promoted = false;
    while tokens[index].kind != TokenKind::End {
        let token = tokens[index];
        if token.kind != TokenKind::Number {
            parts.push(token.text.to_owned());
            index += 1;
            continue;
        }
        let (literal, consumed, explicit) = numeric_literal(&tokens[index..], units);
        let suffix = if explicit {
            ""
        } else if promote_first && !promoted {
            bare_unit
        } else {
            "ul"
        };
        if !explicit && promote_first && !promoted {
            promoted = true;
        }
        parts.push(if suffix.is_empty() {
            literal
        } else {
            format!("{literal} {suffix}")
        });
        index += consumed;
    }
    parts.join(" ")
}
fn numeric_literal(tokens: &[Token<'_>], units: &UnitsOfMeasure) -> (String, usize, bool) {
    let number = tokens[0].text;
    let next = tokens[1];
    if next.kind != TokenKind::Name {
        return (number.into(), 1, false);
    }
    let power = tokens.len() >= 4
        && tokens[2].kind == TokenKind::Caret
        && tokens[3].kind == TokenKind::Number
        && next.position + next.text.len() == tokens[2].position
        && tokens[2].position + 1 == tokens[3].position;
    let suffix = if power {
        format!("{}^{}", next.text, tokens[3].text)
    } else {
        next.text.into()
    };
    if units
        .parse(&format!("{number} {suffix}"), Dimension::Unitless)
        .is_err()
    {
        return (number.into(), 1, false);
    }
    (
        format!("{number} {suffix}"),
        if power { 4 } else { 2 },
        true,
    )
}
