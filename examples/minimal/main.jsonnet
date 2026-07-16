// The smallest useful chartwright entrypoint: one component → Deployment + Service.
// Run:  stamp --jsonnet examples/minimal/main.jsonnet --out ./chart
//
// You describe components as data; generators shape them; helm.value() marks the
// few things that stay tunable at install time. No Helm templates, no YAML by hand.
local cw = import '../../lib/chartwright/chart.libsonnet';
local deployment = import '../../lib/chartwright/generators/deployment.libsonnet';
local service = import '../../lib/chartwright/generators/service.libsonnet';

cw.render(
  { name: 'hello', version: '0.1.0', appVersion: '1.0.0' },
  {
    web: {
      generators: ['deployment', 'service'],
      image: 'nginx:1.27',
      replicas: 2,
      ports: [{ name: 'http', port: 80 }],
    },
  },
  { deployment: deployment, service: service },
)
