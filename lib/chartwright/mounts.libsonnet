// mounts.libsonnet — shared helpers that translate a component's `configs[]`
// descriptor data into pod volumes, container volumeMounts, and rollout
// annotations. Read by the workload generators (deployment, statefulset) and
// paired with the configmap generator, which emits the ConfigMaps/Secrets — no
// generator-to-generator coupling; both read the same `configs[]` data (§8).
local helm = import 'helm.libsonnet';

{
  // isSecret reports whether a config entry is backed by a Secret (vs ConfigMap).
  isSecret(cfg):: std.get(cfg, 'kind', 'ConfigMap') == 'Secret',

  // objectName is the ConfigMap/Secret name for a component's config entry.
  objectName(c, cfg):: c.name + '-' + cfg.name,

  // volumes returns the pod `volumes` for a component's configs.
  volumes(c):: [
    {
      name: cfg.name,
      [if $.isSecret(cfg) then 'secret' else 'configMap']:
        if $.isSecret(cfg)
        then { secretName: $.objectName(c, cfg) }
        else { name: $.objectName(c, cfg) },
    }
    for cfg in std.get(c, 'configs', [])
  ],

  // mounts returns the container `volumeMounts` for a component's configs.
  mounts(c):: [
    {
      name: cfg.name,
      mountPath: cfg.mountPath,
      [if std.objectHas(cfg, 'subPath') then 'subPath']: std.get(cfg, 'subPath', null),
    }
    for cfg in std.get(c, 'configs', [])
  ],

  // checksumAnnotations returns pod-template annotations that change when a
  // config's content changes, triggering a rollout. Only for configs that opt
  // in with checksumRollout: true.
  checksumAnnotations(c):: {
    ['checksum/' + cfg.name]: helm.raw(
      c.name + '.configs.' + cfg.name + '.checksum',
      '{{ .Values.%s.configs.%s | toYaml | sha256sum }}' % [c.name, cfg.name],
    )
    for cfg in std.get(c, 'configs', [])
    if std.get(cfg, 'checksumRollout', false)
  },
}
