package kubernetes

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_5"
	"k8s.io/kubernetes/pkg/util/intstr"
)

func TestAccKubernetesService_basic(t *testing.T) {
	var conf api.Service
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_service.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists("kubernetes_service.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.1724915162.name", ""),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.1724915162.node_port", "0"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.1724915162.port", "8080"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.1724915162.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.1724915162.target_port", "80"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.session_affinity", "None"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.type", "ClusterIP"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
						{
							Port:       int32(8080),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(80),
						},
					}),
				),
			},
			{
				Config: testAccKubernetesServiceConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists("kubernetes_service.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.annotations.Different", "1234"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "Different": "1234"}),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.1521549010.name", ""),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.1521549010.node_port", "0"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.1521549010.port", "8081"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.1521549010.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.1521549010.target_port", "80"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.session_affinity", "None"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.type", "ClusterIP"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
						{
							Port:       int32(8081),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(80),
						},
					}),
				),
			},
		},
	})
}

func TestAccKubernetesService_importBasic(t *testing.T) {
	resourceName := "kubernetes_service.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_basic(name),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesService_generatedName(t *testing.T) {
	var conf api.Service
	prefix := "tf-acc-test-gen-"

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_service.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists("kubernetes_service.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_service.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesService_importGeneratedName(t *testing.T) {
	resourceName := "kubernetes_service.test"
	prefix := "tf-acc-test-gen-import-"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_generatedName(prefix),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckServicePorts(svc *api.Service, expected []api.ServicePort) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(expected) == 0 && len(svc.Spec.Ports) == 0 {
			return nil
		}

		if !reflect.DeepEqual(svc.Spec.Ports, expected) {
			return fmt.Errorf("Service ports don't match.\nExpected: %#v\nGiven: %#v",
				expected, svc.Spec.Ports)
		}

		return nil
	}
}

func testAccCheckKubernetesServiceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_service" {
			continue
		}
		namespace, name := idParts(rs.Primary.ID)
		resp, err := conn.CoreV1().Services(namespace).Get(name)
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Service still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesServiceExists(n string, obj *api.Service) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*kubernetes.Clientset)
		namespace, name := idParts(rs.Primary.ID)
		out, err := conn.CoreV1().Services(namespace).Get(name)
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesServiceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_service" "test" {
	metadata {
		annotations {
			TestAnnotationOne = "one"
			TestAnnotationTwo = "two"
		}
		labels {
			TestLabelOne = "one"
			TestLabelTwo = "two"
			TestLabelThree = "three"
		}
		name = "%s"
	}
	spec {
		port {
			port = 8080
			target_port = 80
		}
	}
}`, name)
}

func testAccKubernetesServiceConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_service" "test" {
	metadata {
		annotations {
			TestAnnotationOne = "one"
			Different = "1234"
		}
		labels {
			TestLabelOne = "one"
			TestLabelThree = "three"
		}
		name = "%s"
	}
	spec {
		port {
			port = 8081
			target_port = 80
		}
	}
}`, name)
}

func testAccKubernetesServiceConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_service" "test" {
	metadata {
		generate_name = "%s"
	}
	spec {
		port {
			port = 8080
			target_port = 80
		}
	}
}`, prefix)
}
