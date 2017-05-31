package google

import (
	"fmt"
	"reflect"
	"testing"

	computeBeta "google.golang.org/api/compute/v0.beta"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccInstanceGroupManagerBeta_basic(t *testing.T) {
	var manager computeBeta.InstanceGroupManager

	template := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	target := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	igm1 := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	igm2 := fmt.Sprintf("igm-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceGroupManagerBetaDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceGroupManagerBeta_basic(template, target, igm1, igm2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerBetaExists(
						"google_compute_beta_instance_group_manager.igm-basic", &manager),
					testAccCheckInstanceGroupManagerBetaExists(
						"google_compute_beta_instance_group_manager.igm-no-tp", &manager),
				),
			},
		},
	})
}

func TestAccInstanceGroupManagerBeta_update(t *testing.T) {
	var manager computeBeta.InstanceGroupManager

	template1 := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	target := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	template2 := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	igm := fmt.Sprintf("igm-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceGroupManagerBetaDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceGroupManagerBeta_update(template1, target, igm),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerBetaExists(
						"google_compute_beta_instance_group_manager.igm-update", &manager),
					testAccCheckInstanceGroupManagerBetaNamedPorts(
						"google_compute_beta_instance_group_manager.igm-update",
						map[string]int64{"customhttp": 8080},
						&manager),
				),
			},
			resource.TestStep{
				Config: testAccInstanceGroupManagerBeta_update2(template1, target, template2, igm),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerBetaExists(
						"google_compute_beta_instance_group_manager.igm-update", &manager),
					testAccCheckInstanceGroupManagerBetaUpdated(
						"google_compute_beta_instance_group_manager.igm-update", 3,
						"google_compute_beta_target_pool.igm-update", template2),
					testAccCheckInstanceGroupManagerBetaNamedPorts(
						"google_compute_beta_instance_group_manager.igm-update",
						map[string]int64{"customhttp": 8080, "customhttps": 8443},
						&manager),
				),
			},
		},
	})
}

func TestAccInstanceGroupManagerBeta_updateLifecycle(t *testing.T) {
	var manager computeBeta.InstanceGroupManager

	tag1 := "tag1"
	tag2 := "tag2"
	igm := fmt.Sprintf("igm-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceGroupManagerBetaDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceGroupManagerBeta_updateLifecycle(tag1, igm),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerBetaExists(
						"google_compute_beta_instance_group_manager.igm-update", &manager),
				),
			},
			resource.TestStep{
				Config: testAccInstanceGroupManagerBeta_updateLifecycle(tag2, igm),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerBetaExists(
						"google_compute_beta_instance_group_manager.igm-update", &manager),
					testAccCheckInstanceGroupManagerBetaTemplateTags(
						"google_compute_beta_instance_group_manager.igm-update", []string{tag2}),
				),
			},
		},
	})
}

func TestAccInstanceGroupManagerBeta_updateStrategy(t *testing.T) {
	var manager computeBeta.InstanceGroupManager
	igm := fmt.Sprintf("igm-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceGroupManagerBetaDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceGroupManagerBeta_updateStrategy(igm),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerBetaExists(
						"google_compute_beta_instance_group_manager.igm-update-strategy", &manager),
					testAccCheckInstanceGroupManagerBetaUpdateStrategy(
						"google_compute_beta_instance_group_manager.igm-update-strategy", "NONE"),
				),
			},
		},
	})
}

func TestAccInstanceGroupManagerBeta_separateRegions(t *testing.T) {
	var manager computeBeta.InstanceGroupManager

	igm1 := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	igm2 := fmt.Sprintf("igm-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceGroupManagerBetaDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceGroupManagerBeta_separateRegions(igm1, igm2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerBetaExists(
						"google_compute_beta_instance_group_manager.igm-basic", &manager),
					testAccCheckInstanceGroupManagerBetaExists(
						"google_compute_beta_instance_group_manager.igm-basic-2", &manager),
				),
			},
		},
	})
}

func testAccCheckInstanceGroupManagerBetaDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_instance_group_manager" {
			continue
		}
		_, err := config.clientComputeBeta.InstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("InstanceGroupManager still exists")
		}
	}

	return nil
}

func testAccCheckInstanceGroupManagerBetaExists(n string, manager *computeBeta.InstanceGroupManager) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientComputeBeta.InstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("InstanceGroupManager not found")
		}

		*manager = *found

		return nil
	}
}

func testAccCheckInstanceGroupManagerBetaUpdated(n string, size int64, targetPool string, template string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		manager, err := config.clientComputeBeta.InstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		// Cannot check the target pool as the instance creation is asynchronous.  However, can
		// check the target_size.
		if manager.TargetSize != size {
			return fmt.Errorf("instance count incorrect")
		}

		// check that the instance template updated
		instanceTemplate, err := config.clientComputeBeta.InstanceTemplates.Get(
			config.Project, template).Do()
		if err != nil {
			return fmt.Errorf("Error reading instance template: %s", err)
		}

		if instanceTemplate.Name != template {
			return fmt.Errorf("instance template not updated")
		}

		return nil
	}
}

func testAccCheckInstanceGroupManagerBetaNamedPorts(n string, np map[string]int64, instanceGroupManager *computeBeta.InstanceGroupManager) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		manager, err := config.clientComputeBeta.InstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		var found bool
		for _, namedPort := range manager.NamedPorts {
			found = false
			for name, port := range np {
				if namedPort.Name == name && namedPort.Port == port {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("named port incorrect")
			}
		}

		return nil
	}
}

func testAccCheckInstanceGroupManagerBetaTemplateTags(n string, tags []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		manager, err := config.clientComputeBeta.InstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		// check that the instance template updated
		instanceTemplate, err := config.clientComputeBeta.InstanceTemplates.Get(
			config.Project, resourceSplitter(manager.InstanceTemplate)).Do()
		if err != nil {
			return fmt.Errorf("Error reading instance template: %s", err)
		}

		if !reflect.DeepEqual(instanceTemplate.Properties.Tags.Items, tags) {
			return fmt.Errorf("instance template not updated")
		}

		return nil
	}
}

func testAccCheckInstanceGroupManagerBetaUpdateStrategy(n, strategy string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		if rs.Primary.Attributes["update_strategy"] != strategy {
			return fmt.Errorf("Expected strategy to be %s, got %s",
				strategy, rs.Primary.Attributes["update_strategy"])
		}
		return nil
	}
}

func testAccInstanceGroupManagerBeta_basic(template, target, igm1, igm2 string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance_template" "igm-basic" {
		name = "%s"
		machine_type = "n1-standard-1"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
			auto_delete = true
			boot = true
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}

		service_account {
			scopes = ["userinfo-email", "compute-ro", "storage-ro"]
		}
	}

	resource "google_compute_target_pool" "igm-basic" {
		description = "Resource created for Terraform acceptance testing"
		name = "%s"
		session_affinity = "CLIENT_IP_PROTO"
	}

	resource "google_compute_beta_instance_group_manager" "igm-basic" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-basic.self_link}"
		target_pools = ["${google_compute_target_pool.igm-basic.self_link}"]
		base_instance_name = "igm-basic"
		zone = "us-central1-c"
		target_size = 2
	}

	resource "google_compute_beta_instance_group_manager" "igm-no-tp" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-basic.self_link}"
		base_instance_name = "igm-no-tp"
		zone = "us-central1-c"
		target_size = 2
	}
	`, template, target, igm1, igm2)
}

func testAccInstanceGroupManagerBeta_update(template, target, igm string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance_template" "igm-update" {
		name = "%s"
		machine_type = "n1-standard-1"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
			auto_delete = true
			boot = true
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}

		service_account {
			scopes = ["userinfo-email", "compute-ro", "storage-ro"]
		}
	}

	resource "google_compute_target_pool" "igm-update" {
		description = "Resource created for Terraform acceptance testing"
		name = "%s"
		session_affinity = "CLIENT_IP_PROTO"
	}

	resource "google_compute_beta_instance_group_manager" "igm-update" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-update.self_link}"
		target_pools = ["${google_compute_target_pool.igm-update.self_link}"]
		base_instance_name = "igm-update"
		zone = "us-central1-c"
		target_size = 2
		named_port {
			name = "customhttp"
			port = 8080
		}
	}`, template, target, igm)
}

// Change IGM's instance template and target size
func testAccInstanceGroupManagerBeta_update2(template1, target, template2, igm string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance_template" "igm-update" {
		name = "%s"
		machine_type = "n1-standard-1"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
			auto_delete = true
			boot = true
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}

		service_account {
			scopes = ["userinfo-email", "compute-ro", "storage-ro"]
		}
	}

	resource "google_compute_target_pool" "igm-update" {
		description = "Resource created for Terraform acceptance testing"
		name = "%s"
		session_affinity = "CLIENT_IP_PROTO"
	}

	resource "google_compute_instance_template" "igm-update2" {
		name = "%s"
		machine_type = "n1-standard-1"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
			auto_delete = true
			boot = true
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}

		service_account {
			scopes = ["userinfo-email", "compute-ro", "storage-ro"]
		}
	}

	resource "google_compute_beta_instance_group_manager" "igm-update" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-update2.self_link}"
		target_pools = ["${google_compute_target_pool.igm-update.self_link}"]
		base_instance_name = "igm-update"
		zone = "us-central1-c"
		target_size = 3
		named_port {
			name = "customhttp"
			port = 8080
		}
		named_port {
			name = "customhttps"
			port = 8443
		}
	}`, template1, target, template2, igm)
}

func testAccInstanceGroupManagerBeta_updateLifecycle(tag, igm string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance_template" "igm-update" {
		machine_type = "n1-standard-1"
		can_ip_forward = false
		tags = ["%s"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
			auto_delete = true
			boot = true
		}

		network_interface {
			network = "default"
		}

		service_account {
			scopes = ["userinfo-email", "compute-ro", "storage-ro"]
		}

		lifecycle {
			create_before_destroy = true
		}
	}

	resource "google_compute_beta_instance_group_manager" "igm-update" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-update.self_link}"
		base_instance_name = "igm-update"
		zone = "us-central1-c"
		target_size = 2
		named_port {
			name = "customhttp"
			port = 8080
		}
	}`, tag, igm)
}

func testAccInstanceGroupManagerBeta_updateStrategy(igm string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance_template" "igm-update-strategy" {
		machine_type = "n1-standard-1"
		can_ip_forward = false
		tags = ["terraform-testing"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
			auto_delete = true
			boot = true
		}

		network_interface {
			network = "default"
		}

		service_account {
			scopes = ["userinfo-email", "compute-ro", "storage-ro"]
		}

		lifecycle {
			create_before_destroy = true
		}
	}

	resource "google_compute_beta_instance_group_manager" "igm-update-strategy" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-update-strategy.self_link}"
		base_instance_name = "igm-update-strategy"
		zone = "us-central1-c"
		target_size = 2
		update_strategy = "NONE"
		named_port {
			name = "customhttp"
			port = 8080
		}
	}`, igm)
}

func testAccInstanceGroupManagerBeta_separateRegions(igm1, igm2 string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance_template" "igm-basic" {
		machine_type = "n1-standard-1"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
			auto_delete = true
			boot = true
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}

		service_account {
			scopes = ["userinfo-email", "compute-ro", "storage-ro"]
		}
	}

	resource "google_compute_beta_instance_group_manager" "igm-basic" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-basic.self_link}"
		base_instance_name = "igm-basic"
		zone = "us-central1-c"
		target_size = 2
	}

	resource "google_compute_beta_instance_group_manager" "igm-basic-2" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-basic.self_link}"
		base_instance_name = "igm-basic-2"
		zone = "us-west1-b"
		target_size = 2
	}
	`, igm1, igm2)
}
