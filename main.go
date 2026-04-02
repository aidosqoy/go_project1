package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

type tokenType int

const (
	wordToken tokenType = iota
	commandToken
	punctuationToken
	quoteToken
	newlineToken
)

type token struct {
	kind  tokenType
	value string
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run . sample.txt result.txt")
		return
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	text := string(data)
	words := splitWords(text)
	processed := processWords(words)
	final := formatText(processed)

	err = os.WriteFile(outputFile, []byte(final), 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return
	}
}

func splitWords(text string) []token {
	var words []token
	runes := []rune(text)
	for i := 0; i < len(runes); {
		current := runes[i]
		if current == '\r' {
			i++
			continue
		}
		if current == '\n' {
			words = append(words, token{kind: newlineToken, value: "\n"})
			i++
			continue
		}
		if unicode.IsSpace(current) {
			i++
			continue
		}
		if current == '(' {
			if value, next, ok := readCommand(runes, i); ok {
				words = append(words, token{kind: commandToken, value: value})
				i = next
				continue
			}
		}
		if current == '\'' {
			words = append(words, token{kind: quoteToken, value: "'"})
			i++
			continue
		}
		if isPunctuation(current) {
			start := i
			for i < len(runes) && isPunctuation(runes[i]) {
				i++
			}
			words = append(words, token{kind: punctuationToken, value: string(runes[start:i])})
			continue
		}

		start := i
		for i < len(runes) &&
			!unicode.IsSpace(runes[i]) &&
			runes[i] != '(' &&
			runes[i] != '\'' &&
			!isPunctuation(runes[i]) {
			i++
		}
		words = append(words, token{kind: wordToken, value: string(runes[start:i])})
	}
	return words
}

func processWords(words []token) []token {
	var result []token

	for _, word := range words {
		if word.kind != commandToken {
			result = append(result, word)
			continue
		}

		action, count, ok := parseCommand(word.value)
		if !ok {
			result = append(result, word)
			continue
		}

		applied := 0
		for j := len(result) - 1; j >= 0 && applied < count; j-- {
			if result[j].kind != wordToken {
				continue
			}

			switch action {
			case "cap":
				result[j].value = capWord(result[j].value)
			case "up":
				result[j].value = upWord(result[j].value)
			case "low":
				result[j].value = lowWord(result[j].value)
			case "hex":
				result[j].value = hexToDec(result[j].value)
			case "bin":
				result[j].value = binToDec(result[j].value)
			}
			applied++
		}
	}

	return fixArticles(result)
}

func formatText(words []token) string {
	var sb strings.Builder
	inQuote := false

	for i, word := range words {
		switch word.kind {
		case newlineToken:
			trimSpace(&sb)
			if i > 0 && sb.Len() > 0 {
				sb.WriteByte('\n')
			}
		case punctuationToken:
			trimSpace(&sb)
			sb.WriteString(word.value)
		case quoteToken:
			if inQuote {
				trimSpace(&sb)
				sb.WriteString("'")
				inQuote = false
			} else {
				if needSpaceBeforeWord(&sb) {
					sb.WriteByte(' ')
				}
				sb.WriteString("'")
				inQuote = true
			}
		case wordToken:
			if needSpaceBeforeToken(&sb, inQuote) {
				sb.WriteByte(' ')
			}
			sb.WriteString(word.value)
		}
	}

	return sb.String()
}

func parseCommand(word string) (string, int, bool) {
	if !isCommand(word) {
		return "", 0, false
	}
	cmd := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(word, "("), ")"))
	parts := strings.Split(cmd, ",")
	if len(parts) == 0 || len(parts) > 2 {
		return "", 0, false
	}

	action := strings.TrimSpace(parts[0])
	switch action {
	case "cap", "up", "low", "hex", "bin":
	default:
		return "", 0, false
	}

	count := 1
	if len(parts) == 2 {
		n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || n < 1 {
			return "", 0, false
		}
		count = n
	}

	return action, count, true
}

func isCommand(word string) bool {
	return strings.HasPrefix(word, "(") && strings.HasSuffix(word, ")")
}

func capWord(text string) string {
	if text == "" {
		return text
	}

	runes := []rune(strings.ToLower(text))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func upWord(text string) string  { return strings.ToUpper(text) }
func lowWord(text string) string { return strings.ToLower(text) }

func hexToDec(text string) string {
	n, e := strconv.ParseInt(text, 16, 64)
	if e != nil {
		return text
	}
	return strconv.FormatInt(n, 10)
}

func binToDec(text string) string {
	n, e := strconv.ParseInt(text, 2, 64)
	if e != nil {
		return text
	}
	return strconv.FormatInt(n, 10)
}

func readCommand(runes []rune, start int) (string, int, bool) {
	for i := start + 1; i < len(runes); i++ {
		if runes[i] == ')' {
			word := string(runes[start : i+1])
			if _, _, ok := parseCommand(word); ok {
				return word, i + 1, true
			}
			return "", start, false
		}
		if runes[i] == '\n' {
			return "", start, false
		}
	}
	return "", start, false
}

func fixArticles(words []token) []token {
	result := make([]token, len(words))
	copy(result, words)

	for i := 0; i < len(result); i++ {
		if result[i].kind != wordToken || !strings.EqualFold(result[i].value, "a") {
			continue
		}

		next, ok := nextWord(result, i+1)
		if !ok || next == "" {
			continue
		}

		first := []rune(next)[0]
		if !startsWithVowelOrH(first) {
			continue
		}

		if result[i].value == "A" {
			result[i].value = "An"
		} else {
			result[i].value = "an"
		}
	}

	return result
}

func nextWord(words []token, start int) (string, bool) {
	for i := start; i < len(words); i++ {
		if words[i].kind == newlineToken {
			return "", false
		}
		if words[i].kind == wordToken {
			return words[i].value, true
		}
	}
	return "", false
}

func startsWithVowelOrH(char rune) bool {
	switch unicode.ToLower(char) {
	case 'a', 'e', 'i', 'o', 'u', 'h':
		return true
	default:
		return false
	}
}

func trimSpace(sb *strings.Builder) {
	for sb.Len() > 0 {
		current := sb.String()
		if current[len(current)-1] != ' ' {
			return
		}
		sb.Reset()
		sb.WriteString(current[:len(current)-1])
	}
}

func needSpaceBeforeWord(sb *strings.Builder) bool {
	if sb.Len() == 0 {
		return false
	}
	last := sb.String()[sb.Len()-1]
	return last != ' ' && last != '\n'
}

func needSpaceBeforeToken(sb *strings.Builder, inQuote bool) bool {
	if sb.Len() == 0 {
		return false
	}
	last := sb.String()[sb.Len()-1]
	if inQuote {
		return last != ' ' && last != '\n' && last != '\''
	}
	return last != ' ' && last != '\n' && last != '\''
}

func isPunctuation(char rune) bool {
	return strings.ContainsRune(".,!?:;", char)
}
