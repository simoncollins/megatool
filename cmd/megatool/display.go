package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/megatool/internal/utils"
)

// listAvailableServers prints a list of available MCP servers with optional indentation
func listAvailableServers(indent string) error {
	// Get available servers
	servers, err := utils.GetAvailableServers()
	if err != nil {
		return err
	}

	// Check if any servers were found
	if len(servers) == 0 {
		fmt.Println(indent + "No MCP servers available")
		return nil
	}

	// Print each server on a separate line
	for _, server := range servers {
		fmt.Println(indent + server)
	}

	return nil
}

// displayServerRecords formats and displays server records
func displayServerRecords(records []utils.ServerRecord, format, fields string, showHeader bool) error {
	// Split requested fields
	fieldList := strings.Split(fields, ",")
	
	// Check if we have any records
	if len(records) == 0 {
		fmt.Println("No running MCP servers found")
		return nil
	}
	
	// Generate output based on format
	switch format {
	case "table":
		return displayServerTable(records, fieldList, showHeader)
	case "json":
		return displayServerJSON(records, fieldList)
	case "csv":
		return displayServerCSV(records, fieldList, showHeader)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

// displayServerTable displays server records in a table format
func displayServerTable(records []utils.ServerRecord, fields []string, showHeader bool) error {
	// Create a new tabwriter
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	
	// Print header if requested
	if showHeader {
		var headers []string
		for _, field := range fields {
			switch field {
			case "name":
				headers = append(headers, "NAME")
			case "pid":
				headers = append(headers, "PID")
			case "uptime":
				headers = append(headers, "UPTIME")
			default:
				headers = append(headers, strings.ToUpper(field))
			}
		}
		fmt.Fprintln(w, strings.Join(headers, "\t"))
	}
	
	// Count instances of each server
	serverCounts := make(map[string]int)
	for _, record := range records {
		serverCounts[record.Name]++
	}
	
	// Print each record
	for _, record := range records {
		var values []string
		for _, field := range fields {
			switch field {
			case "name":
				// Add instance indicator if there are multiple instances
				if serverCounts[record.Name] > 1 {
					values = append(values, fmt.Sprintf("%s (instance %d of %d)", 
						record.Name, 
						getInstanceNumber(records, record),
						serverCounts[record.Name]))
				} else {
					values = append(values, record.Name)
				}
			case "pid":
				values = append(values, fmt.Sprintf("%d", record.PID))
			case "uptime":
				values = append(values, utils.FormatUptime(record.StartTime))
			default:
				values = append(values, "N/A")
			}
		}
		fmt.Fprintln(w, strings.Join(values, "\t"))
	}
	
	return w.Flush()
}

// getInstanceNumber returns the instance number of a record among records with the same name
// sorted by start time (oldest first)
func getInstanceNumber(records []utils.ServerRecord, target utils.ServerRecord) int {
	// Filter records with the same name
	var sameNameRecords []utils.ServerRecord
	for _, record := range records {
		if record.Name == target.Name {
			sameNameRecords = append(sameNameRecords, record)
		}
	}
	
	// Sort by start time (oldest first)
	sort.Slice(sameNameRecords, func(i, j int) bool {
		return sameNameRecords[i].StartTime.Before(sameNameRecords[j].StartTime)
	})
	
	// Find the index of the target record
	for i, record := range sameNameRecords {
		if record.PID == target.PID {
			return i + 1
		}
	}
	
	return 0 // Should not happen
}

// displayServerJSON displays server records in JSON format
func displayServerJSON(records []utils.ServerRecord, fields []string) error {
	// Create a slice of maps for JSON output
	var result []map[string]interface{}
	
	// Count instances of each server
	serverCounts := make(map[string]int)
	for _, record := range records {
		serverCounts[record.Name]++
	}
	
	for _, record := range records {
		item := make(map[string]interface{})
		
		for _, field := range fields {
			switch field {
			case "name":
				// Add instance information if there are multiple instances
				if serverCounts[record.Name] > 1 {
					item["name"] = record.Name
					item["instance_number"] = getInstanceNumber(records, record)
					item["total_instances"] = serverCounts[record.Name]
				} else {
					item["name"] = record.Name
				}
			case "pid":
				item["pid"] = record.PID
			case "uptime":
				item["uptime"] = utils.FormatUptime(record.StartTime)
			}
		}
		
		result = append(result, item)
	}
	
	// Marshal to JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	
	fmt.Println(string(jsonData))
	return nil
}

// displayServerCSV displays server records in CSV format
func displayServerCSV(records []utils.ServerRecord, fields []string, showHeader bool) error {
	// Create a new CSV writer
	w := csv.NewWriter(os.Stdout)
	
	// Print header if requested
	if showHeader {
		var headers []string
		for _, field := range fields {
			switch field {
			case "name":
				headers = append(headers, "NAME")
			case "pid":
				headers = append(headers, "PID")
			case "uptime":
				headers = append(headers, "UPTIME")
			default:
				headers = append(headers, strings.ToUpper(field))
			}
		}
		if err := w.Write(headers); err != nil {
			return err
		}
	}
	
	// Count instances of each server
	serverCounts := make(map[string]int)
	for _, record := range records {
		serverCounts[record.Name]++
	}
	
	// Print each record
	for _, record := range records {
		var values []string
		for _, field := range fields {
			switch field {
			case "name":
				// Add instance indicator if there are multiple instances
				if serverCounts[record.Name] > 1 {
					values = append(values, fmt.Sprintf("%s (instance %d of %d)", 
						record.Name, 
						getInstanceNumber(records, record),
						serverCounts[record.Name]))
				} else {
					values = append(values, record.Name)
				}
			case "pid":
				values = append(values, fmt.Sprintf("%d", record.PID))
			case "uptime":
				values = append(values, utils.FormatUptime(record.StartTime))
			default:
				values = append(values, "N/A")
			}
		}
		if err := w.Write(values); err != nil {
			return err
		}
	}
	
	w.Flush()
	return w.Error()
}
