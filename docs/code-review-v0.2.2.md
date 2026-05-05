# pe — コードレビュー

**リポジトリ:** github.com/Shin-R2un/pe  
**バージョン:** 0.2.2-dev (7コミット)  
**言語:** Go 1.18+ (CIは1.22)  
**総LOC:** 2538行 (本番1537 / テスト1001)  
**テスト:** 全45テスト PASS (0 fail)  
**ライセンス:** MIT

---

## 概要

「ぺっと貼る」をコンセプトにした、CLI向けスニペットランチャー。
キーを登録してワンコマンドでクリップボードにコピーするだけのツール。
テキスト展開ツールやクリップボード履歴とは明確に異なるポジション。

- 単一バイナリ (Go)
- `~/.pe/pe.json` にJSONで永続化 (0600, atomic write)
- クリップボード: pbcopy/wl-copy/xclip/xsel/clip → OSC 52 フォールバック
- インストール: go install / Scoop (Windows) / pre-built binaries / goreleaser

---

## アーキテクチャ評価

### 良い点

1. **パッケージ分割が適切** — `cli`, `store`, `clip`, `editor` の4パッケージに責務が明確に分かれている。`cmd/pe/main.go` はエントリポイントのみ (39行)

2. **DI可能なApp構造体** — `App` が `Path`, `Out`, `Err`, `Copy`, `Now` を持つことで、テスト時にモック注入が容易。テストコードが `t.TempDir()` + fake clipboard で完結しているのは美しい

3. **atomic write** — `store.Save()` が `.tmp` 経由の `rename(2)` でクラッシュセーフ

4. **OSC 52対応** — SSH先/headless環境でもクリップボードにコピー可能。tmux内ではDCS passthroughでESC doubling。実装が正確

5. **"did you mean"サジェスト** — prefix → substring → Levenshtein距離の3段階。閾値 `max(2, len(query)/3)` が適切

6. **予約語ガード** — サブコマンド名をkeyとして登録できない。`reserved` map + `validateKey()` で一貫性を保証

7. **シェル補完** — bash/zsh/fish 3シェル対応。`pe __complete` 経由で動的キー補完

8. **自己更新** — `pe update` で `go install` をラップ。goreleaser + Scoop パイプラインも完備

9. **テスト品質** — 45テスト、1001行。エッジケース (WSL検出、tmux OSC 52、broken JSON、重複key、予約語) を網羅

### 改善提案

#### Critical (セキュリティ/正確性)

- **`editor.go` にテストがない** — `internal/editor` パッケージは唯一テストがない。外部エディタを呼ぶためハードだが、`pick()` の環境変数解決ロジックは純関数なのでテスト可能

- **`edit.go` の `stripJSONComments` が先頭コメントのみ削除** — JSON内の `//` コメントを正しく扱わない。ユーザーがJSON内部に `//` を含む値を書くと壊れる可能性がある。ドキュメントに「valueに `//` を含めないでください」と明記するか、JSON5パーサーを使うか

#### High (UX/信頼性)

- **インタラクティブモードで非ASCII入力が扱えない** — `interactive.go` の `b >= 0x20 && b < 0x7f` で日本語キー入力が弾かれる。UTF-8多バイトシーケンスを読み飛ばしてしまう。日本語keyをサポートするなら `b >= 0x20` で受け入れ、UTF-8デコーダを通す必要がある

- **検索が部分一致のみ** — スペース区切りAND検索や正規表現がない。スニペットが増えると絞り込みが困難になる。Roadmapに `pe tag <tag>` があるが、これも実装待ち

- **`pe a` でdescription/tagsを登録できない** — CLI引数だけではvalueのみ。`--description` / `--tags` フラグか `pe a --editor` が必要

#### Medium (保守性)

- **`go.mod` が Go 1.18 を要求しつつCIが1.22** — 最低バージョンを1.18のままにしている意図が不明。`golang.org/x/term` v0.18.0 は Go 1.18+ で動くが、CIを1.22に揃えるなら `go.mod` も上げてよい

- **`store.File.Get()` がO(n)線形探索** — スニペット数が少ないうちは問題ないが、数百件を超えると `map[string]int` でインデックスを張る方が良い

- **`cmdList` / `cmdSearch` で `width` 計算がバイト単位** — 日本語keyの場合 `len(k)` はバイト数なので表示幅がずれる。`utf8.RuneCountInString(k)` を使うべき

- **`preview()` の `truncate()` はruneベース** — こちらは正しい。一貫性がないのが少し気になる

#### Low (スタイル/ドキュメント)

- **READMEのRoadmapに「shell completions」があるが既に実装済み** — 削除または「✅ 実装済み」に更新すべき

- **`helpText` がハードコード文字列** — `commands.go` のテーブルから生成するか、少なくとも `helpText` と実際の `switch` case の同期をテストで担保したい

- **`go.mod` に `indirect` コメントがない** — `golang.org/x/sys` が indirect 依存だがコメントがない。`go mod tidy` が自動管理する範囲

---

## ファイル別詳細

| ファイル | 行数 | 役割 | 評価 |
|---------|------|------|------|
| `cmd/pe/main.go` | 39 | エントリポイント、バージョン解決 | 素晴らしい。`resolveVersion` で go install / goreleaser 両方に対応 |
| `internal/cli/cli.go` | 162 | ルーター、App構造体、ヘルプ | 責務が明確。`errorf` / `usage` のexit code分離 (1 vs 2) が良い |
| `internal/cli/commands.go` | 232 | add/copy/list/search/show/delete | 実装がシンプルで正確。`notFound` → suggest の流れがUX上級 |
| `internal/cli/edit.go` | 118 | JSONエディタ編集 | editFormの設計が良い。メタデータ保存が透過的 |
| `internal/cli/interactive.go` | 138 | raw mode TUI フィルタ | 最小限の実装で十分。非ASCIIの壁がある |
| `internal/cli/suggest.go` | 103 | "did you mean"エンジン | 3段階マッチング + Levenshtein。閾値設計が適切 |
| `internal/cli/completion.go` | 147 | bash/zsh/fish補完スクリプト | 3シェル網羅。動的 `__complete` が優秀 |
| `internal/cli/reserved.go` | 23 | 予約語マップ | シンプルで正確 |
| `internal/cli/update.go` | 110 | 自己更新 (`pe update`) | go install ラッパー。エラー時のGOPROXY=direct ヒントが親切 |
| `internal/clip/clip.go` | 157 | クリップボード抽象 | OS検出→OSC 52のチェーンが正確。tmux対応も完璧 |
| `internal/editor/editor.go` | 68 | 外部エディタ起動 | `pick()` の環境変数優先度が明確 |
| `internal/store/store.go` | 240 | JSON永続化 | atomic write + 0600。`Search` が多フィールド検索 |
| `.goreleaser.yml` | 80 | リリースパイプライン | Scoop連携含めて成熟 |
| `.github/workflows/ci.yml` | 81 | CI/CD | matrix 3OS + cross-build。concurrency設定あり |
| `.github/workflows/release.yml` | 38 | リリース | goreleaser v6 + workflow_dispatch |

---

## 総評

**品質: A-** (個人プロジェクト・v0.2段階として非常に高い水準)

GoのCLIツールとして教科書的な品質。パッケージ分割、DI、atomic write、OSC 52対応、テストカバレッジ、CI/CD、Scoop配布まで揃っている。7コミットでここまで纏まっているのは印象的。

最大の改善点は **日本語対応** (非ASCII key入力 + 表示幅) と **editor テスト**。次のマイルストーンで `pe a --editor` / `pe tag` / `pe export/import` を実装すれば実用度がさらに上がる。
