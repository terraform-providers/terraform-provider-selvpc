---
layout: "selectel"
page_title: "Selectel: selectel_dbaas_datastore_v1"
sidebar_current: "docs-selectel-resource-dbaas-datastore-v1"
description: |-
  Manages a V1 datastore resource within Selectel Managed Databases Service.
---

# selectel\_dbaas\_datastore\_v1

Manages a V1 datastore resource within Selectel Managed Databases Service.

## Example usage

```hcl
resource "selectel_vpc_project_v2" "project_1" {
  auto_quotas = true
}

resource "selectel_vpc_subnet_v2" "subnet" {
  project_id   = "${selectel_vpc_project_v2.project_1.id}"
  region       = "ru-3"
}

data "selectel_dbaas_datastore_type_v1" "dt" {
  project_id   = "${selectel_vpc_project_v2.project_1.id}"
  region       = "ru-3"
  filter {
    engine  = "postgresql"
    version = "12"
  }
}

resource "selectel_dbaas_datastore_v1" "datastore_1" {
  name         = "datastore-1"
  project_id   = "${selectel_vpc_project_v2.project_1.id}"
  region       = "ru-3"
  type_id      = data.selectel_dbaas_datastore_type_v1.dt.datastore_types[0].id
  subnet_id    = "${selectel_vpc_subnet_v2.subnet.subnet_id}"
  node_count   = 3
  flavor {
    vcpus = 4
    ram   = 4096
    disk  = 32
  }
  pooler {
    mode = "transaction"
    size = 50
  }
  config = {
    xmloption = "content"
    work_mem = 512
    session_replication_role = "replica"
    vacuum_cost_delay = 25
    transform_null_equals = false
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A name of the datastore.
  Changing this creates a new datastore.

* `project_id` - (Required) An associated Selectel VPC project.
  Changing this creates a new datastore.

* `region` - (Required) A Selectel VPC region of where the datastore is located.
  Changing this creates a new datastore.

* `subnet_id` - (Required) Associated OpenStack Networking service subnet ID.
  Changing this creates a new datastore.

* `type_id` - (Required) The datastore type for the datastore.
  Changing this creates a new datastore.

* `node_count` - (Required) Number of nodes to create for the datastore.

* `flavor_id` - (Optional) Flavor identifier for the datastore. It can be omitted in cases when `flavor` is set.

* `flavor` - (Optional) Flavor configuration for the datastore. It's a complex value. See description below.

* `pooler` - (Optional) Pooler configuration for the datastore (only for PostgreSQL datastore). It's a complex value. See description below.

* `firewall` - (Optional) List of the ips to allow access from.

* `restore` - (Optional) Restore parameters for the datastore. It's a complex value. See description below.
  Changing this creates a new datastore.

* `config` - (Optional) Configuration parameters for the datastore.

**flavor**

- `vcpus` - (Required) CPU count for the flavor.
- `ram` - (Required) RAM count for the flavor.
- `disk` - (Required) Disk size for the flavor.

**pooler**

- `mode` - (Required) Mode for the pooler. Valid values: ["session", "transaction", "statement"].
- `size` - (Required) Size of the pooler.

**restore**

- `datastore_id` - (Optional) - Datastore ID to restore from.
- `target_time` - (Optional) - Restore by the target time.

## Attributes Reference

The following attributes are exported:

* `status` - Shows the current status of the datastore.

* `connections` - Shows DNS connection strings for the datastore.

## Import

Datastore can be imported using the `id`, e.g.

```shell
$ env SEL_TOKEN=SELECTEL_API_TOKEN SEL_PROJECT_ID=SELECTEL_VPC_PROJECT_ID SEL_REGION=SELECTEL_VPC_REGION terraform import selectel_dbaas_datastore_v1.datastore_1 b311ce58-2658-46b5-b733-7a0f418703f2
```
