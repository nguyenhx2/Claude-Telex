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
const patchMarker = "/* Vietnamese IME fix v9 */"

// We keep legacy markers so `Restore()` can strip older patches
var legacyMarkers = []string{
	"/* Vietnamese IME fix */",
	"/* Vietnamese IME fix v2 */",
	"/* Vietnamese IME fix v3 */",
	"/* Vietnamese IME fix v4 */",
	"/* Vietnamese IME fix v5 */",
	"/* Vietnamese IME fix v6 */",
	"/* Vietnamese IME fix v7 */",
	"/* Vietnamese IME fix v8 */",
}

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
// Root cause: G6 (onInput handler) reads `y` (cursor state) from a React
// closure captured at component render time. When Gõ Nhanh sends a burst
// (N × \x7f + replacement chars), Ink splits them into individual key events.
// React defers re-rendering, so `y` remains stale across all G6 calls
// within one burst.
//
// Strategy (v9 — engine-aware):
//  1. stateSync preamble: restore `y` from globalThis.__imeState, clear on
//     control keys (detected as o==="" && !backspace)
//  2. Raw \x7f handler: process the full IME burst atomically
//  3. Backspace interception: handle Ink-split \x7f events with state bridge
//  4. Normal path inherits correct `y` from stateSync — no regex injection
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

	// Step 1: Find the bug block to extract variable names and key-info var
	blockStart, blockEnd, block, err := findBugBlock(src)
	if err != nil {
		return err
	}

	// Step 2: Extract dynamic variable names (including key-info var)
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
	if guardIdx > -1 {
		// Guard found: Inject before the guard
		before := src[:blockStart-200+guardIdx]
		// Skip the guard and the original bug block
		after := src[blockEnd:]
		patched = before + fixCode + guardSearch[guardIdx:] + after
	} else {
		// Guard not found: Just replace the bug block
		patched = src[:blockStart] + fixCode + src[blockEnd:]
	}

	// Step 5: (Removed in v9 — stateSync preamble handles recovery)
	// Step 6: (Removed in v9 — no more fragile regex injection into Claude's code)

	// Write patched file to disk
	if err := os.WriteFile(path, []byte(patched), 0o644); err != nil {
		return fmt.Errorf("write patched cli.js: %w", err)
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
	keyInfo    string // the key info parameter (e.g. "J6") with backspace/ctrl/meta
	state      string // the mutable state variable (e.g. "H6")
	curState   string // the current (snapshot) state variable (e.g. "y")
	updateText string // function that updates text (e.g. "q")
	updateOfs  string // function that updates cursor offset (e.g. "v")
	cleanup1   string // first cleanup function (e.g. "XI6")
	cleanup2   string // second cleanup function (e.g. "MI6")
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

	// NOTE: v7 now DOES extract the key info parameter (e.g. J6) because
	// the backspace interception handler needs to check J6.backspace, J6.ctrl, J6.meta.
	reKeyInfo := regexp.MustCompile(`if\(!([\w$]+)\.backspace`)
	m4 := reKeyInfo.FindStringSubmatch(block)
	keyInfo := ""
	if m4 != nil {
		keyInfo = m4[1]
	}

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
	}, nil
}

// generateFix produces the replacement JavaScript code.
//
// It contains TWO sequential handlers:
//  1. Raw \x7f path: for when IME data arrives as a single chunk containing \x7f.
//     Processes each character sequentially: \x7f → .backspace(), else → .insert(c).
//  2. Backspace interception: for when Ink parses \x7f as backspace
//     and sets backspace=true, input="". We handle backspace ourselves with state
//     bridge to maintain coherence across the IME burst.
//
// IMPORTANT: We use `let _s` (locally scoped) to avoid corrupting any outer-scope
// variable with the same minified name.
func generateFix(v *variables) string {
	bsExpr := "_s=_s.backspace();"
	// Bridge stores {s: EditorState, t: creationTimestamp}.
	// On UPDATE, preserve the original creation time so the 200ms expiry
	// counts from when the burst STARTED, not from each event.
	bridge := "globalThis.__imeState={s:_s,t:globalThis.__imeState?globalThis.__imeState.t:Date.now()};"

	// Build the cleanup calls
	cleanup := ""
	if v.cleanup1 != "" && v.cleanup2 != "" {
		cleanup = v.cleanup1 + "()," + v.cleanup2 + "();"
	}

	// stateSync preamble: restore or clear globalThis.__imeState.
	//
	// CRITICAL: Do NOT use text comparison (y.text === __imeState.s.text)
	// to detect React catch-up. After bsHandler sets both y=_s and
	// __imeState.s=_s, they're the SAME object — comparison always
	// returns true, which would prematurely clear the bridge mid-burst.
	//
	// Instead, use ONLY:
	// 1. Timestamp expiry (>200ms) — IME bursts complete in ~10ms
	// 2. Control key detection — Enter, arrows, Escape, etc.

	stateSync := fmt.Sprintf(
		`if(globalThis.__imeState){`+
			`if(Date.now()-globalThis.__imeState.t>200)globalThis.__imeState=null;`+
			`else if(%s.length===0&&!%s.backspace)globalThis.__imeState=null;`+
			`else %s=globalThis.__imeState.s;`+
			`}`,
		v.input, v.keyInfo, v.curState,
	)

	// Part 1: Raw \x7f handler (when IME data arrives as one chunk)
	rawHandler := fmt.Sprintf(
		`if(%s.includes("\x7f")){`+
			`let _s=%s;`+
			`for(const _c of %s){`+
			`if(_c==="\x7f"){%s}`+
			`else{_s=_s.insert(_c);}`+
			`}`+
			`if(!%s.equals(_s)){`+
			`if(%s.text!==_s.text)%s(_s.text);`+
			`%s(_s.offset);`+
			`%s=_s;`+
			`%s`+ // Bridge
			`}%sreturn;}`,
		v.input,
		v.curState,
		v.input,
		bsExpr,
		v.curState,
		v.curState,
		v.updateText,
		v.updateOfs,
		v.curState,
		bridge,
		cleanup,
	)

	// Part 2: Backspace interception (when Ink parses \x7f as backspace)
	// Ink sets J6.backspace=true and input="" for \x7f bytes.
	// We handle this BEFORE the normal path to maintain state coherence.
	// If keyInfo was not extracted, skip — fall back to v6 behavior.
	bsHandler := ""
	if v.keyInfo != "" {
		bsHandler = fmt.Sprintf(
			`if(%s.backspace&&!%s.ctrl&&!%s.meta){`+
				`let _s=%s.backspace();`+
				`if(!%s.equals(_s)){`+
				`if(%s.text!==_s.text)`+
				`%s(_s.text);`+
				`%s(_s.offset);`+
				`%s=_s;`+
				`%s`+ // Bridge
				`}%sreturn;}`,
			v.keyInfo, v.keyInfo, v.keyInfo,
			v.curState,
			v.curState,
			v.curState,
			v.updateText,
			v.updateOfs,
			v.curState,
			bridge,
			cleanup,
		)
	}

	// Part 3: Char insertion during IME burst (when Ink splits replacement chars)
	//
	// When __imeState is active, replacement characters (e.g. "ất" from "rất")
	// must NOT fall through to P6(J6)(t) — P6 captures state from the React
	// closure, which is STALE during the burst. Instead, we insert directly
	// using y.insert() with the correct bridged state from __imeState.
	charHandler := ""
	if v.keyInfo != "" {
		charHandler = fmt.Sprintf(
			`if(globalThis.__imeState&&%s.length>0&&!%s.backspace){`+
				`let _s=%s;`+
				`for(const _c of %s){_s=_s.insert(_c);}`+
				`if(!%s.equals(_s)){`+
				`if(%s.text!==_s.text)%s(_s.text);`+
				`%s(_s.offset);`+
				`%s=_s;`+
				`%s`+ // Bridge
				`}%sreturn;}`,
			v.input, v.keyInfo,
			v.curState,
			v.input,
			v.curState,
			v.curState,
			v.updateText,
			v.updateOfs,
			v.curState,
			bridge,
			cleanup,
		)
	}

	return patchMarker + stateSync + rawHandler + bsHandler + charHandler
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
