# API Coverage Gaps — write surface

Internal roadmap. **Excluded from the public snapshot** (`scripts/publish-public.sh`).

Generated 2026-09-28 by diffing every resource's TrueNAS `create`/`update`
accepts-schema (live, TrueNAS 27.0 `core.get_methods`) against the provider
resource schema, then classifying each unmatched field against the resource
code. Only fields verified as genuinely unmodeled writable input are listed
("REAL_GAP"); renames, nested attributes, apply-time create flags, and
documented omissions were filtered out.

## Scope / limits (this is a floor, not the whole picture)

Covered: **top-level** `create`/`update` input fields for the 81 resources that
have such methods, vs. top-level + nested resource schema attributes.

NOT covered by this pass:
- Data-source read parity (only `smb_share` was ever properly checked and fixed).
- Missing **sub-fields inside** already-modeled nested objects.
- Enum-value completeness on modeled fields.
- Resources driven by methods other than `create`/`update`.

## Verified gaps (~100 fields, 15 resources)

### Storage / ZFS

`dataset` and `zvol` are largely the **same** ZFS-property body of work
(GH-16). Both go through `pool.dataset`.

| resource | n | fields |
|---|---|---|
| `zvol` | 22 | aclmode, acltype, atime, checksum, copies, exec, readonly, recordsize, snapdev, snapdir, special_small_block_size, quota, refquota, reservation, refreservation, quota_warning, quota_critical, refquota_warning, refquota_critical, managedby, user_properties, user_properties_update |
| `dataset` | 20 | aclmode, atime, checksum, copies, deduplication, exec, readonly, recordsize, sync, snapdev, snapdir, special_small_block_size, refreservation, quota_warning, quota_critical, refquota_warning, refquota_critical, managedby, user_properties, user_properties_update |
| `replication` | 15 | compressed, embed, large_block, encryption, encryption_inherit, encryption_key_format, encryption_key_location, allow_from_scratch, hold_pending_snapshots, logging_level, only_matching_schedule, restrict_schedule, properties_exclude, properties_override, lifetimes |
| `pool` | 7 | encryption, encryption_options, deduplication, checksum, dedup_table_quota, dedup_table_quota_value, all_sed |
| `cloudsync` | 7 | args, bwlimit, transfers, encryption, filename_encryption, follow_symlinks, create_empty_src_dirs |
| `cloud_backup` | 1 | args |
| `periodic_snapshot` | 1 | fixate_removal_date |

### Compute / network

| resource | n | fields |
|---|---|---|
| `vm` | 13 | machine_type, enable_secure_boot, trusted_platform_module, bootloader_ovmf, cpuset, nodeset, pin_vcpus, arch_type, command_line_args, hyperv_enlightenments, hide_from_msr, enable_cpu_topology_extension, suspend_on_snapshot |
| `network_interface` | 5 | enable_learning, fec_mode, lacpdu_rate, vlan_pcp, xmit_hash_policy |
| `iscsi_global` | 2 | direct_config, mode |
| `nfs` | 1 | aliases |

### Identity

| resource | n | fields |
|---|---|---|
| `user` | 3 | home_mode, userns_idmap, webshare |
| `group` | 2 | userns_idmap, users |
| `directoryservices` | 1 | force |

### Write-only secrets (need write-only attributes; counted separately)

- `replication.encryption_key`
- `cloudsync.encryption_password`, `cloudsync.encryption_salt`
- `pool.encryption_options` (passphrase)

## Filtered out (verified NOT gaps)

- HA/failover-only: all `network_config` (`activity`, `hostname_b`, `hostname_virtual`), `network_interface.failover_*`.
- Apply-time create flags: `*.force`, `*.force_size`, `create_ancestors`, `force_topology`, `allow_duplicate_serials`, `user.home_create`, `user.random_password`, `system_general.rollback_timeout`, `system_general.ui_restart_delay`, `docker_config.migrate_applications`, `vm.ensure_display_device`, `api_key.reset`.
- Renames: `service.enable`→`enabled`, `certificate.CSR`→`csr`, `app.app_name`→`name`, `app.custom_compose_config`→`custom_compose_config_string`, `cloudsync_credentials.provider`→`provider_config`.
- Server-generated / read-only: `vm.uuid`.
- Documented omissions: `certificate.cert_extensions`, `mail.oauth`, `directoryservices.configuration` (modeled as the flattened per-service-type union).

## Priority

1. `dataset` + `zvol` ZFS properties (GH-16; external contributor PR was opened twice — #17, #19).
2. `vm` hardware/boot/CPU options.
3. `replication` send/encryption options.
4. `pool` encryption + dedup + checksum.
5. `cloudsync` rclone options.
6. The tail (`network_interface`, `user`, `group`, `nfs`, `iscsi_global`, `cloud_backup`, `periodic_snapshot`, `directoryservices.force`).
