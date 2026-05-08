# llama-swap-tool

`llama-swap-tool` は、`llama-swap` を使う場合に、個人的に必要な機能を、まとめて追加するためのランチャーです。
以下の二つの機能で構成されています。

- ol-proxy
- llama-launcher  

## 基本機能

### ol-proxy (プロキシ)

`ol-proxy` は、Ollama 互換の API エンドポイントを提供し、アップストリームの OpenAI 互換サーバーへリクエストを転送するプロキシです。  
ollama API を使うクライアントからのリクエストを `ol-proxy` で受けOpenAI互換のリクエストに変更したあと `llama-swap` に投げることを想定しています。  
API キーについては対応していません。

- サポートする Ollama 互換エンドポイント:
  - `/api/chat`
  - `/api/generate`
  - `/api/tags`
  - `/api/show`
  - `/api/embed` (および `/api/embeddings`)
  - `/api/version`

  ※ ストリーム対応

#### 実行引数

- `-d`, `--debug`  
    デバッグログを有効にする (デフォルト:`false`)
- `-port`  
    リッスンするポート (デフォルト:`11434`)
- `-upstream`  
    アップストリームの OpenAI 互換サーバー URL(デフォルト:`http://localhost:8080`)

### llama-launcher (ランチャー)

`llama-launcher` は、`llama-swap.exe` と `ol-proxy.exe` をバックグラウンドで管理し、システムトレイから操作できるようにするランチャーです。
`llama-launcher.yaml` を使用して、各プロセスの起動可否、パス、引数の設定を行うことができます。

- システムトレイ操作:
  - Open Web UI: `llama-swap` のプレイグラウンドをブラウザで開きます。
  - Open log file: `ol-proxy` のログファイルを開きます。
  - Open config file: `llama-swap` のconfigファイルを開きます。
  - Restart: `llama-swap` と `ol-proxy` を再起動します。
  - Exit: プログラムを終了します（子プロセスも終了します）。

※ 設定ファイルで無効化されているプロセスのメニュー項目（Web UIやログ）は、グレーアウトされ選択できなくなります。

#### 設定ファイル (llama-launcher.yaml)

`llama-launcher.exe` と同じディレクトリに `llama-launcher.yaml` を配置することで、以下の設定が可能です。

```yaml
llama_swap:
  enabled: true          # 起動するかどうか
  path: "llama-swap.exe" # 実行ファイルのパス
  useConfigArgs: false   # true: 下記 args を使用 / false: ランチャーに渡された引数をそのまま使用
  args: []               # useConfigArgs が true の場合に使用される引数リスト

ol_proxy:
  enabled: true
  path: "ol-proxy.exe"
  useConfigArgs: true
  args: ["-d"]
```

#### 引数について

`llama-launcher` に渡された引数は、設定ファイルの `useConfigArgs` が `false` の場合に `llama-swap` に引き渡されます。
また、ランチャーは以下の引数を確認して動作を調整します。

- `ol-proxy` の `-upstream` に与えるURLを特定するため（`llama-swap` の待ち受け先を把握するため）:
  - `-listen`
  - `-tls-cert-file`
  - `-tls-key-file`
- システムトレイから開く `llama-swap` の config ファイルのパスを特定するため:
  - `-config`  

## 実行方法

- `llama-swap` をインストールしたフォルダに、`ol-proxy.exe`、`llama-launcher.exe`、および `llama-launcher.yaml` をコピー
- `llama-launcher.exe` を起動

※ `llama-swap` に与えたい引数は、`llama-launcher` の引数に指定するか、`llama-launcher.yaml` の `args` に記載します。

## ビルド方法

### 前提条件

- Go (Golang) のビルド環境が必要
- WSL や MSYS などの Linux 互換環境でのビルドを想定しています。

### 全体のビルド

プロジェクトのルートディレクトリで以下のコマンドを実行します。

```bash
make
```

ビルドされたバイナリは `build/` ディレクトリに集約されます。

### 個別のビルド

各コンポーネントを個別にビルドする場合は、それぞれのディレクトリに移動して実行します。

#### llama-launcher のビルド

```bash
cd llama-launcher
make llama-launcher.exe
```

#### ol-proxy のビルド

```bash
cd ol-proxy
make ol-proxy.exe
```
