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
  //   chartGenerators: { <genName>: { gvk, build(chart, components) }, ... }
  //                     release-scoped generators, evaluated once, that may see
  //                     ALL components — e.g. shared RBAC, a gateway, cluster-wide
  //                     NetworkPolicy. Optional. (DESIGN §15)
  render(chart, components, generators, chartGenerators={}):: {
    chart: chart,

    components: {
      [name]: {
        enabled: std.get(components[name], 'enabled', true),
        workload: std.get(components[name], 'workload', 'Deployment'),
      }
      for name in std.objectFields(components)
    },

    // release-scoped resources: one pass over chartGenerators, no owning
    // component, no per-component gate.
    local chartScoped = std.flattenArrays([
      (
        local built = chartGenerators[genName].build(chart, components);
        local items = if std.isArray(built) then built else [built];
        [
          {
            file: if std.length(items) == 1
            then 'templates/%s.yaml' % genName
            else 'templates/%s-%d.yaml' % [genName, i],
            gvk: chartGenerators[genName].gvk,
            manifest: items[i],
          }
          for i in std.range(0, std.length(items) - 1)
        ]
      )
      for genName in std.objectFields(chartGenerators)
    ]),

    resources: chartScoped + std.flattenArrays([
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
