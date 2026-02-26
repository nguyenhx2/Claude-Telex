package patcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Test fixtures ---

// padding creates filler content before the bug block (matches real cli.js structure).
const padding = `"use strict";
/* Claude Code CLI - minified bundle (c) Anthropic */
var __webpack_module_cache__={};function __webpack_require__(e){var n=__webpack_module_cache__[e];if(void 0!==n)return n.exports}
var G=function(a,b){return a};
`

// simulateCliJS creates a realistic minified cli.js that matches the ACTUAL
// Claude Code 2.x structure: deleteTokenBefore, cleanup functions, key-info guard.
func simulateCliJS(t *testing.T, dir string) string {
	t.Helper()
	content := padding + `function onInput(o,J6){
let t=G?G(o,J6):o;
if(t===""&&o!=="")return;
if(!J6.backspace&&!J6.delete&&o.includes("` + "\x7f" + `")){let d4=(o.match(/` + "\x7f" + `/g)||[]).length,e3=y;for(let q2=0;q2<d4;q2++)e3=e3.deleteTokenBefore()??e3.backspace();if(!y.equals(e3)){if(y.text!==e3.text)h7(e3.text);S5(e3.offset)}XI6(),MI6();return}
if(!N6(J6,t))XI6();if(!R6(J6,t))MI6();let c=P6(J6)(t);if(c){if(!y.equals(c)){if(y.text!==c.text)h7(c.text);S5(c.offset)}}
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
if(!J6.backspace&&!J6.delete&&o.includes("` + "\x7f" + `")){let d4=(o.match(/` + "\x7f" + `/g)||[]).length,e3=y;for(let q2=0;q2<d4;q2++)e3=e3.deleteTokenBefore()??e3.backspace();if(!y.equals(e3)){if(y.text!==e3.text)h7(e3.text);S5(e3.offset)}XI6(),MI6();return}
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
func simulateCliJSAlternateVars(t *testing.T, dir string) string {
	t.Helper()
	content := padding + `function onInput(abc,QR7){
let t=G?G(abc,QR7):abc;
if(t===""&&abc!=="")return;
if(!QR7.backspace&&!QR7.delete&&abc.includes("` + "\x7f" + `")){let cnt=(abc.match(/` + "\x7f" + `/g)||[]).length,xyz=snap;for(let i=0;i<cnt;i++)xyz=xyz.deleteTokenBefore()??xyz.backspace();if(!snap.equals(xyz)){if(snap.text!==xyz.text)fn1(xyz.text);fn2(xyz.offset)}CL1(),CL2();return}
console.log("normal input");
}
module.exports={onInput};`

	path := filepath.Join(dir, "cli.js")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// simulateCliJSLegacy creates a cli.js with the OLD (v1) patch applied.
func simulateCliJSLegacy(t *testing.T, dir string) string {
	t.Helper()
	// This simulates a file that was patched with the old v1 code
	content := padding + `function onInput(o,J6){
let t=G?G(o,J6):o;
/* Vietnamese IME fix */if(o.includes("` + "\x7f" + `")){let _s=y;for(const _c of o){if(_c==="` + "\x7f" + `"){_s=_s.backspace();}else{_s=_s.insert(_c);}}if(!y.equals(_s)){if(y.text!==_s.text)h7(_s.text);S5(_s.offset)}return;}if(t===""&&o!=="")return;
if(!N6(J6,t))XI6();if(!R6(J6,t))MI6();let c=P6(J6)(t);if(c){if(!y.equals(c)){if(y.text!==c.text)h7(c.text);S5(c.offset)}}
}
module.exports={onInput};`

	// Also write a backup with the original (unpatched) code
	bakContent := padding + `function onInput(o,J6){
let t=G?G(o,J6):o;
if(t===""&&o!=="")return;
if(!J6.backspace&&!J6.delete&&o.includes("` + "\x7f" + `")){let d4=(o.match(/` + "\x7f" + `/g)||[]).length,e3=y;for(let q2=0;q2<d4;q2++)e3=e3.deleteTokenBefore()??e3.backspace();if(!y.equals(e3)){if(y.text!==e3.text)h7(e3.text);S5(e3.offset)}XI6(),MI6();return}
if(!N6(J6,t))XI6();if(!R6(J6,t))MI6();let c=P6(J6)(t);if(c){if(!y.equals(c)){if(y.text!==c.text)h7(c.text);S5(c.offset)}}
}
module.exports={onInput};`

	path := filepath.Join(dir, "cli.js")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".bak", []byte(bakContent), 0o644); err != nil {
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

	if !strings.Contains(src, patchMarker) {
		t.Error("patch marker not found in patched file")
	}

	if !strings.Contains(src, "for(const _c of ") {
		t.Error("fix for-loop not found in patched file")
	}

	// Must still contain the guard
	if !strings.Contains(src, `if(t===""&&o!=="")return;`) {
		t.Error("guard was removed, should be preserved")
	}

	// Must NOT contain the original batch processing
	if strings.Contains(src, `.match(/`+"\x7f"+`/g)||[]).length`) {
		t.Error("original bug block still present after patching")
	}
}

// TestPatch_UsesBackspaceOnly verifies the fix uses .backspace() per \x7f,
// NOT deleteTokenBefore() which deletes whole words (like Ctrl+W) and overshoots
// when Vietnamese IME sends multiple \x7f chars for tone mark corrections.
func TestPatch_UsesBackspaceOnly(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	if err := Patch(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)

	// Must use plain backspace() — one char per \x7f
	if !strings.Contains(src, "_s=_s.backspace();") {
		t.Error("fix must use _s.backspace() for each \\x7f")
	}
	// Must NOT use deleteTokenBefore (word-level delete, overshoots for Vietnamese)
	if strings.Contains(src, "deleteTokenBefore") {
		t.Error("fix must NOT use deleteTokenBefore (deletes whole words, breaks Vietnamese tone marks)")
	}
}

func TestPatch_CallsCleanupFunctions(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	if err := Patch(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)

	// Must call cleanup functions XI6(),MI6() from the original
	if !strings.Contains(src, "XI6(),MI6();") {
		t.Error("fix must call cleanup functions from original code")
	}
}

// TestPatch_NoKeyInfoGuard verifies the fix does NOT include the J6.backspace guard.
// Ink's key parser sets backspace=true for standalone \x7f bytes, which blocks the fix
// when Node.js splits the IME's multi-char input into separate events.
func TestPatch_NoKeyInfoGuard(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	if err := Patch(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)

	// Must NOT have the key-info guard — it blocks the fix for IME events
	if strings.Contains(src, "!J6.backspace&&!J6.delete&&") {
		t.Error("fix must NOT include key-info guard (blocks IME backspace events)")
	}
	// Must still include the \x7f check as the only guard
	if !strings.Contains(src, `o.includes("`) {
		t.Error("fix must check o.includes as its only guard")
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

	if !strings.Contains(src, "for(const _c of abc)") {
		t.Error("fix should use input var 'abc'")
	}
	if !strings.Contains(src, "let _s=snap") {
		t.Error("fix should declare local _s from curState 'snap'")
	}
	if !strings.Contains(src, "_s=_s.backspace();") {
		t.Error("fix should use backspace() for each \\x7f")
	}
	if !strings.Contains(src, "fn1(_s.text)") {
		t.Error("fix should use updateText func 'fn1'")
	}
	if !strings.Contains(src, "fn2(_s.offset)") {
		t.Error("fix should use updateOfs func 'fn2'")
	}
	if !strings.Contains(src, "CL1(),CL2()") {
		t.Error("fix should use cleanup funcs 'CL1','CL2'")
	}
	// v7+: backspace interception SHOULD use key-info (QR7) for backspace handler
	if !strings.Contains(src, "QR7.backspace&&!QR7.ctrl&&!QR7.meta") {
		t.Error("fix must include backspace interception using extracted keyInfo")
	}
}

func TestPatch_AlreadyPatched(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	if err := Patch(path); err != nil {
		t.Fatalf("first patch failed: %v", err)
	}

	err := Patch(path)
	if err != ErrAlreadyPatched {
		t.Errorf("expected ErrAlreadyPatched, got: %v", err)
	}
}

func TestPatch_LegacyAutoUpgrade(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJSLegacy(t, dir)

	// File has old v1 patch. Calling Patch should auto-upgrade.
	err := Patch(path)
	if err != nil {
		t.Fatalf("legacy auto-upgrade patch failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)

	// Should have new marker
	if !strings.Contains(src, patchMarker) {
		t.Error("new patch marker not found after upgrade")
	}
	// Should NOT have old marker
	if strings.Contains(src, "/* Vietnamese IME fix */if") {
		t.Error("old patch marker still present after upgrade")
	}
	// Should use backspace() not deleteTokenBefore
	if !strings.Contains(src, "_s=_s.backspace();") {
		t.Error("upgraded patch should use backspace()")
	}
	if strings.Contains(src, "deleteTokenBefore") {
		t.Error("upgraded patch must not use deleteTokenBefore")
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

	patched, _ := os.ReadFile(path)
	if string(patched) == string(original) {
		t.Fatal("file was not modified by patch")
	}

	if err := Restore(path); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

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

	os.Remove(path + ".bak")

	if err := Restore(path); err != nil {
		t.Fatalf("restore (strip) failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), patchMarker) {
		t.Error("patch marker still present after strip-restore")
	}
}

// --- findBugBlock Tests ---

func TestFindBugBlock_Found(t *testing.T) {
	src := `some code before;if(!J6.backspace&&!J6.delete&&o.includes("` + "\x7f" + `")){let d=(o.match(/` + "\x7f" + `/g)||[]).length;return;}after`
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
	block := `if(!J6.backspace&&!J6.delete&&o.includes("` + "\x7f" + `")){let d4=(o.match(/` + "\x7f" + `/g)||[]).length,e3=y;for(let q2=0;q2<d4;q2++)e3=e3.deleteTokenBefore()??e3.backspace();if(!y.equals(e3)){if(y.text!==e3.text)h7(e3.text);S5(e3.offset)}XI6(),MI6();return}`

	vars, err := extractVariables(block)
	if err != nil {
		t.Fatalf("extractVariables failed: %v", err)
	}

	if vars.input != "o" {
		t.Errorf("expected input='o', got '%s'", vars.input)
	}
	if vars.keyInfo != "J6" {
		t.Errorf("expected keyInfo='J6', got '%s'", vars.keyInfo)
	}
	// keyInfo is extracted (v7: backspace interception uses it)
	if vars.state != "e3" {
		t.Errorf("expected state='e3', got '%s'", vars.state)
	}
	if vars.curState != "y" {
		t.Errorf("expected curState='y', got '%s'", vars.curState)
	}
	if vars.updateText != "h7" {
		t.Errorf("expected updateText='h7', got '%s'", vars.updateText)
	}
	if vars.updateOfs != "S5" {
		t.Errorf("expected updateOfs='S5', got '%s'", vars.updateOfs)
	}
	if vars.cleanup1 != "XI6" {
		t.Errorf("expected cleanup1='XI6', got '%s'", vars.cleanup1)
	}
	if vars.cleanup2 != "MI6" {
		t.Errorf("expected cleanup2='MI6', got '%s'", vars.cleanup2)
	}
}

func TestExtractVariables_DifferentNames(t *testing.T) {
	block := `if(!QR7.backspace&&!QR7.delete&&abc.includes("` + "\x7f" + `")){let cnt=(abc.match(/` + "\x7f" + `/g)||[]).length,xyz=snap;for(let i=0;i<cnt;i++)xyz=xyz.deleteTokenBefore()??xyz.backspace();if(!snap.equals(xyz)){if(snap.text!==xyz.text)fn1(xyz.text);fn2(xyz.offset)}CL1(),CL2();return}`

	vars, err := extractVariables(block)
	if err != nil {
		t.Fatalf("extractVariables failed: %v", err)
	}

	if vars.input != "abc" {
		t.Errorf("expected input='abc', got '%s'", vars.input)
	}
	if vars.keyInfo != "QR7" {
		t.Errorf("expected keyInfo='QR7', got '%s'", vars.keyInfo)
	}
	// keyInfo is extracted (v7: backspace interception uses it)
	if vars.state != "xyz" {
		t.Errorf("expected state='xyz', got '%s'", vars.state)
	}
	if vars.curState != "snap" {
		t.Errorf("expected curState='snap', got '%s'", vars.curState)
	}
	if vars.cleanup1 != "CL1" {
		t.Errorf("expected cleanup1='CL1', got '%s'", vars.cleanup1)
	}
	if vars.cleanup2 != "CL2" {
		t.Errorf("expected cleanup2='CL2', got '%s'", vars.cleanup2)
	}
}

// --- generateFix Tests ---

func TestGenerateFix_ContainsMarker(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6",
	}
	fix := generateFix(v)

	if !strings.HasPrefix(fix, patchMarker) {
		t.Error("fix does not start with patch marker")
	}
}

func TestGenerateFix_SequentialProcessing(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6",
	}
	fix := generateFix(v)

	if !strings.Contains(fix, "for(const _c of o)") {
		t.Error("fix must iterate over each char of input")
	}

	if !strings.Contains(fix, `_c==="\x7f"`) {
		t.Error("fix must check each char for \\x7f")
	}

	// Must use plain backspace — one char per \x7f (NOT deleteTokenBefore which deletes whole words)
	if !strings.Contains(fix, "_s=_s.backspace();") {
		t.Error("fix must use _s.backspace() per \\x7f")
	}
	if strings.Contains(fix, "deleteTokenBefore") {
		t.Error("fix must NOT use deleteTokenBefore")
	}

	if !strings.Contains(fix, "_s=_s.insert(_c)") {
		t.Error("fix must call .insert() for non-backspace chars")
	}
}

func TestGenerateFix_UpdatesTextAndOffset(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6",
	}
	fix := generateFix(v)

	if !strings.Contains(fix, "h7(_s.text)") {
		t.Error("fix must call updateText with _s.text")
	}
	if !strings.Contains(fix, "S5(_s.offset)") {
		t.Error("fix must call updateOfs with _s.offset")
	}
	if !strings.Contains(fix, "y=_s;") {
		t.Error("fix must mutate curState to _s")
	}
}

func TestGenerateFix_ComparesWithSnapshot(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6",
	}
	fix := generateFix(v)

	if !strings.Contains(fix, "!y.equals(_s)") {
		t.Error("fix must compare curState.equals(_s)")
	}
}

func TestGenerateFix_EndsWithReturn(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6",
	}
	fix := generateFix(v)

	if !strings.HasSuffix(fix, "return;}") {
		t.Error("fix must end with return;}")
	}
}

func TestGenerateFix_IncludesCleanup(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6",
	}
	fix := generateFix(v)

	if !strings.Contains(fix, "XI6(),MI6();") {
		t.Error("fix must call cleanup functions")
	}
}

// TestGenerateFix_NoOriginalGuard verifies the fix does NOT include the ORIGINAL key-info guard pattern.
// The original pattern is `!J6.backspace&&!J6.delete&&` which BLOCKS the fix for IME events.
// v7 uses a DIFFERENT pattern: `J6.backspace&&!J6.ctrl&&!J6.meta` (positive, not negative).
func TestGenerateFix_NoOriginalGuard(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6",
	}
	fix := generateFix(v)

	// Must NOT have the ORIGINAL key-info guard (it blocks IME backspace events)
	if strings.Contains(fix, "!J6.backspace&&!J6.delete&&") {
		t.Error("fix must NOT include original key-info guard pattern")
	}
	// Must have the includes check for raw \x7f
	if !strings.Contains(fix, `o.includes("\x7f")`) {
		t.Error("fix must gate on o.includes for raw \\x7f")
	}
	// v7: Must have the POSITIVE backspace check for interception
	if !strings.Contains(fix, "J6.backspace&&!J6.ctrl&&!J6.meta") {
		t.Error("fix must include v7 backspace interception")
	}
}

// TestGenerateFix_AlwaysBackspaceOnly verifies we NEVER use deleteTokenBefore.
func TestGenerateFix_AlwaysBackspaceOnly(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "", cleanup2: "",
	}
	fix := generateFix(v)

	if !strings.Contains(fix, "_s=_s.backspace();") {
		t.Error("fix must use plain backspace()")
	}
	if strings.Contains(fix, "deleteTokenBefore") {
		t.Error("fix must NEVER use deleteTokenBefore")
	}
}

// --- Full lifecycle tests ---

func TestPatch_FixPreservesNormalInput(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	if err := Patch(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)

	// Normal input path should still exist
	if !strings.Contains(src, `let c=P6(J6)(t)`) {
		t.Error("normal input code path was corrupted by patch")
	}
}

func TestPatch_PatchReapplyAfterRestore(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	if err := Patch(path); err != nil {
		t.Fatalf("first patch: %v", err)
	}

	if err := Restore(path); err != nil {
		t.Fatalf("restore: %v", err)
	}

	if IsPatched(path) {
		t.Error("file still appears patched after restore")
	}

	if err := Patch(path); err != nil {
		t.Fatalf("re-patch: %v", err)
	}

	if !IsPatched(path) {
		t.Error("file not patched after re-patch")
	}
}

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

func TestPatch_FixNotCorruptSurroundings(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	if err := Patch(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)

	if !strings.Contains(src, `"use strict"`) {
		t.Error("file header corrupted")
	}
	if !strings.Contains(src, `module.exports={onInput}`) {
		t.Error("module export corrupted")
	}
	if !strings.Contains(src, "function onInput(") {
		t.Error("function declaration corrupted")
	}
}

func TestGenerateFix_StateSyncClearsOnControlKey(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6",
	}
	fix := generateFix(v)

	// Must detect control keys as o.length===0 && !J6.backspace
	if !strings.Contains(fix, "o.length===0&&!J6.backspace") {
		t.Error("fix must detect control keys via o.length===0 && !J6.backspace")
	}
	// Must clear __imeState on control key
	if !strings.Contains(fix, "globalThis.__imeState=null") {
		t.Error("fix must clear __imeState on control key")
	}
}

func TestGenerateFix_StateSyncRestoresState(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6",
	}
	fix := generateFix(v)

	// Must restore y from __imeState.s
	if !strings.Contains(fix, "y=globalThis.__imeState.s") {
		t.Error("fix must restore curState from __imeState.s")
	}
	// Must expire stale bridges after 200ms
	if !strings.Contains(fix, "Date.now()-globalThis.__imeState.t>200") {
		t.Error("fix must expire stale bridges after 200ms")
	}
	// Must NOT use text comparison to clear bridge (it's always true after bsHandler)
	if strings.Contains(fix, "__imeState.s.text)globalThis.__imeState=null") {
		t.Error("fix must NOT use text comparison to clear bridge")
	}
}

func TestGenerateFix_IncludesImeStateBridge(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6",
	}
	fix := generateFix(v)

	if !strings.Contains(fix, "globalThis.__imeState={s:_s,t:globalThis.__imeState?globalThis.__imeState.t:Date.now()};") {
		t.Error("fix must set globalThis.__imeState with preserved timestamp")
	}
	if strings.Contains(fix, "setTimeout") {
		t.Error("fix must NOT include setTimeout")
	}
}

// --- v7 Backspace Interception Tests ---

func TestGenerateFix_BackspaceInterception(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6",
	}
	fix := generateFix(v)

	// Must contain backspace interception block
	if !strings.Contains(fix, "J6.backspace&&!J6.ctrl&&!J6.meta") {
		t.Error("fix must contain backspace interception with ctrl/meta guard")
	}
	// Must call y.backspace() in the handler
	if !strings.Contains(fix, "let _s=y.backspace();") {
		t.Error("fix must call curState.backspace() in backspace handler")
	}
	// Must save state to globalThis.__imeState in ALL THREE handlers
	if c := strings.Count(fix, "globalThis.__imeState={s:_s,t:globalThis.__imeState?globalThis.__imeState.t:Date.now()};"); c < 3 {
		t.Errorf("fix must set __imeState in raw, backspace, AND char handlers, got %d occurrences", c)
	}
	// Must call cleanup in ALL THREE handlers
	if c := strings.Count(fix, "XI6(),MI6();"); c < 3 {
		t.Errorf("fix must call cleanup in all 3 handlers, got %d occurrences", c)
	}
}

func TestGenerateFix_BackspaceInterceptionReturns(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6",
	}
	fix := generateFix(v)

	// The backspace handler must return; to prevent fallthrough to normal path
	// Count returns — should have 3: raw, backspace, and char handlers
	if c := strings.Count(fix, "return;}"); c < 3 {
		t.Errorf("fix must have return;} in all 3 handlers, got %d", c)
	}
}

func TestGenerateFix_NoBackspaceWithoutKeyInfo(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6",
	}
	fix := generateFix(v)

	// Without keyInfo, backspace handler should not be generated
	if strings.Contains(fix, ".backspace&&") {
		t.Error("fix must NOT include backspace handler when keyInfo is empty")
	}
	// Raw \x7f handler should still be present
	if !strings.Contains(fix, `o.includes("\x7f")`) {
		t.Error("raw \\x7f handler must still be present")
	}
}

func TestPatch_BackspaceInterceptionInPatched(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	if err := Patch(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)

	// Verify backspace interception is in the patched file
	if !strings.Contains(src, "J6.backspace&&!J6.ctrl&&!J6.meta") {
		t.Error("patched file must contain backspace interception")
	}
	if !strings.Contains(src, "let _s=y.backspace();") {
		t.Error("patched file must contain y.backspace() call")
	}
}

func TestGenerateFix_AlternateVarsBackspace(t *testing.T) {
	v := &variables{
		input: "abc", keyInfo: "QR7", state: "xyz", curState: "snap",
		updateText: "fn1", updateOfs: "fn2",
		cleanup1: "CL1", cleanup2: "CL2",
	}
	fix := generateFix(v)

	// Check that alternate key-info var is used correctly
	if !strings.Contains(fix, "QR7.backspace&&!QR7.ctrl&&!QR7.meta") {
		t.Error("backspace handler must use alternate keyInfo var")
	}
	if !strings.Contains(fix, "let _s=snap.backspace();") {
		t.Error("backspace handler must use alternate curState var")
	}
}

func TestGenerateFix_CharHandlerDuringBurst(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6",
	}
	fix := generateFix(v)

	// Must intercept chars when __imeState is active
	if !strings.Contains(fix, "globalThis.__imeState&&o.length>0&&!J6.backspace") {
		t.Error("fix must intercept replacement chars during IME burst")
	}
	// Must use insert() for each char
	if !strings.Contains(fix, "_s=_s.insert(_c)") {
		t.Error("char handler must use insert() for replacement chars")
	}
}
