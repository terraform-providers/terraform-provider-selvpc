package selectel

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/selectel/dbaas-go"
	"github.com/selectel/go-selvpcclient/selvpcclient/resell/v2/projects"
)

func TestAccDBaaSConfigurationParametersV1Basic(t *testing.T) {
	var (
		dbaasConfigurationParameters []dbaas.ConfigurationParameter
		project                      projects.Project
	)

	projectName := acctest.RandomWithPrefix("tf-acc")
	datastoreTypeEngine := "postgresql"
	datastoreTypeVersion := "12"
	parameterName := "work_mem"
	parameterNameWithChoices := "session_replication_role"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccSelectelPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckVPCV2ProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDBaaSConfigurationParametersV1Basic(projectName, datastoreTypeEngine, datastoreTypeVersion, parameterName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCV2ProjectExists("selectel_vpc_project_v2.project_tf_acc_test_1", &project),
					testAccDBaaSConfigurationParametersV1Exists("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", &dbaasConfigurationParameters),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.name", parameterName),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.type", "int"),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.unit", "kB"),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.min", "64"),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.max", "2147483647"),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.default_value", "32768"),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.choices.#", "0"),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.is_restart_required", "false"),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.is_changeable", "true"),
				),
			},
			{
				Config: testAccDBaaSConfigurationParametersV1Basic(projectName, datastoreTypeEngine, datastoreTypeVersion, parameterNameWithChoices),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCV2ProjectExists("selectel_vpc_project_v2.project_tf_acc_test_1", &project),
					testAccDBaaSConfigurationParametersV1Exists("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", &dbaasConfigurationParameters),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.name", parameterNameWithChoices),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.type", "str"),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.unit", ""),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.min", ""),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.max", ""),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.default_value", "origin"),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.choices.#", "3"),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.choices.0", "origin"),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.choices.1", "replica"),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.choices.2", "local"),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.is_restart_required", "false"),
					resource.TestCheckResourceAttr("data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1", "configuration_parameters.0.is_changeable", "true"),
				),
			},
		},
	})
}

func testAccDBaaSConfigurationParametersV1Exists(n string, dbaasConfigurationParameters *[]dbaas.ConfigurationParameter) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		ctx := context.Background()

		dbaasClient, err := baseTestAccCheckDBaaSV1EntityExists(ctx, rs, testAccProvider)
		if err != nil {
			return err
		}

		configurationParameters, err := dbaasClient.ConfigurationParameters(ctx)
		if err != nil {
			return err
		}

		*dbaasConfigurationParameters = configurationParameters

		return nil
	}
}

func testAccDBaaSConfigurationParametersV1Basic(projectName, engine, version, name string) string {
	return fmt.Sprintf(`
resource "selectel_vpc_project_v2" "project_tf_acc_test_1" {
  name        = "%s"
  auto_quotas = true
}

data "selectel_dbaas_datastore_type_v1" "dt" {
  project_id = "${selectel_vpc_project_v2.project_tf_acc_test_1.id}"
  region     = "ru-3"
  filter {
	engine = "%s"
	version = "%s"
  }
}

data "selectel_dbaas_configuration_parameters_v1" "configuration_param_tf_acc_test_1" {
  project_id = "${selectel_vpc_project_v2.project_tf_acc_test_1.id}"
  region     = "ru-3"
  filter {
    datastore_type_id = "${data.selectel_dbaas_datastore_type_v1.dt.datastore_types[0].id}"
    name = "%s"
  }
}

output "config" {
  value = data.selectel_dbaas_configuration_parameters_v1.configuration_param_tf_acc_test_1.configuration_parameters[0]
}
`, projectName, engine, version, name)
}
