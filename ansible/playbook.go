package ansible

import (
	"github.com/alecthomas/kingpin/v2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

type Config struct {
	Hosts string         `yaml:"hosts"`
	Vars  map[string]any `yaml:"vars"`
}

func MakeCLIFromAnsiblePlaybook(configFile string) (string, map[string]any) {
	// Read the YAML configuration file
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var config []Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	// Create the application
	app := kingpin.New("MyApp", "My application description")
	app.Version("1.0.0")

	commandsObj, commandsObjExists := config[0].Vars["commands"]
	globalOptionsObj, globalOptionsObjExists := config[0].Vars["globals"]

	// Adding this for future extensibility
	ExistingCLIArguments := make(map[string]any)

	cliArguments := parseOptions(app, commandsObj, commandsObjExists, ExistingCLIArguments, globalOptionsObjExists, globalOptionsObj)

	cli := kingpin.MustParse(app.Parse(os.Args[1:]))

	return cli, cliArguments
}
