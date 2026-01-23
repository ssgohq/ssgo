package repl

import (
	"fmt"

	"github.com/manifoldco/promptui"
)

// SelectService prompts user to select a service from the list
func SelectService(services []GrpcService) (*GrpcService, error) {
	if len(services) == 0 {
		return nil, fmt.Errorf("no gRPC services found")
	}

	if len(services) == 1 {
		return &services[0], nil
	}

	// Build selection items
	items := make([]string, len(services))
	for i, svc := range services {
		items[i] = fmt.Sprintf("%s (%s)", svc.Name, svc.Address)
	}

	prompt := promptui.Select{
		Label: "Select gRPC service",
		Items: items,
		Size:  10,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	return &services[idx], nil
}
