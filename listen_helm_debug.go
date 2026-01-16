package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
)

var debugLineRegex = regexp.MustCompile(`.+\[debug\]\s*(?P<debug_log>.*)$`)
var manifestStartRegex = regexp.MustCompile(`^(---\s*$|#\s*Source:|apiVersion:(\s|$)|kind:(\s|$)|metadata:(\s|$)|spec:(\s|$)|data:(\s|$)|stringData:(\s|$)|type:(\s|$))`)

// TODO: also warnings?
func printIfDebugLine(output io.Writer, line []byte) bool {
	result := debugLineRegex.FindSubmatch(line)
	if len(result) > 0 {
		debugLog := result[len(result)-1]
		_, err := output.Write(append(debugLog, []byte("\n")...))
		panicOnError(err)
		return true
	}
	return false
}

func stripHelmPrefix(line string) string {
	if !strings.HasPrefix(line, "helm:") {
		return line
	}
	idx := strings.Index(line, ">")
	if idx == -1 || idx+1 >= len(line) {
		return line
	}
	remaining := line[idx+1:]
	if strings.HasPrefix(remaining, " ") {
		return remaining[1:]
	}
	return remaining
}

func isManifestLine(line string) bool {
	lineNoPrefix := stripHelmPrefix(line)
	trimmed := strings.TrimSpace(lineNoPrefix)
	if trimmed == "" {
		return true
	}
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "---") {
		return true
	}
	if strings.HasPrefix(lineNoPrefix, " ") || strings.HasPrefix(lineNoPrefix, "\t") {
		return true
	}
	return manifestStartRegex.MatchString(trimmed)
}

func isHelmPrefixed(line string) bool {
	return strings.HasPrefix(line, "helm:")
}

func listenHelmDebug(input *bufio.Scanner, output io.Writer, args []string) {
	flag := flag.NewFlagSet(LISTEN_HELM_DEBUG_CMD_NAME, flag.ExitOnError)
	dropHelmManifests := flag.Bool("drop-helm-manifests", false, "Drop Helm/Kubernetes manifest output but keep other non-debug lines")
	dropHelmPrefixed := flag.Bool("drop-helm-prefixed", false, "Drop any non-debug lines prefixed with 'helm:…>'")
	err := flag.Parse(args)
	if err != nil {
		log.Fatal(err)
	}

	statusMessage := "\x1b[0;33mcicdbox: Only Helm debug logs will be displayed...\x1b[0m\n"
	if *dropHelmManifests || *dropHelmPrefixed {
		statusMessage = "\x1b[0;33mcicdbox: Helm debug logs and filtered non-debug output will be displayed...\x1b[0m\n"
	}
	_, err = output.Write([]byte(statusMessage))
	panicOnError(err)

	var outputToCollapse []string
	inManifest := false
	for input.Scan() {
		rawLine := input.Bytes()
		isDebug := printIfDebugLine(output, rawLine)
		if !*dropHelmManifests && !*dropHelmPrefixed {
			// Keep for the end
			outputToCollapse = append(outputToCollapse, string(rawLine))
		}
		if (*dropHelmManifests || *dropHelmPrefixed) && !isDebug {
			line := string(rawLine)
			trimmed := strings.TrimSpace(stripHelmPrefix(line))
			if *dropHelmPrefixed && isHelmPrefixed(line) {
				continue
			}
			if *dropHelmManifests {
				if inManifest {
					if isManifestLine(line) {
						continue
					}
					inManifest = false
				}
				if manifestStartRegex.MatchString(trimmed) {
					inManifest = true
					continue
				}
			}
			_, err := output.Write(append(rawLine, []byte("\n")...))
			panicOnError(err)
		}
	}
	if !*dropHelmManifests && !*dropHelmPrefixed {
		echoCollapsed(output, "\x1b[0;33mExpand to see the entire output.\x1b[0m", outputToCollapse)
	}

	if err := input.Err(); err != nil {
		log.Fatal(err)
	}
}

func ListenHelmDebug(args []string) {
	// read stdin
	input := bufio.NewScanner(os.Stdin)
	// write to stdout
	output := os.Stdout
	listenHelmDebug(input, output, args)
}
