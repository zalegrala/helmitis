// Demonstrates gating on Kubernetes version / API capability — the two forms:
//
//   1. FIELD-LEVEL switch  — pick a field's value (here apiVersion) by capability,
//                            via helm.raw() injecting a Helm conditional verbatim.
//   2. WHOLE-RESOURCE gate  — emit a resource only on clusters that satisfy an
//                            API/version condition, via a generator's gate(c).
//
// Note: nothing here sets k8s *feature-gate flags* (that's cluster config). A
// chart can only react to the resulting capabilities — which is what these do.
//
//   stamp --jsonnet examples/version-gating/main.jsonnet --out ./chart
//
local cw = import '../../lib/chartwright/chart.libsonnet';
local helm = import '../../lib/chartwright/helm.libsonnet';
local deployment = import '../../lib/chartwright/generators/deployment.libsonnet';
local pdb = import '../../lib/chartwright/generators/pdb.libsonnet';

// An inline HPA generator showing both forms at once.
local hpa = {
  gvk: 'autoscaling/v2/HorizontalPodAutoscaler',

  // (2) WHOLE-RESOURCE gate: only emit on clusters that expose autoscaling/v2
  // AND are Kubernetes >= 1.23. Renders as:
  //   {{- if and (.Capabilities.APIVersions.Has "autoscaling/v2")
  //              (semverCompare ">=1.23-0" .Capabilities.KubeVersion.Version) }}
  gate(c):: helm.gate.all([
    helm.gate.hasAPI('autoscaling/v2'),
    helm.gate.kubeAtLeast('1.23-0'),
  ]),

  build(c):: {
    // (1) FIELD-LEVEL switch: apiVersion chosen by which API the cluster exposes.
    // Renders as:
    //   apiVersion: {{ if .Capabilities.APIVersions.Has "autoscaling/v2" }}autoscaling/v2{{ else }}autoscaling/v2beta2{{ end }}
    apiVersion: helm.raw(
      c.name + '.hpaApiVersion',
      '{{ if .Capabilities.APIVersions.Has "autoscaling/v2" }}autoscaling/v2{{ else }}autoscaling/v2beta2{{ end }}',
    ),
    kind: 'HorizontalPodAutoscaler',
    metadata: { name: c.name, labels: { 'app.kubernetes.io/name': c.name } },
    spec: {
      scaleTargetRef: { apiVersion: 'apps/v1', kind: 'Deployment', name: c.name },
      minReplicas: helm.value(c.name + '.hpa.minReplicas', 1, { schema: { type: 'integer', minimum: 1 } }),
      maxReplicas: helm.value(c.name + '.hpa.maxReplicas', 5, { schema: { type: 'integer', minimum: 1 } }),
      metrics: [{
        type: 'Resource',
        resource: { name: 'cpu', target: { type: 'Utilization', averageUtilization: 80 } },
      }],
    },
  },
};

cw.render(
  { name: 'version-gating', version: '0.1.0', appVersion: '1.0.0', kubeVersion: '>=1.21-0' },
  {
    web: {
      // pdb also demonstrates a whole-resource capability gate (policy/v1).
      generators: ['deployment', 'hpa', 'pdb'],
      image: 'nginx:1.27',
      replicas: 2,
      pdb: { minAvailable: 1 },
    },
  },
  { deployment: deployment, hpa: hpa, pdb: pdb },
)
