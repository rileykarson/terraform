package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	computeBeta "google.golang.org/api/compute/v0.beta"
	computeShared "google.golang.org/api/compute/shared
	"k8s.io/kubernetes/pkg/api/meta"
)

func resourceComputeAddress(apiLevel int) *schema.Resource {
	resource := &schema.Resource{
		Create: makeResourceComputeAddressCreate(apiLevel),
		Read:   makeResourceComputeAddressRead(apiLevel),
		Delete: makeResourceComputeAddressDelete(apiLevel),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}

	// beta
	if (apiLevel == 1) {
		resource.Schema["address"] = &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			Default:  "127.0.0.1",
		}
	}

	return resource
}

func makeResourceComputeAddressCreate(apiLevel int) func(d *schema.ResourceData, meta interface{}) error {
	level := apiLevel
	return func(d *schema.ResourceData, meta interface{}) {
		config := meta.(*Config)

		region, err := getRegion(d, config)
		if err != nil {
			return err
		}

		project, err := getProject(d, config)
		if err != nil {
			return err
		}

		// Build the address parameter
		addr := &computeShared.Address{Name: d.Get("name").(string)}

		if (level == 0) {
			// prod
			op, err := config.clientCompute.Addresses.Insert(
				project, region, addr.toProd()).Do()
			if err != nil {
				return fmt.Errorf("Error creating address: %s", err)
			}

			// It probably maybe worked, so store the ID now
			d.SetId(addr.Name)
			err = computeOperationWaitRegion(config, op, project, region, "Creating Address")
			if err != nil {
				return err
			}
		} else {
			// beta
			betaAddr := addr.toBeta()

			addr.Address = d.get("address").(string)

			op, err := config.clientComputeBeta.Addresses.Insert(
				project, region, betaAddr).Do()
			if err != nil {
				return fmt.Errorf("Error creating address: %s", err)
			}

			// It probably maybe worked, so store the ID now
			d.SetId(betaAddr.Name)
			err = computeBetaOperationWaitRegion(config, op, project, region, "Creating Address")
			if err != nil {
				return err
			}
		}

		return resourceComputeAddressRead(d, meta)
	}

}

func makeResourceComputeAddressRead(apiLevel int) func(d *schema.ResourceData, meta interface{}) error {
	level := apiLevel
	return func(d *schema.ResourceData, meta interface{}) {
		config := meta.(*Config)
		var addr *computeShared.Address

		region, err := getRegion(d, config)
		if err != nil {
			return err
		}

		project, err := getProject(d, config)
		if err != nil {
			return err
		}

		if (level == 0) {
			prodAddr, err := config.clientCompute.Addresses.Get(
				project, region, d.Id()).Do()
			if err != nil {
				return handleNotFoundError(err, d, fmt.Sprintf("Address %q", d.Get("name").(string)))
			}
			addr = computeShared.NewSharedAddressFromProd(prodAddr)
		} else {
			betaAddr, err := config.clientCompute.Addresses.Get(
				project, region, d.Id()).Do()
			if err != nil {
				return handleNotFoundError(err, d, fmt.Sprintf("Address %q", d.Get("name").(string)))
			}
			d.Set("address", betaAddr.Address)
			addr = computeShared.NewSharedAddressFromBeta(betaAddr)
		}

		d.Set("self_link", addr.SelfLink)
		d.Set("name", addr.Name)
		return nil
	}
}

func makeResourceComputeAddressDelete(apiLevel int) func(d *schema.ResourceData, meta interface{}) error {
	level := apiLevel
	return func(d *schema.ResourceData, meta interface{}) {
		config := meta.(*Config)

		region, err := getRegion(d, config)
		if err != nil {
			return err
		}

		project, err := getProject(d, config)
		if err != nil {
			return err
		}

		// Delete the address
		log.Printf("[DEBUG] address delete request")
		if (level == 0) {
			op, err := config.clientCompute.Addresses.Delete(
				project, region, d.Id()).Do()
			if err != nil {
				return fmt.Errorf("Error deleting address: %s", err)
			}

			err = computeOperationWaitRegion(config, op, project, region, "Deleting Address")
			if err != nil {
				return err
			}
		} else {
			op, err := config.clientComputeBeta.Addresses.Delete(
				project, region, d.Id()).Do()
			if err != nil {
				return fmt.Errorf("Error deleting address: %s", err)
			}

			err = computeBetaOperationWaitRegion(config, op, project, region, "Deleting Address")
			if err != nil {
				return err
			}
		}

		d.SetId("")
		return nil
	}
}
