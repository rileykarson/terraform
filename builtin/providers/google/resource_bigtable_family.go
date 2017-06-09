package google

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"

	"cloud.google.com/go/bigtable"
	"golang.org/x/net/context"
)

func resourceBigtableFamily() *schema.Resource {
	return &schema.Resource{
		Create: resourceBigtableFamilyCreate,
		Read:   resourceBigtableFamilyRead,
		Update: resourceBigtableFamilyUpdate,
		Delete: resourceBigtableFamilyDestroy,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"table_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"version_policy": {
				Type:     schema.TypeInt,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value < 1 {
						errors = append(errors, fmt.Errorf("%q must be at least 1.", k))
					}
					return
				},
			},

			"project": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"gc_policy": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceBigtableFamilyCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ctx := context.Background()

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	instanceName := d.Get("instance_name").(string)
	tableName := d.Get("table_name").(string)
	name := d.Get("name").(string)

	c, err := config.clientFactoryBigTable.NewAdminClient(project, instanceName)
	if err != nil {
		return fmt.Errorf("Error starting admin client. %s", err)
	}

	defer c.Close()

	_, err = c.TableInfo(ctx, tableName)
	if err != nil {
		return fmt.Errorf("Error retrieving table. %s", err)
	}

	err = c.CreateColumnFamily(ctx, tableName, name)
	if err != nil {
		return fmt.Errorf("Error creating family. %s", err)
	}

	gcPolicy := bigtable.MaxVersionsPolicy(d.Get("version_policy").(int))
	err = c.SetGCPolicy(ctx, tableName, name, gcPolicy)
	if err != nil {
		return fmt.Errorf("Error setting GC policy. %s", err)
	}

	d.SetId(name)

	return resourceBigtableFamilyRead(d, meta)
}

func resourceBigtableFamilyRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ctx := context.Background()

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	instanceName := d.Get("instance_name").(string)
	tableName := d.Get("table_name").(string)
	name := d.Get("name").(string)

	c, err := config.clientFactoryBigTable.NewAdminClient(project, instanceName)
	if err != nil {
		return fmt.Errorf("Error starting admin client. %s", err)
	}

	defer c.Close()

	tableInfo, err := c.TableInfo(ctx, tableName)
	if err != nil {
		return fmt.Errorf("Error retrieving table. %s", err)
	}

	var familyInfo bigtable.FamilyInfo
	found := false
	for _, v := range tableInfo.FamilyInfos {
		if v.Name == name {
			familyInfo = v
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Error retrieving family. %s", err)
	}

	d.Set("name", familyInfo.Name)
	d.Set("gc_policy", familyInfo.GCPolicy)

	return nil
}

func resourceBigtableFamilyUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ctx := context.Background()

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	instanceName := d.Get("instance_name").(string)
	tableName := d.Get("table_name").(string)
	name := d.Get("name").(string)

	c, err := config.clientFactoryBigTable.NewAdminClient(project, instanceName)
	if err != nil {
		return fmt.Errorf("Error starting admin client. %s", err)
	}

	defer c.Close()

	_, err = c.TableInfo(ctx, tableName)
	if err != nil {
		return fmt.Errorf("Error retrieving table. %s", err)
	}

	gcPolicy := bigtable.MaxVersionsPolicy(d.Get("version_policy").(int))
	err = c.SetGCPolicy(ctx, tableName, name, gcPolicy)
	if err != nil {
		return fmt.Errorf("Error setting GC policy. %s", err)
	}

	return resourceBigtableFamilyRead(d, meta)
}

func resourceBigtableFamilyDestroy(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ctx := context.Background()

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	instanceName := d.Get("instance_name").(string)
	tableName := d.Get("table_name").(string)
	name := d.Get("name").(string)

	c, err := config.clientFactoryBigTable.NewAdminClient(project, instanceName)
	if err != nil {
		return fmt.Errorf("Error starting admin client. %s", err)
	}

	defer c.Close()

	_, err = c.TableInfo(ctx, tableName)
	if err != nil {
		return fmt.Errorf("Error retrieving table. %s", err)
	}

	err = c.DeleteColumnFamily(ctx, tableName, name)
	if err != nil {
		return fmt.Errorf("Error deleting family. %s", err)
	}

	d.SetId("")

	return nil
}
