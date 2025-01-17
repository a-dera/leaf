package sources

import (
	"context"
	"fmt"
	"leaf/utils"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func generateConfig(source, includeContainer, project, feed, token string) string {
	return fmt.Sprintf(`
%s
include_containers = [ "%s" ]

%s
`, strings.TrimSpace(source), includeContainer, utils.DefaultConfig(project, feed, token))
}

func DockerLogs(feed utils.Feed, project utils.Project) error {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return utils.ParsedError(err, "Failed to connect to Docker daemon", true)
	}

	containers, err := client.ContainerList(context.Background(), types.ContainerListOptions{})

	if err != nil {
		return utils.ParsedError(err, "Failed to list containers", true)
	}

	var containerNames = []string{}
	for _, container := range containers {
		containerNames = append(containerNames, fmt.Sprintf("%s (%s)", container.Names[0][1:], container.ID[:12]))
	}

	var selectedContainer string
	survey.AskOne(&survey.Select{
		Message: "Select a container:",
		Options: containerNames,
	}, &selectedContainer, survey.WithValidator(survey.Required))

	selectedContainer = strings.Split(selectedContainer, " ")[0]

	cmd := exec.Command("vector", "generate", "docker_logs")
	vectorGeneratedConfig, err := cmd.Output()
	if err != nil {
		return utils.ParsedError(err, "Failed to generate config", true)
	}

	stringedVectorGeneratedConfig := string(vectorGeneratedConfig[:])

	lines := strings.Split(stringedVectorGeneratedConfig, "\n")
	lines = lines[2:] // remove first 2 lines of generatedConfig (irrelevant to config)

	stringedVectorGeneratedConfig = strings.Join(lines, "\n")

	state, err := utils.GetState()
	if err != nil {
		return utils.ParsedError(err, "Failed to get state", true)
	}

	config := generateConfig(stringedVectorGeneratedConfig, selectedContainer, project.Namespace, feed.Name, state.Token)

	finalPath, err := utils.WriteToPath(fmt.Sprintf("configs/%s-%s.toml", feed.Name, selectedContainer), config)
	if err != nil {
		return utils.ParsedError(err, "Failed to write config", true)
	}

	fmt.Println("Config generated and saved to", finalPath)

	return nil
}
