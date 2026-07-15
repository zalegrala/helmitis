// vpa generator — a VerticalPodAutoscaler for a component (the DESIGN §6 worked
// example). Applies only when the component declares a `vpa` block. targetRef
// reads c.workload (shared descriptor data — no generator-to-generator coupling).
local helm = import '../helm.libsonnet';

{
  gvk: 'autoscaling.k8s.io/v1/VerticalPodAutoscaler',
  when(c):: std.get(c, 'vpa', null) != null,
  build(c):: {
    apiVersion: 'autoscaling.k8s.io/v1',
    kind: 'VerticalPodAutoscaler',
    metadata: {
      name: c.name,
      labels: { 'app.kubernetes.io/name': c.name },
    },
    spec: {
      targetRef: {
        apiVersion: 'apps/v1',
        kind: std.get(c, 'workload', 'Deployment'),
        name: c.name,
      },
      updatePolicy: {
        updateMode: helm.value(c.name + '.vpa.updateMode', std.get(c.vpa, 'updateMode', 'Auto')),
      },
      resourcePolicy: {
        containerPolicies: [{
          containerName: '*',
          minAllowed: helm.blockValue(c.name + '.vpa.minAllowed',
                                      std.get(c.vpa, 'minAllowed', { cpu: '100m', memory: '128Mi' })),
          maxAllowed: helm.blockValue(c.name + '.vpa.maxAllowed',
                                      std.get(c.vpa, 'maxAllowed', { cpu: '2', memory: '4Gi' })),
        }],
      },
    },
  },
}
