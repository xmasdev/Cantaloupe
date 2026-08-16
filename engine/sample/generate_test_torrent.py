from pathlib import Path
import hashlib

content = b"Hello from Cantaloupe!\n"
piece_hash = hashlib.sha1(content).digest()

def bstr(x: bytes) -> bytes:
    return str(len(x)).encode() + b":" + x

def bint(x: int) -> bytes:
    return b"i" + str(x).encode() + b"e"

def blist(items: list[bytes]) -> bytes:
    return b"l" + b"".join(items) + b"e"

def bdict(items: list[tuple[bytes, bytes]]) -> bytes:
    # Canonical bencode dictionary ordering.
    return b"d" + b"".join(
        bstr(k) + v for k, v in sorted(items, key=lambda x: x[0])
    ) + b"e"

info = bdict([
    (b"length", bint(len(content))),
    (b"name", bstr(b"hello.txt")),
    (b"piece length", bint(16384)),
    (b"pieces", bstr(piece_hash)),
])

announce_list = blist([
    blist([bstr(b"https://tracker1.example.com/announce")]),
    blist([
        bstr(b"https://tracker2.example.com/announce"),
        bstr(b"https://tracker3.example.com/announce"),
    ]),
])

torrent = bdict([
    (b"announce", bstr(b"https://tracker.example.com/announce")),
    (b"announce-list", announce_list),
    (b"comment", bstr(b"Test torrent for Cantaloupe")),
    (b"created by", bstr(b"Cantaloupe")),
    (b"info", info),
])

path = Path("./engine/sample/test.torrent")
path.write_bytes(torrent)

print("Created:", path)
print("Size:", len(torrent), "bytes")
print("Info hash:", hashlib.sha1(info).hexdigest())
print("Piece hash:", piece_hash.hex())
