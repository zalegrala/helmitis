// chart.libsonnet — the assembler. Turns chart metadata + component descriptors
// + a generator registry into a Level-0 interchange document (manifests carry
// inline helm.value markers; the stamper lowers them). See DESIGN.md §4, §6.
{
  // render assembles the interchange document.
  //   chart:      { name, version, appVersion?, description?, kubeVersion? }
  //   components: { <name>: descriptor, ... }
  //               descriptor: { enabled?=true, workload?='Deployment',
  //                             generators: [<genName>, ...], ...arbitrary data }
  //   generators: { <genName>: { gvk, when(c)?, build(c) }, ... } registry
  render(chart, components, generators):: {
    chart: chart,

    components: {
      [name]: {
        enabled: std.get(components[name], 'enabled', true),
        workload: std.get(components[name], 'workload', 'Deployment'),
      }
      for name in std.objectFields(components)
    },

    resources: std.flattenArrays([
      // inject the component's own name so generators can reference c.name
      local c = components[name] { name: name };
      [
        {
          file: 'templates/%s/%s.yaml' % [name, genName],
          component: name,
          gvk: generators[genName].gvk,
          gate: name + '.enabled',
          manifest: generators[genName].build(c),
        }
        for genName in std.get(components[name], 'generators', [])
        if !std.objectHas(generators[genName], 'when') || generators[genName].when(c)
      ]
      for name in std.objectFields(components)
    ]),
  },
}
