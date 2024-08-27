# Overview

This is a Go library for defining CLI interfaces using yaml files.

The underlying CLI library is [alecthomas/kingpin](https://github.com/alecthomas/kingpin).

# Features

So far, this library supports creating a cli using an ansible playbook with specially formatted vars.

# Usage

```go
package main

import (
  "fmt"
  "github.com/berttejeda/bert.yamlcli/ansible"
  logger "github.com/sirupsen/logrus"
)

func main() {
  cli, _ := ansible.MakeCLIFromAnsiblePlaybook("Taskfile.yaml")
  fmt.Println(cli)
}

```

# Examples

Consult the [examples](examples) directory.

