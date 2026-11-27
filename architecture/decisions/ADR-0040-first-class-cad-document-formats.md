# ADR-0040: First-class CAD document formats

Date: 2026-11-07. Status: Accepted by the user's format instruction.
Implementation evidence: pending provider qualification and application BDD.

## Decision

SolidWorks and Autodesk Inventor files are first-class document formats beside
Oblikovati YAML. This extends the document workflows in ADR-0020 and supersedes
the import/export-only treatment of these formats in ADR-0018. Parser and
format-specific conversion ownership remains in Martensite.

| Document family | Provider | File types |
| --- | --- | --- |
| Oblikovati | Application-owned YAML persistence | Existing native document types |
| SolidWorks | [Bessemer](https://github.com/Oblikovati/Bessemer) | `.sldprt`, `.sldasm`, `.slddrw` |
| Inventor | [Corliss](https://github.com/Oblikovati/Corliss) | `.ipt`, `.iam`, `.idw`; additional provider document kinds require their own workspace support |

DWG and DXF continue to use acadrust through Martensite.

## Application behavior

- Open files directly into their document workspace. Keep their source format,
  path, references, and provider-owned content with the document session.
- Use the same editing, history, browser, scene, and API operations across
  supported document formats. Do not require conversion to YAML before editing.
- Save back to the current format. Save As selects a destination format
  explicitly. Report unsupported edits or writes before replacing the source.
- Preserve content that an edit does not change. Resolve referenced documents
  through the same project and document-opening workflow.
- Keep provider types behind project-owned interfaces. Public additions start
  in Oblikovati.API; GUI and API call the same host operations.

This replaces a separate legacy-translator workflow for these document families.
It does not add another document model or a second feature recompute pipeline.

## Qualification

Provider selection is not implementation evidence. Qualify each file version,
document kind, read direction, write direction, and edit capability independently.
The provider READMEs describe growing support; Corliss currently excludes ACIS
body writes. A reader or metadata writer does not prove general model editing.

BDD acceptance must cover direct open, browser and scene content, parameter and
feature edits, undo/redo, save in the source format, close/reopen, and referenced
documents. Test unchanged-content preservation and failed-write source safety.
Unsupported content and fidelity loss must reach the API and UI as diagnostics.
Missing capabilities remain required work rather than silent YAML conversion.
