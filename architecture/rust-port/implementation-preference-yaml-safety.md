# Preference YAML write safety

Application, marking-menu, theme, orientation, and keymap writes check their
encoded YAML before staging a file. A mapping key spelled `<<` is refused
because the current YAML value representation cannot distinguish a literal key
from a merge key. Literal text in values remains supported. The original file
and the in-memory preference remain unchanged on refusal. (#4007)

The preference encoder checks the output of the existing serializer and writes
those same bytes. Project configuration uses a closed typed schema; project
asset libraries use the document YAML guard. Fake-filesystem tests cover the
preference writes and project configuration. They do not prove synchronization
with another process. External changes during preference saves remain tracked in issue #4008.
Syntax-preserving merge support remains tracked in issue #4005.
