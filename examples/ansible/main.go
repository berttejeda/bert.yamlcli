package main

import (
	"fmt"
	"github.com/berttejeda/yamlcli/ansible"
	"os"
)

func main() {
	// First positional parameter is the path to the playbook
	yamlfile := os.Args[1]
  // Remove the first positional parameter by re-slicing os.Args
  os.Args = append(os.Args[:1], os.Args[2:]...)	
	cli, cliArguments := ansible.MakeCLIFromAnsiblePlaybook(yamlfile)
	fmt.Println(cliArguments)
	fmt.Println(cli)
}
