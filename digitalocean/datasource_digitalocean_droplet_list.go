package digitalocean

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

//			"volume_ids": {
// Type:        schema.TypeSet,
// Elem:        &schema.Schema{Type: schema.TypeString},
// Computed:    true,
// Description: "list of volumes attached to the droplet",
// },

func dataSourceDigitalOceanDropletList() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDigitalOceanDropletListRead,
		Schema: map[string]*schema.Schema{
			"tag": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "tag of the droplet(s) to find",
				ValidateFunc: validation.NoZeroValues,
			},
			// computed attributes
			"droplets": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The droplets found",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "id of the Droplet",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "name of the Droplet",
						},
						"created_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the creation date for the Droplet",
						},
						"urn": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the uniform resource name for the Droplet",
						},
						"region": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the region that the Droplet instance is deployed in",
						},
						"image": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the image id or slug of the Droplet",
						},
						"size": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the current size of the Droplet",
						},
						"disk": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "the size of the droplets disk in gigabytes",
						},
						"vcpus": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "the number of virtual cpus",
						},
						"memory": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "memory of the droplet in megabytes",
						},
						"price_hourly": {
							Type:        schema.TypeFloat,
							Computed:    true,
							Description: "the droplets hourly price",
						},
						"price_monthly": {
							Type:        schema.TypeFloat,
							Computed:    true,
							Description: "the droplets monthly price",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "state of the droplet instance",
						},
						"locked": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "whether the droplet has been locked",
						},
						"ipv4_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the droplets public ipv4 address",
						},
						"ipv4_address_private": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the droplets private ipv4 address",
						},
						"ipv6_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the droplets public ipv6 address",
						},
						"ipv6_address_private": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the droplets private ipv4 address",
						},
						"backups": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "whether the droplet has backups enabled",
						},
						"ipv6": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "whether the droplet has ipv6 enabled",
						},
						"private_networking": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "whether the droplet has private networking enabled",
						},
						"monitoring": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "whether the droplet has monitoring enabled",
						},
						"volume_ids": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Computed:    true,
							Description: "list of volumes attached to the droplet",
						},
						"tags": tagsDataSourceSchema(),
					},
				},
			},
		},
	}
}

func dataSourceDigitalOceanDropletListRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*CombinedConfig).godoClient()

	opts := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	dropletList := []godo.Droplet{}
	for {
		droplets, resp, err := client.Droplets.List(context.Background(), opts)

		if err != nil {
			return fmt.Errorf("Error retrieving droplets: %s", err)
		}

		for _, droplet := range droplets {
			dropletList = append(dropletList, droplet)
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return fmt.Errorf("Error retrieving droplets: %s", err)
		}

		opts.Page = page + 1
	}

	tag := d.Get("tag")

	dropletList = findDropletListByTag(dropletList, tag.(string))

	err := exportDropletListProperties(d, dropletList)
	if err != nil {
		return err
	}
	return nil
}

func exportDropletListProperties(d *schema.ResourceData, droplets []godo.Droplet) error {
	items := make([]map[string]interface{}, 0)

	for _, droplet := range droplets {
		item := make(map[string]interface{})
		item["id"] = strconv.Itoa(droplet.ID)
		item["name"] = droplet.Name
		item["urn"] = droplet.URN()
		item["region"] = droplet.Region.Slug
		item["size"] = droplet.Size.Slug
		item["price_hourly"] = droplet.Size.PriceHourly
		item["price_monthly"] = droplet.Size.PriceMonthly
		item["disk"] = droplet.Disk
		item["vcpus"] = droplet.Vcpus
		item["memory"] = droplet.Memory
		item["status"] = droplet.Status
		item["locked"] = droplet.Locked
		item["created_at"] = droplet.Created

		if droplet.Image.Slug == "" {
			item["image"] = droplet.Image.ID
		} else {
			item["image"] = droplet.Image.Slug
		}

		if publicIPv4 := findIPv4AddrByType(&droplet, "public"); publicIPv4 != "" {
			item["ipv4_address"] = publicIPv4
		}

		if privateIPv4 := findIPv4AddrByType(&droplet, "private"); privateIPv4 != "" {
			item["ipv4_address_private"] = privateIPv4
		}

		if publicIPv6 := findIPv6AddrByType(&droplet, "public"); publicIPv6 != "" {
			item["ipv6_address"] = strings.ToLower(publicIPv6)
		}

		if privateIPv6 := findIPv6AddrByType(&droplet, "private"); privateIPv6 != "" {
			item["ipv6_address_private"] = strings.ToLower(privateIPv6)
		}

		if features := droplet.Features; features != nil {
			item["backups"] = containsDigitalOceanDropletFeature(features, "backups")
			item["ipv6"] = containsDigitalOceanDropletFeature(features, "ipv6")
			item["private_networking"] = containsDigitalOceanDropletFeature(features, "private_networking")
			item["monitoring"] = containsDigitalOceanDropletFeature(features, "monitoring")
		}

		item["volume_ids"] = flattenDigitalOceanDropletVolumeIds(droplet.VolumeIDs)

		item["tags"] = flattenTags(droplet.Tags)

		items = append(items, item)
	}

	d.Set("droplets", items)

	return nil
}

func findDropletListByTag(droplets []godo.Droplet, tag string) []godo.Droplet {
	results := make([]godo.Droplet, 0)
	for _, droplet := range droplets {
		for _, candidatetag := range droplet.Tags {
			if tag == candidatetag {
				results = append(results, droplet)
			}
		}
	}

	return results
}
