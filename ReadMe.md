# llama-swap-tool

`llama-swap-tool` は、llama-swap に対する追加機能を行なうラップツールです。
以下の二つの機能で構成されています。

- ol-proxy
- llama-launcher  

## 基本機能

### ol-proxy (プロキシ)

`ol-proxy` は、Ollama 互換の API エンドポイントを提供し、アップストリームの OpenAI 互換サーバーへリクエストを転送するプロキシです。
ollama API を使うクライアントからのリクエストを ol-proxy で受け、llama-swap に投げることを想定しています。

- サポートする Ollama 互換エンドポイント:
  - `/api/chat`
  - `/api/generate`
  - `/api/tags`
  - `/api/embed`
- **ストリーミング対応**: アップストリームからのストリーミングレスポンスを適切に処理します。

#### 実行引数

- `-d`, `--debug`  
    デバッグログを有効にする (デフォルト:`false`)
- `-port`  
    リッスンするポート (デフォルト:`11434`)
- `-upstream`  
    アップストリームの OpenAI 互換サーバー URL(デフォルト:`http://localhost:8080`)

### llama-launcher (ランチャー)

`llama-launcher` は、`llama-swap.exe` と `ol-proxy.exe` をバックグラウンドで同時に起動し、システムトレイから操作できるようにするランチャーです。

- システムトレイ操作:
  - Open Web UI: llama-swap のプレイグラウンドをブラウザで開きます。
  - Open log file: `ol-proxy` のログファイルを開きます。
  - Restart: `llama-swap` と `ol-proxy` を再起動します。
  - Exit: プログラムを終了します（子プロセスも終了します）。

#### 引数について

llama-launcher 自身の動作を設定する引数はありません。与えられた実行引数は、全て llama-swap に引き渡します。  
ただし、ol-proxy の -upstream に与える URL を同時に起動する llama-swap に設定するため、以下の引数を確認します。

- `-listen`
- `-tls-cert-file`
- `-tls-key-file`

## 実行方法

- llama-swap をインストールしたフォルダに、ol-proxy.exe および、llama-launcher.exe をコピー
- llama-launcher を起動

※ llama-swap に与えたい引数があれば、llama-launcher の引数に指定する。

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
