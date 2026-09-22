// Copyright TrueNAS 2026
// SPDX-License-Identifier: MPL-2.0

package app_start

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/truenas/terraform-provider-truenas/internal/client"
)

var _ action.Action = &Action{}
var _ action.ActionWithConfigure = &Action{}

// Action starts an app (app.start; a job).
type Action struct{ client *client.Client }

func New() action.Action { return &Action{} }

type model struct {
	AppName types.String `tfsdk:"app_name"`
}

func (a *Action) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_start"
}

func (a *Action) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Starts an app (app.start).",
		Attributes: map[string]schema.Attribute{
			"app_name": schema.StringAttribute{
				Required:    true,
				Description: "App name (truenas_app.<name>.name).",
			},
		},
	}
}

func (a *Action) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	a.client = c
}

func (a *Action) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var cfg model
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Starting app…"})
	}
	if _, err := a.client.CallJob(ctx, "app.start", buildParams(cfg)...); err != nil {
		resp.Diagnostics.AddError("App start failed", err.Error())
		return
	}
	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "App started."})
	}
}

func buildParams(m model) []any {
	return []any{m.AppName.ValueString()}
}
