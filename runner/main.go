package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
)

// Phases configs
type PhasesConfig struct {
	BuildPhase
	ArtifactsPhase
	FinalizePhase
}

type BuildPhase struct {
	Build []string `json:"build"`
}

type ArtifacsNestedPhase struct {
	LocalTargetDir  string `json:"local_target_dir"`
	RemoteTargetDir string `json:"remote_target_dir"`
}
type ArtifactsPhase struct {
	Artifacts ArtifacsNestedPhase `json:"artifacts"`
}

type FinalizePhase struct {
	Finalize []string `json:"finalize"`
}

// Pod Specific Configs
type PodConfig struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type JsonConfigStruct struct {
	PodCfg PodConfig    `json:"pod"`
	Phases PhasesConfig `json:"phases"`
}

func executeBuildPhase(buildCommands []string) error {
	log.Println("Starting execution in Phase: Build")
	for _, cmd := range buildCommands {
		log.Println("Execution Command: ", cmd)
		_cmd := exec.Command("sh", "-c", cmd)
		tOutput, err := _cmd.CombinedOutput()
		if err != nil {
			return err
		}
		log.Println(tOutput)
	}
	return nil

}

func executeArtifactsPhase(local_target_dir string, remote_target_dir string) error {
	log.Println("Starting execution in Phase: Upload Artifacts")
	log.Println("Uploading files in ", local_target_dir, "to ", remote_target_dir)
	return nil
}

func executeFinalizePhase(finalizeCommands []string) error {
	log.Println("Starting execution in Phase: Finalize")
	for _, cmd := range finalizeCommands {
		log.Println("Execution Command: ", cmd)
		_cmd := exec.Command("sh", "-c", cmd)
		tOutput, err := _cmd.CombinedOutput()
		if err != nil {
			return err
		}
		log.Println(tOutput)
	}
	return nil

}

func handlePhases(phases PhasesConfig) error {

	// Build Phase
	err := executeBuildPhase(phases.Build)
	if err != nil {
		return err
	}

	// Artifacs Phase
	err = executeArtifactsPhase(phases.Artifacts.LocalTargetDir, phases.Artifacts.RemoteTargetDir)
	if err != nil {
		return err
	}

	// Finalize Phase
	err = executeFinalizePhase(phases.Finalize)
	if err != nil {
		return err
	}
	return nil
}

func readConfigJson(filePath string) (PhasesConfig, error) {
	// Read the file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println(err)
		return PhasesConfig{}, err
	}
	var jsonconfig PhasesConfig
	err = json.Unmarshal(data, &jsonconfig)

	if err != nil {
		fmt.Println(err)
		return PhasesConfig{}, err
	}
	return jsonconfig, nil

}

func main() {
	configJson, err := readConfigJson("/config/config.json")

	fmt.Println(configJson)
	// Execute the phases
	err = handlePhases(configJson)
	if err != nil {
		log.Fatalf("Error executing phases: %v", err)
	}

	fmt.Println("All phases executed successfully.")
}
