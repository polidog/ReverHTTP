# ReverHTTP

HTTP フローの宣言的記述フォーマット **ReverHTTP** の DSL パーサー・IR ジェネレーター。

WHAT（処理の流れ）を `.rever` ファイルに記述し、JSON IR（中間表現）に変換します。生成された IR から AI やコードジェネレーターが任意のフレームワーク向けの実装コードを生成できます。

```
WHAT の層 (安定・人が書く)         HOW の層 (変化する・AIが生成)
──────────────────────          ──────────────────────────
HTTPフローの定義                  Express / Hono / FastAPI
データ取得の意図                  Prisma / Drizzle / sqlx
レスポンスの形                    ライブラリ固有のAPI
     │                                ▲
     │   JSON IR                      │
     └───────────→  AI / Generator ───┘
```

## 特徴

- **WHAT と HOW の分離** — HTTP フローの意図を記述し、実装詳細は含まない
- **パイプライン構造** — `|>` で繋がれたステップとしてデータの流れを表現
- **エラーフロー** — `~>` で正常フローとエラーフローを視覚的に分離
- **JSON IR 出力** — 言語・フレームワーク非依存の中間表現
- **外部依存なし** — Go 標準ライブラリのみで実装

## インストール

```bash
go install github.com/polidog/reverhttp/cmd/reverc@latest
```

またはソースからビルド:

```bash
git clone https://github.com/polidog/http-ir.git
cd http-ir
go build -o reverc ./cmd/reverc
```

## 使い方

```bash
# JSON IR を標準出力に表示
reverc input.rever

# ファイルに出力
reverc input.rever -o output.json

# 複数ファイルをマージ
reverc routes.rever types.rever

# コンパクト JSON（インデントなし）
reverc input.rever -indent=false
```

## DSL の例

```
import fetch  = github.com/reverhttp/std-fetch@0.1.0
import create = github.com/reverhttp/std-create@0.1.0

type Article {
  id: int
  title: string
  body: string
  status: string
}

defaults
  cors(origins: ["https://blog.example.com"], credentials)
  auth(bearer)

GET /articles/{id}
  cache(max-age: 300, public, etag: hash(article))
  |> input(id: path.id)
  |> validate(id: int & min(1))                    ~> 400 { error: "invalid id" }
  |> transform(id: int(id))
  |> fetch(Article, id) as article                  ~> 404 { error: "article not found" }
  |> respond 200 { id: article.id, title: article.title, body: article.body }

POST /articles
  auth(bearer, roles: ["author", "admin"]) as current_user
  |> input(title: body.title, body: body.body)
  |> validate(
       title: string & min(1) & max(200),
       body: string & min(1)
     )                                              ~> 400 { error: "validation failed" }
  |> transform(title: trim(title), body: trim(body))
  |> create(Article, { title, body }) as article    ~> 500 { error: "creation failed" }
  |> respond 201 { id: article.id, title: article.title }
```

完全な例は [examples/blog.rever](examples/blog.rever) を参照してください。

## コンパイルパイプライン

```
.rever ファイル → Lexer (字句解析) → Parser (構文解析) → Generator (IR生成) → JSON IR
```

## プロジェクト構成

```
cmd/reverc/        CLI ツール
cmd/rever-lsp/     Language Server (LSP)
internal/
  token/           トークン定義
  lexer/           字句解析器
  ast/             抽象構文木
  parser/          構文解析器 (再帰下降)
  ir/              IR データ構造
  gen/             AST → IR 変換
  lsp/             LSP サーバー実装
editors/vscode/    VS Code 拡張
examples/          サンプル .rever ファイル
testdata/          テスト用データ
spec.md            ReverHTTP v0.1 仕様書
```

## エディタ連携 (Language Server)

ReverHTTP LSP サーバーにより、エディタ上でリアルタイムの構文エラー表示とキーワード補完が利用できます。

### LSP サーバーのインストール

```bash
go install github.com/polidog/reverhttp/cmd/rever-lsp@latest
```

### VS Code

`editors/vscode/` に VS Code 拡張が含まれています。

```bash
cd editors/vscode
npm install
npm run compile
```

拡張をインストールするには、上記でビルド後、VS Code の「拡張機能: VSIX からインストール」を使うか、開発中は以下で起動できます:

```bash
code --extensionDevelopmentPath=./editors/vscode
```

`rever-lsp` が PATH に入っていれば自動的に接続されます。パスをカスタマイズする場合は VS Code の設定で `reverhttp.serverPath` を指定してください。

### Neovim (LazyVim)

`~/.config/nvim/lua/plugins/reverhttp.lua` を作成:

```lua
vim.filetype.add({
  extension = {
    rever = "rever",
  },
})

vim.api.nvim_create_autocmd("FileType", {
  pattern = "rever",
  callback = function()
    vim.bo.commentstring = "# %s"
  end,
})

return {
  {
    "neovim/nvim-lspconfig",
    opts = {
      servers = {
        rever_lsp = {},
      },
      setup = {
        rever_lsp = function()
          local configs = require("lspconfig.configs")
          if not configs.rever_lsp then
            configs.rever_lsp = {
              default_config = {
                cmd = { "rever-lsp" },
                filetypes = { "rever" },
                root_dir = require("lspconfig.util").root_pattern(".git", "go.mod"),
                settings = {},
              },
            }
          end
        end,
      },
    },
  },
}
```

シンタックスハイライトを追加するには `~/.config/nvim/syntax/rever.vim` を作成してください。サンプルはリポジトリの `editors/vscode/` を参考にしてください。

### Neovim (nvim-lspconfig のみ)

LazyVim を使わない場合は `init.lua` に直接記述できます:

```lua
vim.filetype.add({ extension = { rever = "rever" } })

local configs = require("lspconfig.configs")
if not configs.rever_lsp then
  configs.rever_lsp = {
    default_config = {
      cmd = { "rever-lsp" },
      filetypes = { "rever" },
      root_dir = require("lspconfig.util").root_pattern(".git", "go.mod"),
    },
  }
end
require("lspconfig").rever_lsp.setup({})
```

## テスト

```bash
go test ./...
```

## 仕様

DSL の詳細な仕様は [spec.md](spec.md) を参照してください。

## ライセンス

MIT
