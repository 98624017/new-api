package openai

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageGenerationAspectRatio(t *testing.T) {
	const fields = `"model":"seedream-5.5","prompt":"产品摄影","size":"2K","n":1,"watermark":false,"image":["https://example.com/product.png"]`
	for _, tc := range []struct {
		name        string
		channelType int
		extra       string
		wantExtra   string
	}{
		{"explicit ratio", constant.ChannelTypeOpenAI, `,"aspect_ratio":"9:16","unknown":true`, `,"aspect_ratio":"9:16"`},
		{"omitted ratio", constant.ChannelTypeOpenAI, "", ""},
		{"other channel", constant.ChannelTypeAzure, `,"aspect_ratio":"9:16"`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var request dto.ImageRequest
			require.NoError(t, common.UnmarshalJsonStr("{"+fields+tc.extra+"}", &request))
			copied, err := common.DeepCopy(&request)
			require.NoError(t, err)

			adaptor := Adaptor{ChannelType: tc.channelType}
			converted, err := adaptor.ConvertImageRequest(nil, &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImagesGenerations}, *copied)
			require.NoError(t, err)
			upstream, err := common.Marshal(converted)
			require.NoError(t, err)
			assert.JSONEq(t, "{"+fields+tc.wantExtra+"}", string(upstream))
		})
	}
}
