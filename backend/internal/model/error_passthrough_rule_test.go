package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllPlatformsIncludesEveryConcretePlatform(t *testing.T) {
	require.ElementsMatch(t, []string{
		"anthropic",
		"openai",
		"gemini",
		"antigravity",
		"grok",
		"kimi",
		"zhipu",
		"deepseek",
		"minimax",
		"opencode_go",
		"typesafe",
		"command_code",
		"cline",
		// 本地视频平台：登记在平台清单末尾，同样属于具体平台。
		"xiaoapi",
	}, AllPlatforms())
}
