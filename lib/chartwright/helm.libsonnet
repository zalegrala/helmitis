// helm.libsonnet — helpers for marking "holes": variable points in a manifest
// that become Helm {{ .Values.x }} expressions at install time. Each helper
// returns an inline marker object that the chartwright stamper lowers into the
// interchange holes[] (with a computed JSON Pointer). See DESIGN.md §6, §10.
{
  // value marks a hole at the given dotted values path.
  //   path:    dotted values path, e.g. "distributor.replicas"
  //   default: default value folded into values.yaml (null = no default)
  //   opts:    optional { schema, render, required } overrides
  //            render is one of "scalar" | "block" | "quote" | "with"
  value(path, default=null, opts={}):: {
    __cw_hole__: { path: path }
                 + (if default != null then { default: default } else {})
                 + opts,
  },

  // required marks a hole with no default that must be supplied at install time.
  required(path, opts={}):: self.value(path, null, { required: true } + opts),

  // raw marks a hole whose value is a literal Helm expression, dropped in verbatim.
  raw(path, expr):: { __cw_hole__: { path: path, render: 'raw', raw: expr } },

  // blockValue is sugar for a structured (object/array) hole rendered as
  // `toYaml | nindent` — e.g. resources, nodeSelector, or a whole config object.
  blockValue(path, default=null, opts={}):: self.value(path, default, { render: 'block' } + opts),

  // gate builds Helm boolean expressions for a resource's whole-resource gate.
  // A generator that defines `gate(c)` overrides the default `<name>.enabled`
  // gate with one of these (or any custom expression string). See DESIGN.md §15.
  gate:: {
    // enabled: the component's install-time on/off flag (the default gate).
    enabled(c):: '.Values.%s.enabled' % c.name,
    // hasAPI: true when the cluster exposes the given group/version[/kind].
    hasAPI(api):: '(.Capabilities.APIVersions.Has "%s")' % api,
    // kubeAtLeast: true when the cluster's Kubernetes version is >= v (e.g. "1.26-0").
    kubeAtLeast(v):: '(semverCompare ">=%s" .Capabilities.KubeVersion.Version)' % v,
    // all: AND several gate expressions together.
    all(exprs):: if std.length(exprs) == 1 then exprs[0] else 'and ' + std.join(' ', exprs),
  },
}
