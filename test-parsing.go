package main

import (
	"fmt"
	"github.com/hypr-technologies/hyprknot/internal/knot"
)

func main() {
	// Test parsing the problematic records from your log
	testRecords := []string{
		"[143.31.194.in-addr.arpa.] 143.31.194.in-addr.arpa. 900 NS dash.hypr.tech.",
		"[143.31.194.in-addr.arpa.] 143.31.194.in-addr.arpa. 900 SOA dns.hypr.tech. admin.hypr.tech. 2024052701 900 300 604800 900",
		"[143.31.194.in-addr.arpa.] 0.143.31.194.in-addr.arpa. 900 PTR hypr.tech.",
		"[143.31.194.in-addr.arpa.] 210.143.31.194.in-addr.arpa. 900 PTR mail.hypr.tech.",
		"[143.31.194.in-addr.arpa.] 221.143.31.194.in-addr.arpa. 900 PTR dns.hypr.tech.",
	}

	fmt.Println("Testing KnotDNS record parsing...")
	fmt.Println("================================")

	for i, line := range testRecords {
		fmt.Printf("\nTest %d: %s\n", i+1, line)
		
		record, err := knot.ParseKnotRecord(line)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Printf("✅ Parsed successfully:\n")
			fmt.Printf("   Name: %s\n", record.Name)
			fmt.Printf("   Type: %s\n", record.Type)
			fmt.Printf("   TTL: %d\n", record.TTL)
			fmt.Printf("   Data: %s\n", record.Data)
			
			// Test round-trip
			formatted := record.ToKnotFormat()
			fmt.Printf("   Formatted: %s\n", formatted)
		}
	}

	// Test creating a new PTR record
	fmt.Println("\n\nTesting PTR record creation...")
	fmt.Println("==============================")
	
	newRecord := &knot.DNSRecord{
		Name: "100.143.31.194.in-addr.arpa.",
		Type: knot.RecordTypePTR,
		TTL:  900,
		Data: "test.hypr.tech.",
	}

	if err := newRecord.Validate(); err != nil {
		fmt.Printf("❌ Validation error: %v\n", err)
	} else {
		fmt.Printf("✅ New PTR record valid:\n")
		fmt.Printf("   Formatted: %s\n", newRecord.ToKnotFormat())
	}
}
