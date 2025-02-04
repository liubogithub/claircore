package postgres

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/quay/claircore"
	"github.com/quay/claircore/internal/indexer"
	"github.com/quay/claircore/test"
	"github.com/quay/claircore/test/integration"
)

// mockScnr is a kind-agnostic scanner we will
// use for testing purposes.
type mockScnr struct {
	name    string
	kind    string
	version string
}

func (m mockScnr) Name() string {
	return m.name
}
func (m mockScnr) Kind() string {
	return m.kind
}
func (m mockScnr) Version() string {
	return m.version
}

type e2e struct {
	name       string
	failed     bool
	store      indexer.Store
	ctx        context.Context
	manifest   claircore.Manifest
	scnrs      indexer.VersionedScanners
	packageGen int
	distGen    int
	repoGen    int
}

func TestE2E(t *testing.T) {
	integration.Skip(t)
	ctx := context.Background()

	e2es := []e2e{
		{
			name: "3 scanners gen small",
			scnrs: indexer.VersionedScanners{
				mockScnr{
					name:    "test-scanner",
					kind:    "test",
					version: "v0.0.1",
				},
				mockScnr{
					name:    "test-scanner1",
					kind:    "test",
					version: "v0.0.11",
				},
				mockScnr{
					name:    "test-scanner2",
					kind:    "test",
					version: "v0.0.8",
				},
			},
			packageGen: 100,
			distGen:    150,
			repoGen:    50,
		},
		{
			name: "6 scanners gen small",
			scnrs: indexer.VersionedScanners{
				mockScnr{
					name:    "test-scanner",
					kind:    "test",
					version: "v0.0.1",
				},
				mockScnr{
					name:    "test-scanner1",
					kind:    "test",
					version: "v0.0.11",
				},
				mockScnr{
					name:    "test-scanner2",
					kind:    "test",
					version: "v0.0.8",
				},
				mockScnr{
					name:    "test-scanner3",
					kind:    "test",
					version: "v0.0.8",
				},
				mockScnr{
					name:    "test-scanner4",
					kind:    "test",
					version: "v0.0.8",
				},
				mockScnr{
					name:    "test-scanner5",
					kind:    "test",
					version: "v0.0.8",
				},
			},
			packageGen: 100,
			distGen:    150,
			repoGen:    50,
		},
		{
			name: "3 scanners gen large",
			scnrs: indexer.VersionedScanners{
				mockScnr{
					name:    "test-scanner",
					kind:    "test",
					version: "v0.0.1",
				},
				mockScnr{
					name:    "test-scanner1",
					kind:    "test",
					version: "v0.0.11",
				},
				mockScnr{
					name:    "test-scanner2",
					kind:    "test",
					version: "v0.0.8",
				},
			},
			packageGen: 1000,
			distGen:    1500,
			repoGen:    500,
		},
		{
			name: "6 scanners gen large",
			scnrs: indexer.VersionedScanners{
				mockScnr{
					name:    "test-scanner",
					kind:    "test",
					version: "v0.0.1",
				},
				mockScnr{
					name:    "test-scanner1",
					kind:    "test",
					version: "v0.0.11",
				},
				mockScnr{
					name:    "test-scanner2",
					kind:    "test",
					version: "v0.0.8",
				},
				mockScnr{
					name:    "test-scanner3",
					kind:    "test",
					version: "v0.0.8",
				},
				mockScnr{
					name:    "test-scanner4",
					kind:    "test",
					version: "v0.0.8",
				},
				mockScnr{
					name:    "test-scanner5",
					kind:    "test",
					version: "v0.0.8",
				},
			},
			packageGen: 1000,
			distGen:    1500,
			repoGen:    500,
		},
	}

	for _, e := range e2es {
		_, store, teardown := TestStore(ctx, t)
		defer teardown()

		layer := &claircore.Layer{
			Hash: claircore.MustParseDigest(`sha256:5f70bf18a086007016e948b04aed3b82103a36bea41755b6cddfaf10ace3c6ef`),
		}
		manifest := claircore.Manifest{
			Hash:   claircore.MustParseDigest(`sha256:fc92eec5cac70b0c324cec2933cd7db1c0eae7c9e2649e42d02e77eb6da0d15f`),
			Layers: []*claircore.Layer{layer},
		}

		e.store = store
		e.ctx = ctx
		e.manifest = manifest

		t.Run(e.name, e.Run)
	}
}

func (e *e2e) Run(t *testing.T) {
	type subtest struct {
		name string
		do   func(t *testing.T)
	}
	subtests := [...]subtest{
		{"RegisterScanner", e.RegisterScanner},
		{"PersistManifest", e.PersisteManifest},
		{"IndexAndRetrievePackages", e.IndexAndRetrievePackages},
		{"IndexAndRetrieveDistributions", e.IndexAndRetrieveDistributions},
		{"IndexAndRetrieveRepos", e.IndexAndRetrieveRepos},
		{"LayerScanned", e.LayerScanned},
		{"LayerScannedNotExists", e.LayerScannedNotExists},
		{"LayerScannedFalse", e.LayerScannedFalse},
		{"IndexReport", e.IndexReport},
	}
	for i := range subtests {
		subtest := subtests[i]
		t.Run(subtest.name, subtest.do)
		if e.failed {
			t.FailNow()
		}
	}
}

// PersistManifest confirms we create the necessary
// Manifest and Layer identifies so layer code
// foreign key references do not fail.
func (e *e2e) PersisteManifest(t *testing.T) {
	defer func() {
		e.failed = t.Failed()
	}()
	err := e.store.PersistManifest(e.ctx, e.manifest)
	if err != nil {
		t.Fatalf("failed to persist manifest: %v", err)
	}
}

// RegisterScanner confirms a scanner can be registered
// and provides this scanner for other subtests to use
func (e *e2e) RegisterScanner(t *testing.T) {
	defer func() {
		e.failed = t.Failed()
	}()
	err := e.store.RegisterScanners(e.ctx, e.scnrs)
	if err != nil {
		t.Fatalf("failed to register scnr: %v", err)
	}
}

// IndexAndRetreivePackages confirms inserting and
// selecting packages associated with a layer works
// correctly.
func (e *e2e) IndexAndRetrievePackages(t *testing.T) {
	defer func() {
		e.failed = t.Failed()
	}()
	A := test.GenUniquePackages(e.packageGen)

	for _, scnr := range e.scnrs {
		err := e.store.IndexPackages(e.ctx, A, e.manifest.Layers[0], scnr)
		if err != nil {
			t.Fatalf("failed to index package: %v", err)
		}
	}

	B, err := e.store.PackagesByLayer(e.ctx, e.manifest.Layers[0].Hash, e.scnrs)
	if err != nil {
		t.Fatalf("failed to retrieve packages by layer: %v", err)
	}

	if len(e.scnrs)*e.packageGen != len(B) {
		t.Fatalf("wanted len: %v got: %v", len(e.scnrs)*e.packageGen, len(B))
	}
}

// IndexAndRetreiveDistributions confirms inserting and
// selecting distributions associated with a layer works
// correctly.
func (e *e2e) IndexAndRetrieveDistributions(t *testing.T) {
	defer func() {
		e.failed = t.Failed()
	}()
	A := test.GenUniqueDistributions(e.distGen)

	for _, scnr := range e.scnrs {
		err := e.store.IndexDistributions(e.ctx, A, e.manifest.Layers[0], scnr)
		if err != nil {
			t.Fatalf("failed to index distributions: %v", err)
		}
	}

	B, err := e.store.DistributionsByLayer(e.ctx, e.manifest.Layers[0].Hash, e.scnrs)
	if err != nil {
		t.Fatalf("failed to retrieve distributions by layer: %v", err)
	}

	if len(e.scnrs)*e.distGen != len(B) {
		t.Fatalf("wanted len: %v got: %v", len(e.scnrs)*e.distGen, len(B))
	}
}

// IndexAndRetreiveRepos confirms inserting and
// selecting repositories associated with a layer works
// correctly.
func (e *e2e) IndexAndRetrieveRepos(t *testing.T) {
	defer func() {
		e.failed = t.Failed()
	}()
	A := test.GenUniqueRepositories(e.repoGen)

	for _, scnr := range e.scnrs {
		err := e.store.IndexRepositories(e.ctx, A, e.manifest.Layers[0], scnr)
		if err != nil {
			t.Fatalf("failed to index repos: %v", err)
		}
	}

	B, err := e.store.RepositoriesByLayer(e.ctx, e.manifest.Layers[0].Hash, e.scnrs)
	if err != nil {
		t.Fatalf("failed to retrieve repos by layer: %v", err)
	}

	if len(e.scnrs)*e.repoGen != len(B) {
		t.Fatalf("wanted len: %v got: %v", len(e.scnrs)*e.repoGen, len(B))
	}
}

// LayerScanned confirms the book keeping involved in marking a layer
// scanned works correctly.
func (e *e2e) LayerScanned(t *testing.T) {
	defer func() {
		e.failed = t.Failed()
	}()
	for _, scnr := range e.scnrs {
		err := e.store.SetLayerScanned(e.ctx, e.manifest.Layers[0].Hash, scnr)
		if err != nil {
			t.Fatalf("failed to set layer scanned: %v", err)
		}

		b, err := e.store.LayerScanned(e.ctx, e.manifest.Layers[0].Hash, scnr)
		if err != nil {
			t.Fatalf("failed to query if layer is scanned: %v", err)
		}
		if !b {
			t.Fatalf("expected layer to be scanned")
		}
	}
}

// LayerScannedNotExists confirms an error is returned when attempting
// to obtain if a layer was scanned by a non-existent scanner.
func (e *e2e) LayerScannedNotExists(t *testing.T) {
	defer func() {
		e.failed = t.Failed()
	}()
	scnr := mockScnr{
		name:    "invalid",
		kind:    "invalid",
		version: "invalid",
	}

	_, err := e.store.LayerScanned(e.ctx, e.manifest.Layers[0].Hash, scnr)
	if err == nil {
		t.Fatalf("expected error scnr not found error condition")
	}
}

// LayerScannedFalse confirms a false boolean is returned when attempting
// to obtain if a non-exitent layer was scanned by a valid scanner
func (e *e2e) LayerScannedFalse(t *testing.T) {
	defer func() {
		e.failed = t.Failed()
	}()

	// create a layer that has not been persisted to the store
	layer := &claircore.Layer{
		Hash: claircore.MustParseDigest(`sha256:5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03`),
	}

	b, err := e.store.LayerScanned(e.ctx, layer.Hash, e.scnrs[0])
	if err != nil {
		t.Fatalf("failed to query if layer is scanned: %v", err)
	}
	if b {
		t.Fatalf("expected layer not to be scanned")
	}
}

// IndexReport confirms the book keeping around index reports works
// correctly.
func (e *e2e) IndexReport(t *testing.T) {
	defer func() {
		e.failed = t.Failed()
	}()

	A := &claircore.IndexReport{
		Hash:  e.manifest.Hash,
		State: "Testing",
	}

	err := e.store.SetIndexReport(e.ctx, A)
	if err != nil {
		t.Fatalf("failed to set index report: %v", err)
	}
	B, ok, err := e.store.IndexReport(e.ctx, e.manifest.Hash)
	if err != nil {
		t.Fatalf("failed to retrieve index report: %v", err)
	}
	if !ok {
		t.Fatalf("no index report found")
	}
	if !cmp.Equal(A.Hash.String(), B.Hash.String()) {
		t.Fatalf("%v", cmp.Diff(A.Hash.String(), B.Hash.String()))
	}
	if !cmp.Equal(A.State, B.State) {
		t.Fatalf("%v", cmp.Diff(A.Hash.String(), B.Hash.String()))
	}

	A.State = "IndexFinished"
	err = e.store.SetIndexFinished(e.ctx, A, e.scnrs)
	if err != nil {
		t.Fatalf("failed to set index as finished: %v", err)
	}

	b, err := e.store.ManifestScanned(e.ctx, e.manifest.Hash, e.scnrs)
	if err != nil {
		t.Fatalf("failed to query if manifest was scanned: %v", err)
	}
	if !b {
		t.Fatalf("expected manifest to be scanned")
	}

	B, ok, err = e.store.IndexReport(e.ctx, e.manifest.Hash)
	if err != nil {
		t.Fatalf("failed to retrieve index report: %v", err)
	}
	if !ok {
		t.Fatalf("no index report found")
	}
	if !cmp.Equal(A.Hash.String(), B.Hash.String()) {
		t.Fatalf("%v", cmp.Diff(A.Hash.String(), B.Hash.String()))
	}
	if !cmp.Equal(A.State, B.State) {
		t.Fatalf("%v", cmp.Diff(A.Hash.String(), B.Hash.String()))
	}
}
