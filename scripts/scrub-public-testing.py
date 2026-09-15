#!/usr/bin/env python3
"""Scrub internal lab topology from TESTING.md for the public repo.

Run by scripts/publish-public.sh against the working-tree TESTING.md on the
temporary publish branch (never against main). Two passes:
  1. Genericize example values in the env-var table (IPs, realms, DNs).
  2. Replace the lab sections (HA / DS / LDAP / FreeIPA / pve networking
     quirk) with generic "bring your own disposable box" guidance that keeps
     the env-var contracts and guard/skip behavior.
Self-checks fail loudly if any lab token survives.

This script names the lab tokens it strips, so publish-public.sh EXCLUDES it
(and itself) from the published snapshot.
"""
import re
import sys

PATH = "TESTING.md"

# --- Pass 1: env-var table example tokens (order: specific before general) ---
TOKENS = [
    ("https://192.168.1.247:14000/dir", "https://pebble.example.com:14000/dir"),
    ("http://192.168.1.247:8055", "http://pebble.example.com:8055"),
    ("ldap://192.168.1.251", "ldap://ldap.example.com"),
    ("ldaps://192.168.1.251", "ldaps://ldap.example.com"),
    ("dc=tftest-ldap,dc=lan", "dc=example,dc=lan"),
    ("ipa.tfipa.lan", "ipa.example.lan"),
    ("`tfipa.lan`", "`example.lan`"),
    ("`TFIPA.LAN`", "`EXAMPLE.LAN`"),
    ("`TFTEST.LAN`", "`EXAMPLE.LAN`"),
]

# --- Pass 2: replacement for the lab sections (HA header .. Operational) ---
NEW_SECTIONS = """## HA / Enterprise test environment

Tests gated on `TRUENAS_HA=1` (`failover_config`, `ipmi_lan`, `enclosure`,
`enclosure_label`; `truecommand_config`/`vmware` are not HA-gated — they run
on any box) need a licensed Enterprise HA controller pair with a BMC/IPMI
channel and a supported enclosure.

- **Env vars**: `TRUENAS_HA=1` plus `TRUENAS_HA_ALLOWED_ENDPOINT` set to
  exactly `TRUENAS_ENDPOINT` — enforced by `acctest.HACheck`'s DSCheck-pattern
  guard, `t.Fatal` on mismatch or omission, so an HA suite can never point at
  the wrong box. `TRUENAS_DISRUPTIVE=1` is additionally required for the
  Tier 2 set-and-restore tests (`failover_config` `timeout`, `ipmi_lan`
  `vlan`, `truecommand_config` `api_key`).
- **Skip behavior on a non-HA box**: `HACheck` probes `failover.licensed`
  live and `t.Skip`s (not `t.Fatal`s) when false, so the HA-gated set runs
  cleanly against an unlicensed box with zero mutating calls.
- **Use a disposable pair.** These tests can trigger real failover events
  (`failover.become_passive`) and can swap the master/backup assignment.
  Never point them at a production or shared HA pair. Revoke any API key
  created for the run afterward.

## Directory-services test environment

Tests gated on `TRUENAS_DS=1` (Active Directory join/idmap/etc.) need a real
domain controller — for example a disposable Samba `samba-ad-dc` VM. Point the
TrueNAS box under test at the DC and supply the realm credentials:

- **Env vars**: `TRUENAS_DS_DOMAIN` (realm, e.g. `EXAMPLE.LAN`),
  `TRUENAS_DS_USER` (e.g. `Administrator`), `TRUENAS_DS_PASSWORD`.
- **Prerequisite**: the TrueNAS box under test must have its nameserver
  pointed at the DC so it can resolve the realm's SRV/A records — the
  acceptance run flips this via `network.configuration.update` and restores
  the box's original nameserver afterward, pass or fail.
- **Disposable-box guard**: `DSCheck` (`internal/acctest`) requires
  `TRUENAS_DS_ALLOWED_ENDPOINT` to be set and to exactly match
  `TRUENAS_ENDPOINT` whenever `TRUENAS_DS=1` — it `t.Fatal`s rather than
  skipping if the guard is missing or mismatched, since an unintended AD join
  is far more disruptive than a skipped test.
- **Keytab fixture**: `TestAccKerberosKeytab_basic` additionally needs
  `TRUENAS_DS_KEYTAB_B64`, base64 of a real keytab (e.g. `samba-tool domain
  exportkeytab /tmp/tfacc.keytab --principal=Administrator@EXAMPLE.LAN` then
  `base64 -w0 /tmp/tfacc.keytab`). The test self-skips when unset, so plain
  Tier-1 sweeps stay green without a DC.

### Generic LDAP test server (RFC2307)

Plain (non-AD) LDAP directory-service tests need an OpenLDAP (`slapd`) server
seeded with RFC2307 posix data (posixAccount/posixGroup under an `ou=People`
/ `ou=Group` tree):

- **Env vars**: `TRUENAS_DS_LDAP_URL` (`ldap://host` or `ldaps://host`),
  `TRUENAS_DS_LDAP_BASEDN` (e.g. `dc=example,dc=lan`),
  `TRUENAS_DS_LDAP_BINDDN` (e.g. `cn=admin,dc=example,dc=lan`),
  `TRUENAS_DS_LDAP_BINDPW`.
- **TLS**: with a self-signed server cert, clients must tolerate it
  (`LDAPTLS_REQCERT=allow` for `ldapsearch`, or set
  `OPT_X_TLS_REQUIRE_CERT`/`OPT_X_TLS_NEWCTX` globally *before*
  `ldap.initialize()` for python-ldap — those options are process-global, not
  per-connection).

### FreeIPA test server

FreeIPA (Kerberos + LDAP + DNS) tests need an `ipa-server` whose own FQDN is
its hostname and which serves an authoritative DNS zone for its domain
(install with `ipa-server-install --setup-dns`):

- **Env vars**: `TRUENAS_DS_IPA_TARGET` (server FQDN, e.g.
  `ipa.example.lan`), `TRUENAS_DS_IPA_DOMAIN` (e.g. `example.lan`, realm
  `EXAMPLE.LAN`), `TRUENAS_DS_IPA_PASSWORD` (the `admin` password).

Keep all directory-service credentials out of the repo — supply them through
the environment at run time.

"""

# Lab tokens that must NOT survive anywhere in the scrubbed file.
FORBIDDEN = [
    "192.168.1.", "10.220.16.188", "plan20-ha",
    "tftest", "tfipa", "TFTEST", "TFIPA",
    "/root/tftest", "192.168.1.1",
]
# 'pve' as a standalone word (avoid matching substrings).
FORBIDDEN_RE = [r"\bpve\b"]


def main():
    with open(PATH, encoding="utf-8") as fh:
        text = fh.read()

    for old, new in TOKENS:
        text = text.replace(old, new)

    start = text.index("## HA / Enterprise test environment")
    end = text.index("## Operational notes")
    text = text[:start] + NEW_SECTIONS + text[end:]

    problems = [tok for tok in FORBIDDEN if tok in text]
    problems += [pat for pat in FORBIDDEN_RE if re.search(pat, text)]
    if problems:
        print("SCRUB FAILED — lab tokens still present:", problems,
              file=sys.stderr)
        sys.exit(1)

    with open(PATH, "w", encoding="utf-8") as fh:
        fh.write(text)
    print("TESTING.md scrubbed. No lab tokens remain.")


if __name__ == "__main__":
    main()
