package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	COLOR_RESET   = "\x1b[0m"
	RESOURCE_DIFF = "\x1b[0;33m"
	ELLIPSIS      = "..."
)

var DIFF_COLOR_PREFIXES = []string{"\x1b[0;31m-", "\x1b[0;32m+"}

func matchesPattern(str string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(str) {
			return true
		}
	}
	return false
}

func coloredWith(line string, prefix string) bool {
	return strings.HasPrefix(line, prefix) && strings.HasSuffix(line, COLOR_RESET)
}

func diffLine(line string) bool {
	for _, color := range DIFF_COLOR_PREFIXES {
		if coloredWith(line, color) {
			return true
		}
	}
	return false
}

func uncolor(line string) string {
	for _, color := range DIFF_COLOR_PREFIXES {
		if coloredWith(line, color) {
			return strings.TrimSuffix(strings.TrimPrefix(line, color), COLOR_RESET)
		}
	}
	return line
}

func unescapeEcho(str string) string {
	if !strings.Contains(str, "\\") {
		return str
	}
	var out strings.Builder
	out.Grow(len(str))
	for i := 0; i < len(str); i++ {
		ch := str[i]
		if ch != '\\' {
			out.WriteByte(ch)
			continue
		}
		if i+1 >= len(str) {
			out.WriteByte('\\')
			break
		}
		next := str[i+1]
		switch next {
		case 'e':
			out.WriteByte(0x1b)
			i++
		case 'n':
			out.WriteByte('\n')
			i++
		case 'r':
			out.WriteByte('\r')
			i++
		case 't':
			out.WriteByte('\t')
			i++
		case '\\':
			out.WriteByte('\\')
			i++
		case 'x':
			if i+3 < len(str) {
				hex := str[i+2 : i+4]
				val, err := strconv.ParseUint(hex, 16, 8)
				if err == nil {
					out.WriteByte(byte(val))
					i += 3
					break
				}
			}
			out.WriteByte('x')
			i++
		default:
			out.WriteByte(next)
			i++
		}
	}
	return out.String()
}

func echo(w io.Writer, str string) {
	_, err := w.Write([]byte(unescapeEcho(str)))
	panicOnError(err)
	_, err = w.Write([]byte("\n"))
	panicOnError(err)
}

func collapse(w io.Writer, lines []string, patterns []*regexp.Regexp) {
	if len(lines) == 0 {
		return
	}
	header, body := lines[0], lines[1:]
	if matchesPattern(uncolor(header), patterns) {
		header = ELLIPSIS
		body = lines
	}
	echoCollapsed(w, header, body)
}

func processResourceDiff(w io.Writer, diffs []string, patterns []*regexp.Regexp) {
	if len(diffs) == 0 {
		return
	}
	diffHeader, lines := diffs[0], diffs[1:]
	toBeCollapsed := []string{diffHeader}

	for _, line := range lines {
		if diffLine(line) {
			if matchesPattern(uncolor(line), patterns) {
				toBeCollapsed = append(toBeCollapsed, line)
			} else {
				//TODO: Keep more context (one line before?, what if this line matchesPattern)
				//if len(toBeCollapsed) != 0 {
				//	contextLine, rest := toBeCollapsed[len(toBeCollapsed)-1], toBeCollapsed[:len(toBeCollapsed)-1]
				//	collapse(rest)
				//	echo(contextLine)
				//}
				collapse(w, toBeCollapsed, patterns)
				toBeCollapsed = []string{}
				echo(w, line)
			}
		} else {
			toBeCollapsed = append(toBeCollapsed, line)
		}
	}
	collapse(w, toBeCollapsed, patterns)
}

func collapseHelmDiff(input *bufio.Scanner, output io.Writer, args []string) {
	flag := flag.NewFlagSet(COLLAPSE_HELM_DIFF_CMD_NAME, flag.ExitOnError)
	err := flag.Parse(args)
	if err != nil {
		log.Fatal(err)
	}

	regexes := flag.Args()
	if len(regexes) == 0 {
		log.Fatal("No regex was provided. Usage: command regex1 regex2 ...")
	}
	patterns := make([]*regexp.Regexp, len(regexes))
	for i, regex := range regexes {
		pattern := regexp.MustCompile(regex)
		patterns[i] = pattern
	}

	resourceDiff := []string{}
	for input.Scan() {
		rawLine := input.Text()
		if coloredWith(rawLine, RESOURCE_DIFF) {
			processResourceDiff(output, resourceDiff, patterns)
			resourceDiff = []string{}
		}
		resourceDiff = append(resourceDiff, rawLine)
	}
	processResourceDiff(output, resourceDiff, patterns)

	if err := input.Err(); err != nil {
		log.Fatal(err)
	}
}

func CollapseHelmDiff(args []string) {
	// read stdin
	input := bufio.NewScanner(os.Stdin)
	// write to stdout
	output := os.Stdout
	collapseHelmDiff(input, output, args)
}
