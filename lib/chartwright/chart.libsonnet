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
      std.flattenArrays([
        (
          // build() may return a single object or an array of objects (e.g. one
          // ConfigMap per config entry). Normalize to a list and emit one file
          // per object; multi-object generators get an index suffix.
          local built = generators[genName].build(c);
          local items = if std.isArray(built) then built else [built];
          [
            {
              file: if std.length(items) == 1
              then 'templates/%s/%s.yaml' % [name, genName]
              else 'templates/%s/%s-%d.yaml' % [name, genName, i],
              component: name,
              gvk: generators[genName].gvk,
              manifest: items[i],
            } + (
              // A generator may define gate(c) to override the default
              // `<name>.enabled` gate with an arbitrary expression (e.g. a
              // capability/version gate). gateExpr is emitted verbatim.
              // objectHasAll (not objectHas) so hidden `::` methods are seen.
              if std.objectHasAll(generators[genName], 'gate')
              then { gateExpr: generators[genName].gate(c) }
              else { gate: name + '.enabled' }
            )
            for i in std.range(0, std.length(items) - 1)
          ]
        )
        for genName in std.get(components[name], 'generators', [])
        if !std.objectHasAll(generators[genName], 'when') || generators[genName].when(c)
      ])
      for name in std.objectFields(components)
    ]),
  },
}
