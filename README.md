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

### Cuan hosted Google Sheets service (opt-in)

`Dockerfile.cuan` builds a separate HTTP MCP service for deployments where Cuan Insight owns Google credentials. It is independent of the local OAuth server above and registers exactly two tools: `sheets_read_values` and `sheets_update_values`. Hosted mode does not use local OAuth, ADC, or persisted provider tokens.

The server listens on `/mcp` (port `8080` by default) and requires the exact `Host` `sheets-mcp.cuaninsight.com`, one `x-cuan-sheets-ingress-secret` header matching the separately configured ingress secret, and one `x-cuan-mcp-connection-key` header (`ci_mcp_ck_` followed by 64 lowercase hex characters). Configure `CUAN_SHEETS_MCP_ALLOWED_HOST=sheets-mcp.cuaninsight.com` and `CUAN_SHEETS_INGRESS_SECRET` (at least 32 characters) only in the private deployment environment. The gateway must set/preserve the exact Host and inject the ingress header; callers supply only their Connection Key. Cuan validates the key and returns a short-lived, range-bound Sheets credential to the service. Reads require one quoted sheet name and a rectangular A1 range of at most 100 cells, such as `'Data Sheet'!A1:B10`.

`sheets_update_values` is two phase. First call supplies `expectedOldValues` and `newValues` and returns a digest plus preview and execution IDs without writing. A second call must repeat the same values and IDs with `confirmed: true` and the exact returned digest. Cuan's claim policy runs before the provider write; the service rereads before writing, uses `RAW`, verifies the result, and finalizes either `CONFIRMED` or `UNKNOWN_OUTCOME`. A timeout after write dispatch is never retried.

The container defaults to disabled and exits unless explicitly enabled. Configure `CUAN_SHEETS_MCP_ENABLED=true`, `CUAN_SHEETS_RUNTIME_URL` (HTTPS), `GOOGLE_SHEETS_PRIVATE_SERVICE_ID`, and `GOOGLE_SHEETS_PRIVATE_SERVICE_SECRET` only in the private deployment environment. Build with `docker build -f Dockerfile.cuan -t cuan-google-sheets-mcp .`; run behind an authenticated private gateway. No deployment is performed by this repository change.

### Cuan hosted Google Workspace service (opt-in)

`Dockerfile.workspace` builds the separate broad Workspace MCP service. It exposes only `workspace_drive_list_files`, `workspace_drive_get_file`, `workspace_drive_get_text`, `workspace_sheets_read_values`, `workspace_drive_create_text_file`, `workspace_drive_update_text_file`, and `workspace_sheets_update_values`. File discovery and access span the connected Google account, subject to Cuan Connection Key grants and Google permissions. Text content is limited to plain text files and Google Docs exported as plain text. This service has no delete, sharing, or move tools. Existing hosted Sheets and local OAuth modes remain separate.

The service listens on `/mcp` and requires the exact configured Host, one `x-cuan-workspace-ingress-secret`, and one `x-cuan-mcp-connection-key`. Set `CUAN_WORKSPACE_MCP_ALLOWED_HOST`, `CUAN_WORKSPACE_INGRESS_SECRET` (at least 32 characters), `CUAN_WORKSPACE_RUNTIME_URL` (HTTPS), `GOOGLE_WORKSPACE_PRIVATE_SERVICE_ID`, `GOOGLE_WORKSPACE_PRIVATE_SERVICE_SECRET`, and `CUAN_WORKSPACE_MCP_ENABLED=true` in the private environment. Build with `docker build -f Dockerfile.workspace -t cuan-google-workspace-mcp .`. Keep the container disabled until Cuan's broad Workspace provider policy and Google pilot gates pass.

Every write first returns a preview ID and approval digest. The service sends Cuan a bounded summary of current and proposed content for admin review. Repeat the same input with `confirmed: true`, `previewId`, and `approvalDigest` to request a write. Cuan checks the exact key, tool, account, target, digest, and credential version, then atomically claims one execution. After a dispatched provider write, the service reads back Drive content or Sheets values before finalizing `CONFIRMED`; a failed or mismatched readback becomes `UNKNOWN_OUTCOME`. It never retries the write. Access tokens stay in private resolver responses and provider requests, never MCP results.

### Access Modes

This server supports two access modes:

| Mode | Permissions | Setup Required |
|------|-------------|----------------|
| **Basic** | Read-only | None (default) |
| **Full Access** | Read + Write + Delete | Custom OAuth credentials |

#### Basic Mode (Default)

Works out of the box with read-only permissions. No additional setup required.

#### Full Access Mode

To enable write operations (send emails, create documents, etc.), you need to set up your own OAuth credentials.

<details>
<summary><b>Setup Instructions</b></summary>

##### 1. Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project (e.g., `workspace-mcp`)
3. Enable the following APIs:
   - Google Calendar API
   - Gmail API
   - Google Drive API
   - Google Docs API
   - Google Sheets API
   - Google Slides API
   - Google Chat API
   - People API

##### 2. Configure OAuth Consent Screen

1. Go to **APIs & Services** > **OAuth consent screen**
2. Choose User Type:
   - **Internal**: For Google Workspace organizations (all members can use)
   - **External**: For personal Gmail accounts (requires adding test users or Google verification)
3. Fill in the required fields (App name, User support email, Developer contact)
4. Add scopes (or skip - they'll be requested at runtime)
5. If External: Add test users (your Gmail address)

##### 3. Create OAuth Credentials

1. Go to **APIs & Services** > **Credentials**
2. Click **Create Credentials** > **OAuth client ID**
3. Select **Desktop app**
4. Download or copy the **Client ID** and **Client Secret**

##### 4. Configure the MCP Server

Add environment variables to your MCP configuration:

**Claude Code** (`~/.claude.json`):
```json
{
  "mcpServers": {
    "google-workspace": {
      "command": "/usr/local/bin/workspace-server",
      "env": {
        "GOOGLE_CLIENT_ID": "your-client-id.apps.googleusercontent.com",
        "GOOGLE_CLIENT_SECRET": "your-client-secret"
      }
    }
  }
}
```

**Claude Desktop** (`claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "google-workspace": {
      "command": "/usr/local/bin/workspace-server",
      "env": {
        "GOOGLE_CLIENT_ID": "your-client-id.apps.googleusercontent.com",
        "GOOGLE_CLIENT_SECRET": "your-client-secret"
      }
    }
  }
}
```

##### 5. Re-authenticate

After configuration, restart the MCP server and run `auth.login` again.

</details>

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
| `gmail.sendWithAttachments` | Send with file attachments |
| `gmail.createDraft` | Create a draft |
| `gmail.sendDraft` | Send a draft |
| `gmail.modify` | Add/remove labels |
| `gmail.listLabels` | List all labels |
| `gmail.createLabel` | Create a label |
| `gmail.deleteLabel` | Delete a label |
| `gmail.downloadAttachment` | Download attachment |
| `gmail.trashMessage` | Move to trash |
| `gmail.untrashMessage` | Restore from trash |
| `gmail.getVacationSettings` | Get vacation settings |
| `gmail.setVacationSettings` | Set vacation auto-reply |

</details>

<details>
<summary><b>Drive Tools</b></summary>

| Tool | Description |
|------|-------------|
| `drive.search` | Search files and folders |
| `drive.findFolder` | Find folder by name |
| `drive.createFolder` | Create a new folder |
| `drive.downloadFile` | Download a file |
| `drive.uploadFile` | Upload a local file |
| `drive.copyFile` | Copy a file |
| `drive.moveFile` | Move a file to folder |
| `drive.deleteFile` | Trash or permanently delete |
| `drive.getFileInfo` | Get detailed file info |
| `drive.shareFile` | Share with user or public |
| `drive.removeShare` | Remove sharing permission |
| `drive.listTrash` | List files in trash |
| `drive.restoreFile` | Restore from trash |
| `drive.emptyTrash` | Empty trash |

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
| `sheets.create` | Create a new spreadsheet |
| `sheets.updateRange` | Update values in range |
| `sheets.appendRows` | Append rows to sheet |
| `sheets.clearRange` | Clear values in range |
| `sheets.createSheet` | Create new sheet tab |
| `sheets.deleteSheet` | Delete sheet tab |

</details>

<details>
<summary><b>Slides Tools</b></summary>

| Tool | Description |
|------|-------------|
| `slides.getText` | Get presentation text |
| `slides.getMetadata` | Get presentation metadata |
| `slides.find` | Search presentations by title |
| `slides.create` | Create new presentation |
| `slides.addSlide` | Add a new slide |
| `slides.deleteSlide` | Delete a slide |
| `slides.addTextBox` | Add text box to slide |
| `slides.addImage` | Add image to slide |
| `slides.updateText` | Update text in shape |

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

### アクセスモード

このサーバーは2つのアクセスモードをサポートしています:

| モード | 権限 | セットアップ |
|--------|------|-------------|
| **Basic** | 読み取り専用 | 不要（デフォルト） |
| **Full Access** | 読み取り + 書き込み + 削除 | カスタムOAuth認証情報が必要 |

#### Basic モード（デフォルト）

追加設定なしで読み取り専用の権限で動作します。

#### Full Access モード

書き込み操作（メール送信、ドキュメント作成など）を有効にするには、独自のOAuth認証情報を設定する必要があります。

<details>
<summary><b>セットアップ手順</b></summary>

##### 1. Google Cloud プロジェクトを作成

1. [Google Cloud Console](https://console.cloud.google.com/) にアクセス
2. 新しいプロジェクトを作成（例: `workspace-mcp`）
3. 以下のAPIを有効化:
   - Google Calendar API
   - Gmail API
   - Google Drive API
   - Google Docs API
   - Google Sheets API
   - Google Slides API
   - Google Chat API
   - People API

##### 2. OAuth 同意画面を設定

1. **APIとサービス** > **OAuth 同意画面** に移動
2. ユーザータイプを選択:
   - **内部**: Google Workspace 組織向け（組織内の全メンバーが利用可能）
   - **外部**: 個人の Gmail アカウント向け（テストユーザーの追加またはGoogle審査が必要）
3. 必須項目を入力（アプリ名、ユーザーサポートメール、デベロッパー連絡先）
4. スコープを追加（またはスキップ - 実行時に要求されます）
5. 外部の場合: テストユーザーを追加（あなたのGmailアドレス）

##### 3. OAuth 認証情報を作成

1. **APIとサービス** > **認証情報** に移動
2. **認証情報を作成** > **OAuth クライアント ID** をクリック
3. **デスクトップアプリ** を選択
4. **クライアントID** と **クライアントシークレット** をコピー

##### 4. MCP サーバーを設定

MCP設定に環境変数を追加:

**Claude Code** (`~/.claude.json`):
```json
{
  "mcpServers": {
    "google-workspace": {
      "command": "/usr/local/bin/workspace-server",
      "env": {
        "GOOGLE_CLIENT_ID": "your-client-id.apps.googleusercontent.com",
        "GOOGLE_CLIENT_SECRET": "your-client-secret"
      }
    }
  }
}
```

**Claude Desktop** (`claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "google-workspace": {
      "command": "/usr/local/bin/workspace-server",
      "env": {
        "GOOGLE_CLIENT_ID": "your-client-id.apps.googleusercontent.com",
        "GOOGLE_CLIENT_SECRET": "your-client-secret"
      }
    }
  }
}
```

##### 5. 再認証

設定後、MCPサーバーを再起動し、`auth.login` を再実行してください。

</details>

### セキュリティ

このサーバーは AI アシスタントに Google Workspace データの読み取り・変更・削除の権限を付与します。以下の点に注意してください:

- AI アシスタントが行うアクションを確認する
- 信頼できないコンテンツ（不明な送信元からのメール、ドキュメントなど）を処理しない
- 認証情報はシステムキーチェーンに安全に保存されます

### ライセンス

Apache License 2.0
