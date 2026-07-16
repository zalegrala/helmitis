// A Tempo-FLAVORED demonstration — NOT a production chart and NOT parity with
// grafana/tempo-distributed. Its job is to show the machinery works at a
// realistic microservices scale: a few components (Deployment + StatefulSet),
// each mounting ONE structured Tempo config via the config-mount primitive, plus
// a release-wide ServiceAccount. The headline is the config surface: a single
// opaque `tempo.yaml` value (Tempo validates it at runtime), not a 1:1 mapping
// of every Tempo knob — the thing tempo-distributed got wrong (DESIGN §1, §8).
//
//   stamp --jsonnet examples/tempo/main.jsonnet --out ./chart
local cw = import '../../lib/chartwright/chart.libsonnet';
local deployment = import '../../lib/chartwright/generators/deployment.libsonnet';
local statefulset = import '../../lib/chartwright/generators/statefulset.libsonnet';
local service = import '../../lib/chartwright/generators/service.libsonnet';
local configmap = import '../../lib/chartwright/generators/configmap.libsonnet';
local serviceaccount = import '../../lib/chartwright/chartgen/serviceaccount.libsonnet';

// One structured Tempo config, authored once and mounted by every component.
// Opaque passthrough: chartwright never introspects it; Tempo validates at runtime.
local tempoConfig = {
  server: { http_listen_port: 3200 },
  distributor: { receivers: { otlp: { protocols: { grpc: {}, http: {} } } } },
  ingester: { max_block_duration: '5m' },
  storage: { trace: { backend: 'local', 'local': { path: '/var/tempo/blocks' } } },
};

// Shared config-mount entry every component carries.
local configMount = [{
  name: 'config',
  kind: 'ConfigMap',
  value: tempoConfig,
  mountPath: '/conf/tempo.yaml',
  subPath: 'tempo.yaml',
  checksumRollout: true,
}];

local httpGrpc = [{ name: 'http', port: 3200 }, { name: 'grpc', port: 9095 }];

cw.render(
  { name: 'tempo', version: '0.1.0', appVersion: '2.6.0', kubeVersion: '>=1.23-0' },
  {
    distributor: {
      generators: ['deployment', 'service', 'configmap'],
      image: 'grafana/tempo:2.6.0',
      replicas: 3,
      ports: httpGrpc,
      configs: configMount,
    },
    ingester: {
      workload: 'StatefulSet',
      generators: ['statefulset', 'service', 'configmap'],
      image: 'grafana/tempo:2.6.0',
      replicas: 3,
      ports: httpGrpc,
      configs: configMount,
    },
    querier: {
      generators: ['deployment', 'service', 'configmap'],
      image: 'grafana/tempo:2.6.0',
      replicas: 2,
      ports: httpGrpc,
      configs: configMount,
    },
    compactor: {
      generators: ['deployment', 'service', 'configmap'],
      image: 'grafana/tempo:2.6.0',
      replicas: 1,
      ports: httpGrpc,
      configs: configMount,
    },
  },
  { deployment: deployment, statefulset: statefulset, service: service, configmap: configmap },
  { serviceaccount: serviceaccount },
)
