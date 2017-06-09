package google

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccBigtableFamily_basic(t *testing.T) {
	instanceName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	tableName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	familyName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBigtableFamilyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBigtableFamily(instanceName, tableName, familyName),
				Check: resource.ComposeTestCheckFunc(
					testAccBigtableFamilyExists(
						"google_bigtable_family.family"),
				),
			},
		},
	})
}

func TestAccBigtableFamily_update(t *testing.T) {
	instanceName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	tableName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	familyName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBigtableFamilyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBigtableFamily(instanceName, tableName, familyName),
				Check: resource.ComposeTestCheckFunc(
					testAccBigtableFamilyExists(
						"google_bigtable_family.family"),
				),
			},
			{
				Config: testAccBigtableFamily_update(instanceName, tableName, familyName),
				Check: resource.ComposeTestCheckFunc(
					testAccBigtableFamilyExists(
						"google_bigtable_family.family"),
				),
			},
		},
	})
}

// Test that grpc is not retrying properly. This test is going to be flaky, as it is timing sensitive. If the destroy
// op fails, a Bigtable might also be left running.
func TestAccBigtableFamily_grpc(t *testing.T) {
	instanceName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	tableName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	tableName2 := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	familyName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	familyName2 := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	familyName3 := fmt.Sprintf("tf-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBigtableFamilyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBigtableFamily_grpc(instanceName, tableName, tableName2, familyName, familyName2, familyName3),
				Check: resource.ComposeTestCheckFunc(
					testAccBigtableFamilyExists(
						"google_bigtable_family.family"),
				),
			},
		},
	})
}

func testAccCheckBigtableFamilyDestroy(s *terraform.State) error {
	var ctx = context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_bigtable_table" {
			continue
		}

		config := testAccProvider.Meta().(*Config)
		c, err := config.clientFactoryBigtable.NewAdminClient(config.Project, rs.Primary.Attributes["instance_name"])
		if err != nil {
			// The instance is already gone
			return nil
		}

		tableInfo, err := c.TableInfo(ctx, rs.Primary.Attributes["table_name"])
		if err != nil {
			// The instance or table is already gone
			return nil
		}

		found := false
		for _, v := range tableInfo.FamilyInfos {
			if v.Name == rs.Primary.Attributes["name"] {
				found = true
				break
			}
		}

		if found {
			return fmt.Errorf("Family still present. Found %s in %s in %s.", rs.Primary.Attributes["name"], rs.Primary.Attributes["table_name"], rs.Primary.Attributes["instance_name"])
		}

		c.Close()
	}

	return nil
}

func testAccBigtableFamilyExists(n string) resource.TestCheckFunc {
	var ctx = context.Background()
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		config := testAccProvider.Meta().(*Config)
		c, err := config.clientFactoryBigtable.NewAdminClient(config.Project, rs.Primary.Attributes["instance_name"])
		if err != nil {
			return fmt.Errorf("Error starting admin client. %s", err)
		}

		tableInfo, err := c.TableInfo(ctx, rs.Primary.Attributes["table_name"])
		if err != nil {
			return fmt.Errorf("Error retrieving table. %s", err)
		}

		found := false
		for _, v := range tableInfo.FamilyInfos {
			if v.Name == rs.Primary.Attributes["name"] {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Error retrieving family. %s", err)
		}

		c.Close()

		return nil
	}
}

func testAccBigtableFamily(instanceName, tableName, familyName string) string {
	return fmt.Sprintf(`
resource "google_bigtable_instance" "instance" {
  name     = "%s"
  cluster_id = "%s"
  zone = "us-central1-b"
  num_nodes = 3
  storage_type = "HDD"
}

resource "google_bigtable_table" "table" {
  name     = "%s"
  instance_name = "${google_bigtable_instance.instance.name}"
  split_keys = ["a", "b", "c"]
}

resource "google_bigtable_family" "family" {
  name     = "%s"
  instance_name = "${google_bigtable_instance.instance.name}"
  table_name = "${google_bigtable_table.table.name}"
  version_policy = 1
}
`, instanceName, instanceName, tableName, familyName)
}

func testAccBigtableFamily_update(instanceName, tableName, familyName string) string {
	return fmt.Sprintf(`
resource "google_bigtable_instance" "instance" {
  name     = "%s"
  cluster_id = "%s"
  zone = "us-central1-b"
  num_nodes = 3
  storage_type = "HDD"
}

resource "google_bigtable_table" "table" {
  name     = "%s"
  instance_name = "${google_bigtable_instance.instance.name}"
  split_keys = ["a", "b", "c"]
}

resource "google_bigtable_family" "family" {
  name     = "%s"
  instance_name = "${google_bigtable_instance.instance.name}"
  table_name = "${google_bigtable_table.table.name}"
  version_policy = 10
}
`, instanceName, instanceName, tableName, familyName)
}

func testAccBigtableFamily_grpc(instanceName, tableName, tableName2, familyName, familyName2, familyName3 string) string {
	return fmt.Sprintf(`
resource "google_bigtable_instance" "instance" {
  name     = "%s"
  cluster_id = "%s"
  zone = "us-central1-b"
  num_nodes = 3
  storage_type = "HDD"
}

resource "google_bigtable_table" "table" {
  name     = "%s"
  instance_name = "${google_bigtable_instance.instance.name}"
  split_keys = ["a", "b", "c"]
}

resource "google_bigtable_table" "table2" {
  name     = "%s"
  instance_name = "${google_bigtable_instance.instance.name}"
  split_keys = ["a", "b", "c"]
}

resource "google_bigtable_family" "family" {
  name     = "%s"
  instance_name = "${google_bigtable_instance.instance.name}"
  table_name = "${google_bigtable_table.table.name}"
  version_policy = 1
}

resource "google_bigtable_family" "family2" {
  name     = "%s"
  instance_name = "${google_bigtable_instance.instance.name}"
  table_name = "${google_bigtable_table.table.name}"
  version_policy = 10
}

resource "google_bigtable_family" "family3" {
  name     = "%s"
  instance_name = "${google_bigtable_instance.instance.name}"
  table_name = "${google_bigtable_table.table2.name}"
  version_policy = 1
}
`, instanceName, instanceName, tableName, tableName2, familyName, familyName2, familyName3)
}
