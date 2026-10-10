// SPDX-License-Identifier: GPL-2.0-only

use super::{
    Expression, ExpressionError, ReferenceSpan,
    node::{BinaryOperator, Node},
    token::{Token, TokenKind, tokenize},
};
use crate::{quantity::Dimension, units::UnitsOfMeasure};
use std::collections::BTreeMap;

// Limit recursive syntax, not a left-associative formula's length. Evaluation
// and destruction use flat storage regardless of the number of operations.
const MAX_NESTING: usize = 128;

struct Parser<'a> {
    tokens: Vec<Token<'a>>,
    position: usize,
    nodes: Vec<Node>,
    references: Vec<String>,
    spans: Vec<ReferenceSpan>,
    reference_indices: BTreeMap<String, usize>,
    units: &'a UnitsOfMeasure,
}

pub(super) fn compile(source: &str, units: &UnitsOfMeasure) -> Result<Expression, ExpressionError> {
    let mut parser = Parser {
        tokens: tokenize(source)?,
        position: 0,
        nodes: Vec::new(),
        references: Vec::new(),
        spans: Vec::new(),
        reference_indices: BTreeMap::new(),
        units,
    };
    let root = parser.binary(0, 0)?;
    if parser.peek().kind != TokenKind::End {
        return Err(parser.peek().expected("the end of the expression"));
    }
    Ok(Expression {
        source: source.into(),
        nodes: parser.nodes,
        references: parser.references,
        spans: parser.spans,
        root,
    })
}

impl Parser<'_> {
    fn peek(&self) -> Token<'_> {
        self.tokens[self.position]
    }

    fn advance(&mut self) -> Token<'_> {
        let token = self.tokens[self.position];
        if token.kind != TokenKind::End {
            self.position += 1;
        }
        token
    }

    fn append(&mut self, node: Node) -> usize {
        let index = self.nodes.len();
        self.nodes.push(node.fold(&self.nodes));
        index
    }

    fn binary(&mut self, precedence: u8, nesting: usize) -> Result<usize, ExpressionError> {
        if precedence == 2 {
            return self.factor(nesting);
        }
        let mut left = self.binary(precedence + 1, nesting)?;
        while let Some(operator) = operator(self.peek().kind, precedence) {
            self.advance();
            let right = self.binary(precedence + 1, nesting)?;
            left = self.append(Node::Binary(operator, left, right));
        }
        Ok(left)
    }

    fn factor(&mut self, nesting: usize) -> Result<usize, ExpressionError> {
        if nesting > MAX_NESTING {
            return Err(self
                .peek()
                .expected("at most 128 nested groups, calls, or unary operators"));
        }
        match self.peek().kind {
            TokenKind::Minus => {
                self.advance();
                let child = self.factor(nesting + 1)?;
                Ok(self.append(Node::Negate(child)))
            }
            TokenKind::Number => self.number(),
            TokenKind::Name => self.identifier(nesting),
            TokenKind::Open => self.parenthesized(nesting),
            _ => Err(self
                .peek()
                .expected("a number, reference, function call, or parenthesized expression")),
        }
    }

    fn number(&mut self) -> Result<usize, ExpressionError> {
        let token = self.tokens[self.position];
        self.advance();
        let next = self.tokens[self.position];
        let literal = format!("{} {}", token.text, next.text);
        if next.kind == TokenKind::Name
            && let Ok(quantity) = self.units.parse(&literal, Dimension::Unitless)
        {
            self.advance();
            return Ok(self.append(Node::Literal(quantity)));
        }
        let quantity = self
            .units
            .parse(token.text, Dimension::Unitless)
            .map_err(|_| token.expected("a finite decimal literal"))?;
        Ok(self.append(Node::Literal(quantity)))
    }

    fn identifier(&mut self, nesting: usize) -> Result<usize, ExpressionError> {
        let start = self.peek().position;
        let name = self.advance().text.to_owned();
        let end = start + name.len();
        if self.peek().kind == TokenKind::Open {
            let arguments = self.arguments(nesting + 1)?;
            return Ok(self.append(Node::Call(name, arguments)));
        }
        let reference = self.reference_index(name);
        self.spans.push(ReferenceSpan {
            start,
            end,
            reference,
        });
        Ok(self.append(Node::Reference(reference)))
    }

    fn reference_index(&mut self, name: String) -> usize {
        if let Some(index) = self.reference_indices.get(&name) {
            return *index;
        }
        let index = self.references.len();
        self.reference_indices.insert(name.clone(), index);
        self.references.push(name);
        index
    }

    fn parenthesized(&mut self, nesting: usize) -> Result<usize, ExpressionError> {
        self.advance();
        let inner = self.binary(0, nesting + 1)?;
        self.close()?;
        Ok(inner)
    }

    fn arguments(&mut self, nesting: usize) -> Result<Vec<usize>, ExpressionError> {
        self.advance();
        let mut arguments = Vec::new();
        if self.peek().kind != TokenKind::Close {
            arguments = self.argument_list(nesting)?;
        }
        self.close()?;
        Ok(arguments)
    }

    fn argument_list(&mut self, nesting: usize) -> Result<Vec<usize>, ExpressionError> {
        let mut arguments = vec![self.binary(0, nesting)?];
        while self.peek().kind == TokenKind::Comma {
            self.advance();
            arguments.push(self.binary(0, nesting)?);
        }
        Ok(arguments)
    }

    fn close(&mut self) -> Result<(), ExpressionError> {
        if self.peek().kind != TokenKind::Close {
            return Err(self.peek().expected("')'"));
        }
        self.advance();
        Ok(())
    }
}

fn operator(kind: TokenKind, precedence: u8) -> Option<BinaryOperator> {
    match (kind, precedence) {
        (TokenKind::Plus, 0) => Some(BinaryOperator::Add),
        (TokenKind::Minus, 0) => Some(BinaryOperator::Subtract),
        (TokenKind::Star, 1) => Some(BinaryOperator::Multiply),
        (TokenKind::Slash, 1) => Some(BinaryOperator::Divide),
        _ => None,
    }
}
