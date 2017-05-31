package google

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/builtin/providers/google/shared"
	"github.com/hashicorp/terraform/helper/schema"

	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeInstanceGroupManager(apiLevel ApiLevel) *schema.Resource {
	s := &schema.Resource{
		Create: resourceComputeInstanceGroupManagerCreate(apiLevel),
		Read:   resourceComputeInstanceGroupManagerRead(apiLevel),
		Update: resourceComputeInstanceGroupManagerUpdate(apiLevel),
		Delete: resourceComputeInstanceGroupManagerDelete(apiLevel),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"base_instance_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance_template": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: self_link_relative_path_comparator,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"instance_group": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"named_port": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"update_strategy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "RESTART",
			},

			"target_pools": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					DiffSuppressFunc: self_link_relative_path_comparator,
				},
				Set: self_link_relative_path_hash,
			},

			"target_size": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
				Optional: true,
			},
		},
	}

	return s
}

func getNamedPortsShared(nps []interface{}) []*shared.NamedPort {
	namedPorts := make([]*shared.NamedPort, 0, len(nps))
	for _, v := range nps {
		np := v.(map[string]interface{})
		namedPorts = append(namedPorts, &shared.NamedPort{
			Name: np["name"].(string),
			Port: int64(np["port"].(int)),
		})
	}
	return namedPorts
}

func resourceComputeInstanceGroupManagerCreate(apiLevel ApiLevel) func(d *schema.ResourceData, meta interface{}) error {
	level := apiLevel
	return func(d *schema.ResourceData, meta interface{}) error {
		config := meta.(*Config)

		project, err := getProject(d, config)
		if err != nil {
			return err
		}

		// Get group size, default to 1 if not given
		var target_size int64 = 1
		if v, ok := d.GetOk("target_size"); ok {
			target_size = int64(v.(int))
		}

		// Build the parameter
		manager := &shared.InstanceGroupManager{
			Name:             d.Get("name").(string),
			BaseInstanceName: d.Get("base_instance_name").(string),
			InstanceTemplate: d.Get("instance_template").(string),
			TargetSize:       target_size,
		}

		// Set optional fields
		if v, ok := d.GetOk("description"); ok {
			manager.Description = v.(string)
		}

		if v, ok := d.GetOk("named_port"); ok {
			manager.NamedPorts = getNamedPortsShared(v.([]interface{}))
		}

		if attr := d.Get("target_pools").(*schema.Set); attr.Len() > 0 {
			var s []string
			for _, v := range attr.List() {
				s = append(s, v.(string))
			}
			manager.TargetPools = s
		}

		updateStrategy := d.Get("update_strategy").(string)
		if !(updateStrategy == "NONE" || updateStrategy == "RESTART") {
			return fmt.Errorf("Update strategy must be \"NONE\" or \"RESTART\"")
		}

		log.Printf("[DEBUG] InstanceGroupManager insert request: %#v", manager)
		var op interface{}
		switch level {
		case PRODUCTION:
			op, err = config.clientCompute.InstanceGroupManagers.Insert(
				project, d.Get("zone").(string), manager.ToProduction()).Do()
		case BETA:
			op, err = config.clientComputeBeta.InstanceGroupManagers.Insert(
				project, d.Get("zone").(string), manager.ToBeta()).Do()
		}

		if err != nil {
			return fmt.Errorf("Error creating InstanceGroupManager: %s", err)
		}

		// It probably maybe worked, so store the ID now
		d.SetId(manager.Name)

		// Wait for the operation to complete
		err = computeSharedOperationWaitZone(config, op, project, d.Get("zone").(string), "Creating InstanceGroupManager")
		if err != nil {
			return err
		}

		return resourceComputeInstanceGroupManagerRead(level)(d, meta)
	}
}

func flattenNamedPorts(namedPorts []*shared.NamedPort) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(namedPorts))
	for _, namedPort := range namedPorts {
		namedPortMap := make(map[string]interface{})
		namedPortMap["name"] = namedPort.Name
		namedPortMap["port"] = namedPort.Port
		result = append(result, namedPortMap)
	}
	return result

}

func resourceComputeInstanceGroupManagerRead(apiLevel ApiLevel) func(d *schema.ResourceData, meta interface{}) error {
	level := apiLevel
	return func(d *schema.ResourceData, meta interface{}) error {
		config := meta.(*Config)

		project, err := getProject(d, config)
		if err != nil {
			return err
		}

		region, err := getRegion(d, config)
		if err != nil {
			return err
		}

		var manager *shared.InstanceGroupManager
		switch level {
		case PRODUCTION:
			getInstanceGroupManager := func(zone string) (interface{}, error) {
				return config.clientCompute.InstanceGroupManagers.Get(project, zone, d.Id()).Do()
			}

			var productionManager *compute.InstanceGroupManager
			var e error
			if zone, ok := d.GetOk("zone"); ok {
				productionManager, e = config.clientCompute.InstanceGroupManagers.Get(project, zone.(string), d.Id()).Do()

				if e != nil {
					return handleNotFoundError(e, d, fmt.Sprintf("Instance Group Manager %q", d.Get("name").(string)))
				}
			} else {
				// If the resource was imported, the only info we have is the ID. Try to find the resource
				// by searching in the region of the project.
				var resource interface{}
				resource, e = getZonalResourceFromRegion(getInstanceGroupManager, region, config.clientCompute, project)

				if e != nil {
					return e
				}

				productionManager = resource.(*compute.InstanceGroupManager)
			}

			if productionManager == nil {
				log.Printf("[WARN] Removing Instance Group Manager %q because it's gone", d.Get("name").(string))

				// The resource doesn't exist anymore
				d.SetId("")
				return nil
			}

			manager = shared.InstanceGroupManagerFromProduction(productionManager)
		case BETA:
			getInstanceGroupManager := func(zone string) (interface{}, error) {
				return config.clientComputeBeta.InstanceGroupManagers.Get(project, zone, d.Id()).Do()
			}

			var betaManager *computeBeta.InstanceGroupManager
			var e error
			if zone, ok := d.GetOk("zone"); ok {
				betaManager, e = config.clientComputeBeta.InstanceGroupManagers.Get(project, zone.(string), d.Id()).Do()

				if e != nil {
					return handleNotFoundError(e, d, fmt.Sprintf("Instance Group Manager %q", d.Get("name").(string)))
				}
			} else {
				// If the resource was imported, the only info we have is the ID. Try to find the resource
				// by searching in the region of the project.
				var resource interface{}
				resource, e = getZonalBetaResourceFromRegion(getInstanceGroupManager, region, config.clientComputeBeta, project)

				if e != nil {
					return e
				}

				betaManager = resource.(*computeBeta.InstanceGroupManager)
			}

			if betaManager == nil {
				log.Printf("[WARN] Removing Instance Group Manager %q because it's gone", d.Get("name").(string))

				// The resource doesn't exist anymore
				d.SetId("")
				return nil
			}

			manager = shared.InstanceGroupManagerFromBeta(betaManager)
		}

		zoneUrl := strings.Split(manager.Zone, "/")
		d.Set("base_instance_name", manager.BaseInstanceName)
		d.Set("instance_template", manager.InstanceTemplate)
		d.Set("name", manager.Name)
		d.Set("zone", zoneUrl[len(zoneUrl)-1])
		d.Set("description", manager.Description)
		d.Set("project", project)
		d.Set("target_size", manager.TargetSize)
		d.Set("target_pools", manager.TargetPools)
		d.Set("named_port", flattenNamedPorts(manager.NamedPorts))
		d.Set("fingerprint", manager.Fingerprint)
		d.Set("instance_group", manager.InstanceGroup)
		d.Set("target_size", manager.TargetSize)
		d.Set("self_link", manager.SelfLink)
		update_strategy, ok := d.GetOk("update_strategy")
		if !ok {
			update_strategy = "RESTART"
		}
		d.Set("update_strategy", update_strategy.(string))

		return nil
	}
}
func resourceComputeInstanceGroupManagerUpdate(apiLevel ApiLevel) func(d *schema.ResourceData, meta interface{}) error {
	level := apiLevel
	return func(d *schema.ResourceData, meta interface{}) error {
		config := meta.(*Config)

		project, err := getProject(d, config)
		if err != nil {
			return err
		}

		d.Partial(true)

		// If target_pools changes then update
		if d.HasChange("target_pools") {
			var targetPools []string
			if attr := d.Get("target_pools").(*schema.Set); attr.Len() > 0 {
				for _, v := range attr.List() {
					targetPools = append(targetPools, v.(string))
				}
			}

			// Build the parameter
			setTargetPools := &shared.InstanceGroupManagersSetTargetPoolsRequest{
				Fingerprint: d.Get("fingerprint").(string),
				TargetPools: targetPools,
			}

			var op interface{}
			switch level {
			case PRODUCTION:
				op, err = config.clientCompute.InstanceGroupManagers.SetTargetPools(
					project, d.Get("zone").(string), d.Id(), setTargetPools.ToProduction()).Do()
			case BETA:
				op, err = config.clientComputeBeta.InstanceGroupManagers.SetTargetPools(
					project, d.Get("zone").(string), d.Id(), setTargetPools.ToBeta()).Do()

			}

			if err != nil {
				return fmt.Errorf("Error updating InstanceGroupManager: %s", err)
			}

			// Wait for the operation to complete
			err = computeSharedOperationWaitZone(config, op, project, d.Get("zone").(string), "Updating InstanceGroupManager")
			if err != nil {
				return err
			}

			d.SetPartial("target_pools")
		}

		// If instance_template changes then update
		if d.HasChange("instance_template") {
			// Build the parameter
			setInstanceTemplate := &shared.InstanceGroupManagersSetInstanceTemplateRequest{
				InstanceTemplate: d.Get("instance_template").(string),
			}

			var op interface{}
			switch level {
			case PRODUCTION:
				op, err = config.clientCompute.InstanceGroupManagers.SetInstanceTemplate(
					project, d.Get("zone").(string), d.Id(), setInstanceTemplate.ToProduction()).Do()
			case BETA:
				op, err = config.clientComputeBeta.InstanceGroupManagers.SetInstanceTemplate(
					project, d.Get("zone").(string), d.Id(), setInstanceTemplate.ToBeta()).Do()
			}

			if err != nil {
				return fmt.Errorf("Error updating InstanceGroupManager: %s", err)
			}

			// Wait for the operation to complete
			err = computeSharedOperationWaitZone(config, op, project, d.Get("zone").(string), "Updating InstanceGroupManager")
			if err != nil {
				return err
			}

			if d.Get("update_strategy").(string) == "RESTART" {
				// While there is an opportunity to have more shared code here, the ManagedInstances
				// type goes 5 levels deep. For now at least, it isn't worth converting this section.
				var managedInstanceCount int
				switch level {
				case PRODUCTION:
					managedInstances, err := config.clientCompute.InstanceGroupManagers.ListManagedInstances(
						project, d.Get("zone").(string), d.Id()).Do()

					managedInstanceCount = len(managedInstances.ManagedInstances)
					instances := make([]string, managedInstanceCount)
					for i, v := range managedInstances.ManagedInstances {
						instances[i] = v.Instance
					}

					recreateInstances := &compute.InstanceGroupManagersRecreateInstancesRequest{
						Instances: instances,
					}

					op, err = config.clientCompute.InstanceGroupManagers.RecreateInstances(
						project, d.Get("zone").(string), d.Id(), recreateInstances).Do()

					if err != nil {
						return fmt.Errorf("Error restarting instance group managers instances: %s", err)
					}
				case BETA:
					managedInstances, err := config.clientComputeBeta.InstanceGroupManagers.ListManagedInstances(
						project, d.Get("zone").(string), d.Id()).Do()

					managedInstanceCount = len(managedInstances.ManagedInstances)
					instances := make([]string, managedInstanceCount)
					for i, v := range managedInstances.ManagedInstances {
						instances[i] = v.Instance
					}

					recreateInstances := &computeBeta.InstanceGroupManagersRecreateInstancesRequest{
						Instances: instances,
					}

					op, err = config.clientComputeBeta.InstanceGroupManagers.RecreateInstances(
						project, d.Get("zone").(string), d.Id(), recreateInstances).Do()

					if err != nil {
						return fmt.Errorf("Error restarting instance group managers instances: %s", err)
					}
				}

				// Wait for the operation to complete
				err = computeSharedOperationWaitZoneTime(config, op, project, d.Get("zone").(string),
					managedInstanceCount*4, "Restarting InstanceGroupManagers instances")
				if err != nil {
					return err
				}
			}

			d.SetPartial("instance_template")
		}

		// If named_port changes then update:
		if d.HasChange("named_port") {

			// Build the parameters for a "SetNamedPorts" request:
			namedPorts := getNamedPortsShared(d.Get("named_port").([]interface{}))
			setNamedPorts := &shared.InstanceGroupsSetNamedPortsRequest{
				NamedPorts: namedPorts,
			}

			// Make the request:
			var op interface{}
			switch level {
			case PRODUCTION:
				op, err = config.clientCompute.InstanceGroups.SetNamedPorts(
					project, d.Get("zone").(string), d.Id(), setNamedPorts.ToProduction()).Do()
			case BETA:

				op, err = config.clientComputeBeta.InstanceGroups.SetNamedPorts(
					project, d.Get("zone").(string), d.Id(), setNamedPorts.ToBeta()).Do()
			}

			if err != nil {
				return fmt.Errorf("Error updating InstanceGroupManager: %s", err)
			}

			// Wait for the operation to complete:
			err = computeSharedOperationWaitZone(config, op, project, d.Get("zone").(string), "Updating InstanceGroupManager")
			if err != nil {
				return err
			}

			d.SetPartial("named_port")
		}

		// If size changes trigger a resize
		if d.HasChange("target_size") {
			if v, ok := d.GetOk("target_size"); ok {
				// Only do anything if the new size is set
				target_size := int64(v.(int))

				var op interface{}
				switch level {
				case PRODUCTION:
					op, err = config.clientCompute.InstanceGroupManagers.Resize(
						project, d.Get("zone").(string), d.Id(), target_size).Do()
				case BETA:
					op, err = config.clientComputeBeta.InstanceGroupManagers.Resize(
						project, d.Get("zone").(string), d.Id(), target_size).Do()
				}

				if err != nil {
					return fmt.Errorf("Error updating InstanceGroupManager: %s", err)
				}

				// Wait for the operation to complete
				err = computeSharedOperationWaitZone(config, op, project, d.Get("zone").(string), "Updating InstanceGroupManager")
				if err != nil {
					return err
				}
			}

			d.SetPartial("target_size")
		}
		d.Partial(false)

		return resourceComputeInstanceGroupManagerRead(level)(d, meta)
	}
}

func resourceComputeInstanceGroupManagerDelete(apiLevel ApiLevel) func(d *schema.ResourceData, meta interface{}) error {
	level := apiLevel
	return func(d *schema.ResourceData, meta interface{}) error {
		config := meta.(*Config)

		project, err := getProject(d, config)
		if err != nil {
			return err
		}

		zone := d.Get("zone").(string)

		var op interface{}
		switch level {
		case PRODUCTION:
			op, err = config.clientCompute.InstanceGroupManagers.Delete(project, zone, d.Id()).Do()
			attempt := 0
			for err != nil && attempt < 20 {
				attempt++
				time.Sleep(2000 * time.Millisecond)
				op, err = config.clientCompute.InstanceGroupManagers.Delete(project, zone, d.Id()).Do()
			}
		case BETA:
			op, err = config.clientComputeBeta.InstanceGroupManagers.Delete(project, zone, d.Id()).Do()
			attempt := 0
			for err != nil && attempt < 20 {
				attempt++
				time.Sleep(2000 * time.Millisecond)
				op, err = config.clientComputeBeta.InstanceGroupManagers.Delete(project, zone, d.Id()).Do()
			}
		}

		if err != nil {
			return fmt.Errorf("Error deleting instance group manager: %s", err)
		}

		currentSize := int64(d.Get("target_size").(int))

		// Wait for the operation to complete
		err = computeSharedOperationWaitZone(config, op, project, d.Get("zone").(string), "Deleting InstanceGroupManager")

		for err != nil && currentSize > 0 {
			if !strings.Contains(err.Error(), "timeout") {
				return err
			}

			var instanceGroupSize int64
			switch level {
			case PRODUCTION:
				instanceGroup, err := config.clientCompute.InstanceGroups.Get(
					project, d.Get("zone").(string), d.Id()).Do()
				if err != nil {
					return fmt.Errorf("Error getting instance group size: %s", err)
				}

				instanceGroupSize = instanceGroup.Size
			case BETA:
				instanceGroup, err := config.clientComputeBeta.InstanceGroups.Get(
					project, d.Get("zone").(string), d.Id()).Do()
				if err != nil {
					return fmt.Errorf("Error getting instance group size: %s", err)
				}

				instanceGroupSize = instanceGroup.Size
			}

			if instanceGroupSize >= currentSize {
				return fmt.Errorf("Error, instance group isn't shrinking during delete")
			}

			log.Printf("[INFO] timeout occured, but instance group is shrinking (%d < %d)", instanceGroupSize, currentSize)
			currentSize = instanceGroupSize
			err = computeSharedOperationWaitZone(config, op, project, d.Get("zone").(string), "Deleting InstanceGroupManager")
		}

		d.SetId("")
		return nil
	}
}

func getZonalResourceFromRegion(getResource func(string) (interface{}, error), region string, compute *compute.Service, project string) (interface{}, error) {
	zoneList, err := compute.Zones.List(project).Do()
	if err != nil {
		return nil, err
	}
	var resource interface{}
	for _, zone := range zoneList.Items {
		if strings.Contains(zone.Name, region) {
			resource, err = getResource(zone.Name)
			if err != nil {
				if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
					// Resource was not found in this zone
					continue
				}
				return nil, fmt.Errorf("Error reading Resource: %s", err)
			}
			// Resource was found
			return resource, nil
		}
	}
	// Resource does not exist in this region
	return nil, nil
}

func getZonalBetaResourceFromRegion(getResource func(string) (interface{}, error), region string, computeBeta *computeBeta.Service, project string) (interface{}, error) {
	zoneList, err := computeBeta.Zones.List(project).Do()
	if err != nil {
		return nil, err
	}
	var resource interface{}
	for _, zone := range zoneList.Items {
		if strings.Contains(zone.Name, region) {
			resource, err = getResource(zone.Name)
			if err != nil {
				if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
					// Resource was not found in this zone
					continue
				}
				return nil, fmt.Errorf("Error reading Resource: %s", err)
			}
			// Resource was found
			return resource, nil
		}
	}
	// Resource does not exist in this region
	return nil, nil
}
