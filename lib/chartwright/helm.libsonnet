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
}
