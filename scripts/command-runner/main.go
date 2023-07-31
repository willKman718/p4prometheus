package main

/*
//TODO CLEANING CLEANING CLEANING CLEANING CLEANING CLEANING CLEANING CLEANING CLEANING, and some more CLEANING
//TODO LOGGING
//TODO AZURE
//TODO Fix directory paths they're currently super ugly but hey this is the uglystick
//TODO file parser  --- needs fix for -server flag but can be used just the same may not even need a -server flag period come to think of it
//TODO SWARM
//TODO HAS
//TODO Error parsing files at the instance level breaks it so if it cant parse the file its looking for its stops.
*/

import (
	"command-runner/tools"
	"flag"
	"fmt"
	"os"
)

var (
	outputJSONFilePath   string
	yamlCommandsFilePath string
	cloudProvider        string
)

func init() {
	flag.StringVar(&outputJSONFilePath, "output", "out.json", "Path to the output JSON file")
	flag.StringVar(&yamlCommandsFilePath, "comyaml", "commands.yaml", "Path to the YAML file containing shell commands")
	flag.StringVar(&cloudProvider, "cloud", "", "Cloud provider (aws, gcp, or azure)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
}

func main() {
	var instanceArg string
	var serverArg bool

	flag.StringVar(&instanceArg, "instance", "", "Instance argument for the command-runner")
	flag.BoolVar(&serverArg, "server", false, "Server argument for the command-runner")

	flag.Parse()

	// If -cloud is provided, check if it's a valid cloud provider
	if cloudProvider != "" {
		switch cloudProvider {
		case "aws":
			//Logic to handle AWS-related functionality
			err := tools.GetAWSInstanceIdentityInfo(outputJSONFilePath)
			if err != nil {
				fmt.Println("Error getting AWS instance identity info:", err)
				os.Exit(1)
			}
		case "gcp":
			//Logic to handle GCP-related functionality
			err := tools.GetGCPInstanceIdentityInfo(outputJSONFilePath)
			if err != nil {
				fmt.Println("Error getting GCP instance identity info:", err)
				os.Exit(1)
			}
		case "azure":
			// Add logic to handle Azure-related functionality
		default:
			fmt.Println("Error: Invalid cloud provider. Please specify aws, gcp, or azure.")
			os.Exit(1)
		}
	}

	// Check if the server argument is provided
	if serverArg {
		fmt.Println("Server ARG passed")
		// Execute and encode server commands
		serverCommands, err := tools.ReadServerCommandsFromYAML(yamlCommandsFilePath)
		if err != nil {
			fmt.Println("Error reading server commands from YAML:", err)
			os.Exit(1)
		}

		base64ServerOutputs, err := tools.ExecuteAndEncodeCommands(serverCommands)
		if err != nil {
			fmt.Println("Error executing server commands:", err)
			os.Exit(1)
		}

		// Create JSON data for server commands
		var serverJSONData []tools.JSONData
		for i, cmd := range serverCommands {
			serverJSONData = append(serverJSONData, tools.JSONData{
				Command:     cmd.Description,
				Description: cmd.Description,
				Output:      base64ServerOutputs[i],
				MonitorTag:  cmd.MonitorTag,
			})
		}

		// Get the existing JSON data from the file (if it exists)
		existingJSONData, err := tools.ReadJSONFromFile(outputJSONFilePath)
		if err != nil && !os.IsNotExist(err) {
			fmt.Printf("Error reading existing JSON data from %s: %s\n", outputJSONFilePath, err)
			os.Exit(1)
		}

		// Append server JSON data to existing data
		allJSONData := append(existingJSONData, serverJSONData...)
		err = tools.AppendParsedDataToFile(serverJSONData, outputJSONFilePath)
		if err != nil {
			fmt.Printf("Error appending server JSON data to %s: %s\n", outputJSONFilePath, err)
			os.Exit(1)
		}
		// Write the updated JSON data back to the file
		if err := tools.WriteJSONToFile(allJSONData, outputJSONFilePath); err != nil {
			fmt.Printf("Error writing server JSON data to %s: %s\n", outputJSONFilePath, err)
			os.Exit(1)
		}

		// Get the hostname of the server
		hostname, err := os.Hostname()
		if err != nil {
			fmt.Println("Error getting hostname:", err)
			os.Exit(1)
		}
		// File parsing for the server level
		err = tools.FileParserFromYAMLConfigServer("/home/perforce/workspace/command-runner/configs/fileparser.yaml", outputJSONFilePath)
		if err != nil {
			fmt.Println("Error parsing files at the server level:", err)
			os.Exit(1)
		}
		fmt.Printf("%s Server commands executed and output appended to %s.\n", hostname, outputJSONFilePath)
	}

	// Check if the instance argument is provided
	if instanceArg != "" {
		fmt.Println("Instance ARG passed")

		instanceCommands, err := tools.ReadInstanceCommandsFromYAML(yamlCommandsFilePath)
		if err != nil {
			fmt.Println("Error reading instance commands from YAML:", err)
			os.Exit(1)
		}

		base64InstanceOutputs, err := tools.ExecuteAndEncodeCommands(instanceCommands)
		if err != nil {
			fmt.Println("Error executing instance commands:", err)
			os.Exit(1)
		}

		// Create JSON data for instance commands
		var instanceJSONData []tools.JSONData
		for i, cmd := range instanceCommands {
			instanceJSONData = append(instanceJSONData, tools.JSONData{
				Command:     cmd.Description,
				Description: cmd.Description,
				Output:      base64InstanceOutputs[i],
				MonitorTag:  cmd.MonitorTag,
			})
		}

		// Get the existing JSON data from the file (if it exists)
		existingJSONData, err := tools.ReadJSONFromFile(outputJSONFilePath)
		if err != nil && !os.IsNotExist(err) {
			fmt.Printf("Error reading existing JSON data from %s: %s\n", outputJSONFilePath, err)
			os.Exit(1)
		}

		// Append instance JSON data to existing data
		allJSONData := append(existingJSONData, instanceJSONData...)

		// Write the updated JSON data back to the file
		if err := tools.WriteJSONToFile(allJSONData, outputJSONFilePath); err != nil {
			fmt.Printf("Error writing instance JSON data to %s: %s\n", outputJSONFilePath, err)
			os.Exit(1)
		}
		// File parsing for the instance level
		err = tools.FileParserFromYAMLConfigInstance("/home/perforce/workspace/command-runner/configs/fileparser.yaml", outputJSONFilePath, instanceArg)
		if err != nil {
			fmt.Println("Error parsing files at the instance level:", err)
			os.Exit(1)
		}
		fmt.Printf("Instance %s commands executed and output appended to %s.\n", instanceArg, outputJSONFilePath)
	}

	if flag.NFlag() == 0 {
		flag.Usage()
	}
}