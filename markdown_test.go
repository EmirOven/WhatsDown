package main

import (
	"testing"
)

func TestWhatsAppMarkdownBubbleFormatting(t *testing.T) {
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
		out, _ := FormatWhatsAppMarkdownForBubble(c.input, nil, nil)
		if out != c.expected {
			t.Errorf("input %q: expected %q, got %q", c.input, c.expected, out)
		}
	}
}

func TestWhatsAppMarkdownForEntry(t *testing.T) {
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
		attrList, err := FormatWhatsAppMarkdownForEntry(text)
		if err != nil {
			t.Errorf("FormatWhatsAppMarkdownForEntry(%q) error: %v", text, err)
		}
		if HasWhatsAppMarkdown(text) && attrList == nil {
			t.Errorf("expected non-nil attrList for %q", text)
		}
	}
}
