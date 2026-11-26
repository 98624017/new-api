package common

import (
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
)

const (
	paramOverrideAuditValueMaxRunes = 200
	paramOverrideAuditMaxBytes      = 1024
)

type paramOverrideAuditRecorder struct {
	lines     []string
	jsonBytes int
	full      bool
}

func (r *paramOverrideAuditRecorder) recordOperation(mode, path, from, to string, value interface{}) {
	if r == nil || r.full {
		return
	}
	line := buildParamOverrideAuditLine(mode, path, from, to, value, paramOverrideAuditValueMaxRunes)
	if line == "" || slices.Contains(r.lines, line) {
		return
	}

	available := paramOverrideAuditMaxBytes - r.jsonBytes
	if len(r.lines) == 0 {
		available -= 2 // 数组两端的方括号。
	} else {
		available-- // 新增行前的逗号。
	}
	if available <= 2 {
		r.full = true
		return
	}

	encoded, _ := common.Marshal(line) // 字符串序列化不会失败。
	if len(encoded) > available {
		r.full = true
		// 总量不足时只缩短值预览，操作名和字段路径保持完整。
		line = buildParamOverrideAuditLine(mode, path, from, to, value, 0)
		encoded, _ = common.Marshal(line)
		if len(encoded) > available {
			return
		}
		low, high := 0, paramOverrideAuditValueMaxRunes
		for low < high {
			mid := (low + high + 1) / 2
			candidate, _ := common.Marshal(buildParamOverrideAuditLine(mode, path, from, to, value, mid))
			if len(candidate) <= available {
				low = mid
			} else {
				high = mid - 1
			}
		}
		line = buildParamOverrideAuditLine(mode, path, from, to, value, low)
		if line == "" || slices.Contains(r.lines, line) {
			return
		}
		encoded, _ = common.Marshal(line)
	}
	if len(r.lines) == 0 {
		r.jsonBytes = 2
	} else {
		r.jsonBytes++
	}
	r.lines = append(r.lines, line)
	r.jsonBytes += len(encoded)
}

func formatParamOverrideAuditValue(value interface{}, maxRunes int) string {
	switch typed := value.(type) {
	case nil:
		return paramOverrideAuditPrefix("<empty>", maxRunes)
	case string:
		return paramOverrideAuditPrefix(typed, maxRunes)
	default:
		preview := paramOverrideAuditPreview{remaining: maxRunes}
		preview.writeJSON(value)
		return preview.text.String()
	}
}

func paramOverrideAuditPrefix(text string, limit int) string {
	for index := range text {
		if limit == 0 {
			return text[:index]
		}
		limit--
	}
	return text
}

type paramOverrideAuditPreview struct {
	text      strings.Builder
	remaining int
}

func (p *paramOverrideAuditPreview) write(text string) {
	prefix := paramOverrideAuditPrefix(text, p.remaining)
	p.text.WriteString(prefix)
	p.remaining -= utf8.RuneCountInString(prefix)
}

// 只格式化审计需要的 JSON 前缀，避免为长 messages 再序列化一份完整提示词。
func (p *paramOverrideAuditPreview) writeJSON(value interface{}) {
	if p.remaining == 0 {
		return
	}
	switch typed := value.(type) {
	case string:
		value = paramOverrideAuditPrefix(typed, p.remaining)
	case map[string]interface{}:
		if typed == nil {
			break
		}
		p.write("{")
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for index, key := range keys {
			if p.remaining == 0 {
				return
			}
			if index > 0 {
				p.write(",")
			}
			p.writeJSON(key)
			p.write(":")
			p.writeJSON(typed[key])
		}
		p.write("}")
		return
	case []interface{}:
		writeParamOverrideAuditArray(p, typed)
		return
	case []string:
		writeParamOverrideAuditArray(p, typed)
		return
	}
	encoded, _ := common.Marshal(value)
	p.write(string(encoded))
}

func writeParamOverrideAuditArray[T any](p *paramOverrideAuditPreview, items []T) {
	if items == nil {
		p.write("null")
		return
	}
	p.write("[")
	for index, item := range items {
		if p.remaining == 0 {
			return
		}
		if index > 0 {
			p.write(",")
		}
		p.writeJSON(item)
	}
	p.write("]")
}
