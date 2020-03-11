package timeseriesinsights

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/timeseriesinsights/mgmt/2018-08-15-preview/timeseriesinsights"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	azValidate "github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/validate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/timeseriesinsights/parse"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	azSchema "github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tf/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmTimeSeriesInsightsEnvironment() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmTimeSeriesInsightsEnvironmentCreateUpdate,
		Read:   resourceArmTimeSeriesInsightsEnvironmentRead,
		Update: resourceArmTimeSeriesInsightsEnvironmentCreateUpdate,
		Delete: resourceArmTimeSeriesInsightsEnvironmentDelete,
		Importer: azSchema.ValidateResourceIDPriorToImport(func(id string) error {
			_, err := parse.TimeSeriesInsightsEnvironmentID(id)
			return err
		}),

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringMatch(
					regexp.MustCompile(`^[-\w\._\(\)]+$`),
					"Time Series Insights Environment name must be 1 - 90 characters long, contain only word characters and underscores.",
				),
			},

			"location": azure.SchemaLocation(),

			"resource_group_name": azure.SchemaResourceGroupName(),

			"sku_name": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					"S1_1",
					"S1_2",
					"S1_3",
					"S1_4",
					"S1_5",
					"S1_6",
					"S1_7",
					"S1_8",
					"S1_9",
					"S1_10",
					"S2_1",
					"S2_2",
					"S2_3",
					"S2_4",
					"S2_5",
					"S2_6",
					"S2_7",
					"S2_8",
					"S2_9",
					"S2_10",
				}, false),
			},

			"storage_limited_exceeded_behavior": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  string(timeseriesinsights.PurgeOldData),
				ValidateFunc: validation.StringInSlice([]string{
					string(timeseriesinsights.PurgeOldData),
					string(timeseriesinsights.PauseIngress),
				}, false),
			},

			"data_retention_time": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: azValidate.ISO8601Duration,
			},

			"tags": tags.ForceNewSchema(),
		},
	}
}

func resourceArmTimeSeriesInsightsEnvironmentCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).TimeSeriesInsights.EnvironmentsClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	name := d.Get("name").(string)
	location := azure.NormalizeLocation(d.Get("location").(string))
	resourceGroup := d.Get("resource_group_name").(string)
	t := d.Get("tags").(map[string]interface{})
	sku, err := expandEnvironmentSkuName(d.Get("sku_name").(string))
	if err != nil {
		return fmt.Errorf("expanding sku: %+v", err)
	}

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := client.Get(ctx, resourceGroup, name, "")
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("checking for presence of existing Time Series Insights Environment %q (Resource Group %q): %s", name, resourceGroup, err)
			}
		}

		if existing.Value != nil {
			environment, ok := existing.Value.AsStandardEnvironmentResource()
			if !ok {
				return fmt.Errorf("exisiting resource was not a standard Time Series Insights Environment %q (Resource Group %q)", name, resourceGroup)
			}

			if environment.ID != nil && *environment.ID != "" {
				return tf.ImportAsExistsError("azurerm_time_series_insights_environment", *environment.ID)
			}
		}
	}

	props := &timeseriesinsights.StandardEnvironmentCreationProperties{
		StorageLimitExceededBehavior: timeseriesinsights.StorageLimitExceededBehavior(d.Get("storage_limited_exceeded_behavior").(string)),
		DataRetentionTime:            utils.String(d.Get("data_retention_time").(string)),
	}

	environment := timeseriesinsights.StandardEnvironmentCreateOrUpdateParameters{
		Location:                              &location,
		Tags:                                  tags.Expand(t),
		Sku:                                   sku,
		StandardEnvironmentCreationProperties: props,
	}

	future, err := client.CreateOrUpdate(ctx, resourceGroup, name, environment)
	if err != nil {
		return fmt.Errorf("creating/updating Time Series Insights Environment %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("waiting for completion of Time Series Insights Environment %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	resp, err := client.Get(ctx, resourceGroup, name, "")
	if err != nil {
		return fmt.Errorf("retrieving Time Series Insights Environment %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	resource, ok := resp.Value.AsStandardEnvironmentResource()
	if !ok {
		return fmt.Errorf("resource was not a standard Time Series Insights Environment %q (Resource Group %q)", name, resourceGroup)
	}

	if resource.ID == nil {
		return fmt.Errorf("cannot read Time Series Insights Environment %q (Resource Group %q) ID", name, resourceGroup)
	}

	d.SetId(*resource.ID)

	return resourceArmTimeSeriesInsightsEnvironmentRead(d, meta)
}

func resourceArmTimeSeriesInsightsEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).TimeSeriesInsights.EnvironmentsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.TimeSeriesInsightsEnvironmentID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, id.ResourceGroup, id.Name, "")
	if err != nil || resp.Value == nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("retrieving Time Series Insights Environment %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}

	environment, ok := resp.Value.AsStandardEnvironmentResource()
	if !ok {
		return fmt.Errorf("exisiting resource was not a standard Time Series Insights Environment %q (Resource Group %q)", id.Name, id.ResourceGroup)
	}

	d.Set("name", environment.Name)
	d.Set("resource_group_name", id.ResourceGroup)
	d.Set("sku_name", flattenEnvironmentSkuName(environment.Sku))
	if location := environment.Location; location != nil {
		d.Set("location", azure.NormalizeLocation(*location))
	}

	if props := environment.StandardEnvironmentResourceProperties; props != nil {
		d.Set("storage_limited_exceeded_behavior", string(props.StorageLimitExceededBehavior))
		d.Set("data_retention_time", props.DataRetentionTime)
	}

	return tags.FlattenAndSet(d, environment.Tags)
}

func resourceArmTimeSeriesInsightsEnvironmentDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).TimeSeriesInsights.EnvironmentsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.TimeSeriesInsightsEnvironmentID(d.Id())
	if err != nil {
		return err
	}

	response, err := client.Delete(ctx, id.ResourceGroup, id.Name)
	if err != nil {
		if !utils.ResponseWasNotFound(response) {
			return fmt.Errorf("deleting Time Series Insights Environment %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
		}
	}

	return nil
}

func expandEnvironmentSkuName(skuName string) (*timeseriesinsights.Sku, error) {
	parts := strings.Split(skuName, "_")
	if len(parts) != 2 {
		return nil, fmt.Errorf("sku_name (%s) has the worng number of parts (%d) after splitting on _", skuName, len(parts))
	}

	var name timeseriesinsights.SkuName
	switch parts[0] {
	case "S1":
		name = timeseriesinsights.S1
	case "S2":
		name = timeseriesinsights.S2
	default:
		return nil, fmt.Errorf("sku_name %s has unknown sku tier %s", skuName, parts[0])
	}

	capacity, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("cannot convert skuname %s capcity %s to int", skuName, parts[2])
	}

	return &timeseriesinsights.Sku{
		Name:     name,
		Capacity: utils.Int32(int32(capacity)),
	}, nil
}

func flattenEnvironmentSkuName(input *timeseriesinsights.Sku) string {
	if input == nil || input.Capacity == nil {
		return ""
	}

	return fmt.Sprintf("%s_%d", string(input.Name), *input.Capacity)
}