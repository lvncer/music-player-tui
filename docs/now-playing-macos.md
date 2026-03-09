# macOS で Now Playing（再生中情報）を取る方法

このアプリでは、macOS 15.4 以降でも再生中の曲情報を表示するために **mediaremote-adapter** を Perl 経由で使っています。

## なぜ Perl 経由なのか

### 背景（取れなくなった理由）

- **macOS 15.4 以降**、Apple が MediaRemote の「Now Playing を読む」API に **entitlement チェック** を入れた。
- 通常のアプリ（この TUI や Electron アプリなど）が直接 `MRMediaRemoteGetNowPlayingInfo` や `MRNowPlayingRequest` を叩くと、**権限がなくて nil が返る**。
- **再生制御**（再生/一時停止/次/前）は今も使えるので、止めたり飛ばしたりは動く。**取得だけ** が制限されている。

### 解決策: platform binary 経由

- **platform binary**（システムが特権付きで持っている実行ファイル）のプロセスだと、MediaRemote の「読む」API が通る。
- 例: `/usr/bin/perl` は platform binary。
- **mediaremote-adapter** は次のように動かす:
  1. **Perl** を起動する（`/usr/bin/perl`）。
  2. Perl が **DynaLoader** で `MediaRemoteAdapter.framework` をロードする。
  3. その Framework が MediaRemote を叩く → **Perl プロセスの権限** で読めるので、Now Playing が取れる。
  4. 結果を **JSON で stdout に出す**。この TUI はそれをパースして表示する。

つまり「**自分のプロセスでは読めないので、読める権限を持つ Perl にやってもらう**」形です。

## このリポジトリでの構成

```sh
music-player-tui/
├── music-player-tui          # ビルドした実行ファイル（ここから起動する）
├── mediaremote-adapter/      # 実行ファイルの「隣」に必要
│   ├── mediaremote-adapter.pl
│   └── MediaRemoteAdapter.framework/
├── adapter.go                # Perl を spawn して get/stream/send を呼ぶ
├── main.go
└── ...
```

- **起動時**: 実行ファイルの隣に `mediaremote-adapter` があるかを見て、あれば **adapter 経由**、なければ **C の MediaRemote 直接**（古い macOS 用）にフォールバックする。
- **曲情報の取得**: `perl mediaremote-adapter.pl <FrameworkPath> get --now` を実行し、stdout の JSON をパース。
- **経過時間の更新**: 1 秒ごとの tick で `get --now` を再度実行し、`elapsedTimeNow` で現在秒数を更新。
- **再生制御**: `perl mediaremote-adapter.pl <FrameworkPath> send <commandId>`（2=play/pause, 4=next, 5=prev）。
- **曲が変わったときの通知**: `perl ... stream --debounce=300` でストリームし、行ごとの JSON で変更を検知。

## mediaremote-adapter の由来

- 本体: [ungive/mediaremote-adapter](https://github.com/ungive/mediaremote-adapter)
- このプロジェクトでは、[lvncer/home](https://github.com/lvncer/home) の Electron 用に vendored していた `mediaremote-adapter` をコピーして利用している（同じ仕組みで macOS 26 でも動く）。

## サムネイル（アートワーク）の表示

- adapter の `get` は `artworkData`（base64）と `artworkMimeType` を返す場合がある。
- **Kitty**・**WezTerm**・**Warp** など、Kitty graphics protocol に対応したターミナルでは、再生中曲のアートワークをインライン画像として表示する（`TERM` に `kitty` / `wezterm` / `warp` が含まれるときのみ有効）。
- サムネがない、または表示できない場合は「Artwork: -」と表示する。
- プロトコルは PNG を前提（`f=100`）。JPEG も試すが、ターミナルによっては表示されないことがある。画像が表示されない場合は `MUSIC_PLAYER_TUI_ARTWORK=1` で強制的に画像出力を試せる（`TERM` 判定をスキップ）。

## 参考

- [LyricFever #94 — MRMediaRemoteGetNowPlayingInfo return nil in latest MacOS](https://github.com/aviwad/LyricFever/issues/94)（entitlement と adapter の説明）
- [mediaremote-adapter README](https://github.com/ungive/mediaremote-adapter#readme)
- [Kitty graphics protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/)
