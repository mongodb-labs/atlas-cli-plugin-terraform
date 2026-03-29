package modulegen

import (
	"net"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/logger"
	"github.com/zclconf/go-cty/cty"
)

var _ ModuleGenerator = ProjectGenerator{}

type ProjectGenerator struct{}

func (g ProjectGenerator) ModuleType() ModuleType {
	return ModuleTypeProject
}

func (g ProjectGenerator) CheckInput(input *Input) []string {
	return CheckRequiredInputStr([]RequiredStr{
		{input.ProjectID, "project_id"},
	})
}

func (g ProjectGenerator) GetResourcesToFetch(_ *Input, resources *ResourcesToFetch) {
	resources.Atlas[AtlasResourceTypeProject] = true
	resources.Atlas[AtlasResourceTypeProjectSettings] = true
	resources.Atlas[AtlasResourceTypeProjectIPAccessList] = true
	resources.Atlas[AtlasResourceTypeProjectMaintenanceWindow] = true
	// TODO@non-spike: See comment in Generate()
	// resources.Atlas[AtlasResourceTypeProjectLimits] = true
}

func (g ProjectGenerator) Generate(_ *Input, store *ResourceStore) (*GenerateModuleResult, error) {
	projectRs := store.Atlas.Project
	result := GenerateModuleResult{
		ModuleType:       g.ModuleType(),
		TerraformVersion: Version{Operator: ">=", Major: 1, Minor: 9},
		Providers: []ProviderRequirement{{
			ProviderType: ProviderTypeAtlas,
			Version:      Version{Operator: "~>", Major: 2, Minor: 1},
		}},
		ModuleBlocks: []*ModuleBlock{{
			Name:    "atlas_project",
			Source:  "terraform-mongodbatlas-modules/project/mongodbatlas",
			Version: &Version{Operator: "~>", Major: 0, Minor: 1},
		}},
	}
	var importBlocks []*ImportBlock
	var attributes []Attribute
	var variables []*Variable

	importBlocks = append(importBlocks, &ImportBlock{
		ID: *projectRs.Id,
		To: "module.atlas_project.mongodbatlas_project.this",
	})

	attributes = append(attributes, StringAttr("name", projectRs.Name))

	v := NewStringVar("org_id", "Atlas Organization ID", projectRs.OrgId)
	variables = append(variables, v)
	attributes = append(attributes, VarAttr(v.Name, v))

	if attr := generateProjectSettings(store); attr != nil {
		attributes = append(attributes, *attr)
	}

	if projectRs.WithDefaultAlertsSettings != nil {
		value := *projectRs.WithDefaultAlertsSettings
		attributes = append(attributes, BoolAttr("with_default_alerts_settings", value, IsDefault(value)))
	}

	if projectRs.RegionUsageRestrictions != nil {
		attributes = append(attributes, StringAttr("region_usage_restrictions", *projectRs.RegionUsageRestrictions))
	}

	// TODO@project-import-readiness: Module default for tags is `{}`, which does not match the API default `nil`.
	//  - The tags variable default Value on the module should be changed to `nil`.
	if projectRs.Tags != nil && len(*projectRs.Tags) > 0 {
		tagsMap := make(map[string]cty.Value, len(*projectRs.Tags))
		for _, tag := range *projectRs.Tags {
			tagsMap[tag.Key] = cty.StringVal(tag.Value)
		}
		attributes = append(attributes, LiteralAttr("tags", cty.MapVal(tagsMap)))
	}

	if attr, importBlock := generateIPAccessList(store); attr != nil {
		importBlocks = append(importBlocks, importBlock)
		attributes = append(attributes, *attr)
	}

	if attr, importBlock := generateMaintenanceWindow(store); attr != nil {
		importBlocks = append(importBlocks, importBlock)
		attributes = append(attributes, *attr)
	}

	// TODO@project-import-readiness
	//  The provider's project import does not consider limits.
	//  So limits defined in the config always create a diff until applied. Even if they match the ones in Atlas.
	//  Two options:
	//    1. [Preferred] Keep current provider Import behavior
	//      - Do not emit limits by default. Include a `project_limits = true` flag in the user input to emit limits.
	//      - Limits always have an expected diff.
	//    2. Provider/API side fix
	//      - Could modify the provider Import behavior to import project limits with non-default values.
	//        - However, this is bound to cause unexpected diffs given that the Atlas default values may change.
	//      - The problem comes from the limits being an embedded resource in the project resource instead of
	//         a separate singleton resource, which causes unclear client vs server ownership on the limits block.
	/* TODO@non-spike: Commenting out limits for the time being
	if len(store.Atlas.ProjectLimits) > 0 {
		limitsMap := make(map[string]cty.Value)
		for _, limit := range store.Atlas.ProjectLimits {
			// Currently only emitting limits with non-default values.
			// The DefaultLimit returned by the API is the prod one regardless of which environment is being called.
			//	So non-prod environments may have a Value different from the DefaultValue even if no change was made.
			if limit.DefaultLimit == nil || limit.Value != *limit.DefaultLimit {
				limitsMap[limit.Name] = cty.NumberIntVal(limit.Value)
			}
		}
		if len(limitsMap) > 0 {
			attributes = append(attributes, Attribute{
				Name:  "limits",
				Value: AttributeValue{Literal: new(cty.MapVal(limitsMap))},
			})
		}
	}
	*/

	result.ImportBlocks = importBlocks
	result.ModuleBlocks[0].Attributes = attributes
	result.Variables = variables
	return &result, nil
}

func generateProjectSettings(store *ResourceStore) *Attribute {
	// Project settings are always generated regardless of value since we don't know what the default Atlas value may be.
	var attrs []Attribute
	ps := store.Atlas.ProjectSettings
	if ps.IsSchemaAdvisorEnabled != nil {
		attrs = append(attrs, BoolAttr("is_schema_advisor_enabled", *ps.IsSchemaAdvisorEnabled))
	}
	if ps.IsCollectDatabaseSpecificsStatisticsEnabled != nil {
		attrs = append(attrs, BoolAttr(
			"is_collect_database_specifics_enabled", *ps.IsCollectDatabaseSpecificsStatisticsEnabled,
		))
	}
	if ps.IsDataExplorerEnabled != nil {
		attrs = append(attrs, BoolAttr("is_data_explorer_enabled", *ps.IsDataExplorerEnabled))
	}
	if ps.IsPerformanceAdvisorEnabled != nil {
		attrs = append(attrs, BoolAttr("is_performance_advisor_enabled", *ps.IsPerformanceAdvisorEnabled))
	}
	if ps.IsRealtimePerformancePanelEnabled != nil {
		attrs = append(attrs, BoolAttr("is_realtime_performance_panel_enabled", *ps.IsRealtimePerformancePanelEnabled))
	}
	if ps.IsExtendedStorageSizesEnabled != nil {
		attrs = append(attrs, BoolAttr("is_extended_storage_sizes_enabled", *ps.IsExtendedStorageSizesEnabled))
	}
	if len(attrs) == 0 {
		return nil
	}
	attr := ObjectAttr("project_settings", attrs, Comment(
		"You can remove any of the following settings without any secondary effects.\n"+
			"Their current value will remain unchanged in Atlas.",
	))
	return &attr
}

func generateIPAccessList(store *ResourceStore) (*Attribute, *ImportBlock) {
	if store.Atlas.ProjectIPAccessList.Results == nil {
		return nil, nil
	}

	var list [][]Attribute
	var sources []string
	for _, entry := range *store.Atlas.ProjectIPAccessList.Results {
		if entry.DeleteAfterDate != nil {
			continue // Temporary entry, shouldn't be managed by Terraform. Skipping.
		}

		var source string
		switch {
		case entry.CidrBlock != nil:
			source = *entry.CidrBlock
			// Atlas accepts CIDRs with host bits set, but the provider validates it.
			// Apply the same validation and only emit values that can be terraform-managed.
			if _, cidr, err := net.ParseCIDR(source); err != nil || cidr == nil || source != cidr.String() {
				// Note: We could emit a comment here to clarify that the entry was skipped.
				continue
			}
		case entry.IpAddress != nil:
			source = *entry.IpAddress
		case entry.AwsSecurityGroup != nil:
			source = *entry.AwsSecurityGroup
		default:
			// Cannot happen, either cidr, ip or sec group are set in Atlas.
			logger.Warning("access list entry does not contain a cidr, ip address or aws security group, skipping...")
			continue
		}

		sources = append(sources, source)
		element := []Attribute{StringAttr("source", source)}
		if entry.Comment != nil && *entry.Comment != "" {
			element = append(element, StringAttr("comment", *entry.Comment))
		}
		list = append(list, element)
	}

	if len(list) == 0 {
		return nil, nil
	}

	attr := &Attribute{Name: "ip_access_list", Value: AttributeValue{ObjectList: list}}
	importBlock := &ImportBlock{
		ForEach: sources,
		ID:      *store.Atlas.Project.Id + "-${each.value}",
		To:      "module.atlas_project.module.ip_access_list[0].mongodbatlas_project_ip_access_list.this[each.value]",
	}

	return attr, importBlock
}

func generateMaintenanceWindow(store *ResourceStore) (*Attribute, *ImportBlock) {
	mw := store.Atlas.ProjectMaintenanceWindow
	if mw == nil || mw.DayOfWeek == 0 {
		return nil, nil
	}

	attrs := []Attribute{
		BoolAttr("enabled", true),
		IntAttr("day_of_week", mw.DayOfWeek),
	}

	if mw.HourOfDay != nil {
		attrs = append(attrs, IntAttr("hour_of_day", *mw.HourOfDay))
	}

	if mw.AutoDeferOnceEnabled != nil {
		value := *mw.AutoDeferOnceEnabled
		attrs = append(attrs, BoolAttr("auto_defer_once_enabled", value, IsDefault(!value)))
	}

	if mw.ProtectedHours != nil {
		var phAttrs []Attribute
		if mw.ProtectedHours.StartHourOfDay != nil {
			phAttrs = append(phAttrs, IntAttr("start_hour_of_day", *mw.ProtectedHours.StartHourOfDay))
		}
		if mw.ProtectedHours.EndHourOfDay != nil {
			phAttrs = append(phAttrs, IntAttr("end_hour_of_day", *mw.ProtectedHours.EndHourOfDay))
		}
		if len(phAttrs) > 0 {
			attrs = append(attrs, ObjectAttr("protected_hours", phAttrs))
		}
	}

	attr := &Attribute{Name: "maintenance_window", Value: AttributeValue{Object: attrs}}
	importBlock := &ImportBlock{
		ID: *store.Atlas.Project.Id,
		To: "module.atlas_project.module.maintenance_window[0].mongodbatlas_maintenance_window.this",
	}
	return attr, importBlock
}
