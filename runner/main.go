package main

import (
	"fmt"
	"log"
	"os/exec"
)

type PhaseConfig struct {
	Name   string
	Script string
}

func executePhase(phase PhaseConfig) error {
	fmt.Printf("Executing phase: %s\n", phase.Name)
	cmd := exec.Command("sh", "-c", phase.Script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute phase %s: %s, error: %v", phase.Name, string(output), err)
	}
	return nil
}

func handlePhases(phases []PhaseConfig) error {
	for _, phase := range phases {
		err := executePhase(phase)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	// Define the phases
	phases := []PhaseConfig{
		{Name: "Download Source", Script: "echo 'Downloading source'"},
		{Name: "Environment", Script: "echo 'Setting up environment'"},
		{Name: "Install", Script: "echo 'Installing dependencies'"},
		{Name: "Build", Script: "echo 'Building the project'"},
		{Name: "Upload artifacts", Script: "echo 'Uploading artifacts'"},
		{Name: "Finalize", Script: "echo 'Finalizing the process'"},
	}

	// Execute the phases
	err := handlePhases(phases)
	if err != nil {
		log.Fatalf("Error executing phases: %v", err)
	}

	fmt.Println("All phases executed successfully.")
}
