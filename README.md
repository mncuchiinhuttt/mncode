# mncode (Claude Code CLI Clone in Golang)

`mncode` là một phiên bản AI coding assistant CLI hiệu năng cao, viết hoàn toàn bằng **Golang**, tương thích với cơ chế hoạt động của **Claude Code** và tích hợp sâu với toàn bộ hệ sinh thái **ClaudeKit Engineer** cùng bộ **Claude Skills (Open Agent Skills Specification)** trong folder `.claude`.

---

## 🌟 Tính năng nổi bật

1. **Multi-Account Manager & 9router-style Smart Routing**:
   - **Đăng nhập nhiều tài khoản Codex & Antigravity** không giới hạn số lượng.
   - **Cơ chế xoay tua (Round-robin)** và tự động cân bằng tải (Load Balancing).
   - **Tự động Cooldown khi chạm Rate Limit (429)**: Tự động phát hiện 429 / 401 và chuyển sang tài khoản kế tiếp ngay lập tức mà không làm gián đoạn câu lệnh của người dùng!
   - Tự động import credentials có sẵn từ `~/.gemini/` hoặc `~/.openai/`.
   - Lưu trữ an toàn tại `~/.mncode/accounts.json`.
2. **Tương thích toàn diện với `.claude`**:
   - Tự động scan và nạp **60+ Claude Skills** (`.claude/skills/*/SKILL.md`) theo chuẩn Open Agent Skills Spec (YAML frontmatter + Markdown).
   - Nạp các **ClaudeKit Specialized Subagents** (`.claude/agents/*.md`): `planner`, `researcher`, `code-reviewer`, `tester`, `debugger`, `docs-manager`, `fullstack-developer`, v.v.
   - Nạp các **Workspace Rules** (`.claude/rules/*.md`) và `.claude/.ck.json`.
3. **Bộ công cụ (Built-in Tools Suite) tiêu chuẩn**:
   - `run_command` (`bash`): Thực thi lệnh shell với timeout & output streaming.
   - `view_file`: Xem file với số dòng và slicing (`StartLine`, `EndLine`).
   - `replace_file_content`: Sửa code chính xác theo từng đoạn chunk.
   - `write_to_file`: Tạo hoặc ghi đè file với tự động tạo thư mục cha.
   - `grep_search`: Tìm kiếm regex / pattern trong toàn bộ dự án.
   - `find_by_name`: Tìm kiếm tên file / glob pattern.
   - `list_dir`: Xem danh sách file / thư mục.
   - `read_url_content`: Đọc nội dung web.
   - `ask_user`: Tương tác hỏi người dùng khi cần làm rõ.
   - `invoke_subagent`: Điều phối các subagent chuyên biệt để giải quyết task.
4. **Multi-Provider LLM Engine**:
   - **Anthropic Claude**: Hỗ trợ Messages API, streaming SSE, tool calling, Extended Thinking (`claude-3-7-sonnet-20250219`).
   - **OpenRouter / OpenAI**: Hỗ trợ OpenRouter, DeepSeek, ChatGPT, Ollama.
   - **Google Gemini / Antigravity**: Hỗ trợ Gemini 2.5 / 3 streaming & tool calling.
4. **Giao diện Interactive Terminal REPL**:
   - Quản lý lịch sử lệnh (History, Arrow Keys, Autocompletion).
   - Streaming token & hiển thị khối Thinking (`[Thinking]`).
   - Slash commands: `/skills`, `/agents`, `/rules`, `/model`, `/clear`, `/status`, `/help`, `/exit`.
   - Cơ chế cấp quyền xác nhận trước khi chạy lệnh (Permission Guard).

---

## 🚀 Hướng dẫn cài đặt & Build

```bash
# Clone hoặc vào thư mục dự án
cd /Users/vominhlong/mncode

# Tải dependencies & Build binary
go build -o bin/mncode ./cmd/mncode
```

---

## 🔑 Cấu hình API Key

Tạo file `.env` tại thư mục root hoặc cấu hình biến môi trường:

```env
# Dùng Anthropic Claude (Khuyên dùng)
ANTHROPIC_API_KEY=your_anthropic_api_key
ANTHROPIC_MODEL=claude-3-7-sonnet-20250219

# Hoặc dùng OpenRouter
OPENROUTER_API_KEY=your_openrouter_api_key
OPENROUTER_MODEL=anthropic/claude-3.7-sonnet

# Hoặc dùng Google Gemini
GEMINI_API_KEY=your_gemini_api_key
GEMINI_MODEL=gemini-2.5-pro
```

---

## 💻 Cách sử dụng

### 1. Khởi động Interactive REPL:
```bash
./bin/mncode
```

### 2. Tự động duyệt lệnh (Auto-approve mode):
```bash
./bin/mncode -y
```

### 3. Chỉ định Model hoặc Provider:
```bash
./bin/mncode -p anthropic -m claude-3-7-sonnet-20250219
./bin/mncode -p gemini -m gemini-2.5-pro
```

### 4. Chạy một câu lệnh trực tiếp (Non-interactive mode):
```bash
./bin/mncode -e "Hãy kiểm tra cấu trúc thư mục và liệt kê các skill đang có"
```

---

## 🕹️ Các lệnh Slash trong REPL

- `/account list`: Xem danh sách tất cả các tài khoản Antigravity & Codex (trạng thái ACTIVE/COOLDOWN, số lần gọi, lỗi gần nhất).
- `/login antigravity`: Đăng nhập thêm tài khoản Antigravity / Gemini OAuth token.
- `/login codex`: Đăng nhập thêm tài khoản OpenAI / Codex token.
- `/account add <provider> <id> <token>`: Thêm thủ công tài khoản vào pool.
- `/account remove <id>`: Xóa tài khoản khỏi pool.
- `/account import`: Tự động quét và import credentials có sẵn từ `~/.gemini/` hoặc `~/.openai/`.
- `/skills`: Liệt kê tất cả các Claude skills đã nạp từ `.claude/skills/`.
- `/agents`: Xem danh sách các ClaudeKit subagents (`planner`, `tester`, `code-reviewer`, v.v.).
- `/rules`: Xem các quy tắc phát triển trong `.claude/rules/`.
- `/model [tên_model]`: Xem hoặc thay đổi model đang chạy trong phiên.
- `/clear`: Xóa lịch sử hội thoại hiện tại.
- `/status`: Xem trạng thái cấu hình và số lượng tin nhắn trong phiên.
- `/help`: Xem hướng dẫn lệnh.
- `/exit` hoặc `/quit`: Thoát chương trình.

---

## 📁 Cấu trúc mã nguồn

```
mncode/
├── cmd/
│   └── mncode/
│       └── main.go                 # Điểm khởi chạy CLI
├── pkg/
│   ├── config/                     # Quản lý config, .env, .ck.json
│   ├── skills/                     # Bộ parser & loader cho Claude Skills, Agents & Rules
│   ├── tools/                      # Bộ tools: Bash, View, Edit, Write, Grep, Find, Subagents
│   ├── provider/                   # LLM engine (Anthropic, OpenRouter, OpenAI, Gemini)
│   ├── agent/                      # ReAct execution loop, prompt builder, subagent runner
│   └── ui/                         # Terminal REPL, colors, spinner, slash commands
└── bin/
    └── mncode                      # File binary thực thi
```
