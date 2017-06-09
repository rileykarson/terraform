---
layout: "google"
page_title: "Google: google_bigtable_table"
sidebar_current: "docs-google-bigtable_table"
description: |-
  Creates a column family inside a Bigtable table.
---

# google_bigtable_family

Creates a column family inside a Bigtable table. For more information see
[the official documentation](https://cloud.google.com/bigtable/) and
[API](https://cloud.google.com/bigtable/docs/go/reference).


## Example Usage

```hcl
resource "google_bigtable_instance" "instance" {
  name     = "tf-instance"
  cluster_id = "tf-instance-cluster"
  zone = "us-central1-b"
  num_nodes = 3
  storage_type = "HDD"
}

resource "google_bigtable_table" "table" {
  name     = "tf-table"
  instance_name = "${google_bigtable_instance.instance.name}"
  split_keys = ["a", "b", "c"]
}

resource "google_bigtable_family" "family" {
  name     = "tf-family"
  instance_name = "${google_bigtable_instance.instance.name}"
  table_name = "${google_bigtable_table.table.name}"
  version_policy = 10
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the column family.

* `instance_name` - (Required) The name of the Bigtable instance.

* `table_name` - (Required) The name of the Bigtable table inside the instance.

* `version_policy` - (Required) The maximum number of versions held before GC. Must be at least 1.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

## Attributes Reference


In addition to the arguments listed above, the following computed attributes are
exported:

* `gc_policy` - A human-readable string representation of the family's GCPolicy.