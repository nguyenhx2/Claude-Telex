<p align="center">
  <a href="https://github.com/nguyenhx2/Claude-Telex"><img src="https://img.shields.io/badge/GitHub-Claude--Telex-181717?logo=github&logoColor=white" alt="GitHub"></a>
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey" alt="Platform">
  <img src="https://img.shields.io/github/license/nguyenhx2/Claude-Telex" alt="License">
</p>

<h1 align="center">⌨️ Claude Telex</h1>

<p align="center">
  <a href="#vietnamese">🇻🇳 Tiếng Việt</a> ·
  <a href="#english">🇬🇧 English</a>
</p>

---

<a id="vietnamese"></a>

## 🇻🇳 Tiếng Việt

### Vấn đề

Khi gõ tiếng Việt bằng bộ gõ TELEX (EVKey, UniKey, GoTiengViet...) trong Claude Code CLI, ký tự bị **mất** hoặc **hiển thị sai** vì Claude Code xử lý ký tự xoá (backspace `\x7F`) theo nhóm thay vì từng ký tự.

> **Ví dụ:** Gõ `banj` mong đợi `bạn`, nhưng nhận được `bn` hoặc text bị lỗi.

### Nguyên nhân gốc

Phân tích source code thực của Claude Code (`cli.js`) cho thấy hàm `onInput` xử lý TELEX theo logic sau:

1. **Đếm** tổng số ký tự `\x7F` (backspace)
2. **Xoá** N lần bằng `deleteTokenBefore()??backspace()`  
3. **Không chèn** các ký tự thay thế (bug gốc của Anthropic — bị drop hoàn toàn!)

Kết quả: chữ bị xoá nhưng ký tự tiếng Việt đúng không bao giờ được ghi vào.

### Giải pháp — Thuật toán Patch v2

Claude Telex **patch trực tiếp** vào `cli.js`, thay thế logic lỗi bằng vòng lặp **xử lý tuần tự từng ký tự**:

```mermaid
flowchart LR
    subgraph "❌ Gốc - Lỗi (Anthropic)"
        A["EVKey gửi: ⌫⌫ + ạn"] --> B["Đếm 2 backspace"]
        B --> C["deleteTokenBefore() × 2"]
        C --> D["❌ Drop ký tự 'ạn' — không insert!"]
    end
    subgraph "✅ Claude Telex v2 - Fix"
        E["EVKey gửi: ⌫⌫ + ạn"] --> F["for..of từng ký tự"]
        F --> G["⌫ → deleteTokenBefore() ?? backspace()"]
        F --> H["ạ → .insert('ạ')"]
        F --> I["n → .insert('n')"]
        G & H & I --> J["✅ Hiển thị đúng 'bạn'"]
    end
```

**Các chi tiết quan trọng của patch v2:**

| Thành phần | Mô tả |
|---|---|
| `deleteTokenBefore()??backspace()` | Xoá đúng cấp độ token (ký tự tổ hợp Unicode) thay vì byte đơn lẻ |
| `!J6.backspace&&!J6.delete` guard | Không can thiệp khi người dùng nhấn phím Backspace/Delete thật |
| Cleanup functions (`XI6(),MI6()`) | Gọi đúng các hàm cleanup nội bộ của Claude Code sau mỗi lần xử lý |
| `let _s = curState` | Biến local có scope an toàn — không ô nhiễm biến minified bên ngoài |
| Auto-upgrade v1→v2 | Tự phát hiện và nâng cấp patch cũ khi restart |

### Cài đặt

#### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/nguyenhx2/Claude-Telex/main/install.ps1 | iex
```

#### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/nguyenhx2/Claude-Telex/main/install.sh | bash
```

#### Từ Source

```bash
go install github.com/nguyenhx2/claude-telex/cmd/claude-telex@latest
```

### Sử dụng

Chạy `claude-telex` - app sẽ:

1. 🔍 Tự động tìm `cli.js` của Claude Code
2. 🩹 Patch logic xử lý backspace (v2)
3. 🖥️ Hiển thị icon ở system tray (cam = bật, xám = tắt)
4. ⚙️ Mở Settings UI tại `http://127.0.0.1:9315`

| Thao tác | Cách thực hiện |
|---|---|
| **Bật/Tắt fix** | Click tray icon → Settings, hoặc `Ctrl+Alt+V` |
| **Re-patch** | Settings UI → "Re-patch ngay" |
| **Khởi động cùng máy** | Settings UI → toggle "Khởi động cùng hệ thống" |
| **Thoát** | Right-click tray icon → Thoát |

### ⌨️ Bộ gõ được hỗ trợ

| Bộ gõ | Hệ điều hành | Trạng thái |
|---|---|---|
| **EVKey** | Windows | ✅ Hỗ trợ đầy đủ |
| **UniKey** | Windows | ✅ Hỗ trợ đầy đủ |
| **GoTiengViet** | Windows / macOS | ✅ Hỗ trợ đầy đủ |
| **ibus-bamboo** | Linux | ✅ Hỗ trợ đầy đủ |
| Bộ gõ khác (gửi `\x7F`) | Tất cả | ✅ Hoạt động |

### 🖥️ Tương thích

| Thành phần | Phiên bản | Trạng thái |
|---|---|---|
| 🤖 **Claude Code** | Mọi phiên bản (npm `@anthropic-ai/claude-code`) | ✅ Hỗ trợ |
| 🪟 **Windows** | 10 / 11 (amd64, arm64) | ✅ Hỗ trợ |
| 🍎 **macOS** | 12 Monterey+ (Intel & Apple Silicon) | ✅ Hỗ trợ |
| 🐧 **Linux** | Ubuntu 20.04+, Fedora 36+, Arch (amd64, arm64) | ✅ Hỗ trợ |

### Kiến trúc

```mermaid
graph TD
    subgraph "⌨️ Claude Telex Binary"
        M[main.go] --> T[tray<br/>System Tray Icon]
        M --> S[settings<br/>HTTP Server :9315]
        M --> HK[hotkey<br/>Ctrl+Alt+V]
        
        T --> P[patcher<br/>Find & Patch cli.js]
        T --> IC[icon<br/>ICO/PNG Generator]
        S --> P
        S --> AS[autostart<br/>Registry/LaunchAgent/XDG]
        S --> ST[state<br/>JSON Config]
        S --> OI[osinfo<br/>Platform-specific]
    end
    
    S --> |serves| UI[Settings UI<br/>index.html]
    P --> |patches| CLI[Claude Code<br/>cli.js]
    IC --> |renders| TRAY[System Tray<br/>🟠 On / ⚪ Off]
    OI --> |Win: Registry<br/>Mac: sw_vers<br/>Lin: /etc/os-release| OS[OS Info]
```

#### Tổng quan Package

| Package | Chức năng |
|---|---|
| `cmd/claude-telex` | Entry point, single-instance lock, orchestration |
| `internal/patcher` | Tìm `cli.js`, trích xuất biến động bằng regex, inject fix v2 |
| `internal/tray` | System tray (ICO trên Windows, PNG trên macOS/Linux) |
| `internal/settings` | HTTP server tại port 9315, JSON API |
| `internal/icon` | Vẽ icon programmatically (vòng tròn + chữ "VN") |
| `internal/hotkey` | Global hotkey `Ctrl+Alt+V` |
| `internal/autostart` | Tự khởi động: Windows Registry / macOS LaunchAgent / Linux XDG |
| `internal/state` | Lưu config JSON tại `~/.claude-telex/config.json` |
| `assets/ui` | Embedded HTML Settings UI (dark theme, Inter font) |

### Luồng Patching (v2)

```mermaid
sequenceDiagram
    participant U as User
    participant CT as ⌨️ Claude Telex
    participant JS as cli.js

    U->>CT: Launch
    CT->>JS: FindCliJS()
    CT->>JS: ReadFile()
    CT->>CT: Kiểm tra patch cũ (v1)?<br/>→ Restore backup, rồi re-patch
    CT->>CT: findBugBlock()<br/>Tìm if(!key.backspace&&...includes("⌫"))
    CT->>CT: extractVariables()<br/>Regex: input, keyInfo, curState,<br/>updateText, updateOfs, cleanup1/2, hasDTB
    CT->>CT: generateFix()<br/>for..of: ⌫→deleteTokenBefore()??backspace()<br/>else→insert(_c)<br/>+ cleanup() + return
    CT->>JS: Inject fix TRƯỚC early-return guard
    CT->>JS: WriteFile() + Verify marker
    CT-->>U: ✅ Patched v2! System tray 🟠
```

### Build & Chạy

#### Yêu cầu

- **Go** 1.22+ ([tải tại đây](https://go.dev/dl/))
- **Git** ([tải tại đây](https://git-scm.com/))
- **Linux**: cần thêm `gcc`, `libgtk-3-dev`, `libappindicator3-dev`

#### Build

```bash
# Clone repo
git clone https://github.com/nguyenhx2/Claude-Telex.git
cd Claude-Telex

# Build binary
go build -ldflags="-s -w -H windowsgui" -o claude-telex.exe ./cmd/claude-telex   # Windows
go build -ldflags="-s -w" -o claude-telex ./cmd/claude-telex                      # macOS / Linux

# Hoặc dùng Make
make build
```

#### Stop & Restart (Development — Windows)

```powershell
# Stop app đang chạy
Get-Process claude-telex -ErrorAction SilentlyContinue | Stop-Process -Force

# Build lại và chạy
go build -ldflags="-s -w -H windowsgui" -o claude-telex.exe ./cmd/claude-telex && `
  Start-Process -FilePath ".\claude-telex.exe" -WindowStyle Hidden

# Stop + Build + Restart trong một lệnh
Get-Process claude-telex -ErrorAction SilentlyContinue | Stop-Process -Force; `
  go build -ldflags="-s -w -H windowsgui" -o claude-telex.exe ./cmd/claude-telex && `
  Start-Process -FilePath ".\claude-telex.exe" -WindowStyle Hidden
```

#### Chạy (Development)

```bash
# Chạy trực tiếp (có console output)
go run ./cmd/claude-telex

# Chạy binary đã build
./claude-telex        # macOS / Linux
.\claude-telex.exe    # Windows
```

#### Release (snapshot)

```bash
goreleaser release --snapshot --clean
```

---

<a id="english"></a>

## 🇬🇧 English

### The Problem

When typing Vietnamese using TELEX IME (EVKey, UniKey, GoTiengViet...) in Claude Code CLI, characters are **lost** or **displayed incorrectly** because Claude Code processes delete characters (backspace `\x7F`) in batches instead of one-by-one.

> **Example:** Typing `banj` expecting `bạn`, but getting `bn` or garbled text.

### Root Cause

Analysis of the actual Claude Code source (`cli.js`) revealed that the `onInput` function handles TELEX like this:

1. **Count** total `\x7F` (backspace) characters
2. **Delete** N times using `deleteTokenBefore()??backspace()`
3. **Never insert** the replacement characters (Anthropic's bug — they are dropped entirely!)

Result: characters get deleted but the correct Vietnamese character is never written.

### The Solution — Patch Algorithm v2

Claude Telex **directly patches** `cli.js`, replacing the broken logic with a loop that **processes each character sequentially**:

```mermaid
flowchart LR
    subgraph "❌ Original - Broken (Anthropic)"
        A["EVKey sends: ⌫⌫ + ạn"] --> B["Count 2 backspaces"]
        B --> C["deleteTokenBefore() × 2"]
        C --> D["❌ Drop 'ạn' chars — never inserted!"]
    end
    subgraph "✅ Claude Telex v2 - Fixed"
        E["EVKey sends: ⌫⌫ + ạn"] --> F["for..of each char"]
        F --> G["⌫ → deleteTokenBefore() ?? backspace()"]
        F --> H["ạ → .insert('ạ')"]
        F --> I["n → .insert('n')"]
        G & H & I --> J["✅ Correctly shows 'bạn'"]
    end
```

**Key v2 patch details:**

| Component | Description |
|---|---|
| `deleteTokenBefore()??backspace()` | Deletes at token level (composed Unicode chars), not individual bytes |
| `!J6.backspace&&!J6.delete` guard | Does not intercept real Backspace/Delete key presses |
| Cleanup functions (`XI6(),MI6()`) | Calls Claude Code's internal cleanup functions correctly after processing |
| `let _s = curState` | Locally scoped variable — never pollutes outer minified scope |
| Auto-upgrade v1→v2 | Auto-detects and upgrades old patches on restart |

### Installation

#### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/nguyenhx2/Claude-Telex/main/install.ps1 | iex
```

#### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/nguyenhx2/Claude-Telex/main/install.sh | bash
```

#### From Source

```bash
go install github.com/nguyenhx2/claude-telex/cmd/claude-telex@latest
```

### Usage

Run `claude-telex` - the app will:

1. 🔍 Auto-detect Claude Code's `cli.js`
2. 🩹 Patch the backspace handling logic (v2)
3. 🖥️ Show a system tray icon (orange = on, grey = off)
4. ⚙️ Open Settings UI at `http://127.0.0.1:9315`

| Action | How |
|---|---|
| **Toggle fix** | Click tray icon → Settings, or `Ctrl+Alt+V` |
| **Re-patch** | Settings UI → "Re-patch now" |
| **Start with OS** | Settings UI → toggle "Start with system" |
| **Exit** | Right-click tray icon → Exit |

### ⌨️ Supported IME

| IME | OS | Status |
|---|---|---|
| **EVKey** | Windows | ✅ Fully supported |
| **UniKey** | Windows | ✅ Fully supported |
| **GoTiengViet** | Windows / macOS | ✅ Fully supported |
| **ibus-bamboo** | Linux | ✅ Fully supported |
| Other IMEs (sending `\x7F`) | All | ✅ Works |

### 🖥️ Compatibility

| Component | Version | Status |
|---|---|---|
| 🤖 **Claude Code** | All versions (npm `@anthropic-ai/claude-code`) | ✅ Supported |
| 🪟 **Windows** | 10 / 11 (amd64, arm64) | ✅ Supported |
| 🍎 **macOS** | 12 Monterey+ (Intel & Apple Silicon) | ✅ Supported |
| 🐧 **Linux** | Ubuntu 20.04+, Fedora 36+, Arch (amd64, arm64) | ✅ Supported |

### Architecture

```mermaid
graph TD
    subgraph "⌨️ Claude Telex Binary"
        M[main.go] --> T[tray<br/>System Tray Icon]
        M --> S[settings<br/>HTTP Server :9315]
        M --> HK[hotkey<br/>Ctrl+Alt+V]
        
        T --> P[patcher<br/>Find & Patch cli.js]
        T --> IC[icon<br/>ICO/PNG Generator]
        S --> P
        S --> AS[autostart<br/>Registry/LaunchAgent/XDG]
        S --> ST[state<br/>JSON Config]
        S --> OI[osinfo<br/>Platform-specific]
    end
    
    S --> |serves| UI[Settings UI<br/>index.html]
    P --> |patches| CLI[Claude Code<br/>cli.js]
    IC --> |renders| TRAY[System Tray<br/>🟠 On / ⚪ Off]
    OI --> |Win: Registry<br/>Mac: sw_vers<br/>Lin: /etc/os-release| OS[OS Info]
```

#### Package Overview

| Package | Description |
|---|---|
| `cmd/claude-telex` | Entry point, single-instance lock, orchestration |
| `internal/patcher` | Find `cli.js`, extract dynamic vars via regex, inject fix v2 |
| `internal/tray` | System tray (ICO on Windows, PNG on macOS/Linux) |
| `internal/settings` | HTTP server at port 9315, JSON API |
| `internal/icon` | Programmatic icon rendering (circle + "VN" text) |
| `internal/hotkey` | Global hotkey `Ctrl+Alt+V` |
| `internal/autostart` | Auto-start: Windows Registry / macOS LaunchAgent / Linux XDG |
| `internal/state` | JSON config at `~/.claude-telex/config.json` |
| `assets/ui` | Embedded HTML Settings UI (dark theme, Inter font, copy CLI path) |

### Patching Flow (v2)

```mermaid
sequenceDiagram
    participant U as User
    participant CT as ⌨️ Claude Telex
    participant JS as cli.js

    U->>CT: Launch
    CT->>JS: FindCliJS()
    CT->>JS: ReadFile()
    CT->>CT: Legacy patch detected (v1)?<br/>→ Restore backup, then re-patch
    CT->>CT: findBugBlock()<br/>Find if(!key.backspace&&...includes("⌫"))
    CT->>CT: extractVariables()<br/>Regex: input, keyInfo, curState,<br/>updateText, updateOfs, cleanup1/2, hasDTB
    CT->>CT: generateFix()<br/>for..of: ⌫→deleteTokenBefore()??backspace()<br/>else→insert(_c)<br/>+ cleanup() + return
    CT->>JS: Inject fix BEFORE early-return guard
    CT->>JS: WriteFile() + Verify marker
    CT-->>U: ✅ Patched v2! System tray 🟠
```

### Build & Run

#### Prerequisites

- **Go** 1.22+ ([download](https://go.dev/dl/))
- **Git** ([download](https://git-scm.com/))
- **Linux**: also needs `gcc`, `libgtk-3-dev`, `libappindicator3-dev`

#### Build

```bash
# Clone the repo
git clone https://github.com/nguyenhx2/Claude-Telex.git
cd Claude-Telex

# Build binary
go build -ldflags="-s -w -H windowsgui" -o claude-telex.exe ./cmd/claude-telex   # Windows
go build -ldflags="-s -w" -o claude-telex ./cmd/claude-telex                      # macOS / Linux

# Or use Make
make build
```

#### Stop & Restart (Development — Windows)

```powershell
# Stop running instance
Get-Process claude-telex -ErrorAction SilentlyContinue | Stop-Process -Force

# Build and start
go build -ldflags="-s -w -H windowsgui" -o claude-telex.exe ./cmd/claude-telex && `
  Start-Process -FilePath ".\claude-telex.exe" -WindowStyle Hidden

# Stop + Build + Restart in one command
Get-Process claude-telex -ErrorAction SilentlyContinue | Stop-Process -Force; `
  go build -ldflags="-s -w -H windowsgui" -o claude-telex.exe ./cmd/claude-telex && `
  Start-Process -FilePath ".\claude-telex.exe" -WindowStyle Hidden
```

#### Run (Development)

```bash
# Run directly (with console output)
go run ./cmd/claude-telex

# Run the built binary
./claude-telex        # macOS / Linux
.\claude-telex.exe    # Windows
```

#### Release (snapshot)

```bash
goreleaser release --snapshot --clean
```

---

<p align="center">

**⌨️ Claude Telex** - Vietnamese TELEX Support for Claude Code CLI

</p>

<p align="center">
  <a href="https://github.com/nguyenhx2">@nguyenhx2</a> ·
  Go 1.22+ ·
  <a href="https://github.com/nguyenhx2/Claude-Telex">GitHub</a> ·
  <a href="LICENSE">MIT License</a>
</p>

<p align="center">
  <b>Thư viện / Libraries:</b>
  <a href="https://github.com/getlantern/systray">getlantern/systray</a> -
  <a href="https://pkg.go.dev/golang.design/x/hotkey">golang.design/x/hotkey</a> -
  <a href="https://pkg.go.dev/golang.org/x/image">golang.org/x/image</a>
</p>

<p align="center">
  <i>Cảm hứng / Inspired by: Vietnamese IME bug reports from the Claude Code Vietnam community</i>
</p>
