package markdown

import "strings"

type textSegment struct {
	isCode      bool
	isBlockCode bool
	content     string
	delims      string
}

// splitByCodeBlocks splits a string into segments of code blocks and normal text.
func splitByCodeBlocks(text string) []textSegment {
	var segments []textSegment
	n := len(text)
	lastIdx, i := 0, 0

	for i < n {
		if i+3 <= n && text[i:i+3] == "```" {
			end := strings.Index(text[i+3:], "```")
			if end != -1 {
				closeIdx := i + 3 + end
				if i > lastIdx {
					segments = append(segments, textSegment{content: text[lastIdx:i]})
				}
				segments = append(segments, textSegment{
					isCode:      true,
					isBlockCode: true,
					delims:      "```",
					content:     text[i+3 : closeIdx],
				})
				i = closeIdx + 3
				lastIdx = i
				continue
			}
		} else if text[i] == '`' {
			end := -1
			for j := i + 1; j < n; j++ {
				if text[j] == '\n' {
					break
				}
				if text[j] == '`' {
					end = j
					break
				}
			}
			if end != -1 {
				if i > lastIdx {
					segments = append(segments, textSegment{content: text[lastIdx:i]})
				}
				segments = append(segments, textSegment{
					isCode:      true,
					isBlockCode: false,
					delims:      "`",
					content:     text[i+1 : end],
				})
				i = end + 1
				lastIdx = i
				continue
			}
		}
		i++
	}

	if lastIdx < n {
		segments = append(segments, textSegment{content: text[lastIdx:]})
	}
	return segments
}
