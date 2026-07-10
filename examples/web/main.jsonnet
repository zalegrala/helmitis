// Example chartwright entrypoint. Emits a Level-0 interchange document on stdout;
// feed it to the stamper:
//
//   stamp --jsonnet examples/web/main.jsonnet --out ./chart
//
// Demonstrates a stateless component (Deployment + Service) and a stateful one
// (StatefulSet + Service + PodDisruptionBudget) from the same descriptor shape.
local cw = import '../../lib/chartwright/chart.libsonnet';
local deployment = import '../../lib/chartwright/generators/deployment.libsonnet';
local service = import '../../lib/chartwright/generators/service.libsonnet';
local statefulset = import '../../lib/chartwright/generators/statefulset.libsonnet';
local pdb = import '../../lib/chartwright/generators/pdb.libsonnet';

cw.render(
  { name: 'acceptance', version: '0.1.0', appVersion: '2.6.0' },
  {
    web: {
      workload: 'Deployment',
      generators: ['deployment', 'service'],
      ports: [{ name: 'http', port: 3200 }],
      image: 'grafana/tempo:2.6.0',
      replicas: 1,
    },
    store: {
      workload: 'StatefulSet',
      generators: ['statefulset', 'service', 'pdb'],
      ports: [{ name: 'http', port: 3200 }],
      image: 'grafana/tempo:2.6.0',
      replicas: 3,
      pdb: { minAvailable: 2 },
    },
  },
  { deployment: deployment, service: service, statefulset: statefulset, pdb: pdb },
)
