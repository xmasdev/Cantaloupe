package engine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/xmasdev/Cantaloupe/engine/download"
	"github.com/xmasdev/Cantaloupe/engine/metainfo"
	"github.com/xmasdev/Cantaloupe/engine/peer"
	"github.com/xmasdev/Cantaloupe/engine/tracker"
	"github.com/xmasdev/Cantaloupe/engine/types"
)

type Engine struct {
	Metadata *types.TorrentMetadata
	InfoHash [20]byte
	PeerID   [20]byte

	Port int

	Download *download.TorrentDownload
	Storage  *download.Storage

	Listener net.Listener
	Peers    []*peer.PeerSession
}

func NewEngine(torrentPath string, outputDir string) (*Engine, error) {
	data, err := os.ReadFile(torrentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read torrent file: %w", err)
	}

	metadata, err := metainfo.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse torrent metadata: %w", err)
	}

	infoHash, err := metainfo.GetInfoHash(data)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate info hash: %w", err)
	}

	peerID, err := peer.GeneratePeerID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate peer ID: %w", err)
	}

	torrentDownload, err := download.NewTorrentDownload(&metadata.Info)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize download: %w", err)
	}

	storage, err := download.NewStorage(outputDir, &metadata.Info)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, fmt.Errorf("failed to start peer listener: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	return &Engine{
		Metadata: metadata,
		InfoHash: infoHash,
		PeerID:   peerID,
		Port:     port,
		Download: torrentDownload,
		Storage:  storage,
		Listener: listener,
		Peers:    make([]*peer.PeerSession, 0),
	}, nil
}

func (e *Engine) Close() error {
	if e == nil {
		return nil
	}

	var firstErr error

	for _, session := range e.Peers {
		if session == nil {
			continue
		}

		if err := session.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if e.Listener != nil {
		if err := e.Listener.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (e *Engine) Announce() (*types.AnnounceResponse, error) {
	return e.announceWithStats(0, 0)
}

func (e *Engine) ConnectToPeers(peers []*types.Peer) error {
	if e == nil {
		return errors.New("engine is nil")
	}

	var lastErr error

	for _, p := range peers {
		if p == nil {
			continue
		}

		address := net.JoinHostPort(
			p.IP.String(),
			strconv.Itoa(int(p.Port)),
		)

		session, err := peer.NewPeerSession(
			address,
			e.InfoHash,
			e.PeerID,
		)
		if err != nil {
			lastErr = fmt.Errorf(
				"failed to connect to peer %s: %w",
				address,
				err,
			)
			continue
		}

		e.Peers = append(e.Peers, session)
	}

	if len(e.Peers) == 0 {
		if lastErr != nil {
			return lastErr
		}

		return errors.New("failed to connect to any peers")
	}

	return nil
}

func (e *Engine) preparePeers() error {
	if len(e.Peers) == 0 {
		return errors.New("no peers available")
	}

	type result struct {
		session *peer.PeerSession
	}

	results := make(chan result, len(e.Peers))

	var wg sync.WaitGroup

	for _, session := range e.Peers {
		if session == nil {
			continue
		}

		wg.Add(1)

		go func(session *peer.PeerSession) {
			defer wg.Done()

			if err := session.SendInterested(); err != nil {
				_ = session.Close()
				return
			}

			if err := session.WaitForBitfield(); err != nil {
				_ = session.Close()
				return
			}

			if err := session.WaitForUnchoke(); err != nil {
				_ = session.Close()
				return
			}

			results <- result{session: session}
		}(session)
	}

	wg.Wait()
	close(results)

	ready := make([]*peer.PeerSession, 0, len(e.Peers))

	for result := range results {
		if result.session != nil {
			ready = append(ready, result.session)
		}
	}

	e.Peers = ready

	if len(e.Peers) == 0 {
		return errors.New("no peers became ready")
	}

	return nil
}

func (e *Engine) DownloadTorrent() error {
	if e == nil {
		return errors.New("engine is nil")
	}

	if e.Download == nil {
		return errors.New("engine has no download state")
	}

	if e.Storage == nil {
		return errors.New("engine has no storage")
	}

	if e.Download.NextMissingPiece() == nil {
		return nil
	}

	if len(e.Peers) == 0 {
		return errors.New("engine has no connected peers")
	}

	if err := e.preparePeers(); err != nil {
		return err
	}

	scheduler := newPieceScheduler(e.Download)

	var wg sync.WaitGroup

	errCh := make(chan error, len(e.Peers))

	for _, session := range e.Peers {
		if session == nil {
			continue
		}

		wg.Add(1)

		go func(session *peer.PeerSession) {
			defer wg.Done()

			for {
				piece := scheduler.Next(session)

				if piece == nil {
					return
				}

				err := download.DownloadPieceFor(
					e.Download,
					session,
					piece,
				)

				if err != nil {
					scheduler.Release(piece.Index)

					errCh <- fmt.Errorf(
						"peer failed downloading piece %d: %w",
						piece.Index,
						err,
					)

					return
				}

				if err := e.Storage.WritePiece(piece); err != nil {
					scheduler.Release(piece.Index)

					errCh <- fmt.Errorf(
						"failed to store piece %d: %w",
						piece.Index,
						err,
					)

					return
				}

				scheduler.Release(piece.Index)
			}
		}(session)
	}

	wg.Wait()
	close(errCh)

	if e.Download.NextMissingPiece() == nil {
		return nil
	}

	for err := range errCh {
		return err
	}

	return errors.New("download incomplete")
}

func (e *Engine) ReannounceLoop(
	ctx context.Context,
	uploaded int64,
	downloaded int64,
) {
	for {
		response, err := e.announceWithStats(
			uploaded,
			downloaded,
		)

		if err == nil && response != nil {
			timer := time.NewTimer(
				time.Duration(response.Interval) * time.Second,
			)

			select {
			case <-ctx.Done():
				timer.Stop()
				return

			case <-timer.C:
			}
		} else {
			select {
			case <-ctx.Done():
				return

			case <-time.After(time.Minute):
			}
		}
	}
}

func (e *Engine) announceWithStats(
	uploaded int64,
	downloaded int64,
) (*types.AnnounceResponse, error) {
	req := &types.AnnounceRequest{
		InfoHash:   e.InfoHash,
		PeerID:     e.PeerID,
		Port:       uint16(e.Port),
		Uploaded:   uploaded,
		Downloaded: downloaded,
		Left:       e.Download.TotalLength(),
	}

	return tracker.Announce(
		e.Metadata.Announce,
		*req,
	)
}

func (e *Engine) Start() error {
	if e == nil {
		return errors.New("engine is nil")
	}

	if e.Download == nil {
		return errors.New("engine has no download state")
	}

	if e.Storage == nil {
		return errors.New("engine has no storage")
	}

	// Torrent is already complete.
	if e.Download.NextMissingPiece() == nil {
		return nil
	}

	response, err := e.Announce()
	if err != nil {
		return fmt.Errorf("announce failed: %w", err)
	}

	if len(response.Peers) == 0 {
		return errors.New("tracker returned no peers")
	}

	if err := e.ConnectToPeers(response.Peers); err != nil {
		return fmt.Errorf("failed to connect to peers: %w", err)
	}

	if err := e.DownloadTorrent(); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	return nil
}

func (e *Engine) AcceptPeers(ctx context.Context) {
	for {
		conn, err := e.Listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}

			continue
		}

		go e.handleIncomingPeer(conn)
	}
}

func (e *Engine) handleIncomingPeer(conn net.Conn) {
	connection := peer.NewConnection(conn)

	if err := peer.Handshake(
		connection,
		e.InfoHash,
		e.PeerID,
	); err != nil {
		_ = conn.Close()
		return
	}

	session := peer.NewPeerSessionFromConnection(connection)

	// Add the session if your current peer API supports
	// constructing a session from an already-connected Connection.
	//
	// If it doesn't, leave incoming connections for the
	// next protocol pass rather than duplicating connection logic.
	_ = session
}
