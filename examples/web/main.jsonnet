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
local configmap = import '../../lib/chartwright/generators/configmap.libsonnet';

cw.render(
  { name: 'acceptance', version: '0.1.0', appVersion: '2.6.0' },
  {
    web: {
      workload: 'Deployment',
      generators: ['deployment', 'service', 'configmap'],
      ports: [{ name: 'http', port: 3200 }],
      image: 'grafana/tempo:2.6.0',
      replicas: 1,
      // config-mount primitive: a structured config → ConfigMap → mounted file,
      // with a checksum annotation so content changes roll the pods.
      configs: [
        {
          name: 'config',
          kind: 'ConfigMap',
          value: { server: { http_listen_port: 3200 } },
          mountPath: '/conf/tempo.yaml',
          subPath: 'tempo.yaml',
          checksumRollout: true,
        },
      ],
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
  {
    deployment: deployment,
    service: service,
    statefulset: statefulset,
    pdb: pdb,
    configmap: configmap,
  },
)
