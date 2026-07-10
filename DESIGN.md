# Helm Stamper — Design

> Project name: `chartwright` — a maker of charts (as in playwright/shipwright).
> Status: design agreed, pre-implementation. This document guides future work sessions.
> Date: 2026-06-12.

## 1. Problem & motivation

We want a Helm chart for Tempo, but the team does not want to *hand-maintain* one.
Two failure modes from the existing `tempo-distributed` chart inform this design:

1. **Exposed config surface too wide.** The chart attempts a 1:1 mapping between chart
   values and Tempo config fields. Every config change shipped in Tempo then has to be
   chased into the chart elsewhere — pure maintenance tax — and Helm only "validates" by
   blowing up at render time, while Tempo already validates its own config at runtime.
2. **Cognitive load is hidden.** Nobody wants to own the consequences of chart changes
   (both Kubernetes deployment practices and Tempo-specific ones), so consequences go
   unreviewed and the community inherits questionable defaults.

The goal is a **spec-driven stamper**: a consumable description of components renders a
complete Helm chart (Deployments, StatefulSets, Services, etc.) on disk, ready to install.
The same approach should serve both the microservices topology and the single-binary, and
be adaptable to projects other than Tempo. Tempo is the only current target.

The intended pipeline (the user's original framing, now concretized):

```
jsonnet  →  structured interchange format  →  stamper tool  →  Helm chart on disk
```

## 2. Goals / non-goals

**Goals**
- Spec-driven generation of a complete, installable Helm chart.
- Reuse the structure of Tempo's existing microservices jsonnet rather than re-describing it.
- A *deliberately minimal*, opinionated config surface — one structured way to pass config.
- Extensible to new Kubernetes resource types without changing the core tool.
- Deterministic output, so a CI drift check is trustworthy.
- Project-agnostic core; Tempo is the first consumer, not a hardcoded assumption.

**Non-goals (v1)**
- Inferring the chart/templates from rendered output (see §3, the DRAIN discussion).
- Auto-generating `values.schema.json` from Tempo's config struct (noted as future, §7).
- Chart signing / provenance (`cosign`, `.prov`) — future hardening.
- A bespoke descriptor DSL, language server, or GUI — jsonnet idioms suffice.
- Owning where the chart lives, committing it, the CI gate, or publishing — those are
  *consumer* concerns (§9).

## 3. Why not DRAIN / inference

The DRAIN idea — take many concrete rendered k8s objects and discover the variable parts —
was considered and **rejected** as the engine, for three reasons:

1. **Wrong tool for structured data.** DRAIN is a text-clustering algorithm for
   *unstructured* logs; it tokenizes by position and collapses high-cardinality tokens to
   wildcards. Kubernetes objects are already structured trees. The structurally-correct
   equivalent is *anti-unification* (least-general-generalization), not DRAIN.
2. **Inference only finds variability present in the samples.** Render Tempo's jsonnet for
   one environment and every field is constant, so nothing becomes a value. Discovering that
   `replicas` is a knob requires rendering variations that differ in `replicas` — exactly the
   manual enumeration the approach was meant to avoid.
3. **Inference produces an *accidental* config surface.** It variabilizes whatever happened
   to differ and couples knobs by coincidence — the opposite of the deliberate, minimal
   surface we want. The config surface is a *design artifact*; it should be authored.

The only acceptable role for inference is a **one-time migration aid** that *suggests*
candidate values for a human to curate. Never the runtime engine.

## 4. Architecture

Three layers, with a deliberately dumb seam between each.

```
┌─ AUTHORING (jsonnet, lives in the target repo, e.g. tempo) ──────────────┐
│  component descriptors  +  generator library                            │
│  (data: what exists)       (functions: component → object-with-holes)   │
└──────────────────────────────┬───────────────────────────────────────────┘
                               │  emits
                               ▼
┌─ INTERCHANGE (JSON on stdout — the stable contract) ─────────────────────┐
│  { chart, components, resources: [ {file, gvk, gate, manifest,          │
│    holes:[{path, pointer, default, schema, render}] } ] }               │
└──────────────────────────────┬───────────────────────────────────────────┘
                               │  consumed by
                               ▼
┌─ STAMPER (Go binary — generic, project-agnostic) ────────────────────────┐
│  interchange → chart on disk: templates/*.yaml (holes re-injected as     │
│  {{ }}), values.yaml, values.schema.json, Chart.yaml, _helpers.tpl       │
│  validate (kubeconform · helm lint) · deterministic emit                 │
└──────────────────────────────────────────────────────────────────────────┘
```

Key property: **the stamper knows nothing about Tempo, or even Kubernetes semantics.** It
knows only the interchange schema. All k8s/Tempo opinions live in the jsonnet generator
library. The Go tool is written once and never grows opinions.

**Why jsonnet emits interchange rather than the chart directly** (unlike the Alloy adapter,
which manifests its target DSL in-language): jsonnet is poor at the chart-assembly chores —
raw `{{ }}` emission, byte-stable formatting for drift checks, `values.schema.json`,
lint/package/publish. Go is good at exactly those. The seam puts each language where it is
strong.

**jsonnet is a preference, not a requirement.** Because the interchange is plain JSON against
a published schema, a hand-written spec — or any future non-jsonnet producer — can feed the
same stamper. That is the "adaptable to other projects" property, for free.

### Reference: the Alloy `alloy-syntax-jsonnet` adapter

The inspiration is `github.com/grafana/alloy/operations/alloy-syntax-jsonnet`: build a
structured object in jsonnet, then `manifest` it into a target DSL, using a special marker
(`alloy.expr(...)`) for parts that must emit as literal, non-quoted expressions. The
`helm.value(...)` helper (§6) is the direct analog — it marks holes.

## 5. The stamp/install boundary (fat values, thin structure)

- **Stamp time (jsonnet):** which components exist, which resources each gets, the *shape* of
  every object. Structural fanout happens here — one file per component per resource.
- **Install time (Helm values):** leaf values (image, replicas, resources, structured
  config), plus a per-component `enabled` gate.

Structural fanout is done **above Helm**, at stamp time — *not* with `{{ range }}` inside a
template. The emitted chart is dumb-flat: `templates/distributor/deployment.yaml`,
`templates/ingester/statefulset.yaml`, each containing only leaf `{{ .Values.x }}` holes and
a one-line `{{ if .Values.distributor.enabled }}` gate. More hardcoded values in the files,
far less maintenance, and a readable chart. Writing control flow inside Helm templates is the
explicit anti-goal — if we went there, the stamper would add nothing over a hand-written
chart.

## 6. The generator model (the extension unit)

A **stamp is a resource generator**: a pure function from a component descriptor to an
object-with-holes, tagged with its GVK. The kernel is tiny and never changes:

```
for component in components:
  for generator in generators:
    if generator.when(component): emit(generator.build(component))
```

A generator — the full "expression necessary" to teach the tool a new resource (e.g. a
VerticalPodAutoscaler) — is GVK + a `build(component)` mapping + inline hole declarations:

```jsonnet
{
  gvk: 'autoscaling.k8s.io/v1/VerticalPodAutoscaler',
  when(c):: c.vpa != null,                 // applicability predicate (optional)
  build(c):: {                             // the mapping — the irreducible human opinion
    apiVersion: 'autoscaling.k8s.io/v1',
    kind: 'VerticalPodAutoscaler',
    metadata: { name: c.name + '-vpa' },
    spec: {
      targetRef: { apiVersion: 'apps/v1', kind: c.workload, name: c.name },
      updatePolicy: { updateMode: helm.value(c.name + '.vpa.updateMode', default='Auto') },
      resourcePolicy: { containerPolicies: [{
        containerName: '*',
        minAllowed: helm.value(c.name + '.vpa.minAllowed', default={ cpu: '100m', memory: '128Mi' }),
        maxAllowed: helm.value(c.name + '.vpa.maxAllowed', default={ cpu: '2',    memory: '4Gi'  }),
      }] },
    },
  },
}
```

Notes:
- **The mapping is irreducible.** No tool can infer that a VPA's `targetRef` comes from a
  component's identity — that is semantic. Generators are where opinions legitimately live.
- **Min/max bounds are *value* schema, not *resource* schema.** They attach to the hole and
  flow into `values.schema.json`. You do not need the k8s OpenAPI schema to *build* a
  resource — only to *validate* the result, done after the fact by `kubeconform` / `helm lint`.
  Generation is schema-free; validation is schema-checked.
- **No generator-to-generator coupling.** When one generator needs something another produced
  (VPA `targetRef` needs to know Deployment vs StatefulSet), it reads shared *descriptor data*
  (`c.workload`), never another generator's output. Every generator stays a pure function of
  the component.
- **Opinions are pluggable, shared libraries.** Keep generators in your repo → project-specific.
  Share them as an importable jsonnet library → opinions improve over time as consumers upgrade,
  *without* coupling opinion-evolution to releases of the core tool. The kernel never grows.

## 7. Component descriptors

You maintain two things, sized very differently:

- **Generators:** write-once, per *resource type*, not per component. Shared across all
  components and ideally across projects. Adding a component costs zero generator code.
- **Descriptors:** sparse, `defaults + overrides` via jsonnet's `+`. This is the anti-tedium
  move.

```jsonnet
local defaults = {
  workload: 'Deployment',
  ports: { http: 3200, grpc: 9095 },
  health: { path: '/ready', port: 'http' },
  generators: ['deployment', 'service', 'servicemonitor'],
  configs: [                                   // see §8 — repeatable config-mount primitive
    { name: 'config',    kind: 'ConfigMap', value: {/* tempo config */},
      mountPath: '/conf/tempo.yaml', subPath: 'tempo.yaml', checksumRollout: true },
    { name: 'overrides', kind: 'ConfigMap', value: {/* runtime overrides */},
      mountPath: '/overrides', checksumRollout: false },
  ],
};

local components = {
  distributor: defaults,                                  // pure default
  ingester: defaults + {                                  // sparse override
    workload: 'StatefulSet',
    generators+: ['vpa', 'pdb'],                          // append, don't restate
    vpa: { maxAllowed: { cpu: '4', memory: '8Gi' } },
  },
  compactor: defaults + { ports+: { grpc: null } },       // drop a field
};
```

Adding a component is a few lines; changing a port is one line; changing the pattern for
everyone is one line in `defaults`. Global configs live in `defaults` (every component gets
them); component-scoped ones go in that component's override block.

**Anti-tedium tooling** (no heavier than this): the `defaults + overrides` idiom, write-once
generators, a tight `make helm-stamp` feedback loop, interchange validation with precise
errors at stamp time, `kubeconform` + `helm lint` on output, and jsonnet `assert` for
descriptor invariants. Explicitly *not* building a bespoke DSL, LSP, or GUI for v1.

## 8. The config-mount primitive

The config story is a **repeatable primitive**, not a single special value:
`$structuredObject → ConfigMap/Secret → mount at $path`, allowed in multiples (Tempo config,
runtime Overrides, and whatever comes next).

- The structured object is an **opaque passthrough value.** Helm never introspects it, never
  maps individual fields, never validates types. It lands in a ConfigMap verbatim; Tempo
  validates it at runtime — where the real validation already lives. This kills the 1:1
  maintenance coupling: a Tempo config change requires *no* chart change.
- It fits the generator model with **no generator-to-generator coupling.** `configs[]` is
  descriptor data read by two independent generators:
  - the **configmap/secret generator** emits one ConfigMap (or Secret) per entry, with the
    `value` as a single (block-rendered) hole so it is install-time tunable as one structured value;
  - the **workload generator** reads the same `configs[]` and injects the volume, volumeMount,
    and — when `checksumRollout` — a `checksum/config` pod-template annotation so content
    changes trigger a rollout.
- **Schema: none by default** (opaque passthrough, Tempo owns validation). If editor
  completion is ever wanted, the *only* acceptable source is a schema **auto-generated from
  Tempo's own config struct** into `values.schema.json` — never hand-maintained. **Out of
  scope for v1.**

## 9. The interchange format

JSON on stdout, one object. The contract — and the validation boundary.

```json
{
  "chart": {
    "name": "tempo",
    "version": "0.1.0",
    "appVersion": "2.6.0",
    "description": "...",
    "kubeVersion": ">=1.28-0"
  },
  "components": {
    "distributor": { "enabled": true, "workload": "Deployment" }
  },
  "resources": [
    {
      "file": "templates/distributor/deployment.yaml",
      "component": "distributor",
      "gvk": "apps/v1/Deployment",
      "gate": "distributor.enabled",
      "manifest": { "apiVersion": "apps/v1", "kind": "Deployment", "...": "..." },
      "holes": [
        { "path": "distributor.replicas", "pointer": "/spec/replicas",
          "default": 3, "schema": { "type": "integer", "minimum": 1 } }
      ]
    }
  ]
}
```

Three decisions:

1. **Holes are carried out-of-band, not inlined as `{{ }}` strings.** Each `manifest` is
   clean, valid structured data with a real placeholder value at the hole site; `holes[]`
   says, via JSON Pointer, "replace the value at `/spec/replicas` with the Helm expression."
   This keeps `manifest` valid the whole way through, so `kubeconform` can validate it
   *before* placeholders are injected, and substitution stays mechanical and unambiguous.
2. **`gate` is first-class, not a hole.** A component's `enabled` toggle wraps the *whole
   file* in `{{ if .Values.distributor.enabled }}...{{ end }}` — distinct from value
   substitutions *within* a manifest.
3. **The interchange is the validation boundary.** The stamper validates this JSON against a
   published schema before doing anything; a hand-written or non-jsonnet producer only has to
   satisfy that schema. This is what makes the stamper reusable rather than Tempo-coupled.

## 10. The stamper (Go binary)

Core contract: **interchange JSON in, chart directory out.** Reads interchange from stdin or
a file; a `--jsonnet main.jsonnet` convenience flag shells out to jsonnet and pipes the
result, so `make helm-stamp` is one command.

Pipeline:
1. **Validate** interchange against the schema → precise errors at stamp time.
2. **Substitute holes.** Walk to each hole's JSON Pointer and replace the placeholder with a
   Helm expression. Because the stamper owns the YAML encoder, it emits the `{{ }}` as a raw
   unquoted node — no string-splicing, no invalid-YAML-as-strings. (The Go-side analog of the
   Alloy `exprMarker`.)
3. **Assemble** `templates/<component>/<kind>.yaml` (gate-wrapped if `gate` is set),
   `values.yaml` (defaults folded from dotted hole paths into nested structure),
   `values.schema.json` (from hole schemas), `Chart.yaml` (from `chart`), `_helpers.tpl`.
4. **Validate output:** `kubeconform` against pre-substitution manifests + `helm lint`.
5. **Emit deterministically** (see §11).

### Hole render modes — a closed set

Two core modes (auto-selected by the value's JSON type), two opt-in modifiers, one escape hatch:

| Mode/flag      | Trigger                      | Emits                                                          |
|----------------|------------------------------|----------------------------------------------------------------|
| scalar inline  | string/number/bool           | `{{ .Values.x \| default 3 }}` (default omitted if required)   |
| block          | object/array                 | `{{ .Values.x \| toYaml \| nindent N }}` (N from pointer depth) |
| `quote` (flag) | scalar needing quoting       | `{{ .Values.x \| quote }}`                                     |
| `with` (flag)  | block that vanishes if empty | `{{- with .Values.x }}{{ toYaml . \| nindent N }}{{- end }}`   |
| `raw` (hatch)  | anything exotic              | literal Helm expression, verbatim                              |

`raw` is the pressure-release valve: composite scalars (`image: "{{ .repo }}:{{ .tag }}"`),
`lookup`, custom pipelines. It guarantees a generator author is never blocked on a stamper
release — keeping faith with "the kernel never grows opinions." Prefer modeling composites as
a single structured hole where possible; fall back to `raw` rather than inventing a templating
mini-language in the interchange.

## 11. Determinism (a first-class correctness property)

Determinism is *required*, not a formatting nicety — it is what makes a CI drift check
trustworthy. Stamping the same input twice must produce identical bytes: stable key ordering,
stable file enumeration, stable formatting, pinned tool versions. If output is
non-deterministic, the drift gate produces false diffs and the team learns to ignore it.

## 12. Capability vs policy — what this repo ships

This repo ships a **capability**, not a **policy** — mirroring how `jsonnet-fmt` formats but
`jsonnet-check` (fmt + `git diff --exit-code`) lives in the *consumer's* Makefile and CI.

**This repo (the stamper) provides:**
- `stamp` — interchange → chart on disk, in place (the `jsonnet-fmt` analog).
- `stamp --check` (optional convenience) — render and compare against what's on disk, nonzero
  exit on difference, *without* writing; gives a chart-scoped diff even with unrelated
  working-tree changes. A consumer may instead just run `git diff --exit-code`.
- The jsonnet helper lib (`helm.value`, manifest helpers).
- A starter generator library: deployment, statefulset, service, configmap, vpa, pdb,
  servicemonitor.
- The interchange JSON schema, docs, and an example wiring.

**The consumer repo (Tempo) owns:**
- The descriptors and any Tempo-specific generators.
- Where the rendered chart lives and committing it.
- `make helm-stamp` (write) and `make helm-stamp-check` (CI gate).
- Versioning and the `helm package` + OCI push on release.

### The hot-path benefit (accrues in the consumer repo)

Committing the rendered chart next to the jsonnet source buys: (1) a **drift gate** —
`helm-stamp` + `git diff --exit-code` fails CI if a change wasn't re-stamped, so the chart
cannot rot; and (2) **review-time visibility** — a jsonnet PR also shows the rendered
Kubernetes diff (new volume mount, changed probe, extra RBAC), moving the deployment
consequences to a reviewable diff at change time. That is the cultural fix for "nobody wants
the cognitive load," enforced by tooling.

### How Helm + OCI publishing works (consumer concern)

Modern OCI registries store arbitrary artifacts, so the same registry hosting the Tempo image
can host the chart; Helm 3.8+ treats this as first-class. At release time:

```
helm package chart/                                  # → tempo-0.1.0.tgz (version from Chart.yaml)
helm push tempo-0.1.0.tgz oci://<registry>/grafana/charts
```

Chart name → repo path; chart version → tag. No `index.yaml`, no chart-repo server. Users:
`helm install tempo oci://<registry>/grafana/charts/tempo --version 0.1.0`. Optional future
hardening: `cosign sign` + `.prov` provenance.

`appVersion` tracks the Tempo version (data, via interchange); the chart's own `version` is
independent semver bumped by the author.

## 13. v1 scope summary

**In:** the Go stamper (`stamp`, `stamp --check`, `--jsonnet` convenience), the interchange
schema + validation, hole substitution with the closed render-mode set, deterministic emit,
`kubeconform` + `helm lint`, the jsonnet helper lib, a starter generator library, the
config-mount primitive, docs + example wiring. First target: Tempo microservices and
single-binary topologies via separate descriptor sets.

**Out:** inference/DRAIN as an engine, auto-generated `values.schema.json` from Tempo's config
struct, chart signing/provenance, a bespoke descriptor DSL/LSP/GUI, and any policy about
committing/gating/publishing (consumer-owned).

## 14. Roadmap & progress (multi-session tracker)

The total effort is split into sequential plans so it survives across sessions. The
interchange schema is the contract; the consumer (stamper) is built first so the contract is
locked before the producer (jsonnet) targets it.

| # | Plan | Scope | Status | Doc |
|---|------|-------|--------|-----|
| 1 | Stamper core | interchange JSON → validated, deterministic Helm chart on disk: types, schema validation, hole substitution (closed render-mode set), values.yaml + values.schema.json, Chart.yaml + _helpers.tpl, deterministic emit, `--check` drift, optional `helm lint`/`kubeconform`, CLI. Tested against hand-written fixtures. | ✅ Done (branch `feat/stamper-core`) | `docs/superpowers/plans/2026-06-29-helm-stamper-core.md` |
| 2a | Lowering pass | Inline `__cw_hole__` markers → Level-1 interchange (holes + JSON Pointers), wired into `Build`. | ✅ Done | `docs/superpowers/plans/2026-07-10-plan2a-lowering.md` |
| 2b | Jsonnet authoring layer | `helm.value`/`required`/`raw`/`blockValue` helper, `chart.libsonnet` assembler, generators (deployment, service, statefulset, pdb), `examples/web`, jsonnet→installable-chart acceptance test. Config-mount primitive (§8) and CRD generators (vpa, servicemonitor) tracked as follow-ups. | 🚧 Core done | this file |
| 3 | Tempo descriptors + example wiring | Tempo microservices and single-binary descriptor sets, `make helm-stamp` example, docs. Lives partly in this repo (example) and partly in the Tempo repo (real descriptors). | ⏳ Not started | _to be written_ |

Status legend: ✅ done · 🚧 in progress · 📋 planned (plan written) · ⏳ not started.

When a plan completes, update its row to ✅ and note the key deliverables here so a future
session can resume without re-reading the whole history.

### Plan 1 deliverables (done)

Packages `interchange` (types + embedded JSON-Schema validation) and `stamp` (the engine:
JSON-Pointer sentinel substitution with the closed render-mode set, gate wrapping, values.yaml
folding, values.schema.json, Chart.yaml, _helpers.tpl, deterministic `Build`, `Write`/`Check`
drift), plus the `cmd/stamp` CLI (`--in`/`--jsonnet`/`--out`/`--check`/`--no-validate-output`).
15 TDD tasks, all committed; a final code review was applied (missing-sentinel now errors,
`deepCopy` is fallible, deterministic component ordering, cached schema).

### Installability acceptance gate (done)

`testdata/installable.json` renders a complete Deployment + Service; `TestInstallableChart`
gates the built chart through `helm lint` → `helm template` → `kubeconform -strict`. This is
the installability bar every Plan 2 generator must meet. Real-cluster `kind` install proof is a
tracked follow-up (issue #8).

### Hardening from Plan 1's final review (done)

- **Sentinel guessability** (#3): sentinel is now `CWHOLE<sha256(file)[:16]><i>END` — a
  per-resource nonce, effectively uncollidable with real content, still deterministic.
- **`quote` + `required`** (#4): confirmed valid (`required … | quote`) and covered by a test.
- **Colon-in-key** (#5): `replaceTokenBlock` preserves the key prefix verbatim, so keys with
  colons/quotes survive.
- **Hole path shadowing a component gate** (#6): `buildValues` now rejects a hole whose path
  equals a `<component>.enabled` gate.
