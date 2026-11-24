# Part replay during sketch inspection

Issue #4079. A real Martensite prism supplies the face for one sketch. The
original prism sketch has independent placement. Editing the prism depth
invalidates the previous part result.

The read-only API initially rejected a Get for the independent sketch
because it tried to use a resolver bound to another document. After the
fix, Get returns the healthy independent sketch and List still rejects
the foreign resolver because the list includes the face sketch.

With the mutable file API and the correct resolution policy, Get for the
independent sketch starts zero named part replays. Get for the face sketch
reports it healthy after two replays, one to refresh the changed solve and
one to publish the resulting generation. Repeating that Get reuses the
current result.

The [native nested-sketch scenario](published/cucumber.json) also passed
against model-browser behavior source
`f590a2cc540119948799f5733a5fc2276e6758cc` on Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe.
