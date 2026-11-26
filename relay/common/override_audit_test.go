package common

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	common2 "github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParamOverrideAuditTruncatesUnicodeValuesWithoutChangingRequests(t *testing.T) {
	originalDebug := common2.DebugEnabled
	common2.DebugEnabled = false
	t.Cleanup(func() { common2.DebugEnabled = originalDebug })

	longValue := strings.Repeat("界🙂", 130) + "完整尾部"
	nestedText := "特殊\"\\\n<>&" + longValue
	for _, test := range []struct {
		name   string
		mode   string
		path   string
		value  interface{}
		legacy bool
	}{
		{name: "legacy string", path: "instructions", value: longValue, legacy: true},
		{name: "operation string", path: "system", value: longValue},
		{name: "prepend messages array", mode: "prepend", path: "messages", value: []interface{}{
			map[string]interface{}{"role": "system", "content": nestedText},
		}},
		{name: "instruction object", path: "systemInstruction", value: map[string]interface{}{
			"parts": []interface{}{map[string]interface{}{"text": nestedText}},
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			mode := test.mode
			if mode == "" {
				mode = "set"
			}
			override := map[string]interface{}{
				"operations": []map[string]interface{}{{"mode": mode, "path": test.path, "value": test.value}},
			}
			if test.legacy {
				override = map[string]interface{}{test.path: test.value}
			}
			configBefore, err := common2.Marshal(override)
			require.NoError(t, err)
			valueText, isString := test.value.(string)
			if !isString {
				encoded, err := common2.Marshal(test.value)
				require.NoError(t, err)
				valueText = string(encoded)
			}
			wantPreview := string([]rune(valueText)[:200])
			info := &RelayInfo{ChannelMeta: &ChannelMeta{ParamOverride: override}}

			output, err := ApplyParamOverrideWithRelayInfo([]byte(`{"model":"existing","messages":[{"role":"user","content":"question"}]}`), info)
			require.NoError(t, err)
			var request map[string]interface{}
			require.NoError(t, common2.Unmarshal(output, &request))
			wantValue := test.value
			wantLine := "set " + test.path + " = " + wantPreview
			if mode == "prepend" {
				wantValue = append(slices.Clone(test.value.([]interface{})), map[string]interface{}{"role": "user", "content": "question"})
				wantLine = "prepend " + test.path + " with " + wantPreview
			}
			assert.Equal(t, wantValue, request[test.path])
			assert.Equal(t, "existing", request["model"])
			assert.Equal(t, []string{wantLine}, info.ParamOverrideAudit)
			configAfter, err := common2.Marshal(override)
			require.NoError(t, err)
			assert.Equal(t, configBefore, configAfter)
		})
	}
}

func TestParamOverrideAuditBoundsEscapedJSONAndKeepsApplyingOperations(t *testing.T) {
	for _, debug := range []bool{false, true} {
		t.Run(fmt.Sprintf("DEBUG=%t", debug), func(t *testing.T) {
			originalDebug := common2.DebugEnabled
			common2.DebugEnabled = debug
			t.Cleanup(func() { common2.DebugEnabled = originalDebug })

			container := "instructions"
			if debug {
				container = "metadata"
			}
			value := strings.Repeat("\"\\\n\t<>&\u2028🙂", 30)
			operations := make([]map[string]interface{}, 0, 5)
			for i := 0; i < 4; i++ {
				operations = append(operations, map[string]interface{}{
					"mode": "set", "path": fmt.Sprintf("%s.audit_%d", container, i), "value": value,
				})
			}
			operations = append(operations, map[string]interface{}{"mode": "set", "path": "model", "value": "after-budget"})
			info := &RelayInfo{ChannelMeta: &ChannelMeta{ParamOverride: map[string]interface{}{"operations": operations}}}

			output, err := ApplyParamOverrideWithRelayInfo([]byte(`{"instructions":{},"metadata":{}}`), info)
			require.NoError(t, err)
			var request struct {
				Instructions map[string]string `json:"instructions"`
				Metadata     map[string]string `json:"metadata"`
				Model        string            `json:"model"`
			}
			require.NoError(t, common2.Unmarshal(output, &request))
			fields := request.Instructions
			if debug {
				fields = request.Metadata
			}
			for i := 0; i < 4; i++ {
				assert.Equal(t, value, fields[fmt.Sprintf("audit_%d", i)])
			}
			assert.Equal(t, "after-budget", request.Model)

			encoded, err := common2.Marshal(info.ParamOverrideAudit)
			require.NoError(t, err)
			assert.LessOrEqual(t, len(encoded), 1024)
			var decoded []string
			require.NoError(t, common2.Unmarshal(encoded, &decoded))
			assert.Equal(t, info.ParamOverrideAudit, decoded)
			require.Len(t, decoded, 2)
			fullPreview := string([]rune(value)[:200])
			for i, line := range decoded {
				prefix := fmt.Sprintf("set %s.audit_%d = ", container, i)
				require.True(t, strings.HasPrefix(line, prefix))
				preview := strings.TrimPrefix(line, prefix)
				assert.True(t, utf8.ValidString(preview))
				assert.True(t, strings.HasPrefix(fullPreview, preview))
				if i == 0 {
					assert.Equal(t, fullPreview, preview)
				} else {
					assert.NotEmpty(t, preview)
					assert.Less(t, utf8.RuneCountInString(preview), 200)
				}
			}
		})
	}
}

func TestParamOverrideAuditKeepsNullOperationAtByteLimit(t *testing.T) {
	padding := strings.Repeat("<", 163)
	info := &RelayInfo{ChannelMeta: &ChannelMeta{ParamOverride: map[string]interface{}{
		"operations": []map[string]interface{}{
			{"mode": "set", "path": "instructions", "value": padding},
			{"mode": "set", "path": "system", "value": nil},
		},
	}}}

	output, err := ApplyParamOverrideWithRelayInfo([]byte(`{"system":"previous"}`), info)
	require.NoError(t, err)
	assert.JSONEq(t, `{"system":null,"instructions":"`+padding+`"}`, string(output))
	assert.Equal(t, []string{"set instructions = " + padding, "set system = <e"}, info.ParamOverrideAudit)
	encoded, err := common2.Marshal(info.ParamOverrideAudit)
	require.NoError(t, err)
	assert.Len(t, encoded, 1024)
}

func TestParamOverrideAuditPreservesOperationFormatAndResetsRetryBudget(t *testing.T) {
	originalDebug := common2.DebugEnabled
	common2.DebugEnabled = false
	t.Cleanup(func() { common2.DebugEnabled = originalDebug })

	longValue := strings.Repeat("界🙂", 130)
	info := &RelayInfo{ChannelMeta: &ChannelMeta{ParamOverride: map[string]interface{}{
		"operations": []map[string]interface{}{
			{"mode": "set", "path": "instructions", "value": longValue},
			{"mode": "set", "path": "system", "value": longValue},
		},
	}}}
	_, err := ApplyParamOverrideWithRelayInfo([]byte(`{}`), info)
	require.NoError(t, err)
	require.Len(t, info.ParamOverrideAudit, 2)
	assert.Less(t, utf8.RuneCountInString(strings.TrimPrefix(info.ParamOverrideAudit[1], "set system = ")), 200)

	key := strings.Repeat("p", 210)
	path := "metadata." + key
	info.ChannelMeta.ParamOverride = map[string]interface{}{
		"operations": []map[string]interface{}{
			{"mode": "set", "path": "instructions", "value": "short"},
			{"mode": "set", "path": "instructions", "value": "short"},
			{"mode": "copy", "from": "instructions", "to": path},
			{"mode": "move", "from": path, "to": "system"},
		},
	}
	wantAudit := []string{
		"set instructions = short",
		"copy instructions -> " + path,
		"move " + path + " -> system",
	}
	for attempt := 0; attempt < 2; attempt++ {
		output, err := ApplyParamOverrideWithRelayInfo([]byte(`{"model":"existing"}`), info)
		require.NoError(t, err)
		var request map[string]interface{}
		require.NoError(t, common2.Unmarshal(output, &request))
		assert.Equal(t, "short", request["instructions"])
		assert.Equal(t, "short", request["system"])
		assert.Equal(t, "existing", request["model"])
		assert.NotContains(t, request["metadata"], key)
		assert.Equal(t, wantAudit, info.ParamOverrideAudit)
	}
}

func TestParamOverrideAuditTruncatesReplacementTextWithoutChangingReplacement(t *testing.T) {
	originalDebug := common2.DebugEnabled
	common2.DebugEnabled = false
	t.Cleanup(func() { common2.DebugEnabled = originalDebug })

	from := strings.Repeat(" ", 200) + "FROM-END"
	to := "\t" + strings.Repeat("界🙂", 115) + "TO-END"
	input, err := common2.Marshal(map[string]string{"instructions": "before:" + from + ":after"})
	require.NoError(t, err)
	for _, mode := range []string{"replace", "regex_replace"} {
		t.Run(mode, func(t *testing.T) {
			override := map[string]interface{}{
				"operations": []map[string]interface{}{{"mode": mode, "path": "instructions", "from": from, "to": to}},
			}
			configBefore, err := common2.Marshal(override)
			require.NoError(t, err)
			info := &RelayInfo{ChannelMeta: &ChannelMeta{ParamOverride: override}}

			output, err := ApplyParamOverrideWithRelayInfo(input, info)
			require.NoError(t, err)
			var request map[string]string
			require.NoError(t, common2.Unmarshal(output, &request))
			assert.Equal(t, "before:"+to+":after", request["instructions"])
			assert.Equal(t, []string{mode + " instructions from " + string([]rune(from)[:200]) + " to " + string([]rune(to)[:200])}, info.ParamOverrideAudit)
			encoded, err := common2.Marshal(info.ParamOverrideAudit)
			require.NoError(t, err)
			assert.LessOrEqual(t, len(encoded), 1024)
			configAfter, err := common2.Marshal(override)
			require.NoError(t, err)
			assert.Equal(t, configBefore, configAfter)
		})
	}
}
