package tracker

import (
	"strings"
	"testing"
)

func TestParseTibiaComWorld(t *testing.T) {
	// Sample HTML from tibia.com world page
	htmlSample := `
<!DOCTYPE html>
<html>
<body>
<table>
<tr class="Odd" style="text-align:right;">
	<td style="width:70%;text-align:left;">
		<a href="https://www.tibia.com/community/?subtopic=characters&name=Test+Player">Test&#160;Player</a>
	</td>
	<td style="width:10%;">123</td>
	<td style="width:20%;">Elite&#160;Knight</td>
</tr>
<tr class="Even" style="text-align:right;">
	<td style="width:70%;text-align:left;">
		<a href="https://www.tibia.com/community/?subtopic=characters&name=Another+One">Another&#160;One</a>
	</td>
	<td style="width:10%;">456</td>
	<td style="width:20%;">Royal&#160;Paladin</td>
</tr>
<tr class="Odd" style="text-align:right;">
	<td style="width:70%;text-align:left;">
		<a href="https://www.tibia.com/community/?subtopic=characters&name=Zafis+Cain">Zafis&#160;Cain</a>
	</td>
	<td style="width:10%;">1680</td>
	<td style="width:20%;">Elite&#160;Knight</td>
</tr>
</table>
</body>
</html>
`

	reader := strings.NewReader(htmlSample)
	players, err := ParseTibiaComWorld(reader)

	if err != nil {
		t.Fatalf("ParseTibiaComWorld failed: %v", err)
	}

	expected := map[string]int{
		"Test Player": 123,
		"Another One": 456,
		"Zafis Cain":  1680,
	}

	if len(players) != len(expected) {
		t.Errorf("Expected %d players, got %d", len(expected), len(players))
	}

	for name, expectedLevel := range expected {
		actualLevel, ok := players[name]
		if !ok {
			t.Errorf("Player %q not found in results", name)
			continue
		}
		if actualLevel != expectedLevel {
			t.Errorf("Player %q: expected level %d, got %d", name, expectedLevel, actualLevel)
		}
	}
}

func TestParseTibiaComWorld_Empty(t *testing.T) {
	htmlSample := `<!DOCTYPE html><html><body><table></table></body></html>`

	reader := strings.NewReader(htmlSample)
	players, err := ParseTibiaComWorld(reader)

	if err != nil {
		t.Fatalf("ParseTibiaComWorld failed: %v", err)
	}

	if len(players) != 0 {
		t.Errorf("Expected 0 players, got %d", len(players))
	}
}

func TestExtractNameFromURL(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{
			"https://www.tibia.com/community/?subtopic=characters&name=Test+Player",
			"Test Player",
		},
		{
			"https://www.tibia.com/community/?subtopic=characters&name=Zafis+Cain",
			"Zafis Cain",
		},
		{
			"https://www.tibia.com/community/?name=Simple",
			"Simple",
		},
		{
			"https://www.tibia.com/community/?subtopic=characters",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := extractNameFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("extractNameFromURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestParseTibiaComWorld_WithQuote(t *testing.T) {
	htmlSample := `
<!DOCTYPE html>
<html>
<body>
<table>
<tr class="Odd" style="text-align:right;">
	<td style="width:70%;text-align:left;">
		<a href="https://www.tibia.com/community/?subtopic=characters&name=Hell%27Draco">Hell&#39;Draco</a>
	</td>
	<td style="width:10%;">123</td>
	<td style="width:20%;">Elite&#160;Knight</td>
</tr>
</table>
</body>
</html>
`

	reader := strings.NewReader(htmlSample)
	players, err := ParseTibiaComWorld(reader)

	if err != nil {
		t.Fatalf("ParseTibiaComWorld failed: %v", err)
	}

	expectedName := "Hell'Draco"
	level, ok := players[expectedName]
	if !ok {
		// Log what keys we actually have
		keys := make([]string, 0, len(players))
		for k := range players {
			keys = append(keys, k)
		}
		t.Errorf("Player %q not found in results. Found: %v", expectedName, keys)
	}
	if level != 123 {
		t.Errorf("Expected level 123, got %d", level)
	}

	// Double check that we don't have the encoded version
	if _, ok := players["Hell%27Draco"]; ok {
		t.Errorf("Found improperly encoded name Hell%%27Draco in results")
	}
}
