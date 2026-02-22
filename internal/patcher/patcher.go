package patcher

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ErrNotFound is returned when cli.js cannot be located.
var ErrNotFound = errors.New("claude code cli.js not found; install with: npm install -g @anthropic-ai/claude-code")

// ErrAlreadyPatched is returned when the file is already patched.
var ErrAlreadyPatched = errors.New("already patched")

// patchMarker is embedded in patched files so we can detect them.
const patchMarker = "/* Vietnamese IME fix */"

// delChar is the character Vietnamese IME sends as backspace (0x7F).
const delChar = "\x7f"

// IsPatched reports whether the file at path has already been patched.
func IsPatched(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), patchMarker)
}

// Patch applies the Vietnamese IME fix to cli.js.
//
// Strategy:
//  1. Find the onInput function (contains .includes("\x7f"))
//  2. Find the early return guard: if(t===""&&o!=="")return;
//  3. Replace the guard AND the original if-block with our fix FIRST,
//     then the guard (so our \x7f check runs before the guard can eat it)
//  4. Our fix processes each char sequentially: \x7f → .backspace(), else → .insert()
func Patch(path string) error {
	if IsPatched(path) {
		return ErrAlreadyPatched
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read cli.js: %w", err)
	}

	// Write backup
	backupPath := path + ".bak"
	if err := os.WriteFile(backupPath, data, 0o644); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}

	src := string(data)

	// Step 1: Find the bug block to extract variable names
	blockStart, blockEnd, block, err := findBugBlock(src)
	if err != nil {
		return err
	}

	// Step 2: Extract dynamic variable names
	vars, err := extractVariables(block)
	if err != nil {
		return fmt.Errorf("extract variables: %w", err)
	}

	// Step 3: Find the early return guard BEFORE the bug block
	// Pattern: if(t===""&&o!=="")return;  (where t and o are the transformed and raw input)
	// It's between "let t=G?G(o,J6):o;" and our bug block
	guardSearch := src[blockStart-200 : blockStart]
	guardPattern := `if(t===""&&` + vars.input + `!=="")return;`
	guardIdx := strings.Index(guardSearch, guardPattern)

	// Step 4: Generate fix and determine injection strategy
	fixCode := generateFix(vars)

	var patched string
	if guardIdx != -1 {
		// We found the guard — inject our fix BEFORE the guard, then remove the old bug block
		guardAbsIdx := (blockStart - 200) + guardIdx

		// Our fix replaces from guardAbsIdx through blockEnd
		// Layout: [fix_code][guard_code][remaining_after_bugblock]
		patched = src[:guardAbsIdx] + fixCode + guardPattern + src[blockEnd:]
	} else {
		// No guard found — just replace the bug block (original approach)
		patched = src[:blockStart] + fixCode + src[blockEnd:]
	}

	if err := os.WriteFile(path, []byte(patched), 0o644); err != nil {
		_ = os.WriteFile(path, data, 0o644)
		return fmt.Errorf("write patched file: %w", err)
	}

	// Verify
	verifyData, _ := os.ReadFile(path)
	if !strings.Contains(string(verifyData), patchMarker) {
		_ = os.WriteFile(path, data, 0o644)
		return errors.New("verify failed: patch marker not found after write")
	}

	return nil
}

// Restore removes the patch and restores from backup.
func Restore(path string) error {
	backupPath := path + ".bak"
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return stripPatch(path)
	}
	return os.WriteFile(path, data, 0o644)
}

// ClaudeVersion reads the version string from the package.json sibling to cli.js.
func ClaudeVersion(cliPath string) string {
	pkgPath := strings.Replace(cliPath, "cli.js", "package.json", 1)
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return "unknown"
	}
	re := regexp.MustCompile(`"version"\s*:\s*"([^"]+)"`)
	m := re.FindStringSubmatch(string(data))
	if len(m) < 2 {
		return "unknown"
	}
	return m[1]
}

// findBugBlock locates the if-block containing .includes("\x7f") in the source.
func findBugBlock(src string) (int, int, string, error) {
	pattern := `.includes("` + delChar + `")`
	idx := strings.Index(src, pattern)
	if idx == -1 {
		return 0, 0, "", errors.New(
			`bug pattern .includes("\x7f") not found — Claude Code may have been fixed by Anthropic`)
	}

	// Find the containing if(
	searchStart := idx - 150
	if searchStart < 0 {
		searchStart = 0
	}
	segment := src[searchStart:idx]
	ifIdx := strings.LastIndex(segment, "if(")
	if ifIdx == -1 {
		return 0, 0, "", errors.New("cannot find enclosing if( for bug pattern")
	}
	blockStart := searchStart + ifIdx

	// Find matching closing brace
	depth := 0
	blockEnd := idx
	limit := blockStart + 800
	if limit > len(src) {
		limit = len(src)
	}
	for i, c := range src[blockStart:limit] {
		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				blockEnd = blockStart + i + 1
				goto found
			}
		}
	}
	return 0, 0, "", errors.New("cannot find matching closing brace for bug block")

found:
	return blockStart, blockEnd, src[blockStart:blockEnd], nil
}

// variables holds the dynamic variable names extracted from the bug block.
type variables struct {
	input      string // the input data variable
	state      string // the mutable state variable
	curState   string // the current (snapshot) state variable
	updateText string // function that updates text
	updateOfs  string // function that updates cursor offset
}

// extractVariables extracts dynamic variable names from the minified bug block.
func extractVariables(block string) (*variables, error) {
	normalized := strings.ReplaceAll(block, delChar, `\x7f`)

	reCountState := regexp.MustCompile(
		`let ([\w$]+)=\(\w+\.match\(/\\x7f/g\)\|\|\[\]\)\.length[,;]([\w$]+)=([\w$]+)[;,]`)
	m := reCountState.FindStringSubmatch(normalized)
	if m == nil {
		return nil, errors.New("cannot extract count/state variables from bug block")
	}
	state := m[2]
	curState := m[3]

	reUpdate := regexp.MustCompile(
		`([\w$]+)\(` + regexp.QuoteMeta(state) + `\.text\);([\w$]+)\(` + regexp.QuoteMeta(state) + `\.offset\)`)
	m2 := reUpdate.FindStringSubmatch(block)
	if m2 == nil {
		return nil, errors.New("cannot extract update functions from bug block")
	}

	reInput := regexp.MustCompile(`([\w$]+)\.includes\("`)
	m3 := reInput.FindStringSubmatch(block)
	if m3 == nil {
		return nil, errors.New("cannot extract input variable from bug block")
	}

	return &variables{
		input:      m3[1],
		state:      state,
		curState:   curState,
		updateText: m2[1],
		updateOfs:  m2[2],
	}, nil
}

// generateFix produces the replacement JavaScript code.
// It processes each character sequentially: \x7f → backspace(), else → insert().
//
// IMPORTANT: We use a locally-scoped variable `_s` (declared with `let`) instead of
// reusing the minified state variable name. In minified JS, the same short name
// (e.g. `e3`) may exist in the outer/parent scope as a completely different variable.
// Assigning without `let` would corrupt that outer variable, causing subsequent
// onInput calls to malfunction (e.g. the second Vietnamese word typed breaks).
func generateFix(v *variables) string {
	return fmt.Sprintf(
		`%s`+
			`if(%s.includes("\x7f")){`+
			`let _s=%s;`+
			`for(const _c of %s){`+
			`if(_c==="\x7f"){_s=_s.backspace();}`+
			`else{_s=_s.insert(_c);}`+
			`}`+
			`if(!%s.equals(_s)){`+
			`if(%s.text!==_s.text)`+
			`%s(_s.text);`+
			`%s(_s.offset)`+
			`}return;}`,
		patchMarker,
		v.input,
		v.curState,
		v.input,
		v.curState,
		v.curState,
		v.updateText,
		v.updateOfs,
	)
}

// stripPatch removes the injected block from src when no backup is available.
func stripPatch(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	src := string(data)
	markerIdx := strings.Index(src, patchMarker)
	if markerIdx == -1 {
		return errors.New("patch marker not found")
	}
	endPattern := "return;}"
	endIdx := strings.Index(src[markerIdx:], endPattern)
	if endIdx == -1 {
		return errors.New("patch end not found")
	}
	stripped := src[:markerIdx] + src[markerIdx+endIdx+len(endPattern):]
	return os.WriteFile(path, []byte(stripped), 0o644)
}
