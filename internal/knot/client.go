package knot

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Client represents a KnotDNS client
type Client struct {
	knotcPath    string
	socketPath   string
	allowedZones []string
	logger       *logrus.Logger
}

// NewClient creates a new KnotDNS client
func NewClient(knotcPath, socketPath string, allowedZones []string, logger *logrus.Logger) *Client {
	return &Client{
		knotcPath:    knotcPath,
		socketPath:   socketPath,
		allowedZones: allowedZones,
		logger:       logger,
	}
}

// normalizeZoneName ensures zone name has proper DNS format
func normalizeZoneName(zone string) string {
	if !strings.HasSuffix(zone, ".") {
		return zone + "."
	}
	return zone
}

// IsZoneAllowed checks if a zone is in the allowed zones list
func (c *Client) IsZoneAllowed(zone string) bool {
	if len(c.allowedZones) == 0 {
		return true // If no restrictions, allow all zones
	}

	// Normalize the zone name to canonical form
	normalizedZone := normalizeZoneName(zone)

	for _, allowedZone := range c.allowedZones {
		normalizedAllowed := normalizeZoneName(allowedZone)
		if normalizedZone == normalizedAllowed || strings.HasSuffix(normalizedZone, "."+normalizedAllowed) {
			return true
		}
	}
	return false
}

// executeKnotc executes a knotc command
func (c *Client) executeKnotc(args ...string) (string, error) {
	cmdArgs := []string{}
	if c.socketPath != "" {
		cmdArgs = append(cmdArgs, "-s", c.socketPath)
	}
	cmdArgs = append(cmdArgs, args...)

	c.logger.Debugf("Executing knotc command: %s %v", c.knotcPath, cmdArgs)

	cmd := exec.Command(c.knotcPath, cmdArgs...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		c.logger.Errorf("knotc command failed: %v, output: %s", err, string(output))
		return "", fmt.Errorf("knotc command failed: %w, output: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	c.logger.Debugf("knotc command output: %s", result)

	return result, nil
}

// GetZones returns a list of configured zones
func (c *Client) GetZones() ([]string, error) {
	output, err := c.executeKnotc("conf-read", "zone")
	if err != nil {
		return nil, fmt.Errorf("failed to get zones: %w", err)
	}

	var zones []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "zone[") {
			// Extract zone name from zone[example.com]
			start := strings.Index(line, "[")
			end := strings.Index(line, "]")
			if start != -1 && end != -1 && end > start {
				zoneName := line[start+1 : end]
				if c.IsZoneAllowed(zoneName) {
					zones = append(zones, zoneName)
				}
			}
		}
	}

	return zones, nil
}

// GetRecords returns all records for a zone
func (c *Client) GetRecords(zone string) ([]DNSRecord, error) {
	if !c.IsZoneAllowed(zone) {
		return nil, fmt.Errorf("zone not allowed: %s", zone)
	}

	// Use normalized zone name for KnotDNS commands
	normalizedZone := normalizeZoneName(zone)
	output, err := c.executeKnotc("zone-read", normalizedZone)
	if err != nil {
		return nil, fmt.Errorf("failed to read zone %s: %w", zone, err)
	}

	var records []DNSRecord
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		record, err := ParseKnotRecord(line)
		if err != nil {
			c.logger.Warnf("Failed to parse record: %s, error: %v", line, err)
			continue
		}

		records = append(records, *record)
	}

	return records, nil
}

// GetRecord returns a specific record
func (c *Client) GetRecord(zone, name string, recordType RecordType) (*DNSRecord, error) {
	if !c.IsZoneAllowed(zone) {
		return nil, fmt.Errorf("zone not allowed: %s", zone)
	}

	records, err := c.GetRecords(zone)
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		if record.Name == name && record.Type == recordType {
			return &record, nil
		}
	}

	return nil, fmt.Errorf("record not found: %s %s in zone %s", name, recordType, zone)
}

// CreateRecord creates a new DNS record (idempotent - replaces existing record)
func (c *Client) CreateRecord(zone string, record *DNSRecord) error {
	if !c.IsZoneAllowed(zone) {
		return fmt.Errorf("zone not allowed: %s", zone)
	}

	if err := record.Validate(); err != nil {
		return fmt.Errorf("invalid record: %w", err)
	}

	// Check if record already exists
	existingRecord, err := c.GetRecord(zone, record.Name, record.Type)
	if err == nil {
		// Record exists, check if it's identical
		if existingRecord.TTL == record.TTL &&
			existingRecord.Data == record.Data &&
			((existingRecord.Priority == nil && record.Priority == nil) ||
				(existingRecord.Priority != nil && record.Priority != nil && *existingRecord.Priority == *record.Priority)) {
			c.logger.Infof("Record already exists with same values: %s %s in zone %s", record.Name, record.Type, zone)
			return nil // Idempotent - record already exists with same values
		}
	}

	// Use normalized zone name for KnotDNS commands
	normalizedZone := normalizeZoneName(zone)

	// Begin transaction
	if _, err := c.executeKnotc("zone-begin", normalizedZone); err != nil {
		return fmt.Errorf("failed to begin transaction for zone %s: %w", zone, err)
	}

	// Add/replace record (zone-set replaces existing records)
	// KnotDNS zone-set expects: zone-set <zone> <owner> <ttl> <type> <rdata>
	// For relative names within the zone, we need to remove the zone suffix
	recordName := record.Name
	if strings.HasSuffix(recordName, "."+normalizedZone) {
		// Convert absolute name to relative by removing zone suffix
		recordName = strings.TrimSuffix(recordName, "."+normalizedZone)
	} else if strings.HasSuffix(recordName, normalizedZone) {
		// Handle case where zone doesn't have trailing dot in record name
		recordName = strings.TrimSuffix(recordName, normalizedZone)
		recordName = strings.TrimSuffix(recordName, ".")
	}

	args := []string{"zone-set", normalizedZone, recordName,
		strconv.FormatUint(uint64(record.TTL), 10), string(record.Type)}

	// Add priority for MX records
	if record.Type == RecordTypeMX && record.Priority != nil {
		args = append(args, strconv.FormatUint(uint64(*record.Priority), 10))
	}

	// Add the record data
	args = append(args, record.Data)

	if _, err := c.executeKnotc(args...); err != nil {
		// Abort transaction on error
		c.executeKnotc("zone-abort", normalizedZone)
		return fmt.Errorf("failed to add record to zone %s: %w", zone, err)
	}

	// Commit transaction
	if _, err := c.executeKnotc("zone-commit", normalizedZone); err != nil {
		return fmt.Errorf("failed to commit transaction for zone %s: %w", zone, err)
	}

	if existingRecord != nil {
		c.logger.Infof("Replaced existing record: %s %s in zone %s", record.Name, record.Type, zone)
	} else {
		c.logger.Infof("Created record: %s %s in zone %s", record.Name, record.Type, zone)
	}
	return nil
}

// UpdateRecord updates an existing DNS record
func (c *Client) UpdateRecord(zone, name string, recordType RecordType, updates *UpdateRecordRequest) error {
	if !c.IsZoneAllowed(zone) {
		return fmt.Errorf("zone not allowed: %s", zone)
	}

	// Get existing record
	existingRecord, err := c.GetRecord(zone, name, recordType)
	if err != nil {
		return fmt.Errorf("record not found: %w", err)
	}

	// Store original record for precise removal
	originalRecordStr := existingRecord.ToKnotFormat()

	// Apply updates
	if updates.TTL != nil {
		existingRecord.TTL = *updates.TTL
	}
	if updates.Data != nil {
		existingRecord.Data = *updates.Data
	}
	if updates.Priority != nil {
		existingRecord.Priority = updates.Priority
	}

	// Validate updated record
	if err := existingRecord.Validate(); err != nil {
		return fmt.Errorf("invalid updated record: %w", err)
	}

	// Use normalized zone name for KnotDNS commands
	normalizedZone := normalizeZoneName(zone)

	// Begin transaction
	if _, err := c.executeKnotc("zone-begin", normalizedZone); err != nil {
		return fmt.Errorf("failed to begin transaction for zone %s: %w", zone, err)
	}

	// Remove old record using full record string for precision
	if _, err := c.executeKnotc("zone-unset", normalizedZone, originalRecordStr); err != nil {
		c.executeKnotc("zone-abort", normalizedZone)
		return fmt.Errorf("failed to remove old record from zone %s: %w", zone, err)
	}

	// Add updated record using separate arguments
	// For relative names within the zone, we need to remove the zone suffix
	recordName := existingRecord.Name
	if strings.HasSuffix(recordName, "."+normalizedZone) {
		// Convert absolute name to relative by removing zone suffix
		recordName = strings.TrimSuffix(recordName, "."+normalizedZone)
	} else if strings.HasSuffix(recordName, normalizedZone) {
		// Handle case where zone doesn't have trailing dot in record name
		recordName = strings.TrimSuffix(recordName, normalizedZone)
		recordName = strings.TrimSuffix(recordName, ".")
	}

	args := []string{"zone-set", normalizedZone, recordName,
		strconv.FormatUint(uint64(existingRecord.TTL), 10), string(existingRecord.Type)}

	// Add priority for MX records
	if existingRecord.Type == RecordTypeMX && existingRecord.Priority != nil {
		args = append(args, strconv.FormatUint(uint64(*existingRecord.Priority), 10))
	}

	// Add the record data
	args = append(args, existingRecord.Data)

	if _, err := c.executeKnotc(args...); err != nil {
		c.executeKnotc("zone-abort", normalizedZone)
		return fmt.Errorf("failed to add updated record to zone %s: %w", zone, err)
	}

	// Commit transaction
	if _, err := c.executeKnotc("zone-commit", normalizedZone); err != nil {
		return fmt.Errorf("failed to commit transaction for zone %s: %w", zone, err)
	}

	c.logger.Infof("Updated record: %s %s in zone %s", existingRecord.Name, existingRecord.Type, zone)
	return nil
}

// DeleteRecord deletes a DNS record
func (c *Client) DeleteRecord(zone, name string, recordType RecordType) error {
	if !c.IsZoneAllowed(zone) {
		return fmt.Errorf("zone not allowed: %s", zone)
	}

	// Check if record exists
	if _, err := c.GetRecord(zone, name, recordType); err != nil {
		return fmt.Errorf("record not found: %w", err)
	}

	// Begin transaction
	if _, err := c.executeKnotc("zone-begin", zone); err != nil {
		return fmt.Errorf("failed to begin transaction for zone %s: %w", zone, err)
	}

	// Remove record
	recordStr := fmt.Sprintf("%s %s", name, recordType)
	if _, err := c.executeKnotc("zone-unset", zone, recordStr); err != nil {
		c.executeKnotc("zone-abort", zone)
		return fmt.Errorf("failed to remove record from zone %s: %w", zone, err)
	}

	// Commit transaction
	if _, err := c.executeKnotc("zone-commit", zone); err != nil {
		return fmt.Errorf("failed to commit transaction for zone %s: %w", zone, err)
	}

	c.logger.Infof("Deleted record: %s %s from zone %s", name, recordType, zone)
	return nil
}

// ReloadZone reloads a zone configuration
func (c *Client) ReloadZone(zone string) error {
	if !c.IsZoneAllowed(zone) {
		return fmt.Errorf("zone not allowed: %s", zone)
	}

	if _, err := c.executeKnotc("zone-reload", zone); err != nil {
		return fmt.Errorf("failed to reload zone %s: %w", zone, err)
	}

	c.logger.Infof("Reloaded zone: %s", zone)
	return nil
}

// CheckHealth checks if KnotDNS is running and accessible
func (c *Client) CheckHealth() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.knotcPath, "status")
	if c.socketPath != "" {
		cmd.Args = append(cmd.Args[:1], append([]string{"-s", c.socketPath}, cmd.Args[1:]...)...)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("KnotDNS health check failed: %w", err)
	}

	return nil
}
