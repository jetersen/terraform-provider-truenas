// Copyright TrueNAS 2026
// SPDX-License-Identifier: MPL-2.0

package dataset

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// zfsEnumAttr builds an Optional+Computed string attribute for a source-aware
// ZFS enum property (coverage audit): values are the uppercase ZFS forms, it reads back
// null when the property is inherited/default, and it cannot be reverted to
// inherited by removing it from config (see zfsprops.go localString).
func zfsEnumAttr(desc string, values ...string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:      true,
		Computed:      true,
		Description:   desc,
		Validators:    []validator.String{stringvalidator.OneOf(values...)},
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

func resourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages a ZFS dataset (filesystem or volume) on TrueNAS.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Dataset name (used as Terraform ID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Full dataset path, e.g. tank/mydata.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Dataset type: FILESYSTEM (default) or VOLUME. Case-insensitive.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"compression": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Compression algorithm. Case-insensitive: lz4, zstd, off, etc.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"acltype": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "ACL type: posix, nfsv4, or off. Case-insensitive.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"share_type": schema.StringAttribute{
				Optional:    true,
				Description: "Optimised share type: UNIX or WINDOWS (write-only, not returned by API).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"comments": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Human-readable description stored as org.freenas:description.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"quota": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Quota in bytes (0 = unlimited).",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"refquota": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Referenced quota in bytes (0 = unlimited).",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"reservation": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Reserved space in bytes.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"volsize": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Volume size in bytes. Required for type=VOLUME.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			// --- Source-aware ZFS tuning properties (coverage audit) ---
			// Each is Optional+Computed and reads back null when the property is
			// inherited from the parent or left at its ZFS default (only a value
			// set LOCAL on this dataset is recorded). Consequence: reverting a
			// locally-set value to inherited cannot be done by removing it from
			// the configuration — change it out of band and refresh.
			"aclmode": zfsEnumAttr("ACL inheritance mode: PASSTHROUGH, RESTRICTED, or DISCARD. Null (unset) inherits from the parent.",
				"PASSTHROUGH", "RESTRICTED", "DISCARD"),
			"atime":    zfsEnumAttr("Update access time on read: ON or OFF. Null inherits.", "ON", "OFF"),
			"exec":     zfsEnumAttr("Allow executing files: ON or OFF. Null inherits.", "ON", "OFF"),
			"readonly": zfsEnumAttr("Mount read-only: ON or OFF. Null inherits.", "ON", "OFF"),
			"sync": zfsEnumAttr("Sync write behaviour: STANDARD, ALWAYS, or DISABLED. Null inherits.",
				"STANDARD", "ALWAYS", "DISABLED"),
			"checksum": zfsEnumAttr("Checksum algorithm: ON, OFF, FLETCHER2, FLETCHER4, SHA256, SHA512, SKEIN, EDONR, or BLAKE3. Null inherits.",
				"ON", "OFF", "FLETCHER2", "FLETCHER4", "SHA256", "SHA512", "SKEIN", "EDONR", "BLAKE3"),
			"snapdir": zfsEnumAttr("Visibility of the .zfs/snapshot directory: VISIBLE, HIDDEN, or DISABLED. Null inherits.",
				"VISIBLE", "HIDDEN", "DISABLED"),
			"dedup": zfsEnumAttr("Deduplication (the ZFS `deduplication` property): ON, VERIFY, or OFF. Null inherits. Named `dedup` to match truenas_zvol.",
				"ON", "VERIFY", "OFF"),
			"recordsize": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Suggested block size for files, e.g. \"128K\" or \"1M\". Null (unset) inherits from the parent. Use the ZFS form (uppercase suffix) to avoid drift.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"copies": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Number of copies of each block (1-3). Null (unset) inherits from the parent.",
				Validators:  []validator.Int64{int64validator.Between(1, 3)},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"special_small_block_size": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Threshold in bytes below which blocks are written to a pool's special allocation-class vdev; 0 disables it. Null (unset) inherits from the parent.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"refreservation": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Referenced reservation in bytes (space guaranteed to this dataset, excluding descendants/snapshots). Null (unset) inherits.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"mountpoint": schema.StringAttribute{
				Computed:    true,
				Description: "Dataset mountpoint path.",
			},
			"encrypted": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the dataset is encrypted.",
			},
			"pool": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the pool containing this dataset.",
			},
		},
	}
}
