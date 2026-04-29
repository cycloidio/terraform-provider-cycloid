package provider

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// lhsFilter represents a single LHS bracket filter parameter.
type lhsFilter struct {
	Attribute string
	Condition string
	Value     string
}

// filterGet makes a GET request to the Cycloid API with LHS bracket filter
// parameters. It builds the query string without percent-encoding regex
// metacharacters in filter values, since the API uses filter values directly
// as regex patterns without URL-decoding them first.
func filterGet(p *CycloidProvider, org string, route []string, filters []lhsFilter, response any) error {
	baseURL, err := url.Parse(p.APIClient.Config.URL)
	if err != nil {
		return fmt.Errorf("invalid API URL: %w", err)
	}

	pathParts := make([]string, 0, len(route)+1)
	pathParts = append(pathParts, baseURL.Path)
	pathParts = append(pathParts, route...)
	baseURL.Path = path.Join(pathParts...)

	if len(filters) > 0 {
		baseURL.RawQuery = buildLHSFilterQuery(filters)
	}

	req, err := http.NewRequest(http.MethodGet, baseURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.APIClient.GetToken(&org))
	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: p.APIClient.Config.Insecure, //nolint:gosec
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Data != nil {
		return json.Unmarshal(envelope.Data, response)
	}
	return json.Unmarshal(body, response)
}

// buildLHSFilterQuery builds a raw query string for LHS bracket filter params.
// Brackets are kept literal in keys; values use lhsEscapeValue.
func buildLHSFilterQuery(filters []lhsFilter) string {
	parts := make([]string, 0, len(filters))
	for _, f := range filters {
		key := f.Attribute + "[" + f.Condition + "]"
		parts = append(parts, key+"="+lhsEscapeValue(f.Value))
	}
	return strings.Join(parts, "&")
}

// lhsEscapeValue percent-encodes only characters that are structurally
// significant in query strings (&, =, #, space, control chars), preserving
// regex metacharacters such as ?, *, +, [, ], (, ), {, }, |, ^, $, \.
func lhsEscapeValue(s string) string {
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '&' || c == '=' || c == '#' || c == ' ':
			fmt.Fprintf(&buf, "%%%02X", c)
		case c < 0x20 || c == 0x7F || c > 0x7E:
			fmt.Fprintf(&buf, "%%%02X", c)
		default:
			buf.WriteByte(c)
		}
	}
	return buf.String()
}
