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

func TestPatch_UsesDeleteTokenBefore(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	if err := Patch(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)

	// Must use deleteTokenBefore()??backspace() not just backspace()
	if !strings.Contains(src, ".deleteTokenBefore()??_s.backspace()") {
		t.Error("fix must use deleteTokenBefore for proper token deletion")
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

func TestPatch_PreservesKeyInfoGuard(t *testing.T) {
	dir := t.TempDir()
	path := simulateCliJS(t, dir)

	if err := Patch(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)

	// Must preserve the !J6.backspace&&!J6.delete guard
	if !strings.Contains(src, "!J6.backspace&&!J6.delete&&") {
		t.Error("fix must preserve key-info guard for real backspace/delete keys")
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
	if !strings.Contains(src, "_s=_s.deleteTokenBefore()??_s.backspace()") {
		t.Error("fix should use deleteTokenBefore")
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
	if !strings.Contains(src, "!QR7.backspace&&!QR7.delete&&") {
		t.Error("fix should use key-info guard with 'QR7'")
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
	// Should have the new fix features
	if !strings.Contains(src, "deleteTokenBefore") {
		t.Error("upgraded patch should use deleteTokenBefore")
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
	if !vars.hasDTB {
		t.Error("expected hasDTB=true")
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
		cleanup1: "XI6", cleanup2: "MI6", hasDTB: true,
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
		cleanup1: "XI6", cleanup2: "MI6", hasDTB: true,
	}
	fix := generateFix(v)

	if !strings.Contains(fix, "for(const _c of o)") {
		t.Error("fix must iterate over each char of input")
	}

	if !strings.Contains(fix, `_c==="\x7f"`) {
		t.Error("fix must check each char for \\x7f")
	}

	if !strings.Contains(fix, "_s=_s.deleteTokenBefore()??_s.backspace()") {
		t.Error("fix must use deleteTokenBefore for backspace handling")
	}

	if !strings.Contains(fix, "_s=_s.insert(_c)") {
		t.Error("fix must call .insert() for non-backspace chars")
	}
}

func TestGenerateFix_UpdatesTextAndOffset(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6", hasDTB: true,
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
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6", hasDTB: true,
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
		cleanup1: "XI6", cleanup2: "MI6", hasDTB: true,
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
		cleanup1: "XI6", cleanup2: "MI6", hasDTB: true,
	}
	fix := generateFix(v)

	if !strings.Contains(fix, "XI6(),MI6();") {
		t.Error("fix must call cleanup functions")
	}
}

func TestGenerateFix_IncludesKeyInfoGuard(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "J6", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "XI6", cleanup2: "MI6", hasDTB: true,
	}
	fix := generateFix(v)

	if !strings.Contains(fix, "!J6.backspace&&!J6.delete&&o.includes") {
		t.Error("fix must include key-info guard")
	}
}

func TestGenerateFix_NoDeleteTokenBefore(t *testing.T) {
	v := &variables{
		input: "o", keyInfo: "", state: "e3", curState: "y",
		updateText: "h7", updateOfs: "S5",
		cleanup1: "", cleanup2: "", hasDTB: false,
	}
	fix := generateFix(v)

	// Without DTB, should use plain backspace
	if !strings.Contains(fix, "_s=_s.backspace();") {
		t.Error("fix without DTB should use plain backspace")
	}
	if strings.Contains(fix, "deleteTokenBefore") {
		t.Error("fix without DTB should NOT use deleteTokenBefore")
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
