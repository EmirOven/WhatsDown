package markdown_test

import (
	"strings"
	"testing"

	"postbetasoftware.com/whatsdown/internal/markdown"
)

func TestBubbleURLs(t *testing.T) {
	cases := []struct{ input, expected string }{
		{"https://example.com", `<a href="https://example.com">https://example.com</a>`},
		{"Visit www.example.com.", `Visit <a href="https://www.example.com">www.example.com</a>.`},
		{"*https://example.com/a_b_c?x=1&y=2*", `<b><a href="https://example.com/a_b_c?x=1&amp;y=2">https://example.com/a_b_c?x=1&amp;y=2</a></b>`},
		{"_https://example.com_", `<i><a href="https://example.com">https://example.com</a></i>`},
		{"(https://example.com/a_(b)).", `(<a href="https://example.com/a_(b)">https://example.com/a_(b)</a>).`},
		{"`https://example.com`", `<tt>https://example.com</tt>`},
		{"<https://example.com>", `&lt;<a href="https://example.com">https://example.com</a>&gt;`},
		{"javascript:alert(1)", "javascript:alert(1)"},
		{"https://example.com/@123456", `<a href="https://example.com/@123456">https://example.com/@123456</a>`},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, mentions := markdown.FormatForBubble(tc.input, nil, nil)
			if got != tc.expected {
				t.Fatalf("got %q, want %q", got, tc.expected)
			}
			if mentions {
				t.Fatal("URL interpreted as a mention")
			}
		})
	}
	got, mentions := markdown.FormatForBubble("@123456 https://example.com", nil, nil)
	if !mentions || !strings.Contains(got, `href="user:123456@s.whatsapp.net"`) || !strings.Contains(got, `href="https://example.com"`) {
		t.Fatalf("mention and URL links did not coexist: %s", got)
	}
}

func TestBubbleFormatting(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"*bold*", "<b>bold</b>"},
		{"_italic_", "<i>italic</i>"},
		{"~strike~", "<s>strike</s>"},
		{"`code`", "<tt>code</tt>"},
		{"*bold* and _italic_", "<b>bold</b> and <i>italic</i>"},
		{"*_bold and italic_*", "<b><i>bold and italic</i></b>"},
		{"*not bold *", "*not bold *"},
		{"* not bold*", "* not bold*"},
		{"> quote text", `<span alpha="70%">▌ <i>quote text</i></span>`},
		{"- item 1", "• item 1"},
	}

	for _, c := range cases {
		out, _ := markdown.FormatForBubble(c.input, nil, nil)
		if out != c.expected {
			t.Errorf("input %q: expected %q, got %q", c.input, c.expected, out)
		}
	}
}

func TestForEntry(t *testing.T) {
	cases := []string{
		"*bold text*",
		"_italic text_",
		"~strike text~",
		"`code inline`",
		"```code block```",
		"*bold* and _italic_ and `code`",
		"> quote line",
		"- bullet item",
		"1. first item",
		"Normal text with no markdown",
	}

	for _, text := range cases {
		attrList, err := markdown.FormatForEntry(text)
		if err != nil {
			t.Errorf("FormatForEntry(%q) error: %v", text, err)
		}
		if markdown.HasWhatsAppMarkdown(text) && attrList == nil {
			t.Errorf("expected non-nil attrList for %q", text)
		}
	}
}
