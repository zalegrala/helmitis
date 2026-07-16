# chartwright

> ⚠️ **Early work in progress.** APIs, formats, and structure are unstable and will break
> without notice.

A spec-driven **Helm chart stamper**. Instead of hand-maintaining a Helm chart, you describe
components once and generate a complete, installable chart deterministically:

```
jsonnet (components + generators)  →  interchange JSON  →  stamper  →  Helm chart on disk
```

The idea: keep Kubernetes/application opinions where they already live (jsonnet), express the
*variable* points as declared "holes", and let a small, project-agnostic Go tool assemble the
chart — templates, `values.yaml`, `values.schema.json`, `Chart.yaml`. Output is byte-stable so
a consumer's CI can fail on uncommitted chart drift. The first target is Grafana Tempo
(microservices and single-binary), but the core is not Tempo-specific.

## Why

Hand-maintained charts rot because nobody wants the cognitive load of owning deployment
consequences, and 1:1 value-to-config mappings are a maintenance tax. This approach makes the
rendered chart a reviewable build artifact and gives components *one* structured way to pass
config rather than exposing every knob. See [`DESIGN.md`](./DESIGN.md) for the full rationale.

## Status

| Component | Status |
|-----------|--------|
| Stamper core (interchange → chart) | ✅ working |
| Hole-marker lowering pass | ✅ working |
| Jsonnet authoring layer (`helm.value`, generators) | ✅ working (deployment/service/statefulset/pdb/configmap/vpa/servicemonitor) |
| Config-mount primitive (structured config → ConfigMap → mount) | ✅ working |
| CRD generators + kubeconform CRD validation | ✅ working |
| Version/capability gating (`.Capabilities`, kubeVersion) | ✅ working |
| Tempo descriptors + example wiring | ⏳ not started |

See [`DESIGN.md` §14](./DESIGN.md) for the roadmap.

## Try it

```bash
# Smallest example — one component → Deployment + Service (start here):
go run ./cmd/stamp --jsonnet examples/minimal/main.jsonnet --out /tmp/chart

# Fuller showcase — config-mount, CRDs, capability gates, chart-scoped RBAC:
go run ./cmd/stamp --jsonnet examples/web/main.jsonnet --out /tmp/chart

# Version/capability gating — apiVersion switch + whole-resource gates by k8s version/API:
go run ./cmd/stamp --jsonnet examples/version-gating/main.jsonnet --out /tmp/chart

# Or from a hand-written interchange JSON document (no jsonnet):
go run ./cmd/stamp --in testdata/installable.json --out /tmp/chart
```

`examples/minimal/main.jsonnet` is ~15 readable lines and the best first read. Renders into an
installable Helm chart under `/tmp/chart`. `--check` compares against an existing chart and
exits non-zero on drift (for CI); `--jsonnet <file>` runs jsonnet and uses its stdout as the
interchange input.

## How this is being built

This project is developed openly and largely with AI assistance (Claude Code). The design
conversations, specs, and step-by-step implementation plans are committed in-repo under
[`DESIGN.md`](./DESIGN.md) and [`docs/`](./docs/) rather than hidden — the process is part of
the artifact. Feedback welcome.

## License

[Apache License 2.0](./LICENSE).
