# ThisVidDownloader
A bunny-style ThisVid downloader built on [go-thisvid](https://github.com/RoyalFlyBy/go-thisvid).

It resolves a video page through the player's license wrapper, asks the CDN for the file size with a one-byte range, then downloads the stream with ranged, resumable threads.

## Install
```sh
go install github.com/RoyalFlyBy/ThisVidDownloader/_bin@latest
```
`go-thisvid` and `go-browser-util` are private modules, so set `GOPRIVATE=github.com/RoyalFlyBy/*` and have git credentials before building.

Or build a binary:
```sh
go build -o thisvid_downloader ./_bin
# cross-compile everything into _bin/:
_resources/scripts/build.sh
```

## Usage

Download a single video (page URL, `/videos/<slug>/` path or bare slug):
```sh
thisvid_downloader --video="https://thisvid.com/videos/gtj-056/"
```

Download one video from a playlist:
```sh
thisvid_downloader --video="https://thisvid.com/playlist/577936/video/three-japanese-girls-bust-his-balls/"
```

Download every video in a text file (one URL/slug per line, `#` comments allowed):
```sh
thisvid_downloader --list="./urls.txt"
```

Download the 10 newest public videos of a member (URL or numeric id):
```sh
thisvid_downloader --member="https://thisvid.com/members/5670837/" --amount=10
```

Download 25 videos from a playlist, skipping the first 5:
```sh
thisvid_downloader --playlist="https://thisvid.com/playlist/577936/video/three-japanese-girls-bust-his-balls/" --amount=25 --offset=5
```

Files are named `<uploader> - <title><ext>`. A file that already has the expected size is skipped; partial downloads resume from their `.partN` files (or from the target itself with `--threads=1`).

### Flags
| Flag | Default | Meaning |
|---|---|---|
| `--video` | | video page URL, path or slug |
| `--list` | | text file with one video URL/slug per line |
| `--member` | | member profile URL or numeric id (public videos, newest first) |
| `--playlist` | | playlist URL or numeric id |
| `--output` | `.` | download directory |
| `--threads` | `4` | simultaneous ranged threads (1..12) |
| `--quality` | `-1` | stream index in the player's list; `-1` picks the best (UHD > HD > first) |
| `--amount` | `20` | videos to download from `--member` / `--playlist` |
| `--offset` | `0` | videos to skip in `--member` / `--playlist` |
| `--pid` | `false` | prepend `[<id>] ` to filenames |
| `--cookies` | | raw Cookie header for friends-only videos |
| `--proxy` | | proxy URL, e.g. `http://127.0.0.1:8888` or `socks5://127.0.0.1:9050` |
| `--debug` | `false` | debug logs in `thisvid_downloader.log` |
| `--version` | `false` | print version information |

The first target flag wins, in the order listed above.

### Private (friends-only) videos
A private page has metadata but no player unless the session is allowed to watch it. Log in to the site, copy the request's Cookie header, and pass it:
```sh
thisvid_downloader --video="..." --cookies="PHPSESSID=...; kt_tcookie=1"
```
A private item in a `--member` or `--playlist` run is reported and skipped.

## Features
* Single videos, list files, member public-video pages and playlists.
* Paged collections with `--amount` / `--offset`.
* Ranged, multi-threaded, resumable downloads with a live progress display.
* Best-quality stream by default, with `--quality` to pick another.
* Browser headers from [go-browser-util](https://github.com/RoyalFlyBy/go-browser-util).
* Proxy support and a version/update hint.

## Known bugs
* Stream URLs are signed and expire; a very long paused download may need to be restarted to get fresh URLs (the partial `.partN` files resume).
* A private video with no stream is skipped in collections, so `--amount=N` can yield fewer files.
