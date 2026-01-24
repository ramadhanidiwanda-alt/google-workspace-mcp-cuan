# Google Workspace MCP

[English](#english) | [日本語](#日本語)

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server that connects AI assistants to Google Workspace APIs.

---

## English

### Features

- **Google Calendar** - Create, update, delete events, find free time
- **Gmail** - Search, read, send emails, manage labels
- **Google Drive** - Search files, create folders, download files
- **Google Docs** - Create, read, edit documents with Markdown support
- **Google Sheets** - Read spreadsheet data in multiple formats
- **Google Slides** - Read presentation content
- **Google Chat** - Send messages, manage spaces
- **People API** - Get user profiles and relations

### Installation

#### Download Binary

Download the latest binary from [Releases](https://github.com/tomohiro-owada/google-workspace-mcp/releases):

| Platform | Architecture | Download |
|----------|-------------|----------|
| macOS | Apple Silicon (M1/M2/M3) | `workspace-server-darwin-arm64` |
| macOS | Intel | `workspace-server-darwin-amd64` |
| Linux | x86_64 | `workspace-server-linux-amd64` |
| Linux | ARM64 | `workspace-server-linux-arm64` |
| Windows | x86_64 | `workspace-server-windows-amd64.exe` |

```bash
# Example: macOS Apple Silicon
curl -L -o workspace-server https://github.com/tomohiro-owada/google-workspace-mcp/releases/latest/download/workspace-server-darwin-arm64
chmod +x workspace-server
sudo mv workspace-server /usr/local/bin/
```

#### Build from Source

```bash
git clone https://github.com/tomohiro-owada/google-workspace-mcp.git
cd google-workspace-mcp
go build -o workspace-server ./cmd/workspace-server/
```

### Configuration

#### Claude Code

Add to `~/.claude.json`:

```json
{
  "mcpServers": {
    "google-workspace": {
      "command": "/usr/local/bin/workspace-server"
    }
  }
}
```

#### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS):

```json
{
  "mcpServers": {
    "google-workspace": {
      "command": "/usr/local/bin/workspace-server"
    }
  }
}
```

### Authentication

This server uses OAuth 2.0 for Google authentication. On first use, you need to authenticate:

```
# Check auth status
auth.status

# Login (opens browser)
auth.login

# Clear credentials
auth.clear
```

### Available Tools

<details>
<summary><b>Auth Tools</b></summary>

| Tool | Description |
|------|-------------|
| `auth.login` | Initiate Google OAuth login |
| `auth.status` | Check authentication status |
| `auth.clear` | Clear stored credentials |
| `auth.refreshToken` | Manually refresh token |

</details>

<details>
<summary><b>Calendar Tools</b></summary>

| Tool | Description |
|------|-------------|
| `calendar.list` | List all calendars |
| `calendar.listEvents` | List events in a time range |
| `calendar.getEvent` | Get event details |
| `calendar.createEvent` | Create a new event |
| `calendar.updateEvent` | Update an existing event |
| `calendar.deleteEvent` | Delete an event |
| `calendar.respondToEvent` | Accept/decline/tentative |
| `calendar.findFreeTime` | Find available time slots |

</details>

<details>
<summary><b>Gmail Tools</b></summary>

| Tool | Description |
|------|-------------|
| `gmail.search` | Search emails |
| `gmail.get` | Get email content |
| `gmail.send` | Send an email |
| `gmail.createDraft` | Create a draft |
| `gmail.sendDraft` | Send a draft |
| `gmail.modify` | Add/remove labels |
| `gmail.listLabels` | List all labels |
| `gmail.downloadAttachment` | Download attachment |

</details>

<details>
<summary><b>Drive Tools</b></summary>

| Tool | Description |
|------|-------------|
| `drive.search` | Search files and folders |
| `drive.findFolder` | Find folder by name |
| `drive.createFolder` | Create a new folder |
| `drive.downloadFile` | Download a file |

</details>

<details>
<summary><b>Docs Tools</b></summary>

| Tool | Description |
|------|-------------|
| `docs.create` | Create a new document |
| `docs.getText` | Get document content |
| `docs.insertText` | Insert text at beginning |
| `docs.appendText` | Append text at end |
| `docs.replaceText` | Find and replace text |
| `docs.move` | Move document to folder |
| `docs.find` | Search documents by title |
| `docs.extractIdFromUrl` | Extract ID from URL |

</details>

<details>
<summary><b>Sheets Tools</b></summary>

| Tool | Description |
|------|-------------|
| `sheets.getText` | Get sheet content (text/csv/json) |
| `sheets.getRange` | Get values from range |
| `sheets.getMetadata` | Get spreadsheet metadata |
| `sheets.find` | Search spreadsheets by title |

</details>

<details>
<summary><b>Slides Tools</b></summary>

| Tool | Description |
|------|-------------|
| `slides.getText` | Get presentation text |
| `slides.getMetadata` | Get presentation metadata |
| `slides.find` | Search presentations by title |

</details>

<details>
<summary><b>Chat Tools</b></summary>

| Tool | Description |
|------|-------------|
| `chat.listSpaces` | List all spaces |
| `chat.findSpaceByName` | Find space by name |
| `chat.findDmByEmail` | Find DM by email |
| `chat.getMessages` | Get messages from space |
| `chat.listThreads` | List threads in space |
| `chat.sendMessage` | Send message to space |
| `chat.sendDm` | Send direct message |
| `chat.setUpSpace` | Create a new space |

</details>

<details>
<summary><b>People Tools</b></summary>

| Tool | Description |
|------|-------------|
| `people.getMe` | Get authenticated user profile |
| `people.getUserProfile` | Get user profile by email |
| `people.getUserRelations` | Get user relations |

</details>

<details>
<summary><b>Time Tools</b></summary>

| Tool | Description |
|------|-------------|
| `time.getCurrentTime` | Get current time |
| `time.getCurrentDate` | Get current date |
| `time.getTimeZone` | Get timezone info |

</details>

### Security

This server grants AI assistants access to read, modify, and delete your Google Workspace data. Use with caution:

- Review actions taken by AI assistants
- Don't process untrusted content (emails, documents from unknown sources)
- Credentials are stored securely using system keychain

### License

Apache License 2.0

---

## 日本語

### 概要

Google Workspace MCP は、AIアシスタント（Claude Code、Claude Desktop など）を Google Workspace API に接続する [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) サーバーです。

### 機能

- **Google カレンダー** - イベントの作成・更新・削除、空き時間の検索
- **Gmail** - メールの検索・閲覧・送信、ラベル管理
- **Google ドライブ** - ファイル検索、フォルダ作成、ファイルダウンロード
- **Google ドキュメント** - ドキュメントの作成・閲覧・編集（Markdown対応）
- **Google スプレッドシート** - データの読み取り（テキスト/CSV/JSON形式）
- **Google スライド** - プレゼンテーション内容の読み取り
- **Google Chat** - メッセージ送信、スペース管理
- **People API** - ユーザープロフィール・組織情報の取得

### インストール

#### バイナリをダウンロード

[Releases](https://github.com/tomohiro-owada/google-workspace-mcp/releases) から最新のバイナリをダウンロード:

| プラットフォーム | アーキテクチャ | ファイル名 |
|--------------|-------------|----------|
| macOS | Apple Silicon (M1/M2/M3) | `workspace-server-darwin-arm64` |
| macOS | Intel | `workspace-server-darwin-amd64` |
| Linux | x86_64 | `workspace-server-linux-amd64` |
| Linux | ARM64 | `workspace-server-linux-arm64` |
| Windows | x86_64 | `workspace-server-windows-amd64.exe` |

```bash
# 例: macOS Apple Silicon
curl -L -o workspace-server https://github.com/tomohiro-owada/google-workspace-mcp/releases/latest/download/workspace-server-darwin-arm64
chmod +x workspace-server
sudo mv workspace-server /usr/local/bin/
```

#### ソースからビルド

```bash
git clone https://github.com/tomohiro-owada/google-workspace-mcp.git
cd google-workspace-mcp
go build -o workspace-server ./cmd/workspace-server/
```

### 設定

#### Claude Code

`~/.claude.json` に追加:

```json
{
  "mcpServers": {
    "google-workspace": {
      "command": "/usr/local/bin/workspace-server"
    }
  }
}
```

#### Claude Desktop

`~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) に追加:

```json
{
  "mcpServers": {
    "google-workspace": {
      "command": "/usr/local/bin/workspace-server"
    }
  }
}
```

### 認証

OAuth 2.0 を使用して Google 認証を行います。初回使用時は認証が必要です:

```
# 認証状態を確認
auth.status

# ログイン（ブラウザが開きます）
auth.login

# 認証情報をクリア
auth.clear
```

### セキュリティ

このサーバーは AI アシスタントに Google Workspace データの読み取り・変更・削除の権限を付与します。以下の点に注意してください:

- AI アシスタントが行うアクションを確認する
- 信頼できないコンテンツ（不明な送信元からのメール、ドキュメントなど）を処理しない
- 認証情報はシステムキーチェーンに安全に保存されます

### ライセンス

Apache License 2.0
