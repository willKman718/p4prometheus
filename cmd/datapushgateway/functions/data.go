package functions

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// var configFile string
var configFile = "mdconfigs/sort.yaml"

// SortConfig represents the structure of the sort.yaml file.
type SortConfig struct {
	FileConfigs []struct {
		FileName    string   `yaml:"file_name"`
		Directory   string   `yaml:"directory"`
		MonitorTags []string `yaml:"monitor_tags"`
	} `yaml:"file_configs"`
}

// CreateMarkdownFiles generates Markdown files based on the grouped data.
func CreateMarkdownFiles(dataDir string, groupedData map[string][]string, sortConfig *SortConfig, logger *logrus.Logger, customer string) error {
	for fileName, items := range groupedData {
		// Find the directory for the current file from the sort.yaml configuration
		var directory string
		for _, fileConfig := range sortConfig.FileConfigs {
			if fileConfig.FileName == fileName {
				directory = fileConfig.Directory
				break
			}
		}

		// If the directory is not found, log an error and continue to the next file
		if directory == "" {
			logger.Errorf("Directory not found for file %s", fileName)
			continue
		}

		// Create the directory if it doesn't exist
		dirPath := filepath.Join(dataDir, customer, directory)
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return fmt.Errorf("error creating directory %s: %v", dirPath, err)
		}

		// Create the Markdown file for the current file
		filePath := filepath.Join(dirPath, fmt.Sprintf("%s.md", fileName))
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("error creating Markdown file %s: %v", filePath, err)
		}
		defer file.Close()

		// Write the Markdown content to the file
		for _, item := range items {
			var itemData map[string]interface{}
			if err := json.Unmarshal([]byte(item), &itemData); err != nil {
				logger.Errorf("Error unmarshaling item data: %v", err)
				continue
			}

			description, _ := itemData["description"].(string)
			output, _ := itemData["output"].(string)
			decodedOutput, err := base64.StdEncoding.DecodeString(output)
			if err != nil {
				logger.Errorf("Error decoding output data: %v", err)
				continue
			}

			_, err = fmt.Fprintf(file, "# %s\n```\n%s\n```\n", description, decodedOutput)
			if err != nil {
				logger.Errorf("Error writing to file %s: %v", filePath, err)
			}
		}
	}
	return nil
}

// LoadSortConfig reads and parses the sort.yaml file and returns the parsed data.
func LoadSortConfig(configFile string) (*SortConfig, error) {
	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read sort.yaml: %v", err)
	}

	var config SortConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse sort.yaml: %v", err)
	}

	return &config, nil
}

// ProcessDataMap is a function to process the JSON data map based on the sort.yaml configuration.
func ProcessDataMap(dataMap map[string]string, configFile, dataDir string, logger *logrus.Logger, customer string) {
	// Load the sort.yaml configuration
	sortConfig, err := LoadSortConfig(configFile)
	if err != nil {
		fmt.Printf("Error loading sort.yaml: %v\n", err)
		return
	}

	// Create a map to group data by monitor tags specified in the sort.yaml
	groupedData := make(map[string][]string)

	// Group the data based on monitor tags
	for key, value := range dataMap {
		// Parse the JSON data for each item
		var itemData map[string]interface{}
		err := json.Unmarshal([]byte(value), &itemData)
		if err != nil {
			fmt.Printf("Error unmarshaling data for key %s: %v\n", key, err)
			continue
		}

		// Extract the monitor tag from the item data
		monitorTag, tagExists := itemData["monitor_tag"].(string)
		if !tagExists {
			fmt.Printf("Missing monitor_tag in item %s\n", key)
			continue
		}

		// Check if the monitor tag is specified in the sort.yaml
		for _, fileConfig := range sortConfig.FileConfigs {
			// Check for each tag in the same order as specified in the sort.yaml
			for _, tag := range fileConfig.MonitorTags {
				if strings.EqualFold(tag, monitorTag) {
					groupedData[fileConfig.FileName] = append(groupedData[fileConfig.FileName], value)
					break
				}
			}
		}
	}

	// Now you can print the grouped data
	Debugf("Printing Grouped Data:")
	for fileName, items := range groupedData {
		Debugf("File Name: %s\n", fileName)
		for _, item := range items {
			Debugf(item)
		}
		Debugf("----")
	}

	// Call the CreateMarkdownFiles function to generate Markdown files
	err = CreateMarkdownFiles(dataDir, groupedData, sortConfig, logger, customer)
	if err != nil {
		fmt.Printf("Error creating Markdown files: %v\n", err)
	}

	// Your specific processing logic goes here.
}

func HandleJSONData(w http.ResponseWriter, req *http.Request, logger *logrus.Logger, dataDir string) {
	// Ensure that the request is a POST request
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, pass, ok := req.BasicAuth()
	if !ok || !VerifyUserPass(user, pass) {
		w.Header().Set("WWW-Authenticate", `Basic realm="api"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	query := req.URL.Query()
	customer := query.Get("customer")
	// Process JSON data
	var jsonData []map[string]interface{}
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&jsonData); err != nil {
		http.Error(w, "Failed to decode JSON data", http.StatusBadRequest)
		return
	}

	// Convert JSON data to a map[string]string for better understanding
	dataMap := make(map[string]string)
	for i, item := range jsonData {
		itemStr, err := json.Marshal(item)
		if err != nil {
			http.Error(w, "Failed to convert JSON data to map", http.StatusInternalServerError)
			return
		}
		dataMap[fmt.Sprintf("Item%d", i+1)] = string(itemStr)
	}

	// Print out the map for better understanding
	logger.Debugf("JSON Data Map:")
	for key, value := range dataMap {
		logger.Debugf("%s: %s", key, value)
	}

	// Call the ProcessDataMap function to work with the data map
	//	ProcessDataMap(dataMap, configFile, dataDir)
	ProcessDataMap(dataMap, configFile, dataDir, logger, customer)
	// Now you can process each object in the JSON array differently.
	//	for _, dataItem := range jsonData {
	//
	//		logger.Debugf("Command: %s", command)
	//		logger.Debugf("Description: %s", description)
	//		logger.Debugf("Output: %s", output)
	//		logger.Debugf("Monitor_Tag: %s", monitorTag)
	//		logger.Debugf("----OUT------")
	//	}

	// Respond with success
	w.Write([]byte("JSON data processed\n"))
}
func SaveData(dataDir, customer, instance, data string, logger *logrus.Logger) error {
	newpath := filepath.Join(dataDir, customer, "servers")
	err := os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		return err
	}
	fname := filepath.Join(newpath, fmt.Sprintf("%s.md", instance))
	f, err := os.Create(fname)
	if err != nil {
		logger.Errorf("Error opening %s: %v", fname, err)
		return err
	}
	f.Write([]byte(data))
	err = f.Close()
	if err != nil {
		logger.Errorf("Error closing file: %v", err)
	}
	return nil
}
func SaveDataV2(dataDir, customer, instance, data string, logger *logrus.Logger) error {
	newpath := filepath.Join(dataDir, customer, "servers")
	err := os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		return err
	}
	fname := filepath.Join(newpath, fmt.Sprintf("%s.md", instance))
	f, err := os.Create(fname)
	if err != nil {
		logger.Errorf("Error opening %s: %v", fname, err)
		return err
	}
	f.Write([]byte(data))
	err = f.Close()
	if err != nil {
		logger.Errorf("Error closing file: %v", err)
	}
	return nil
}
