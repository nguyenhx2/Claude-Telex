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
const patchMarker = "/* Vietnamese IME fix v2 */"

// legacyMarkers lists markers from previous patch versions (for auto-upgrade).
var legacyMarkers = []string{"/* Vietnamese IME fix */"}

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

// hasLegacyPatch checks if the file has an older version of our patch.
func hasLegacyPatch(src string) bool {
	for _, m := range legacyMarkers {
		if strings.Contains(src, m) {
			return true
		}
	}
	return false
}

// Patch applies the Vietnamese IME fix to cli.js.
//
// Strategy:
//  1. Find the onInput function (contains .includes("\x7f"))
//  2. Find the early return guard: if(t===""&&o!=="")return;
//  3. Replace the guard AND the original if-block with our fix FIRST,
//     then the guard (so our \x7f check runs before the guard can eat it)
//  4. Our fix processes each char sequentially:
//     \x7f → .deleteTokenBefore()??.backspace(), else → .insert()
//  5. After processing, call cleanup functions (XI6, MI6 etc.) from the original
func Patch(path string) error {
	if IsPatched(path) {
		return ErrAlreadyPatched
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read cli.js: %w", err)
	}

	src := string(data)

	// Auto-upgrade: if legacy patch exists, restore first
	if hasLegacyPatch(src) {
		if err := Restore(path); err != nil {
			return fmt.Errorf("restore legacy patch: %w", err)
		}
		// Re-read after restore
		data, err = os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read after restore: %w", err)
		}
		src = string(data)
	}

	// Write backup (after potential restore, so backup is always the clean original)
	backupPath := path + ".bak"
	if err := os.WriteFile(backupPath, data, 0o644); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}

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
	// Pattern: if(t===""&&o!=="")return;
	guardSearch := src[blockStart-200 : blockStart]
	guardPattern := `if(t===""&&` + vars.input + `!==""` + `)return;`
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
	input      string // the raw input data variable (e.g. "o")
	keyInfo    string // the key info parameter (e.g. "J6") — has .backspace, .delete props
	state      string // the mutable state variable (e.g. "H6")
	curState   string // the current (snapshot) state variable (e.g. "y")
	updateText string // function that updates text (e.g. "q")
	updateOfs  string // function that updates cursor offset (e.g. "v")
	cleanup1   string // first cleanup function (e.g. "XI6")
	cleanup2   string // second cleanup function (e.g. "MI6")
	hasDTB     bool   // whether .deleteTokenBefore() is used in the original
}

// extractVariables extracts dynamic variable names from the minified bug block.
//
// Real-world pattern (Claude Code 2.x):
//
//	if(!J6.backspace&&!J6.delete&&o.includes("\x7f")){
//	  let Y6=(o.match(/\x7f/g)||[]).length,H6=y;
//	  for(let D6=0;D6<Y6;D6++)H6=H6.deleteTokenBefore()??H6.backspace();
//	  if(!y.equals(H6)){if(y.text!==H6.text)q(H6.text);v(H6.offset)}
//	  XI6(),MI6();return}
func extractVariables(block string) (*variables, error) {
	normalized := strings.ReplaceAll(block, delChar, `\x7f`)

	// Extract count/state/curState: let Y6=(o.match(...)).length,H6=y
	reCountState := regexp.MustCompile(
		`let ([\w$]+)=\(\w+\.match\(/\\x7f/g\)\|\|\[\]\)\.length[,;]([\w$]+)=([\w$]+)[;,]`)
	m := reCountState.FindStringSubmatch(normalized)
	if m == nil {
		return nil, errors.New("cannot extract count/state variables from bug block")
	}
	state := m[2]
	curState := m[3]

	// Extract update functions: q(H6.text);v(H6.offset)
	reUpdate := regexp.MustCompile(
		`([\w$]+)\(` + regexp.QuoteMeta(state) + `\.text\);([\w$]+)\(` + regexp.QuoteMeta(state) + `\.offset\)`)
	m2 := reUpdate.FindStringSubmatch(block)
	if m2 == nil {
		return nil, errors.New("cannot extract update functions from bug block")
	}

	// Extract input variable: o.includes("
	reInput := regexp.MustCompile(`([\w$]+)\.includes\("`)
	m3 := reInput.FindStringSubmatch(block)
	if m3 == nil {
		return nil, errors.New("cannot extract input variable from bug block")
	}

	// Extract key info parameter from the guard: !J6.backspace&&!J6.delete
	keyInfo := ""
	reKeyInfo := regexp.MustCompile(`!([\w$]+)\.backspace&&!([\w$]+)\.delete`)
	m4 := reKeyInfo.FindStringSubmatch(block)
	if m4 != nil && m4[1] == m4[2] {
		keyInfo = m4[1]
	}

	// Check if deleteTokenBefore is used
	hasDTB := strings.Contains(block, ".deleteTokenBefore()")

	// Extract cleanup functions: XI6(),MI6()
	// They appear after the update block, before return
	cleanup1, cleanup2 := "", ""
	reCleanup := regexp.MustCompile(
		regexp.QuoteMeta(state) + `\.offset\)}\s*([\w$]+)\(\)[,;]\s*([\w$]+)\(\)`)
	m5 := reCleanup.FindStringSubmatch(block)
	if m5 != nil {
		cleanup1 = m5[1]
		cleanup2 = m5[2]
	}

	return &variables{
		input:      m3[1],
		keyInfo:    keyInfo,
		state:      state,
		curState:   curState,
		updateText: m2[1],
		updateOfs:  m2[2],
		cleanup1:   cleanup1,
		cleanup2:   cleanup2,
		hasDTB:     hasDTB,
	}, nil
}

// generateFix produces the replacement JavaScript code.
//
// It processes each character sequentially:
//   - \x7f → .deleteTokenBefore()??.backspace() (matches original Claude Code behavior)
//   - else → .insert(c) (THIS is the fix — original code drops replacement chars)
//
// IMPORTANT: We use `let _s` (locally scoped) to avoid corrupting any outer-scope
// variable with the same minified name.
func generateFix(v *variables) string {
	// Build the backspace expression
	bsExpr := "_s=_s.backspace();"
	if v.hasDTB {
		bsExpr = "_s=_s.deleteTokenBefore()??_s.backspace();"
	}

	// Build the cleanup calls
	cleanup := ""
	if v.cleanup1 != "" && v.cleanup2 != "" {
		cleanup = v.cleanup1 + "()," + v.cleanup2 + "();"
	}

	// Build the key-info guard
	guard := ""
	if v.keyInfo != "" {
		guard = "!" + v.keyInfo + ".backspace&&!" + v.keyInfo + ".delete&&"
	}

	return fmt.Sprintf(
		`%s`+
			`if(%s%s.includes("\x7f")){`+
			`let _s=%s;`+
			`for(const _c of %s){`+
			`if(_c==="\x7f"){%s}`+
			`else{_s=_s.insert(_c);}`+
			`}`+
			`if(!%s.equals(_s)){`+
			`if(%s.text!==_s.text)`+
			`%s(_s.text);`+
			`%s(_s.offset)`+
			`}%sreturn;}`,
		patchMarker,
		guard,
		v.input,
		v.curState,
		v.input,
		bsExpr,
		v.curState,
		v.curState,
		v.updateText,
		v.updateOfs,
		cleanup,
	)
}

// stripPatch removes the injected block from src when no backup is available.
func stripPatch(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	src := string(data)

	// Try current marker first, then legacy markers
	allMarkers := append([]string{patchMarker}, legacyMarkers...)
	for _, marker := range allMarkers {
		markerIdx := strings.Index(src, marker)
		if markerIdx == -1 {
			continue
		}
		endPattern := "return;}"
		endIdx := strings.Index(src[markerIdx:], endPattern)
		if endIdx == -1 {
			continue
		}
		stripped := src[:markerIdx] + src[markerIdx+endIdx+len(endPattern):]
		return os.WriteFile(path, []byte(stripped), 0o644)
	}
	return errors.New("no patch marker found (current or legacy)")
}
