# Cantaloupe

Cantaloupe is a small BitTorrent client I am building with Go, Wails, and React. The desktop interface is intentionally simple, while the engine handles peer connections, trackers, piece verification, and storage.

## Features

- Open `.torrent` files from the desktop app
- Download single-file and multi-file torrents
- HTTP, HTTPS, and UDP tracker support
- Tracker tier fallback through `announce-list`
- SHA-1 piece verification before saving
- Pause, resume, and remove torrents
- Multiple torrents in the library
- Progress, verified pieces, peers, and download location details
- Custom Cantaloupe application logo

The project is still being developed. DHT peer discovery, magnet links, upload/seeding, and some detailed torrent views are not finished yet.

## Requirements

- Go 1.25 or newer
- Node.js and npm
- Wails v2
- GTK/WebKitGTK development packages on Linux

On Ubuntu or Debian:

```bash
sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev
```

## Run the app

```bash
wails dev
```

Use **Add torrent**, choose a `.torrent` file, choose a download folder, and press **Start**. The default folder is `~/Downloads`.

## Command-line downloader

The engine can also be run without the desktop UI:

```bash
cd engine
go run ./cmd/cantaloupe /path/to/file.torrent /path/to/output
```

## Development checks

```bash
go test ./...
go test -race ./...
go vet ./...
```

Build the frontend:

```bash
cd engine/frontend
npm install
npm run build
```

Build the desktop application:

```bash
wails build
```

## Project layout

```text
engine/              torrent engine, peers, trackers, and storage
engine/cmd/          command-line downloader
engine/frontend/     Wails React interface
build/               desktop build assets and application icon
```

## Notes

Torrent downloads depend on reachable peers. Trackers can return peers that are offline, choked, or no longer have the torrent, so it can take time for useful data to arrive.

Piece verification can be disabled in Settings, but leaving it enabled is recommended for normal downloads.

## Screenshots

![Cantaloupe screenshot 1](https://raw.githubusercontent.com/xmasdev/Cantaloupe/screenshots/screenshot1.png)

![Cantaloupe screenshot 2](https://raw.githubusercontent.com/xmasdev/Cantaloupe/screenshots/screenshot2.png)
