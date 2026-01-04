package scraper

import (
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

func ParseTibiaComWorld(r io.Reader) (map[string]int, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	players := make(map[string]int)
	var traverse func(*html.Node)

	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			if isPlayerRow(n) {
				name, level := extractPlayerData(n)
				if name != "" && level > 0 {
					players[name] = level
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
	return players, nil
}

func isPlayerRow(n *html.Node) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" && (attr.Val == "Odd" || attr.Val == "Even") {
			return true
		}
	}
	return false
}

func extractPlayerData(tr *html.Node) (string, int) {
	var cells []*html.Node

	for c := tr.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "td" {
			cells = append(cells, c)
		}
	}

	if len(cells) < 2 {
		return "", 0
	}

	name := extractPlayerName(cells[0])
	level := extractLevel(cells[1])

	return name, level
}

func extractPlayerName(td *html.Node) string {
	var link *html.Node

	// Iterate over all children to find the correct link
	for c := td.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "a" {
			// Check if this anchor is the player link
			for _, attr := range c.Attr {
				if attr.Key == "href" && strings.Contains(attr.Val, "name=") {
					link = c
					break
				}
			}
			if link != nil {
				break
			}
		}
	}

	if link == nil {
		return ""
	}

	for _, attr := range link.Attr {
		if attr.Key == "href" && strings.Contains(attr.Val, "name=") {
			return extractNameFromURL(attr.Val)
		}
	}

	return ""
}

func extractNameFromURL(link string) string {
	re := regexp.MustCompile(`[?&]name=([^&]+)`)
	matches := re.FindStringSubmatch(link)
	if len(matches) < 2 {
		return ""
	}

	decoded, err := url.QueryUnescape(matches[1])
	if err != nil {
		return ""
	}
	return decoded
}

func extractLevel(td *html.Node) int {
	text := getTextContent(td)
	text = strings.TrimSpace(text)

	level, err := strconv.Atoi(text)
	if err != nil {
		return 0
	}

	return level
}

func getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(getTextContent(c))
	}

	return text.String()
}
