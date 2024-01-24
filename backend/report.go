package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

type SystemReport struct {
	CPUInfo map[string]string `json:"cpu_info"`
	OS      map[string]string `json:"os"`
	MEMInfo map[string]string `json:"mem_info"`
	Env     map[string]string `json:"environment_variables"`
}

func report(out_file string, stdout bool) {
	// Get CPU Info
	cpuInfo, err := os.Open("/proc/cpuinfo")
	if err != nil {
		fmt.Println("Error opening /proc/cpuinfo:", err)
		return
	}
	defer cpuInfo.Close()

	scanner := bufio.NewScanner(cpuInfo)
	cpuMap := make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			cpuMap[key] = val
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading /proc/cpuinfo:", err)
		return
	}

	// Get OS Info
	osInfo, err := os.Open("/etc/os-release")
	if err != nil {
		fmt.Println("Error opening /etc/os-release:", err)
		return
	}
	defer osInfo.Close()

	scanner = bufio.NewScanner(osInfo)
	osMap := make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			val := strings.Trim(parts[1], `"`)
			osMap[key] = val
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading /etc/os-release:", err)
		return
	}

	// Get mem info
	memInfo, err := os.Open("/proc/meminfo")
	if err != nil {
		fmt.Println("Error opening /proc/meminfo:", err)
		return
	}
	defer cpuInfo.Close()

	scanner = bufio.NewScanner(memInfo)
	memMap := make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			memMap[key] = val
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading /proc/meminfo:", err)
		return
	}

	// Get env
	envMap := make(map[string]string)
	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			val := parts[1]
			envMap[key] = val
		}
	}

	// Create a new system report
	report := &SystemReport{
		CPUInfo: cpuMap,
		OS:      osMap,
		MEMInfo: memMap,
		Env:     envMap,
	}

	// Marshal the report to JSON
	jsonData, err := json.MarshalIndent(report, "", " ")
	if err != nil {
		fmt.Println("Error marshalling report to JSON:", err)
		return
	}

	if stdout {
		fmt.Print(string(jsonData))
		return
	} else {
		// Write the JSON data to the output file
		err = os.WriteFile(out_file, jsonData, fs.ModePerm)
		if err != nil {
			fmt.Println("Error writing report to file:", err)
			return
		}

		fmt.Println("System report completed.")
	}
}
