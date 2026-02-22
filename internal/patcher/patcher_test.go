package patcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Test fixtures ---

// padding creates filler content to ensure there are at least 250 chars
// before the bug block, matching real-world cli.js which is very large.
// This prevents patcher.go's guardSearch (src[blockStart-200:blockStart]) from panicking.
const padding = `"use strict";
/* Claude Code CLI - minified bundle (c) Anthropic */
var __webpack_module_cache__={};function __webpack_require__(e){var n=__webpack_module_cache__[e];if(void 0!==n)return n.exports}
var G=function(a,b){return a};
`

// simulateCliJS creates a fake cli.js file with a minified onInput function
// that contains the bug pattern (batch backspace processing).
// The variable names mimic real Claude Code minified output.
func simulateCliJS(t *testing.T, dir string) string {
	t.Helper()

	// This simulates the real minified cli.js structure:
	// - A guard: if(t===""&&o!=="")return;
	// - The bug block: if(o.includes("\x7f")){...batch processing...}
	// The variable names (o, e3, J6, h7, S5) are representative of real minified code.
	content := padding + `function onInput(o,J6){
let t=G?G(o,J6):o;
if(t===""&&o!=="")return;
if(o.includes("` + "\x7f" + `")){let d4=(o.match(/` + "\x7f" + `/g)||[]).length,e3=J6,q2;for(q2=0;q2<d4;q2++)e3=e3.backspace();let r=o.replaceAll("` + "\x7f" + `","");if(r.length>0)e3=e3.insert(r);if(!J6.equals(e3)){if(J6.text!==e3.text)h7(e3.text);S5(e3.offset)}return;}
console.log("normal input");
}
module.exports={onInput};`

	path := filepath.Join(dir, "cli.js")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// simulateCliJSNoGuard creates a cli.js without the early return guard.
func simulateCliJSNoGuard(t *testing.T, dir string) string {
	t.Helper()

	content := padding + `function onInput(o,J6){
if(o.includes("` + "\x7f" + `")){let d4=(o.match(/` + "\x7f" + `/g)||[]).length,e3=J6,q2;for(q2=0;q2<d4;q2++)e3=e3.backspace();let r=o.replaceAll("` + "\x7f" + `","");if(r.length>0)e3=e3.insert(r);if(!J6.equals(e3)){if(J6.text!==e3.text)h7(e3.text);S5(e3.offset)}return;}
console.log("normal input");
}
module.exports={onInput};`

	path := filepath.Join(dir, "cli.js")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// simulateCliJSAlternateVars creates a cli.js with different variable names.
// Tests that the regex extraction is resilient to variable name changes.
func simulateCliJSAlternateVars(t *testing.T, dir string) string {
	t.Helper()

	// Different var names: input=abc, state=xyz, curState=QR7, updateText=fn1, updateOfs=fn2
	content := padding + `function onInput(abc,QR7){
let t=G?G(abc,QR7):abc;
if(t===""&&abc!=="")return;
if(abc.includes("` + "\x7f" + `")){let cnt=(abc.match(/` + "\x7f" + `/g)||[]).length,xyz=QR7,i;for(i=0;i<cnt;i++)xyz=xyz.backspace();let r=abc.replaceAll("` + "\x7f" + `","");if(r.length>0)xyz=xyz.insert(r);if(!QR7.equals(xyz)){if(QR7.text!==xyz.text)fn1(xyz.text);fn2(xyz.offset)}return;}
console.log("normal input");
}
module.exports={onInput};`

	path := filepath.Join(dir, "cli.js")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// --- Patch Tests ---

func TestPatch_Basic(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	err := Patch(path)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)

	// Must contain patch marker
	if !strings.Contains(src, patchMarker) {
		t.Error("patch marker not found in patched file")
	}

	// Must contain our fix's for-loop pattern
	if !strings.Contains(src, "for(const _c of ") {
		t.Error("fix for-loop not found in patched file")
	}

	// Must still contain the guard
	if !strings.Contains(src, `if(t===""&&o!=="")return;`) {
		t.Error("guard was removed, should be preserved")
	}

	// Must NOT contain the original batch processing (match count pattern)
	if strings.Contains(src, `.match(/`+"\x7f"+`/g)||[]).length`) {
		t.Error("original bug block still present after patching")
	}
}

func TestPatch_NoGuard(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJSNoGuard(t, dir)

	err := Patch(path)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)

	if !strings.Contains(src, patchMarker) {
		t.Error("patch marker not found")
	}
	if !strings.Contains(src, "for(const _c of ") {
		t.Error("fix for-loop not found")
	}
}

func TestPatch_AlternateVariableNames(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJSAlternateVars(t, dir)

	err := Patch(path)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)

	if !strings.Contains(src, patchMarker) {
		t.Error("patch marker not found")
	}

	// Verify the fix uses correct input/update var names but local _s for state
	if !strings.Contains(src, "for(const _c of abc)") {
		t.Error("fix should use input var 'abc'")
	}
	if !strings.Contains(src, "let _s=QR7") {
		t.Error("fix should declare local _s from curState 'QR7'")
	}
	if !strings.Contains(src, "_s=_s.backspace()") {
		t.Error("fix should use local _s for backspace")
	}
	if !strings.Contains(src, "fn1(_s.text)") {
		t.Error("fix should use updateText func 'fn1' with _s")
	}
	if !strings.Contains(src, "fn2(_s.offset)") {
		t.Error("fix should use updateOfs func 'fn2' with _s")
	}
}

func TestPatch_AlreadyPatched(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	// Patch once
	if err := Patch(path); err != nil {
		t.Fatalf("first patch failed: %v", err)
	}

	// Patch again - should return ErrAlreadyPatched
	err := Patch(path)
	if err != ErrAlreadyPatched {
		t.Errorf("expected ErrAlreadyPatched, got: %v", err)
	}
}

func TestPatch_CreatesBackup(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	original, _ := os.ReadFile(path)

	if err := Patch(path); err != nil {
		t.Fatalf("patch failed: %v", err)
	}

	backup, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal("backup file not created")
	}

	if string(backup) != string(original) {
		t.Error("backup content does not match original")
	}
}

func TestIsPatched(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	if IsPatched(path) {
		t.Error("unpatched file reported as patched")
	}

	if err := Patch(path); err != nil {
		t.Fatal(err)
	}

	if !IsPatched(path) {
		t.Error("patched file reported as unpatched")
	}
}

func TestIsPatched_NonExistent(t *testing.T) {
	if IsPatched("/nonexistent/path/cli.js") {
		t.Error("non-existent file reported as patched")
	}
}

func TestRestore(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	original, _ := os.ReadFile(path)

	if err := Patch(path); err != nil {
		t.Fatal(err)
	}

	// File should be changed
	patched, _ := os.ReadFile(path)
	if string(patched) == string(original) {
		t.Fatal("file was not modified by patch")
	}

	// Restore
	if err := Restore(path); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	// File should match original
	restored, _ := os.ReadFile(path)
	if string(restored) != string(original) {
		t.Error("restored content does not match original")
	}
}

func TestRestore_NoBackup_StripsMarker(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	if err := Patch(path); err != nil {
		t.Fatal(err)
	}

	// Delete backup
	os.Remove(path + ".bak")

	// Restore should strip the injected block
	if err := Restore(path); err != nil {
		t.Fatalf("restore (strip) failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	// Marker should be gone
	if strings.Contains(string(data), patchMarker) {
		t.Error("patch marker still present after strip-restore")
	}
}

// --- findBugBlock Tests ---

func TestFindBugBlock_Found(t *testing.T) {
	src := `some code before;if(o.includes("` + "\x7f" + `")){let d=(o.match(/` + "\x7f" + `/g)||[]).length;return;}after`
	start, end, block, err := findBugBlock(src)
	if err != nil {
		t.Fatalf("findBugBlock failed: %v", err)
	}
	if start >= end {
		t.Errorf("invalid range: start=%d end=%d", start, end)
	}
	if !strings.Contains(block, ".includes(") {
		t.Error("block does not contain .includes()")
	}
}

func TestFindBugBlock_NotFound(t *testing.T) {
	src := `normal code without the pattern`
	_, _, _, err := findBugBlock(src)
	if err == nil {
		t.Error("expected error for missing pattern, got nil")
	}
}

// --- extractVariables Tests ---

func TestExtractVariables_StandardBlock(t *testing.T) {
	// Simulate a realistic minified bug block
	block := `if(o.includes("` + "\x7f" + `")){let d4=(o.match(/` + "\x7f" + `/g)||[]).length,e3=J6,q2;for(q2=0;q2<d4;q2++)e3=e3.backspace();let r=o.replaceAll("` + "\x7f" + `","");if(r.length>0)e3=e3.insert(r);if(!J6.equals(e3)){if(J6.text!==e3.text)h7(e3.text);S5(e3.offset)}return;}`

	vars, err := extractVariables(block)
	if err != nil {
		t.Fatalf("extractVariables failed: %v", err)
	}

	if vars.input != "o" {
		t.Errorf("expected input='o', got '%s'", vars.input)
	}
	if vars.state != "e3" {
		t.Errorf("expected state='e3', got '%s'", vars.state)
	}
	if vars.curState != "J6" {
		t.Errorf("expected curState='J6', got '%s'", vars.curState)
	}
	if vars.updateText != "h7" {
		t.Errorf("expected updateText='h7', got '%s'", vars.updateText)
	}
	if vars.updateOfs != "S5" {
		t.Errorf("expected updateOfs='S5', got '%s'", vars.updateOfs)
	}
}

func TestExtractVariables_DifferentNames(t *testing.T) {
	// Different minification output
	block := `if(abc.includes("` + "\x7f" + `")){let cnt=(abc.match(/` + "\x7f" + `/g)||[]).length,xyz=QR7,i;for(i=0;i<cnt;i++)xyz=xyz.backspace();let r=abc.replaceAll("` + "\x7f" + `","");if(r.length>0)xyz=xyz.insert(r);if(!QR7.equals(xyz)){if(QR7.text!==xyz.text)fn1(xyz.text);fn2(xyz.offset)}return;}`

	vars, err := extractVariables(block)
	if err != nil {
		t.Fatalf("extractVariables failed: %v", err)
	}

	if vars.input != "abc" {
		t.Errorf("expected input='abc', got '%s'", vars.input)
	}
	if vars.state != "xyz" {
		t.Errorf("expected state='xyz', got '%s'", vars.state)
	}
	if vars.curState != "QR7" {
		t.Errorf("expected curState='QR7', got '%s'", vars.curState)
	}
	if vars.updateText != "fn1" {
		t.Errorf("expected updateText='fn1', got '%s'", vars.updateText)
	}
	if vars.updateOfs != "fn2" {
		t.Errorf("expected updateOfs='fn2', got '%s'", vars.updateOfs)
	}
}

// --- generateFix Tests ---

func TestGenerateFix_ContainsMarker(t *testing.T) {
	v := &variables{
		input: "o", state: "e3", curState: "J6",
		updateText: "h7", updateOfs: "S5",
	}
	fix := generateFix(v)

	if !strings.HasPrefix(fix, patchMarker) {
		t.Error("fix does not start with patch marker")
	}
}

func TestGenerateFix_SequentialProcessing(t *testing.T) {
	v := &variables{
		input: "o", state: "e3", curState: "J6",
		updateText: "h7", updateOfs: "S5",
	}
	fix := generateFix(v)

	// Must iterate over each character
	if !strings.Contains(fix, "for(const _c of o)") {
		t.Error("fix must iterate over each char of input")
	}

	// Must handle backspace one-by-one
	if !strings.Contains(fix, `_c==="\x7f"`) {
		t.Error("fix must check each char for \\x7f")
	}

	// Must call .backspace() on local _s variable
	if !strings.Contains(fix, "_s=_s.backspace()") {
		t.Error("fix must call .backspace() on local _s var")
	}

	// Must call .insert() on local _s variable
	if !strings.Contains(fix, "_s=_s.insert(_c)") {
		t.Error("fix must call .insert() on local _s var")
	}
}

func TestGenerateFix_UpdatesTextAndOffset(t *testing.T) {
	v := &variables{
		input: "o", state: "e3", curState: "J6",
		updateText: "h7", updateOfs: "S5",
	}
	fix := generateFix(v)

	if !strings.Contains(fix, "h7(_s.text)") {
		t.Error("fix must call updateText with _s.text")
	}
	if !strings.Contains(fix, "S5(_s.offset)") {
		t.Error("fix must call updateOfs with _s.offset")
	}
}

func TestGenerateFix_ComparesWithSnapshot(t *testing.T) {
	v := &variables{
		input: "o", state: "e3", curState: "J6",
		updateText: "h7", updateOfs: "S5",
	}
	fix := generateFix(v)

	// Must compare snapshot (curState) with local _s
	if !strings.Contains(fix, "!J6.equals(_s)") {
		t.Error("fix must compare curState.equals(_s)")
	}
}

func TestGenerateFix_EndsWithReturn(t *testing.T) {
	v := &variables{
		input: "o", state: "e3", curState: "J6",
		updateText: "h7", updateOfs: "S5",
	}
	fix := generateFix(v)

	if !strings.HasSuffix(fix, "return;}") {
		t.Error("fix must end with return;}")
	}
}

// --- TELEX Edge Case Tests ---

// TestPatch_FixPreservesNormalInput ensures normal (non-\x7f) input
// paths are not affected by the patch.
func TestPatch_FixPreservesNormalInput(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	if err := Patch(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)

	// Normal input path should still exist
	if !strings.Contains(src, `console.log("normal input")`) {
		t.Error("normal input code path was corrupted by patch")
	}
}

// TestPatch_PatchReapplyAfterRestore tests the full lifecycle:
// patch -> restore -> patch again
func TestPatch_PatchReapplyAfterRestore(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	// Patch
	if err := Patch(path); err != nil {
		t.Fatalf("first patch: %v", err)
	}

	// Restore
	if err := Restore(path); err != nil {
		t.Fatalf("restore: %v", err)
	}

	// Should be unpatchable again
	if IsPatched(path) {
		t.Error("file still appears patched after restore")
	}

	// Re-patch
	if err := Patch(path); err != nil {
		t.Fatalf("re-patch: %v", err)
	}

	if !IsPatched(path) {
		t.Error("file not patched after re-patch")
	}
}

// TestClaudeVersion_ParsesVersion tests version extraction from package.json.
func TestClaudeVersion_ParsesVersion(t *testing.T) {
	dir := t.TempDir()
	pkgContent := `{"name":"@anthropic-ai/claude-code","version":"1.0.42","description":"Claude Code CLI"}`
	pkgPath := filepath.Join(dir, "package.json")
	if err := os.WriteFile(pkgPath, []byte(pkgContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cliPath := filepath.Join(dir, "cli.js")
	ver := ClaudeVersion(cliPath)
	if ver != "1.0.42" {
		t.Errorf("expected version '1.0.42', got '%s'", ver)
	}
}

func TestClaudeVersion_MissingPackageJSON(t *testing.T) {
	ver := ClaudeVersion("/nonexistent/cli.js")
	if ver != "unknown" {
		t.Errorf("expected 'unknown', got '%s'", ver)
	}
}

// TestPatch_MultipleBackspaces verifies the generated fix handles
// sequences like ⌫⌫⌫ + char correctly (the core TELEX use case).
// This is a structural test: we verify the fix code correctly templates
// the sequential processing pattern.
func TestPatch_MultipleBackspaces(t *testing.T) {
	v := &variables{
		input: "o", state: "e3", curState: "J6",
		updateText: "h7", updateOfs: "S5",
	}
	fix := generateFix(v)

	// The fix must NOT batch-count backspaces
	if strings.Contains(fix, ".length") {
		t.Error("fix should not count backspaces with .length (batch mode)")
	}

	// The fix must NOT use .match() or .replaceAll()
	if strings.Contains(fix, ".match(") {
		t.Error("fix should not use .match() (batch mode)")
	}
	if strings.Contains(fix, ".replaceAll(") {
		t.Error("fix should not use .replaceAll() (batch mode)")
	}

	// The fix MUST use for..of to iterate character by character
	if !strings.Contains(fix, "for(const _c of ") {
		t.Error("fix must use for..of for character-by-character processing")
	}
}

// TestPatch_FixNotCorruptSurroundings ensures patching doesn't corrupt
// the file structure beyond the targeted block.
func TestPatch_FixNotCorruptSurroundings(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	if err := Patch(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)

	// File header
	if !strings.Contains(src, `"use strict"`) {
		t.Error("file header corrupted")
	}

	// Module export
	if !strings.Contains(src, `module.exports={onInput}`) {
		t.Error("module export corrupted")
	}

	// Opening function
	if !strings.Contains(src, "function onInput(") {
		t.Error("function declaration corrupted")
	}
}

// TestPatch_EmptyInputString verifies the fix handles the edge case where
// the input string is empty (no \x7f chars but includes() was checked).
func TestPatch_EmptyInputString(t *testing.T) {
	v := &variables{
		input: "o", state: "e3", curState: "J6",
		updateText: "h7", updateOfs: "S5",
	}
	fix := generateFix(v)

	// The fix starts with an includes check, so empty string won't enter the block
	if !strings.Contains(fix, `if(o.includes("\x7f"))`) {
		t.Error("fix must guard with .includes() check")
	}
}

// TestPatch_SingleCharBackspace tests that a single \x7f is handled correctly
// (simplest TELEX case: type one wrong char then correct it).
func TestPatch_SingleCharBackspace(t *testing.T) {
	v := &variables{
		input: "inp", state: "st", curState: "cur",
		updateText: "ut", updateOfs: "uo",
	}
	fix := generateFix(v)

	// With a single \x7f in the for..of loop:
	// - _c === "\x7f" -> _s = _s.backspace()
	// That's it. No batch counting needed.
	if !strings.Contains(fix, "_s=_s.backspace()") {
		t.Error("fix must handle single backspace via _s.backspace()")
	}
}

// TestPatch_MixedCharsAndBackspaces tests the critical TELEX scenario:
// IME sends ⌫⌫ + "ạ" (delete 2 chars, insert accented char).
// The fix must process: backspace, backspace, insert('ạ') in sequence.
func TestPatch_MixedCharsAndBackspaces(t *testing.T) {
	v := &variables{
		input: "data", state: "s", curState: "snap",
		updateText: "uT", updateOfs: "uO",
	}
	fix := generateFix(v)

	// Verify sequential processing pattern: backspace OR insert for each char
	if !strings.Contains(fix, `if(_c==="\x7f"){_s=_s.backspace();}else{_s=_s.insert(_c);}`) {
		t.Error("fix must handle mixed ⌫ and chars with if/else per character")
	}
}
