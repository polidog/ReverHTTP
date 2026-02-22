# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

ReverHTTP DSL のパーサー・IR ジェネレーター。`.rever` ファイルを JSON IR（中間表現）に変換する Go 製コンパイラ。
モジュール名: `github.com/polidog/reverhttp`、外部依存なし（Go 標準ライブラリのみ）。

## コマンド

```bash
# ビルド
go build -o reverc ./cmd/reverc

# テスト（全パッケージ）
go test ./...

# 単一テスト実行
go test -run TestNextToken_Operators ./internal/lexer/ -v

# 特定パッケージのテスト
go test ./internal/parser -v

# フォーマット・静的解析
go fmt ./...
go vet ./...
```

## アーキテクチャ

4段階のコンパイルパイプライン:

```
.rever → Lexer (字句解析) → Parser (構文解析) → Generator (IR生成) → JSON IR
```

- **token**: トークン型定義。`|>` (パイプ), `~>` (エラーフロー), HTTP メソッド等
- **lexer**: 字句解析器。括弧内の改行抑制、正規表現モード、コメント処理を持つ
- **ast**: 抽象構文木。`File` がルートで `imports`, `types`, `defaults`, `routes` を持つ。`PipelineStep` は `StepKind` で種別を区別する union 型
- **parser**: 再帰下降パーサー。`cur`/`peek` の 2 トークン先読み。エラーを蓄積して複数同時報告
- **ir**: JSON シリアライズ可能な IR 構造体。`omitempty` タグでゼロ値を省略
- **gen**: AST → IR 変換。キャスト (int, string 等) と関数呼び出し (trim, hash 等) を区別

CLI (`cmd/reverc/main.go`) は複数ファイルの入力を受け付け、`mergeIR()` でインポート・型・ルートをマージする。

## テスト構造

- **lexer/parser**: 入力文字列から直接 AST/トークンを検証するユニットテスト
- **gen**: `testdata/*.rever` を入力し `testdata/expected/*.json` と比較するゴールデンファイルテスト
- パーサーテストでは `parse()` / `parseWithErrors()` ヘルパーを使用

## DSL 仕様

詳細は `spec.md` を参照。主要な構文要素:
- `import name = package@version` — パッケージインポート（`@/path` でローカル）
- `type Name { field: type }` — 型定義
- `defaults` — 全ルート共通ディレクティブ（cors, auth）
- `METHOD /path` — ルート定義、`|>` でパイプライン、`~>` でエラーレスポンス
- ディレクティブ: `cache(...)`, `cors(...)`, `auth(...)`
- パイプラインステップ: `input`, `validate`, `transform`, `guard`, `match`, パッケージ呼び出し, `respond`
