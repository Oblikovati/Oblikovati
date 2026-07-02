// SPDX-License-Identifier: GPL-2.0-only

package benchgen

import (
	"fmt"
	"path"

	obkmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// Stats reports the realized shape of a generated assembly so the CLI can print it and
// tests can assert the spec was met. LeafPlacements is ground truth from the flattened
// occurrence tree (PlacedBodies); PerTier records the realized placement count of each
// tier.
type Stats struct {
	Profile        string
	LeafPlacements int
	PerTier        map[Tier]int
	UniqueMeshes   int
	SubAssemblies  int
	Documents      int
	Depth          int
}

// Generate builds the synthetic assembly described by p into ws under dirPrefix and
// returns the root assembly document with the realized stats. Every part and
// sub-assembly is a registered document placed by file reference, so the result both
// drives the in-memory benchmarks and saves to a loadable .obk set. Example:
//
//	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
//	root, stats, err := benchgen.Generate(ws, "car30k", benchgen.Auto30k())
func Generate(ws *doc.Workspace, dirPrefix string, p Profile) (*doc.Document, Stats, error) {
	b, err := newBuilder(ws, dirPrefix, p)
	if err != nil {
		return nil, Stats{}, err
	}
	root, err := b.buildRoot()
	if err != nil {
		return nil, Stats{}, err
	}
	return root, b.stats(root), nil
}

// builder threads the shared generation state (workspace, pools, the shared fastener
// kit, round-robin cursors, and realized counters) through the per-level tree helpers,
// keeping each helper to a single placement loop.
type builder struct {
	ws        *doc.Workspace
	dirPrefix string
	p         Profile
	pools     map[Tier][]*doc.Document
	kit       *doc.Document
	kitSize   int
	cursor    map[Tier]int
	perTier   map[Tier]int
	subAsm    int
	asmSeq    int
	bayCount  int
	bayDone   int
}

// newBuilder constructs the unique part pools and the one shared fastener kit, then
// returns a builder ready to assemble the tree. The kit holds kitSize fasteners and is
// placed once per leaf bay, so it is the assembly's dominant flyweight — a single
// definition reached through every bay path (the DAG node the renderer instances).
func newBuilder(ws *doc.Workspace, dirPrefix string, p Profile) (*builder, error) {
	pools, unique, err := buildPools(ws, dirPrefix, p)
	if err != nil {
		return nil, err
	}
	_ = unique // pools sized from the profile; TotalUniqueMeshes reports it in Stats
	b := &builder{
		ws: ws, dirPrefix: dirPrefix, p: p, pools: pools,
		cursor: map[Tier]int{}, perTier: map[Tier]int{},
		bayCount: p.BayCount(),
	}
	b.kitSize = kitSize(p, b.bayCount)
	if err := b.buildFastenerKit(); err != nil {
		return nil, err
	}
	b.perTier[Fastener] = b.kitSize * b.bayCount
	return b, nil
}

// buildPools builds the unique-definition pool for every tier and reports the total
// unique-mesh count.
func buildPools(ws *doc.Workspace, dirPrefix string, p Profile) (map[Tier][]*doc.Document, int, error) {
	pools := map[Tier][]*doc.Document{}
	unique := 0
	for _, spec := range p.Tiers {
		pool, err := buildPool(ws, dirPrefix, spec)
		if err != nil {
			return nil, 0, err
		}
		pools[spec.Tier] = pool
		unique += len(pool)
	}
	return pools, unique, nil
}

// kitSize is the number of fasteners in the shared kit, chosen so kitSize×bayCount lands
// nearest the fastener placement target (sharing forces the product, so the realized
// total is reported in Stats rather than matched exactly).
func kitSize(p Profile, bayCount int) int {
	spec, ok := p.Tier(Fastener)
	if !ok || bayCount == 0 {
		return 0
	}
	return (spec.Placements + bayCount/2) / bayCount
}

// buildFastenerKit creates the shared kit sub-assembly, round-robin placing kitSize
// fasteners from the fastener pool so a tiny unique set yields a large instanced count.
func (b *builder) buildFastenerKit() error {
	kitDoc, kitDef, err := b.newAssembly("fastenerkit")
	if err != nil {
		return err
	}
	for i := 0; i < b.kitSize; i++ {
		part := b.nextPart(Fastener, false) // counted once via kitSize×bayCount, not per path
		name := fmt.Sprintf("bolt:%d", i)
		if _, err := kitDef.PlaceComponentFromFile(kitDoc, part, name, partTransform(i)); err != nil {
			return err
		}
	}
	b.kit = kitDoc
	return nil
}

// buildRoot creates the root assembly and places the vehicle systems under it.
func (b *builder) buildRoot() (*doc.Document, error) {
	rootDoc, rootDef, err := b.newAssembly("root")
	if err != nil {
		return nil, err
	}
	for i := 0; i < b.p.Systems; i++ {
		if err := b.buildSystem(rootDoc, rootDef, i); err != nil {
			return nil, err
		}
	}
	return rootDoc, nil
}

// buildSystem creates one vehicle-system sub-assembly under the root and fills it with
// modules.
func (b *builder) buildSystem(parentDoc *doc.Document, parentDef *compdef.AssemblyComponentDefinition, idx int) error {
	sysDoc, sysDef, err := b.newAssembly("system")
	if err != nil {
		return err
	}
	b.place(parentDoc, parentDef, sysDoc, fmt.Sprintf("system:%d", idx), bayTransform(idx, b.p.Systems))
	for i := 0; i < b.p.Modules; i++ {
		if err := b.buildModule(sysDoc, sysDef, i); err != nil {
			return err
		}
	}
	return nil
}

// buildModule creates one module sub-assembly under a system and fills it with
// sub-modules.
func (b *builder) buildModule(parentDoc *doc.Document, parentDef *compdef.AssemblyComponentDefinition, idx int) error {
	modDoc, modDef, err := b.newAssembly("module")
	if err != nil {
		return err
	}
	b.place(parentDoc, parentDef, modDoc, fmt.Sprintf("module:%d", idx), partTransform(idx))
	for i := 0; i < b.p.SubModules; i++ {
		if err := b.buildSubModule(modDoc, modDef, i); err != nil {
			return err
		}
	}
	return nil
}

// buildSubModule creates one sub-module sub-assembly under a module and fills it with
// leaf bays.
func (b *builder) buildSubModule(parentDoc *doc.Document, parentDef *compdef.AssemblyComponentDefinition, idx int) error {
	subDoc, subDef, err := b.newAssembly("submodule")
	if err != nil {
		return err
	}
	b.place(parentDoc, parentDef, subDoc, fmt.Sprintf("submodule:%d", idx), partTransform(idx))
	for i := 0; i < b.p.Bays; i++ {
		if err := b.buildBay(subDoc, subDef, i); err != nil {
			return err
		}
	}
	return nil
}

// buildBay creates one leaf bay, scatters its share of the non-fastener tiers into it,
// and places the shared fastener kit — the deepest structural level, where the bulk of
// the placements and the DAG sharing land.
func (b *builder) buildBay(parentDoc *doc.Document, parentDef *compdef.AssemblyComponentDefinition, idx int) error {
	bayDoc, bayDef, err := b.newAssembly("bay")
	if err != nil {
		return err
	}
	b.place(parentDoc, parentDef, bayDoc, fmt.Sprintf("bay:%d", idx), bayTransform(b.bayDone, b.bayCount))
	if err := b.fillBay(bayDoc, bayDef); err != nil {
		return err
	}
	b.bayDone++
	return nil
}

// fillBay places this bay's quota of each non-fastener tier plus the shared kit.
func (b *builder) fillBay(bayDoc *doc.Document, bayDef *compdef.AssemblyComponentDefinition) error {
	slot := 0
	for _, tier := range []Tier{Bracket, Machined, System} {
		n := b.bayQuota(tier)
		if err := b.placeTierInBay(bayDoc, bayDef, tier, n, &slot); err != nil {
			return err
		}
	}
	_, err := bayDef.PlaceComponentFromFile(bayDoc, b.kit, "fastenerkit:0", partTransform(slot))
	return err
}

// bayQuota is this bay's share of a tier's total placements, spreading the remainder
// across the first bays so the realized total equals the tier's target exactly.
func (b *builder) bayQuota(tier Tier) int {
	spec, ok := b.p.Tier(tier)
	if !ok {
		return 0
	}
	return distribute(spec.Placements, b.bayCount, b.bayDone)
}

// placeTierInBay round-robin places n parts of tier into the bay, advancing the shared
// slot so each placement gets a distinct in-bay jitter.
func (b *builder) placeTierInBay(bayDoc *doc.Document, bayDef *compdef.AssemblyComponentDefinition, tier Tier, n int, slot *int) error {
	for i := 0; i < n; i++ {
		part := b.nextPart(tier, true)
		name := fmt.Sprintf("%s:%d", tier, *slot)
		if _, err := bayDef.PlaceComponentFromFile(bayDoc, part, name, partTransform(*slot)); err != nil {
			return err
		}
		*slot++
	}
	return nil
}

// nextPart returns the next part of tier round-robin over its pool. When count is true
// the placement is tallied in perTier (fastener placements are tallied via the kit
// multiplier instead, so the kit build passes false).
func (b *builder) nextPart(tier Tier, count bool) *doc.Document {
	pool := b.pools[tier]
	d := pool[b.cursor[tier]%len(pool)]
	b.cursor[tier]++
	if count {
		b.perTier[tier]++
	}
	return d
}

// newAssembly creates a uniquely-named assembly document and returns it with its
// definition, tallying the sub-assembly count.
func (b *builder) newAssembly(role string) (*doc.Document, *compdef.AssemblyComponentDefinition, error) {
	name := path.Join(b.dirPrefix, "asm", fmt.Sprintf("%s_%06d", role, b.asmSeq)) + doc.Assembly.Extension()
	b.asmSeq++
	d, err := compdef.AddAssembly(b.ws, name, false)
	if err != nil {
		return nil, nil, fmt.Errorf("benchgen: add assembly %q: %w", name, err)
	}
	def, ok := d.Content().(*compdef.AssemblyComponentDefinition)
	if !ok {
		return nil, nil, fmt.Errorf("benchgen: assembly %q content is %T, want *AssemblyComponentDefinition", name, d.Content())
	}
	b.subAsm++
	return d, def, nil
}

// place adds a persistent occurrence of child in parent at tf, panicking only on the
// programmer error of placing non-placeable content (every child here is a part or
// assembly document we just created).
func (b *builder) place(parentDoc *doc.Document, parentDef *compdef.AssemblyComponentDefinition, child *doc.Document, name string, tf obkmath.Matrix4) {
	if _, err := parentDef.PlaceComponentFromFile(parentDoc, child, name, tf); err != nil {
		panic(fmt.Sprintf("benchgen: place %q in %q: %v", name, parentDoc.FullDocumentName(), err))
	}
}

// stats assembles the realized report, taking LeafPlacements from the flattened tree as
// ground truth.
func (b *builder) stats(root *doc.Document) Stats {
	def := root.Content().(*compdef.AssemblyComponentDefinition)
	return Stats{
		Profile:        b.p.Name,
		LeafPlacements: len(def.PlacedBodies()),
		PerTier:        b.perTier,
		UniqueMeshes:   b.p.TotalUniqueMeshes(),
		SubAssemblies:  b.subAsm,
		Documents:      b.ws.Count(),
		Depth:          7, // root→system→module→submodule→bay→kit→fastener
	}
}

// distribute splits total across n buckets as evenly as possible, giving bucket index
// one extra when index < total%n, so summing distribute over all indices equals total.
func distribute(total, n, index int) int {
	if n <= 0 {
		return 0
	}
	base := total / n
	if index < total%n {
		return base + 1
	}
	return base
}
