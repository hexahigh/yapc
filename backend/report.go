package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type SystemReport struct {
	CPUInfo string `json:"cpu_info"`
	OS      string `json:"os"`
	MEMInfo string `json:"mem_info"`
	LSBLK   string `json:"lsblk"`
	DF      string `json:"df"`
}

func report(out_file string) {
	fmt.Println("Starting system report")

	// Get CPU Info
	cpuInfo, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		fmt.Println("Error reading /proc/cpuinfo:", err)
		return
	}

	// Get OS Info
	osInfo, err := os.ReadFile("/etc/os-release")
	if err != nil {
		osInfo = []byte(runtime.GOOS)
		if err != nil {
			osInfo = []byte("failed")
		}
	}

	// Get mem info
	memInfo, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		fmt.Println("Error reading /proc/meminfo:", err)
		return
	}

	// Get lsblk
	lsblkCmd := exec.Command("lsblk")
	lsblkOutput, err := lsblkCmd.Output()
	if err != nil {
		fmt.Println("Error running lsblk:", err)
		return
	}
	lsblk := string(lsblkOutput)

	// Get df
	dfCmd := exec.Command("df", "-h")
	dfOutput, err := dfCmd.Output()
	if err != nil {
		fmt.Println("Error running df:", err)
		return
	}
	df := string(dfOutput)

	// Create a new system report
	report := &SystemReport{
		CPUInfo: strings.TrimSpace(string(cpuInfo)),
		OS:      strings.TrimSpace(string(osInfo)),
		MEMInfo: strings.TrimSpace(string(memInfo)),
		LSBLK:   strings.TrimSpace(lsblk),
		DF:      strings.TrimSpace(df),
	}

	// Marshal the report to JSON
	jsonData, err := json.MarshalIndent(report, "", " ")
	if err != nil {
		fmt.Println("Error marshalling report to JSON:", err)
		return
	}

	// Write the JSON data to the output file
	err = os.WriteFile(out_file, jsonData, fs.ModePerm)
	if err != nil {
		fmt.Println("Error writing report to file:", err)
		return
	}

	fmt.Println("System report completed successfully.")
}
