package knot

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// RecordType represents DNS record types
type RecordType string

const (
	RecordTypeA     RecordType = "A"
	RecordTypeAAAA  RecordType = "AAAA"
	RecordTypePTR   RecordType = "PTR"
	RecordTypeCNAME RecordType = "CNAME"
	RecordTypeMX    RecordType = "MX"
	RecordTypeTXT   RecordType = "TXT"
	RecordTypeNS    RecordType = "NS"
	RecordTypeSOA   RecordType = "SOA"
)

// DNSRecord represents a DNS record
type DNSRecord struct {
	Name     string     `json:"name" yaml:"name"`
	Type     RecordType `json:"type" yaml:"type"`
	TTL      uint32     `json:"ttl" yaml:"ttl"`
	Data     string     `json:"data" yaml:"data"`
	Priority *uint16    `json:"priority,omitempty" yaml:"priority,omitempty"` // For MX records
}

// Zone represents a DNS zone
type Zone struct {
	Name    string      `json:"name" yaml:"name"`
	Records []DNSRecord `json:"records" yaml:"records"`
}

// CreateRecordRequest represents a request to create a DNS record
type CreateRecordRequest struct {
	Name     string     `json:"name" binding:"required"`
	Type     RecordType `json:"type" binding:"required"`
	TTL      uint32     `json:"ttl"`
	Data     string     `json:"data" binding:"required"`
	Priority *uint16    `json:"priority,omitempty"`
}

// UpdateRecordRequest represents a request to update a DNS record
type UpdateRecordRequest struct {
	TTL      *uint32 `json:"ttl,omitempty"`
	Data     *string `json:"data,omitempty"`
	Priority *uint16 `json:"priority,omitempty"`
}

// ValidRecordTypes returns a list of supported record types
func ValidRecordTypes() []RecordType {
	return []RecordType{
		RecordTypeA,
		RecordTypeAAAA,
		RecordTypePTR,
		RecordTypeCNAME,
		RecordTypeMX,
		RecordTypeTXT,
		RecordTypeNS,
	}
}

// IsValidRecordType checks if a record type is valid
func IsValidRecordType(recordType string) bool {
	rt := RecordType(strings.ToUpper(recordType))
	validTypes := ValidRecordTypes()
	for _, valid := range validTypes {
		if rt == valid {
			return true
		}
	}
	return false
}

// Validate validates a DNS record
func (r *DNSRecord) Validate() error {
	// Validate name
	if r.Name == "" {
		return fmt.Errorf("record name cannot be empty")
	}

	// Validate type
	if !IsValidRecordType(string(r.Type)) {
		return fmt.Errorf("invalid record type: %s", r.Type)
	}

	// Validate TTL
	if r.TTL == 0 {
		r.TTL = 300 // Default TTL
	}

	// Validate data based on record type
	if err := r.validateData(); err != nil {
		return fmt.Errorf("invalid record data: %w", err)
	}

	return nil
}

// validateData validates record data based on type
func (r *DNSRecord) validateData() error {
	switch r.Type {
	case RecordTypeA:
		if net.ParseIP(r.Data) == nil || net.ParseIP(r.Data).To4() == nil {
			return fmt.Errorf("invalid IPv4 address: %s", r.Data)
		}
	case RecordTypeAAAA:
		if net.ParseIP(r.Data) == nil || net.ParseIP(r.Data).To4() != nil {
			return fmt.Errorf("invalid IPv6 address: %s", r.Data)
		}
	case RecordTypePTR, RecordTypeCNAME, RecordTypeNS:
		if r.Data == "" {
			return fmt.Errorf("data cannot be empty for %s record", r.Type)
		}
		// Ensure FQDN ends with dot
		if !strings.HasSuffix(r.Data, ".") {
			r.Data += "."
		}
	case RecordTypeMX:
		if r.Priority == nil {
			return fmt.Errorf("priority is required for MX record")
		}
		if !strings.HasSuffix(r.Data, ".") {
			r.Data += "."
		}
	case RecordTypeTXT:
		if r.Data == "" {
			return fmt.Errorf("data cannot be empty for TXT record")
		}
		// Ensure TXT data is properly quoted
		if !strings.HasPrefix(r.Data, "\"") || !strings.HasSuffix(r.Data, "\"") {
			r.Data = fmt.Sprintf("\"%s\"", strings.Trim(r.Data, "\""))
		}
	}

	return nil
}

// ToKnotFormat converts the record to KnotDNS format
func (r *DNSRecord) ToKnotFormat() string {
	var parts []string

	parts = append(parts, r.Name)
	parts = append(parts, strconv.FormatUint(uint64(r.TTL), 10))
	parts = append(parts, string(r.Type))

	if r.Type == RecordTypeMX && r.Priority != nil {
		parts = append(parts, strconv.FormatUint(uint64(*r.Priority), 10))
	}

	parts = append(parts, r.Data)

	return strings.Join(parts, " ")
}

// ParseKnotRecord parses a record from KnotDNS output format
func ParseKnotRecord(line string) (*DNSRecord, error) {
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid record format: %s", line)
	}

	record := &DNSRecord{
		Name: parts[0],
		Type: RecordType(parts[2]),
	}

	// Parse TTL
	ttl, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid TTL: %s", parts[1])
	}
	record.TTL = uint32(ttl)

	// Handle different record types
	switch record.Type {
	case RecordTypeMX:
		if len(parts) < 5 {
			return nil, fmt.Errorf("invalid MX record format: %s", line)
		}
		priority, err := strconv.ParseUint(parts[3], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid MX priority: %s", parts[3])
		}
		p := uint16(priority)
		record.Priority = &p
		record.Data = parts[4]
	default:
		record.Data = strings.Join(parts[3:], " ")
	}

	return record, nil
}

// Validate validates a create record request
func (r *CreateRecordRequest) Validate() error {
	record := &DNSRecord{
		Name:     r.Name,
		Type:     r.Type,
		TTL:      r.TTL,
		Data:     r.Data,
		Priority: r.Priority,
	}
	return record.Validate()
}

// ToRecord converts CreateRecordRequest to DNSRecord
func (r *CreateRecordRequest) ToRecord() *DNSRecord {
	return &DNSRecord{
		Name:     r.Name,
		Type:     r.Type,
		TTL:      r.TTL,
		Data:     r.Data,
		Priority: r.Priority,
	}
}
