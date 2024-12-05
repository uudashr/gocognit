package testdata

import (
	"strings"
)

// Adopt from org.sonar.api.utils.WildcardPattern.java in SonarQube

func ToRegexp(antPattern string, directorySeparator string) string { // want "cognitive complexity 20 of func ToRegexp is high \\(> 0\\)"
	escapedDirectorySeparator := "\\" + directorySeparator

	var sb strings.Builder
	sb.WriteString("^")

	i := 0
	if strings.HasPrefix(antPattern, "/") || // +1 "if", +1 "||"
		strings.HasPrefix(antPattern, "\\") {
		i = 1
	}

	for i < len(antPattern) { // +1
		ch := antPattern[i]

		if strings.ContainsRune(SPECIAL_CHARS, rune(ch)) { // +2 (nesting=1)
			sb.WriteString("\\")
			sb.WriteByte(ch)
		} else if ch == '*' { // +1
			if i+1 < len(antPattern) && antPattern[i+1] == '*' { // +3 "if" (nesting=2), +1 "&&"
				if i+2 < len(antPattern) && isSlash(antPattern[i+2]) { // +4 "if" (nesting=3), +1 "&&"
					sb.WriteString("(?:.*")
					sb.WriteString(escapedDirectorySeparator)
					sb.WriteString("|)")
					i += 2
				} else { // +1
					sb.WriteString(".*")
					i += 1
				}
			} else { // +1
				sb.WriteString("[^" + escapedDirectorySeparator + "]*?")
			}
		} else if ch == '?' { // +1
			sb.WriteString("[^" + escapedDirectorySeparator + "]")
		} else if isSlash(ch) { // +1
			sb.WriteString(escapedDirectorySeparator)
		} else { // +1
			sb.WriteByte(ch)
		}

		i++
	}

	sb.WriteString("$")
	return sb.String()
} // total complelxity = 20

func isSlash(ch byte) bool { // want "cognitive complexity 1 of func isSlash is high \\(> 0\\)"
	return ch == '/' || ch == '\\' // +1
}

var SPECIAL_CHARS = "()[]^$.{}+|"
