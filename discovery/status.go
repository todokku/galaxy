package main

import (
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/litl/galaxy/log"
	"github.com/litl/galaxy/utils"
	"github.com/ryanuber/columnize"
)

func status(c *cli.Context) {

	initOrDie(c)

	containers, err := serviceRuntime.ManagedContainers()
	if err != nil {
		panic(err)
	}

	outputBuffer.Log(strings.Join([]string{
		"CONTAINER ID", "IMAGE",
		"EXTERNAL", "INTERNAL", "CREATED", "EXPIRES",
	}, " | "))

	for _, container := range containers {
		name := serviceRuntime.EnvFor(container)["GALAXY_APP"]
		registered, err := serviceRegistry.GetServiceRegistration(
			utils.GalaxyEnv(c), utils.GalaxyPool(c), container)
		if err != nil {
			log.Printf("ERROR: Unable to determine status of %s: %s\n",
				name, err)
			return
		}

		if registered != nil {
			outputBuffer.Log(strings.Join([]string{
				registered.ContainerID[0:12],
				registered.Image,
				registered.ExternalAddr(),
				registered.InternalAddr(),
				utils.HumanDuration(time.Now().UTC().Sub(registered.StartedAt)) + " ago",
				"In " + utils.HumanDuration(registered.Expires.Sub(time.Now().UTC())),
			}, " | "))

		} else {
			outputBuffer.Log(strings.Join([]string{
				container.ID[0:12],
				container.Image,
				"",
				"",
				utils.HumanDuration(time.Now().Sub(container.Created)) + " ago",
				"",
			}, " | "))
		}

	}

	result, _ := columnize.SimpleFormat(outputBuffer.Output)
	log.Println(result)
}
