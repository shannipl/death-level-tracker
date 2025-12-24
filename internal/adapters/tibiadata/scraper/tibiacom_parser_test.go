package scraper

import (
	"strings"
	"testing"
)

func TestParseTibiaComWorld(t *testing.T) {
	tests := []struct {
		name      string
		htmlInput string
		want      map[string]int
		wantErr   bool
	}{
		{
			name: "Standard Table - Odd and Even Rows",
			htmlInput: `
				<html><body>
				<table border="0" cellpadding="4" cellspacing="1" width="100%">
					<tr class="LabelH"><td colspan="3">Online Players</td></tr>
					<tr class="Odd">
						<td><a href="https://www.tibia.com/community/?subtopic=characters&name=Bubble">Bubble</a></td>
						<td>100</td>
						<td>Elite Knight</td>
					</tr>
					<tr class="Even">
						<td><a href="https://www.tibia.com/community/?subtopic=characters&name=Eternal+Oblivion">Eternal Oblivion</a></td>
						<td>200</td>
						<td>Elite Knight</td>
					</tr>
				</table>
				</body></html>`,
			want: map[string]int{
				"Bubble":           100,
				"Eternal Oblivion": 200,
			},
			wantErr: false,
		},
		{
			name: "Edge Case - Names with Special Characters (Quotes, Spaces)",
			htmlInput: `
				<html><body><table>
					<tr class="Odd">
						<td><a href="https://www.tibia.com/community/?subtopic=characters&name=Hell%27Draco">Hell'Draco</a></td>
						<td>150</td>
					</tr>
					<tr class="Even">
						<td><a href="https://www.tibia.com/community/?subtopic=characters&name=Zafis+Cain">Zafis Cain</a></td>
						<td>300</td>
					</tr>
					<tr class="Odd">
						<td><a href="https://www.tibia.com/community/?name=Complex%20Name%27s+Here&subtopic=characters">Complex Name's Here</a></td>
						<td>404</td>
					</tr>
				</table></body></html>`,
			want: map[string]int{
				"Hell'Draco":          150,
				"Zafis Cain":          300,
				"Complex Name's Here": 404,
			},
			wantErr: false,
		},
		{
			name: "Edge Case - Malformed Level (Non-numeric)",
			htmlInput: `
				<html><body><table>
					<tr class="Odd">
						<td><a href="...&name=Player">Player</a></td>
						<td>NotALevel</td>
					</tr>
				</table></body></html>`,
			want:    map[string]int{},
			wantErr: false, // Should skip row gracefully, not error
		},
		{
			name: "Edge Case - Missing Level Column",
			htmlInput: `
				<html><body><table>
					<tr class="Odd">
						<td><a href="...&name=Player">Player</a></td>
					</tr>
				</table></body></html>`,
			want:    map[string]int{},
			wantErr: false,
		},
		{
			name: "Edge Case - Link Missing Query Param",
			htmlInput: `
				<html><body><table>
					<tr class="Odd">
						<td><a href="https://www.tibia.com/community/?subtopic=characters">No Name Param</a></td>
						<td>100</td>
					</tr>
				</table></body></html>`,
			want:    map[string]int{},
			wantErr: false,
		},
		{
			name:      "Empty - No Table or Rows",
			htmlInput: `<html><body><p>No players online</p></body></html>`,
			want:      map[string]int{},
			wantErr:   false,
		},
		{
			name:      "Empty - Invalid HTML (Closed prematurely)",
			htmlInput: `<html><body><table><tr class="Odd"><td>`,
			want:      map[string]int{},
			wantErr:   false, // html.Parse is forgiving
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.htmlInput)
			got, err := ParseTibiaComWorld(reader)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTibiaComWorld() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("ParseTibiaComWorld() got %d players, want %d", len(got), len(tt.want))
			}

			for name, level := range tt.want {
				gotLevel, ok := got[name]
				if !ok {
					t.Errorf("ParseTibiaComWorld() missing player %q", name)
				}
				if gotLevel != level {
					t.Errorf("ParseTibiaComWorld() player %q level = %d, want %d", name, gotLevel, level)
				}
			}
		})
	}
}

// Test internal logic specifically for URL parsing if needed, but coverage via ParseTibiaComWorld is preferred for exported API.
func TestExtractNameFromURL_Internal(t *testing.T) {
	// Focusing on specific regex/decoding logic that might be tricky
	tests := []struct {
		input string
		want  string
	}{
		{"https://tbia.com/?name=Simple", "Simple"},
		{"?name=Space+Man", "Space Man"},
		{"?name=Encoded%20Man", "Encoded Man"},
		{"?name=Quote%27s", "Quote's"},
		{"?other=1&name=MiddleParam&foo=bar", "MiddleParam"},
		{"https://tibia.com?name=Trailing", "Trailing"},
	}

	for _, tt := range tests {
		if got := extractNameFromURL(tt.input); got != tt.want {
			t.Errorf("extractNameFromURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
