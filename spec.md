# ReverHTTP Specification v0.1 (Draft)

> HTTPフローの宣言的記述フォーマット
>
> WHAT（処理の流れ）を記述し、HOW（実装）はAI/コードジェネレータに委ねる。

---

# 1. 背景と目的

ライブラリやフレームワークの変化は速い。しかし「HTTPリクエストを受けて、どう処理し、どう返すか」という**フロー**は安定している。

ReverHTTP は、このフローを実装から分離して記述するためのフォーマットである。

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

## 主な目的

- HTTPリクエストの処理フローを宣言的に記述する
- フローを JSON IR として表現し、言語・フレームワーク非依存にする
- AI / コードジェネレータが IR から実装コードを生成できるようにする
- 実装技術の変化からフロー定義を守る

---

# 2. 設計原則

1. **WHATとHOWの分離** — フローは処理の流れと意図を記述し、実装詳細を含まない
2. **HTTP が第一級概念** — HTTP リクエスト/レスポンスの記述に特化する
3. **パイプライン構造** — 全ルートを `|>` で繋がれたパイプラインとして記述する。データが各ステップを流れていく様を構文で表現する
4. **正常フローとエラーフローの分離** — `|>` が正常フロー、`~>` がエラーフローを表し、視覚的に区別できる
5. **コアは最小、拡張はパッケージ** — DSL のビルトインはHTTPフロー制御のみ。データ操作や外部連携は全てパッケージとして import する
6. **JSON IR** — DSL は JSON に変換可能であり、それが正式な中間表現となる

---

# 3. 全体構造

ReverHTTP のファイルは4つのセクションで構成される。

```
┌─────────────────────────────┐
│  Imports (パッケージ宣言)     │  外部ステップの読み込み
├─────────────────────────────┤
│  Types (型定義)              │  エンティティの形を宣言
├─────────────────────────────┤
│  Defaults (デフォルト指令)    │  全ルートに適用されるルートレベル指令
├─────────────────────────────┤
│  Routes (フロー定義)         │  HTTPリクエストの処理フローを宣言
└─────────────────────────────┘
```

---

# 4. Types（型定義）

エンティティのデータ構造を定義する。

## DSL

```
type User {
  id: int
  name: string
  email: string
  created_at: datetime
}

type Post {
  id: int
  title: string
  body: string
  author_id: int
}
```

## JSON IR

```json
{
  "types": {
    "User": {
      "id": "int",
      "name": "string",
      "email": "string",
      "created_at": "datetime"
    },
    "Post": {
      "id": "int",
      "title": "string",
      "body": "string",
      "author_id": "int"
    }
  }
}
```

## 型一覧（v0.1）

| 型 | 説明 |
|----|------|
| `int` | 整数 |
| `string` | 文字列 |
| `bool` | 真偽値 |
| `float` | 浮動小数点 |
| `datetime` | 日時（ISO8601文字列として扱う） |

---

# 5. Routes（フロー定義）

HTTP リクエストの処理フローを `|>` パイプで繋いだパイプラインとして記述する。

```
GET /users/{id}
  cache(...)               ルートレベル指令（任意）
  |> input(...)            リクエストから値を取り出す
  |> validate(...)         入力値を検証する               ~> エラーレスポンス
  |> transform(...)        値を変換する
  |> pkg(...)              importしたステップ              ~> エラーレスポンス
  |> guard expr            条件ゲート                     ~> エラーレスポンス
  |> match expr { ... }    値によるパターンマッチ分岐     ~> エラーレスポンス
  |> respond N             レスポンスを返す（ボディなしも可）
```

## ルートレベル指令

ルートレベル指令は、パイプラインステップ（`|>`）ではなく、ルート全体に適用される横断的関心事を宣言する。ルート宣言の直後、最初の `|>` の前にインデントして記述する。

| 指令 | 役割 |
|------|------|
| **cache(...)** | HTTPキャッシュの振る舞いを宣言する（§13） |
| **cors(...)** | CORSヘッダーを宣言する（§14） |
| **auth(...)** | 認証・認可を宣言する（§15） |

## ビルトインステップ一覧

コアDSLが提供するステップ。HTTPフロー制御に特化している。

| ステップ | 役割 |
|----------|------|
| **input(...)** | HTTPリクエストから値を取り出す |
| **validate(...)** | 入力値の形式を検証する。制約は `&` で合成する |
| **transform(...)** | 値を変換する（型変換、文字列処理等） |
| **guard** | 条件を検証し、偽ならエラーフローへ |
| **match** | 値によるパターンマッチで分岐する（各アームはシングルステップ） |
| **respond** | HTTPステータスコードとレスポンスを返す。ボディは任意。`with headers` でカスタムヘッダーを付与できる |

データ操作（fetch、create、update、delete 等）はビルトインではなく、パッケージとして import する。

## DSL 構文要素

| 構文 | 意味 |
|------|------|
| `\|>` | 正常フロー — 次のステップへデータを流す |
| `~>` | エラーフロー — 失敗時のレスポンスを宣言する |
| `as name` | ステップの結果を変数に束縛する |
| `&` | バリデーション制約の合成 |
| `import` | パッケージの読み込みとエイリアス宣言 |
| `with headers { ... }` | respond にカスタムレスポンスヘッダーを付与する |

## respond の構文

`respond` はパイプラインの最終ステップとしてHTTPレスポンスを返す。ボディは任意であり、`with headers` でカスタムヘッダーを付与できる。

### 構文パターン

```
|> respond 204                                              # ボディなし
|> respond 200 { id: user.id, name: user.name }             # ボディあり
|> respond 200 { id: user.id } with headers { x-req: req.id }  # ボディ + ヘッダー
|> respond 301 with headers { location: "/new" }            # リダイレクト
```

### JSON IR

```json
{ "output": { "status": 204 } }
{ "output": { "status": 301, "headers": { "location": "/new" } } }
{ "output": { "status": 200, "body": { "id": "user.id", "name": "user.name" }, "headers": { "x-req": "req.id" } } }
```

---

# 6. 例: GET /users/{id}

ユーザーIDからユーザー情報を取得して返す。

## DSL

```
import fetch = github.com/reverhttp/std-fetch@0.1.0

GET /users/{id}
  |> input(id: path.id)
  |> validate(id: int & min(1))          ~> 400 { error: "invalid id" }
  |> transform(id: int(id))
  |> fetch(User, id) as user             ~> 404 { error: "user not found" }
  |> respond 200 { id: user.id, name: user.name, email: user.email }
```

## JSON IR

```json
{
  "imports": {
    "fetch": {
      "source": "github.com/reverhttp/std-fetch",
      "version": "0.1.0"
    }
  },
  "route": { "method": "GET", "path": "/users/{id}" },
  "input": {
    "id": { "from": "path.id" }
  },
  "validate": {
    "rules": { "id": { "type": "int", "min": 1 } },
    "error": { "status": 400, "body": { "error": "invalid id" } }
  },
  "transform_in": {
    "id": { "cast": "int", "from": "id" }
  },
  "process": {
    "steps": [
      {
        "bind": "user",
        "use": "fetch",
        "input": { "type": "User", "id": "id" },
        "error": { "status": 404, "body": { "error": "user not found" } }
      }
    ]
  },
  "output": {
    "status": 200,
    "body": {
      "id": "user.id",
      "name": "user.name",
      "email": "user.email"
    }
  }
}
```

---

# 7. 例: POST /users

ユーザーを新規作成する。

## DSL

```
import fetch  = github.com/reverhttp/std-fetch@0.1.0
import create = github.com/reverhttp/std-create@0.1.0

POST /users
  |> input(name: body.name, email: body.email)
  |> validate(
       name: string & min(1) & max(100),
       email: string & format(email)
     )                                   ~> 400 { error: "validation failed", details: errors }
  |> transform(name: trim(name), email: lower(email))
  |> fetch(User, email: email) as existing
  |> guard !existing                     ~> 409 { error: "email already taken" }
  |> create(User, { name, email }) as user ~> 500 { error: "creation failed" }
  |> respond 201 { id: user.id, name: user.name, email: user.email }
```

## JSON IR

```json
{
  "imports": {
    "fetch": {
      "source": "github.com/reverhttp/std-fetch",
      "version": "0.1.0"
    },
    "create": {
      "source": "github.com/reverhttp/std-create",
      "version": "0.1.0"
    }
  },
  "route": { "method": "POST", "path": "/users" },
  "input": {
    "name": { "from": "body.name" },
    "email": { "from": "body.email" }
  },
  "validate": {
    "rules": {
      "name": { "type": "string", "min": 1, "max": 100 },
      "email": { "type": "string", "format": "email" }
    },
    "error": { "status": 400, "body": { "error": "validation failed", "details": "errors" } }
  },
  "transform_in": {
    "name": { "fn": "trim", "from": "name" },
    "email": { "fn": "lower", "from": "email" }
  },
  "process": {
    "steps": [
      {
        "bind": "existing",
        "use": "fetch",
        "input": { "type": "User", "email": "email" }
      },
      {
        "guard": { "not": "existing" },
        "error": { "status": 409, "body": { "error": "email already taken" } }
      },
      {
        "bind": "user",
        "use": "create",
        "input": { "type": "User", "data": { "name": "name", "email": "email" } },
        "error": { "status": 500, "body": { "error": "creation failed" } }
      }
    ]
  },
  "output": {
    "status": 201,
    "body": {
      "id": "user.id",
      "name": "user.name",
      "email": "user.email"
    }
  }
}
```

---

# 8. 例: PUT /users/{id}

ユーザー情報を更新する。

## DSL

```
import fetch  = github.com/reverhttp/std-fetch@0.1.0
import update = github.com/reverhttp/std-update@0.1.0

PUT /users/{id}
  |> input(id: path.id, name: body.name, email: body.email)
  |> validate(
       id: int & min(1),
       name: string & min(1) & max(100),
       email: string & format(email)
     )                                   ~> 400 { error: "validation failed" }
  |> transform(id: int(id), name: trim(name), email: lower(email))
  |> fetch(User, id) as user            ~> 404 { error: "user not found" }
  |> update(User, id, { name, email }) as user ~> 500 { error: "update failed" }
  |> respond 200 { id: user.id, name: user.name, email: user.email }
```

---

# 9. 例: DELETE /users/{id}

ユーザーを削除する。

## DSL

```
import fetch  = github.com/reverhttp/std-fetch@0.1.0
import delete = github.com/reverhttp/std-delete@0.1.0

DELETE /users/{id}
  |> input(id: path.id)
  |> validate(id: int & min(1))          ~> 400 { error: "invalid id" }
  |> transform(id: int(id))
  |> fetch(User, id) as user            ~> 404 { error: "user not found" }
  |> delete(User, id)                   ~> 500 { error: "delete failed" }
  |> respond 200 { deleted: true }
```

---

# 10. 例: GET /accounts/{id}（match 式）

ロールに応じて異なる型のデータを取得する。

## DSL

```
import fetch = github.com/reverhttp/std-fetch@0.1.0

GET /accounts/{id}
  |> input(id: path.id, role: header.x-role)
  |> validate(id: int & min(1), role: string)  ~> 400 { error: "invalid input" }
  |> transform(id: int(id))
  |> match role {
       "user":  fetch(User, id)
       "admin": fetch(Admin, id)
       _:                                      ~> 400 { error: "unknown role" }
     } as account                              ~> 404 { error: "account not found" }
  |> respond 200 { id: account.id, name: account.name, role: role }
```

## JSON IR

```json
{
  "imports": {
    "fetch": {
      "source": "github.com/reverhttp/std-fetch",
      "version": "0.1.0"
    }
  },
  "route": { "method": "GET", "path": "/accounts/{id}" },
  "input": {
    "id": { "from": "path.id" },
    "role": { "from": "header.x-role" }
  },
  "validate": {
    "rules": {
      "id": { "type": "int", "min": 1 },
      "role": { "type": "string" }
    },
    "error": { "status": 400, "body": { "error": "invalid input" } }
  },
  "transform_in": {
    "id": { "cast": "int", "from": "id" }
  },
  "process": {
    "steps": [
      {
        "bind": "account",
        "match": {
          "on": "role",
          "arms": [
            { "pattern": { "value": "user" }, "use": "fetch", "input": { "type": "User", "id": "id" } },
            { "pattern": { "value": "admin" }, "use": "fetch", "input": { "type": "Admin", "id": "id" } }
          ],
          "default": {
            "error": { "status": 400, "body": { "error": "unknown role" } }
          }
        },
        "error": { "status": 404, "body": { "error": "account not found" } }
      }
    ]
  },
  "output": {
    "status": 200,
    "body": {
      "id": "account.id",
      "name": "account.name",
      "role": "role"
    }
  }
}
```

---

# 11. match 式

値によるパターンマッチでパイプラインを分岐する。各アームには**シングルステップ**を記述する。

## 構文

```
|> match <expr> {
     <pattern>: <step>
     <pattern>: <step>
     _: <step or ~> error>
   } as <name>                ~> <status> { <body> }
```

- `match <expr>` — 式の値でアームを選択する
- 各アームにはひとつのステップを書く
- `_` はデフォルトアーム（どのパターンにも一致しない場合）
- `as name` — 実行されたアームの結果を変数に束縛する
- `_:` に `~>` を書くと、一致なしをエラーにできる
- `}` の後の `~>` は、アーム内のステップが失敗した場合のエラー

## パターンの種類

| パターン | DSL 例 | 説明 |
|----------|--------|------|
| リテラル | `1`, `"admin"`, `true` | 値の完全一致 |
| 複数値 | `"user", "member"` | いずれかに一致（OR） |
| 範囲 | `1..100` | 数値の範囲（両端を含む） |
| 正規表現 | `/^admin/` | 正規表現による文字列マッチ |
| ワイルドカード | `_` | すべてに一致（デフォルト） |

## 例: 各パターンの使用

```
import fetch = github.com/reverhttp/std-fetch@0.1.0
import call  = github.com/reverhttp/std-call@0.1.0

|> match role {
     "user", "member": fetch(User, id)
     "admin":          fetch(Admin, id)
     _:                                ~> 400 { error: "unknown role" }
   } as account

|> match status_code {
     200..299: call(handleSuccess, response)
     400..499: call(handleClientError, response)
     _:        call(handleServerError, response)
   } as result

|> match path {
     /^\/api\/v1/: fetch(V1Handler, path)
     /^\/api\/v2/: fetch(V2Handler, path)
     _:                                ~> 404 { error: "not found" }
   } as handler
```

## 各アームに個別のエラー

各アームにもステップレベルの `~>` を付与できる。

```
|> match role {
     "user":  fetch(User, id)        ~> 404 { error: "user not found" }
     "admin": fetch(Admin, id)       ~> 404 { error: "admin not found" }
     _:                              ~> 400 { error: "unknown role" }
   } as account
```

## JSON IR マッピング

パターンの種類ごとに JSON IR での表現が異なる。

| DSL パターン | JSON IR |
|-------------|---------|
| `"admin"` | `{ "value": "admin" }` |
| `"user", "member"` | `{ "in": ["user", "member"] }` |
| `1..100` | `{ "range": { "min": 1, "max": 100 } }` |
| `/^admin/` | `{ "regex": "^admin" }` |
| `_` | `"default"` キー |

```json
{
  "bind": "account",
  "match": {
    "on": "role",
    "arms": [
      { "pattern": { "in": ["user", "member"] }, "use": "fetch", "input": { "type": "User", "id": "id" } },
      { "pattern": { "value": "admin" }, "use": "fetch", "input": { "type": "Admin", "id": "id" } }
    ],
    "default": {
      "error": { "status": 400, "body": { "error": "unknown role" } }
    }
  }
}
```

---

# 12. エラー処理

`~>` でエラーフローを宣言する。ステップが失敗した場合、対応するエラーレスポンスを返しパイプラインを中断する。

```
|> validate(id: int & min(1))  ~> 400 { error: "invalid id" }
                               ──  ───  ─────────────────────
                                │   │    └── レスポンスボディ
                                │   └── HTTPステータスコード
                                └── エラーフロー演算子
```

各ステップに個別の `~>` を付与できる。

```
|> fetch(User, id) as user              ~> 404 { error: "user not found" }
|> guard user.active                    ~> 403 { error: "user is deactivated" }
```

`~>` を省略したステップは、失敗してもエラーとならない（`fetch` で見つからなければ `null` が束縛される等）。

---

# 13. HTTP キャッシュ

`cache(...)` はルートレベル指令であり、パイプラインステップ（`|>`）ではなく横断的関心事としてルートに宣言する。レスポンスキャッシュヘッダー（Cache-Control, ETag, Last-Modified, Vary）と条件付きリクエスト（If-None-Match, If-Modified-Since → 304 Not Modified）の振る舞いを宣言的に記述する。

`etag` / `last-modified` の式はパイプライン中の変数を前方参照し、ジェネレータが適切なタイミングで評価する。

## DSL

```
GET /users/{id}
  cache(max-age: 3600, public, etag: hash(user))
  |> input(id: path.id)
  |> validate(id: int & min(1))          ~> 400 { error: "invalid id" }
  |> transform(id: int(id))
  |> fetch(User, id) as user             ~> 404 { error: "user not found" }
  |> respond 200 { id: user.id, name: user.name, email: user.email }
```

`cache(...)` はルート宣言の直後、最初の `|>` の前に書く。

## パラメータ一覧

| パラメータ | 型 | 説明 |
|---|---|---|
| `max-age` | int | Cache-Control max-age（秒） |
| `s-maxage` | int | Cache-Control s-maxage（共有キャッシュ向け） |
| `public` | flag | Cache-Control: public |
| `private` | flag | Cache-Control: private |
| `no-cache` | flag | Cache-Control: no-cache（常に再検証） |
| `no-store` | flag | Cache-Control: no-store（キャッシュ禁止） |
| `etag` | expr | ETag ヘッダーの値。条件付きリクエスト（If-None-Match → 304）を有効化 |
| `last-modified` | expr | Last-Modified ヘッダーの値。条件付きリクエスト（If-Modified-Since → 304）を有効化 |
| `vary` | list | Vary ヘッダー（キャッシュのキーとなるリクエストヘッダーを指定） |

## キャッシュ制御ヘッダー

`cache(...)` のパラメータは以下のレスポンスヘッダーに変換される。

```
cache(max-age: 3600, public, etag: hash(user), vary: [Accept])
```

↓ 生成されるレスポンスヘッダー

```
Cache-Control: public, max-age=3600
ETag: "a1b2c3d4"
Vary: Accept
```

- `max-age`, `s-maxage`, `public`, `private`, `no-cache`, `no-store` → `Cache-Control` ヘッダーのディレクティブとして結合
- `etag` → `ETag` ヘッダー（式の評価結果をダブルクォートで囲む）
- `last-modified` → `Last-Modified` ヘッダー（ISO8601 → HTTP-date 形式に変換）
- `vary` → `Vary` ヘッダー

## 条件付きリクエスト

`etag` または `last-modified` を宣言すると、条件付きリクエストが自動的に有効化される。

### ETag による検証

`etag` 宣言時、ジェネレータは以下の振る舞いを実装する。

1. レスポンスに `ETag` ヘッダーを付与
2. リクエストに `If-None-Match` ヘッダーがある場合、ETag 値と比較
3. 一致すれば `304 Not Modified`（ボディなし）を返し、パイプラインの残りをスキップ

### Last-Modified による検証

`last-modified` 宣言時、ジェネレータは以下の振る舞いを実装する。

1. レスポンスに `Last-Modified` ヘッダーを付与
2. リクエストに `If-Modified-Since` ヘッダーがある場合、日時を比較
3. 変更がなければ `304 Not Modified`（ボディなし）を返し、パイプラインの残りをスキップ

### 優先順位

`etag` と `last-modified` の両方が宣言されている場合、`etag`（If-None-Match）が優先される。両方のヘッダーがリクエストに存在する場合、ETag の一致を先に評価し、一致すれば 304 を返す。

## JSON IR

```json
{
  "cache": {
    "max_age": 3600,
    "visibility": "public",
    "etag": { "fn": "hash", "from": "user" },
    "vary": ["Accept"]
  }
}
```

| DSL | JSON IR |
|-----|---------|
| `max-age: 3600` | `"max_age": 3600` |
| `s-maxage: 600` | `"s_maxage": 600` |
| `public` | `"visibility": "public"` |
| `private` | `"visibility": "private"` |
| `no-cache` | `"no_cache": true` |
| `no-store` | `"no_store": true` |
| `etag: hash(user)` | `"etag": { "fn": "hash", "from": "user" }` |
| `etag: user.version` | `"etag": "user.version"` |
| `last-modified: user.updated_at` | `"last_modified": "user.updated_at"` |
| `vary: [Accept, Authorization]` | `"vary": ["Accept", "Authorization"]` |

## 使用パターン

### 時間ベースキャッシュ

```
GET /public/config
  cache(max-age: 3600, public)
  |> ...
```

### ETag 検証付きキャッシュ

```
GET /users/{id}
  cache(max-age: 3600, public, etag: hash(user))
  |> input(id: path.id)
  |> validate(id: int & min(1))          ~> 400 { error: "invalid id" }
  |> transform(id: int(id))
  |> fetch(User, id) as user             ~> 404 { error: "user not found" }
  |> respond 200 { id: user.id, name: user.name, email: user.email }
```

```json
{
  "route": { "method": "GET", "path": "/users/{id}" },
  "cache": {
    "max_age": 3600,
    "visibility": "public",
    "etag": { "fn": "hash", "from": "user" }
  },
  "input": { "id": { "from": "path.id" } },
  "validate": {
    "rules": { "id": { "type": "int", "min": 1 } },
    "error": { "status": 400, "body": { "error": "invalid id" } }
  },
  "transform_in": {
    "id": { "cast": "int", "from": "id" }
  },
  "process": {
    "steps": [
      {
        "bind": "user",
        "use": "fetch",
        "input": { "type": "User", "id": "id" },
        "error": { "status": 404, "body": { "error": "user not found" } }
      }
    ]
  },
  "output": {
    "status": 200,
    "body": {
      "id": "user.id",
      "name": "user.name",
      "email": "user.email"
    }
  }
}
```

### 常に再検証

```
GET /users/{id}/profile
  cache(no-cache, etag: hash(user))
  |> ...
```

### キャッシュ禁止

```
POST /users
  cache(no-store)
  |> ...
```

### Last-Modified による検証

```
GET /articles/{id}
  cache(max-age: 600, public, last-modified: article.updated_at)
  |> input(id: path.id)
  |> validate(id: int & min(1))            ~> 400 { error: "invalid id" }
  |> transform(id: int(id))
  |> fetch(Article, id) as article         ~> 404 { error: "not found" }
  |> respond 200 { id: article.id, title: article.title, body: article.body }
```

```json
{
  "route": { "method": "GET", "path": "/articles/{id}" },
  "cache": {
    "max_age": 600,
    "visibility": "public",
    "last_modified": "article.updated_at"
  }
}
```

---

# 14. Defaults とルートレベル指令の継承

`defaults` ブロックはルートレベル指令のデフォルト値を宣言する。`defaults` に記述した指令は全ルートに適用され、個別ルートで上書きできる。

## DSL

```
defaults
  cors(origins: ["*"])
  auth(bearer)
```

`defaults` はファイルの Types と Routes の間に記述する。

## CORS

CORS（Cross-Origin Resource Sharing）は Web API でほぼ必須の横断的関心事であり、`defaults` の主要ユースケースである。`cors(...)` はルートレベル指令としてCORSヘッダーの振る舞いを宣言する。

### パラメータ一覧

| パラメータ | 型 | 説明 |
|---|---|---|
| `origins` | list | Access-Control-Allow-Origin（`["*"]` で全許可） |
| `methods` | list | Access-Control-Allow-Methods |
| `headers` | list | Access-Control-Allow-Headers |
| `expose-headers` | list | Access-Control-Expose-Headers |
| `max-age` | int | Access-Control-Max-Age（プリフライトキャッシュ秒数） |
| `credentials` | flag | Access-Control-Allow-Credentials: true |
| `none` | keyword | CORS を無効化（defaults の上書き用） |

### defaults + ルート上書きの例

```
defaults
  cors(origins: ["https://app.example.com"], credentials)

GET /api/users
  |> ...                       # defaults の cors が適用

GET /public/health
  cors(none)                   # このルートだけ CORS 無効化
  |> respond 200 { status: "ok" }

GET /api/admin
  cors(origins: ["https://admin.example.com"])  # 別設定で上書き
  auth(bearer, roles: ["admin"])
  |> ...
```

### JSON IR

```json
{
  "defaults": {
    "cors": {
      "origins": ["https://app.example.com"],
      "credentials": true
    }
  }
}
```

ルート単位の上書き:

```json
{
  "cors": null
}
```

`cors(none)` → `"cors": null` で無効化を表現する。

---

# 15. 認証・認可

`auth(...)` はルートレベル指令として認証・認可の要件を宣言する。`defaults` で全ルートに適用し、個別ルートで上書きできる。

## DSL

```
GET /admin/users
  auth(bearer, roles: ["admin"]) as current_user
  |> fetch(User) as users
  |> respond 200 { users: users, requested_by: current_user.name }
```

- `as` で認証済みエンティティを変数に束縛（任意）
- 認証失敗 → 401 Unauthorized
- 認可失敗（role/permission不足） → 403 Forbidden

## パラメータ一覧

| パラメータ | 型 | 説明 |
|---|---|---|
| (第1引数) | keyword | 認証方式（`bearer`, `api-key`, `basic`） |
| `roles` | list | 必要なロール（いずれかに一致で認可） |
| `permissions` | list | 必要なパーミッション（すべてに一致で認可） |
| `none` | keyword | 認証を無効化（defaults の上書き用） |

## JSON IR

```json
{
  "auth": {
    "method": "bearer",
    "roles": ["admin"],
    "bind": "current_user"
  }
}
```

## 使用パターン

### 認証のみ

```
GET /profile
  auth(bearer)
  |> ...
```

### 認証 + ロール

```
GET /admin/users
  auth(bearer, roles: ["admin", "editor"])
  |> ...
```

### 認証 + パーミッション

```
DELETE /users/{id}
  auth(bearer, permissions: ["users:read", "users:write"])
  |> ...
```

### API キー

```
GET /api/data
  auth(api-key)
  |> ...
```

### 認証無効化

```
GET /public/health
  auth(none)
  |> respond 200 { status: "ok" }
```

---

# 16. カスタムステップ（import）

`import` でパッケージを読み込むと、エイリアス名がパイプラインステップとして使えるようになる。ビルトインステップと同じ感覚で呼び出せる。

## 構文

```
import <alias> = <source>@<version>
```

| 要素 | 説明 |
|------|------|
| `<alias>` | パイプライン内で使う名前 |
| `<source>` | GitHubリポジトリパス or ローカルパス |
| `@<version>` | Gitタグ（`@0.1.0`）または `@latest` |

## データ層の切り替え

import を変えるだけで、DSL のルート定義を一切変えずにデータ層が差し替わる。

```
# Doctrine ORM を使いたい
import fetch = github.com/reverhttp/doctrine-fetch@0.1.0

# Prisma を使いたい
import fetch = github.com/reverhttp/prisma-fetch@0.1.0

# 汎用（AIが生成）
import fetch = github.com/reverhttp/std-fetch@0.1.0
```

## 例: キャッシュ経由でのデータ取得

```
import fetch       = github.com/reverhttp/std-fetch@0.1.0
import redis-cache = github.com/someone/redis-cache@2.0.0

GET /users/{id}/cached
  |> input(id: path.id)
  |> validate(id: int & min(1))                    ~> 400 { error: "invalid id" }
  |> transform(id: int(id))
  |> redis-cache(key: "user:{id}") as cached
  |> match cached {
       null: fetch(User, id)
       _:    cached
     } as user                                     ~> 404 { error: "user not found" }
  |> respond 200 { id: user.id, name: user.name }
```

## JSON IR

```json
{
  "imports": {
    "fetch": {
      "source": "github.com/reverhttp/std-fetch",
      "version": "0.1.0"
    },
    "redis-cache": {
      "source": "github.com/someone/redis-cache",
      "version": "2.0.0"
    }
  }
}
```

---

# 17. パッケージ

カスタムステップはパッケージとして配布する。1パッケージ = 1ステップ。

## パッケージ構造

```
doctrine-fetch/
  step.rever              # インターフェース + 自然言語の説明（必須）
  package.json           # メタデータ（必須）
  impl/                  # 言語別の実装（任意）
    php/handler.php
    ts/handler.ts
    go/handler.go
```

## step.rever — インターフェース定義

ステップの引数・戻り値の型と、自然言語による処理内容の記述。

```
step doctrine-fetch(entity: string, id: int) -> any {
  """
  Doctrine ORMを使ってエンティティを取得する。
  - EntityManager::find() を使用
  - soft deleteされたレコードは除外
  - リレーションはeager loadingで取得
  """
}
```

- 引数と戻り値の型はステップのインターフェースとして公開される
- `"""` 内の自然言語はAI/ジェネレータへの指示となる
- 実装コードは含まない（WHATの原則を維持）

## package.json — メタデータ

```json
{
  "name": "doctrine-fetch",
  "version": "0.1.0",
  "description": "Fetch entity via Doctrine ORM",
  "license": "MIT",
  "implements": ["php"]
}
```

- `implements` — `impl/` に実装が存在する言語の一覧

## impl/ — 言語別実装

対象言語の実装コードを格納する。

```php
// impl/php/handler.php
function doctrineFetch(string $entity, int $id): mixed
{
    return $em->getRepository($entity)->find($id);
}
```

実装内部で必要な依存（`$em` 等）の解決はジェネレータ/フレームワーク側の責務であり、ステップの引数には含めない。

## ジェネレータの解決フロー

```
fetch(User, id)
     │
     ▼
impl/ に対象言語の実装がある？
   ├── Yes → その実装コードを組み込む
   └── No  → step.rever の自然言語記述をもとにAIが生成
```

実装がなくても `step.rever` の記述だけで動く。パッケージ作者が全言語に対応する必要はない。

## バージョン管理

パッケージはGitHubリポジトリとして公開し、Gitタグでバージョンを管理する。

| 指定 | 意味 |
|------|------|
| `@0.1.0` | タグ `0.1.0` に固定 |
| `@latest` | 最新タグを使用 |

## ロックファイル（rever.lock.json）

再現性のために、バージョン解決の結果をロックファイルに記録する。

```json
{
  "packages": {
    "fetch": {
      "source": "github.com/reverhttp/std-fetch",
      "version": "0.1.0",
      "commit": "a1b2c3d4e5f6",
      "hash": "sha256:..."
    },
    "create": {
      "source": "github.com/reverhttp/std-create",
      "version": "0.1.0",
      "commit": "b2c3d4e5f6a1",
      "hash": "sha256:..."
    }
  }
}
```

- `@latest` は解決時点の最新タグでロックされる
- `rever.lock.json` をリポジトリにコミットすれば、チーム全員が同じバージョンで動く

## プロジェクト構成例

```
my-project/
  rever.lock.json                # ロックファイル（自動生成）
  steps/                        # ローカルのカスタムステップ
    custom-fetch/
      step.rever
  routes/
    users.rever                  # ルート定義
    accounts.rever
```

---

# 18. DSL から JSON IR へのマッピング

DSL のパイプラインステップは、JSON IR の対応するセクションに変換される。

| DSL ステップ | JSON IR セクション |
|-------------|-------------------|
| `defaults` | `"defaults"` |
| `cache(...)` | `"cache"` |
| `cors(...)` | `"cors"` |
| `auth(...)` | `"auth"` |
| `input(...)` | `"input"` |
| `validate(...)` | `"validate"` (`"rules"` + `"error"`) |
| `transform(...)` | `"transform_in"` |
| `guard` / `match` / importしたステップ | `"process"` (`"steps"` 配列) |
| `import` | `"imports"` |
| `respond N` | `"output"` (`"status"` のみ) |
| `respond N { ... }` | `"output"` (`"status"` + `"body"`) |
| `with headers { ... }` | `"output"."headers"` |
| `~> N { ... }` | 各セクションの `"error"` |
| `as name` | ステップの `"bind"` |

importしたステップは JSON IR では `"use": "<alias>"` としてマッピングされる。

---

# 19. IR 全体構造

ひとつのアプリケーションの IR 全体像。

```json
{
  "version": "0.1",
  "imports": {
    "fetch": {
      "source": "github.com/reverhttp/std-fetch",
      "version": "0.1.0"
    }
  },
  "types": {
    "User": {
      "id": "int",
      "name": "string",
      "email": "string",
      "created_at": "datetime"
    }
  },
  "defaults": {
    "cors": { "origins": ["*"] },
    "auth": { "method": "bearer" }
  },
  "routes": [
    {
      "route": { "method": "GET", "path": "/users/{id}" },
      "auth": { "method": "bearer", "roles": ["admin"], "bind": "current_user" },
      "cache": {
        "max_age": 3600,
        "visibility": "public",
        "etag": { "fn": "hash", "from": "user" }
      },
      "input": { "id": { "from": "path.id" } },
      "validate": {
        "rules": { "id": { "type": "int", "min": 1 } },
        "error": { "status": 400, "body": { "error": "invalid id" } }
      },
      "transform_in": {
        "id": { "cast": "int", "from": "id" }
      },
      "process": {
        "steps": [
          {
            "bind": "user",
            "use": "fetch",
            "input": { "type": "User", "id": "id" },
            "error": { "status": 404, "body": { "error": "user not found" } }
          }
        ]
      },
      "output": {
        "status": 200,
        "body": {
          "id": "user.id",
          "name": "user.name",
          "email": "user.email"
        }
      }
    }
  ]
}
```

---

# 20. 既存技術との位置づけ

```
        形だけ              流れも書ける         実装も含む
        ←─────────────────────────────────────────→

OpenAPI    Smithy     Step Functions    Ballerina
TypeSpec              Camel
                      Temporal
                                   ↑
                             ここが ReverHTTP
                        「HTTPフローの宣言的記述」
                          実装を含まない
```

| 技術 | 共通点 | 違い |
|------|--------|------|
| OpenAPI / Smithy / TypeSpec | API の形を定義 | フロー（処理の流れ）は書けない |
| AWS Step Functions / Camel | ワークフロー記述 | HTTP API 特化ではない |
| Ballerina | HTTP 統合が第一級 | 完全な言語であり HOW も含む |
| **ReverHTTP** | — | HTTP フローの WHAT のみ。HOW は AI/Generator に委ねる |

---

# 21. 今後の拡張（非 v0.1）

- match アーム内の複数ステップ（パイプラインのネスト）
- ページネーション支援
- middleware / hooks
- バッチ処理・並列処理
- ルートレベル指令の拡充（`rate-limit(...)` 等）
- defaults の拡張（パスプレフィックスによるグルーピング等）

---

# 22. まとめ

ReverHTTP は：

- **HTTP フローの WHAT** を宣言的に記述するフォーマット
- **`|>` パイプライン** でデータの流れを視覚的に表現し、`~>` でエラーフローを分離
- **コアは最小** — ビルトインはHTTPフロー制御のみ。データ操作は全てパッケージとして import
- **パッケージシステム** — GitHub ベースの配布。import を変えるだけでデータ層が差し替わる
- **JSON IR** により言語・フレームワーク非依存
- **AI / コードジェネレータ** が IR から実装を生成する前提の設計
- 実装技術の変化からフロー定義を**分離・保護**する
