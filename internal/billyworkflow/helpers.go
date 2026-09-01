package billyworkflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cortexium-io/api-mcp/apimcp"
)

const (
	initialPageSize = 1000
	maxPages        = 100000
)

type paginationInfo struct {
	Complete             bool   `json:"complete"`
	PagesFetched         int    `json:"pagesFetched"`
	PageSize             int    `json:"pageSize"`
	RecordsFetched       int    `json:"recordsFetched"`
	ReportedTotal        int    `json:"reportedTotal,omitempty"`
	ReportedTotalPresent bool   `json:"reportedTotalPresent"`
	TruncationRetries    int    `json:"truncationRetries"`
	IncompleteReason     string `json:"incompleteReason,omitempty"`
}

func fetchAll(ctx context.Context, rt runtime, profileID, operation, root string, baseQuery map[string]any) ([]map[string]any, paginationInfo, error) {
	pageSize := initialPageSize
	retries := 0

	for {
		records, info, truncated, err := fetchAllAtPageSize(ctx, rt, profileID, operation, root, baseQuery, pageSize)
		if err != nil {
			return nil, paginationInfo{}, err
		}
		if !truncated {
			info.TruncationRetries = retries
			return records, info, nil
		}
		if pageSize == 1 {
			return nil, paginationInfo{}, fmt.Errorf("%s still exceeds the response limit at pageSize 1", operation)
		}
		pageSize = max(1, pageSize/2)
		retries++
	}
}

func fetchAllAtPageSize(ctx context.Context, rt runtime, profileID, operation, root string, baseQuery map[string]any, pageSize int) ([]map[string]any, paginationInfo, bool, error) {
	var records []map[string]any
	reportedTotal := 0
	reportedTotalPresent := false
	pageCount := 0

	for page := 1; page <= maxPages; page++ {
		query := cloneMap(baseQuery)
		query["page"] = page
		query["pageSize"] = pageSize
		response, err := rt.ExecuteRead(ctx, profileID, operation, apimcp.ToolInput{Query: query})
		if err != nil {
			return nil, paginationInfo{}, false, fmt.Errorf("%s page %d: %w", operation, page, err)
		}
		if response.Truncated {
			return nil, paginationInfo{}, true, nil
		}
		if !response.OK {
			return nil, paginationInfo{}, false, apiResponseError(operation, response)
		}

		body, err := objectBody(response.Body)
		if err != nil {
			return nil, paginationInfo{}, false, fmt.Errorf("%s page %d: %w", operation, page, err)
		}
		items, err := objectArray(body[root])
		if err != nil {
			return nil, paginationInfo{}, false, fmt.Errorf("%s page %d field %q: %w", operation, page, root, err)
		}
		records = append(records, items...)

		pagination := response.Pagination
		if pagination == nil {
			return records, paginationInfo{
				Complete:         false,
				PagesFetched:     page,
				PageSize:         pageSize,
				RecordsFetched:   len(records),
				IncompleteReason: "The response did not contain proven page/pageCount pagination metadata.",
			}, false, nil
		}
		if pagination.Page != page {
			return nil, paginationInfo{}, false, fmt.Errorf("%s page %d returned pagination metadata for page %d", operation, page, pagination.Page)
		}
		if pageCount == 0 {
			pageCount = pagination.PageCount
		} else if pagination.PageCount != pageCount {
			return nil, paginationInfo{}, false, fmt.Errorf("%s pageCount changed from %d to %d while paging", operation, pageCount, pagination.PageCount)
		}
		if bodyPaging, ok := nestedMap(body, "meta", "paging"); ok {
			if value, ok := integerValue(bodyPaging["total"]); ok {
				if reportedTotalPresent && value != reportedTotal {
					return nil, paginationInfo{}, false, fmt.Errorf("%s reported total changed from %d to %d while paging", operation, reportedTotal, value)
				}
				reportedTotal = value
				reportedTotalPresent = true
			}
		}

		if pagination.FinalPage {
			complete := true
			incompleteReason := ""
			if reportedTotalPresent && reportedTotal != len(records) {
				complete = false
				incompleteReason = fmt.Sprintf("Billy reported %d records, but %d were fetched.", reportedTotal, len(records))
			}
			info := paginationInfo{
				Complete:             complete,
				PagesFetched:         page,
				PageSize:             pageSize,
				RecordsFetched:       len(records),
				ReportedTotal:        reportedTotal,
				ReportedTotalPresent: reportedTotalPresent,
				IncompleteReason:     incompleteReason,
			}
			return records, info, false, nil
		}
	}

	return nil, paginationInfo{}, false, fmt.Errorf("%s exceeded the %d-page safety limit", operation, maxPages)
}

func executeObjectRead(ctx context.Context, rt runtime, profileID, operation, root string, input apimcp.ToolInput) (map[string]any, apimcp.APIResponse, error) {
	response, err := rt.ExecuteRead(ctx, profileID, operation, input)
	if err != nil {
		return nil, response, err
	}
	if response.Truncated {
		return nil, response, fmt.Errorf("%s response was truncated; the operation cannot be narrowed", operation)
	}
	if !response.OK {
		return nil, response, apiResponseError(operation, response)
	}
	body, err := objectBody(response.Body)
	if err != nil {
		return nil, response, fmt.Errorf("%s: %w", operation, err)
	}
	value, ok := body[root]
	if !ok {
		return nil, response, fmt.Errorf("%s response is missing %q", operation, root)
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, response, fmt.Errorf("%s response field %q is not an object", operation, root)
	}
	return object, response, nil
}

func apiResponseError(operation string, response apimcp.APIResponse) error {
	return fmt.Errorf("%s returned HTTP %d (%s)", operation, response.Status, response.StatusText)
}

func objectBody(value any) (map[string]any, error) {
	body, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("response body is not a JSON object")
	}
	return body, nil
}

func objectArray(value any) ([]map[string]any, error) {
	if value == nil {
		return []map[string]any{}, nil
	}
	values, ok := value.([]any)
	if !ok {
		return nil, errors.New("value is not an array")
	}
	result := make([]map[string]any, 0, len(values))
	for i, value := range values {
		object, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("item %d is not an object", i+1)
		}
		result = append(result, object)
	}
	return result, nil
}

func nestedMap(value map[string]any, path ...string) (map[string]any, bool) {
	current := value
	for _, key := range path {
		next, ok := current[key].(map[string]any)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func cloneMap(source map[string]any) map[string]any {
	result := make(map[string]any, len(source)+2)
	for key, value := range source {
		result[key] = value
	}
	return result
}

func integerValue(value any) (int, bool) {
	switch value := value.(type) {
	case float64:
		if value < 0 || value != math.Trunc(value) || value > math.MaxInt {
			return 0, false
		}
		return int(value), true
	case int:
		return value, value >= 0
	case json.Number:
		parsed, err := strconv.Atoi(value.String())
		return parsed, err == nil && parsed >= 0
	default:
		return 0, false
	}
}

func stringValue(value any) string {
	valueString, _ := value.(string)
	return valueString
}

func boolValue(value any) bool {
	valueBool, _ := value.(bool)
	return valueBool
}

func floatValue(value any) (float64, bool) {
	switch value := value.(type) {
	case float64:
		return value, true
	case json.Number:
		parsed, err := value.Float64()
		return parsed, err == nil
	case int:
		return float64(value), true
	default:
		return 0, false
	}
}

func exactFields(source map[string]any, fields ...string) map[string]any {
	result := make(map[string]any, len(fields))
	for _, field := range fields {
		result[field] = source[field]
	}
	return result
}

func parseDateRange(start, end string) (time.Time, time.Time, error) {
	startDate, err := time.Parse(time.DateOnly, start)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("startDate must use YYYY-MM-DD: %w", err)
	}
	endDate, err := time.Parse(time.DateOnly, end)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("endDate must use YYYY-MM-DD: %w", err)
	}
	if endDate.Before(startDate) {
		return time.Time{}, time.Time{}, errors.New("endDate must not be before startDate")
	}
	return startDate, endDate, nil
}

func dateInRange(value string, start, end time.Time) (bool, bool) {
	date, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return false, false
	}
	return !date.Before(start) && !date.After(end), true
}

func latestDate(records []map[string]any, field string) (string, int) {
	latest := ""
	invalid := 0
	for _, record := range records {
		value := stringValue(record[field])
		if _, err := time.Parse(time.DateOnly, value); err != nil {
			invalid++
			continue
		}
		if value > latest {
			latest = value
		}
	}
	return latest, invalid
}

func responseMap(value any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func normalizedDescription(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func canonicalAmount(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}
