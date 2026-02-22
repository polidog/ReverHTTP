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
internal/
  token/           トークン定義
  lexer/           字句解析器
  ast/             抽象構文木
  parser/          構文解析器 (再帰下降)
  ir/              IR データ構造
  gen/             AST → IR 変換
examples/          サンプル .rever ファイル
testdata/          テスト用データ
spec.md            ReverHTTP v0.1 仕様書
```

## テスト

```bash
go test ./...
```

## 仕様

DSL の詳細な仕様は [spec.md](spec.md) を参照してください。

## ライセンス

MIT
