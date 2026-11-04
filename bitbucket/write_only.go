package bitbucket

import (
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// writeOnlySchemas returns the <key>_wo and <key>_wo_version attributes, a write-only alternative
// to the sensitive attribute key that is never stored in state. Requires Terraform 1.11 or later.
func writeOnlySchemas(key string) (*schema.Schema, *schema.Schema) {
	wo := &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Sensitive:     true,
		WriteOnly:     true,
		ConflictsWith: []string{key},
		RequiredWith:  []string{key + "_wo_version"},
		Description:   fmt.Sprintf("Write-only alternative to `%s`, never stored in state. Requires Terraform 1.11 or later.", key),
	}

	version := &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		RequiredWith: []string{key + "_wo"},
		Description:  fmt.Sprintf("Change this value to update `%s_wo`, Terraform cannot detect changes to write-only attributes.", key),
	}

	return wo, version
}

// writeOnlyOrValue returns the configured <key>_wo value when set, otherwise the value of key.
// Write-only values are only available from the configuration during create and update.
func writeOnlyOrValue(d *schema.ResourceData, key string) string {
	v, diags := d.GetRawConfigAt(cty.GetAttrPath(key + "_wo"))
	if !diags.HasError() && v.IsKnown() && !v.IsNull() && v.Type().Equals(cty.String) {
		return v.AsString()
	}

	return d.Get(key).(string)
}

// usesWriteOnly reports whether the resource is configured with <key>_wo instead of key, in which
// case key must not be populated from the API on read.
func usesWriteOnly(d *schema.ResourceData, key string) bool {
	return d.Get(key+"_wo_version").(string) != ""
}
