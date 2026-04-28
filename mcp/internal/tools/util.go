// Package tools provides MCP tool implementations for Mailpit.
package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/amirhmoradi/mailpit-mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// textResult creates a text content result.
func textResult(text string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, nil, nil
}

// jsonResult creates a JSON content result.
func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	return textResult(string(data))
}

// errorResult creates an error result.
func errorResult(err error) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)},
		},
		IsError: true,
	}, nil, nil
}

// jsonResultWithSummary formats a MessagesSummary result.
func jsonResultWithSummary(result *client.MessagesSummary) (*mcp.CallToolResult, any, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d messages (%d unread) out of %d total\n\n",
		result.MessagesCount, result.MessagesUnreadCount, result.Total))

	for i, msg := range result.Messages {
		from := "unknown"
		if msg.From != nil {
			from = formatAddress(msg.From.Name, msg.From.Address)
		}
		status := "○"
		if msg.Read {
			status = "●"
		}
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1+result.Start, status, msg.Subject))
		sb.WriteString(fmt.Sprintf("   ID: %s\n", msg.ID))
		sb.WriteString(fmt.Sprintf("   From: %s\n", from))
		sb.WriteString(fmt.Sprintf("   Date: %s | Size: %s", msg.Created.Format("2006-01-02 15:04:05"), formatSize(msg.Size)))
		if msg.Attachments > 0 {
			sb.WriteString(fmt.Sprintf(" | Attachments: %d", msg.Attachments))
		}
		if len(msg.Tags) > 0 {
			sb.WriteString(fmt.Sprintf(" | Tags: %s", strings.Join(msg.Tags, ", ")))
		}
		sb.WriteString("\n\n")
	}

	return textResult(sb.String())
}

// jsonResultWithMessage formats a Message result.
func jsonResultWithMessage(msg *client.Message) (*mcp.CallToolResult, any, error) {
	var sb strings.Builder

	// Header section
	sb.WriteString(fmt.Sprintf("Subject: %s\n", msg.Subject))
	sb.WriteString(fmt.Sprintf("ID: %s\n", msg.ID))
	sb.WriteString(fmt.Sprintf("Message-ID: %s\n", msg.MessageID))
	sb.WriteString(fmt.Sprintf("Date: %s\n", msg.Date.Format("2006-01-02 15:04:05 MST")))
	sb.WriteString(fmt.Sprintf("Size: %s\n\n", formatSize(msg.Size)))

	// Addresses
	if msg.From != nil {
		sb.WriteString(fmt.Sprintf("From: %s\n", formatAddress(msg.From.Name, msg.From.Address)))
	}
	if len(msg.To) > 0 {
		addrs := make([]string, len(msg.To))
		for i, a := range msg.To {
			addrs[i] = formatAddress(a.Name, a.Address)
		}
		sb.WriteString(fmt.Sprintf("To: %s\n", strings.Join(addrs, ", ")))
	}
	if len(msg.Cc) > 0 {
		addrs := make([]string, len(msg.Cc))
		for i, a := range msg.Cc {
			addrs[i] = formatAddress(a.Name, a.Address)
		}
		sb.WriteString(fmt.Sprintf("Cc: %s\n", strings.Join(addrs, ", ")))
	}
	if len(msg.ReplyTo) > 0 {
		addrs := make([]string, len(msg.ReplyTo))
		for i, a := range msg.ReplyTo {
			addrs[i] = formatAddress(a.Name, a.Address)
		}
		sb.WriteString(fmt.Sprintf("Reply-To: %s\n", strings.Join(addrs, ", ")))
	}
	if msg.ReturnPath != "" {
		sb.WriteString(fmt.Sprintf("Return-Path: %s\n", msg.ReturnPath))
	}

	// Tags
	if len(msg.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("\nTags: %s\n", strings.Join(msg.Tags, ", ")))
	}

	// Attachments
	if len(msg.Attachments) > 0 {
		sb.WriteString(fmt.Sprintf("\nAttachments (%d):\n", len(msg.Attachments)))
		for _, a := range msg.Attachments {
			sb.WriteString(fmt.Sprintf("  - %s (%s, %s, PartID: %s)\n", a.FileName, a.ContentType, formatSize(a.Size), a.PartID))
		}
	}
	if len(msg.Inline) > 0 {
		sb.WriteString(fmt.Sprintf("\nInline attachments (%d):\n", len(msg.Inline)))
		for _, a := range msg.Inline {
			sb.WriteString(fmt.Sprintf("  - %s (%s, %s, PartID: %s)\n", a.FileName, a.ContentType, formatSize(a.Size), a.PartID))
		}
	}

	// Body content
	sb.WriteString("\n--- Text Body ---\n")
	if msg.Text != "" {
		sb.WriteString(msg.Text)
	} else {
		sb.WriteString("(empty)")
	}
	sb.WriteString("\n\n--- HTML Body ---\n")
	if msg.HTML != "" {
		// Truncate HTML for readability
		html := msg.HTML
		if len(html) > 5000 {
			html = html[:5000] + "\n... (truncated, use get_message_html for full content)"
		}
		sb.WriteString(html)
	} else {
		sb.WriteString("(empty)")
	}

	return textResult(sb.String())
}

// formatAddress formats an address for display.
func formatAddress(name, addr string) string {
	if name != "" {
		return fmt.Sprintf("%s <%s>", name, addr)
	}
	return addr
}

// formatSize formats a byte size for display.
func formatSize(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
