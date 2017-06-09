package google

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"

	"cloud.google.com/go/bigtable"
	"golang.org/x/net/context"
)

func resourceBigtableInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceBigtableInstanceCreate,
		Read:   resourceBigtableInstanceRead,
		Delete: resourceBigtableInstanceDestroy,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cluster_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"zone": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"num_nodes": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"project": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"storage_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceBigtableInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ctx := context.Background()

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	clusterId := d.Get("cluster_id").(string)
	numNodes := int32(d.Get("num_nodes").(int))
	zone := d.Get("zone").(string)

	var storageType bigtable.StorageType
	switch value := d.Get("storage_type"); value {
	case "HDD":
		storageType = bigtable.HDD
	case "SSD":
		storageType = bigtable.SSD
	default:
		return fmt.Errorf("Invalid BigTable Instance storage_type. Saw: %s", value)
	}

	instanceConf := &bigtable.InstanceConf{
		ClusterId:   clusterId,
		DisplayName: name,
		InstanceId:  name,
		NumNodes:    numNodes,
		StorageType: storageType,
		Zone:        zone,
	}

	c, err := config.clientFactoryBigtable.NewInstanceAdminClient(project)
	if err != nil {
		return fmt.Errorf("Error starting instance admin client. %s", err)
	}

	defer c.Close()

	err = c.CreateInstance(ctx, instanceConf)
	if err != nil {
		return fmt.Errorf("Error creating instance. %s", err)
	}

	d.SetId(name)

	return resourceBigtableInstanceRead(d, meta)
}

func resourceBigtableInstanceRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ctx := context.Background()

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	c, err := config.clientFactoryBigtable.NewInstanceAdminClient(project)
	if err != nil {
		return fmt.Errorf("Error starting instance admin client. %s", err)
	}

	defer c.Close()

	name := d.Id()
	instances, err := c.Instances(ctx)
	if err != nil {
		return fmt.Errorf("Error retrieving instances. %s", err)
	}

	var instanceInfo *bigtable.InstanceInfo
	found := false
	for _, i := range instances {
		if i.Name == name {
			instanceInfo = i
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Error retrieving instance. Could not find %s.", name)
	}

	d.Set("name", instanceInfo.Name)

	return nil
}

func resourceBigtableInstanceDestroy(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ctx := context.Background()

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	c, err := config.clientFactoryBigtable.NewInstanceAdminClient(project)
	if err != nil {
		return fmt.Errorf("Error starting instance admin client. %s", err)
	}

	defer c.Close()

	name := d.Id()
	err = c.DeleteInstance(ctx, name)
	if err != nil {
		return fmt.Errorf("Error deleting instance. %s", err)
	}

	d.SetId("")

	return nil
}
