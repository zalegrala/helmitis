// chart-scoped serviceaccount generator — a single release-wide ServiceAccount,
// not tied to any one component. Demonstrates the chart-scoped generator tier:
// build(chart, components) runs once and can see the whole component set
// (DESIGN §15). A real chart would use this shape for shared RBAC, a gateway, a
// cluster-wide NetworkPolicy, etc.
{
  gvk: 'v1/ServiceAccount',
  build(chart, components):: {
    apiVersion: 'v1',
    kind: 'ServiceAccount',
    metadata: {
      name: chart.name,
      labels: { 'app.kubernetes.io/name': chart.name },
      annotations: {
        // proof the generator sees all components (illustrative)
        'chartwright.dev/components': std.join(',', std.objectFields(components)),
      },
    },
  },
}
