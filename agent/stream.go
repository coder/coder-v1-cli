package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"cdr.dev/slog"
	"github.com/hashicorp/yamux"
	"github.com/pion/webrtc/v3"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/internal/x/xwebrtc"
	"cdr.dev/coder-cli/pkg/proto"
)

type stream struct {
	stream *yamux.Stream
	logger slog.Logger

	rtc *webrtc.PeerConnection
}

// writes an error and closes.
func (s *stream) fatal(err error) {
	_ = s.write(proto.Message{
		Error: err.Error(),
	})
	s.logger.Error(context.Background(), err.Error(), slog.Error(err))
	_ = s.stream.Close()
}

func (s *stream) listen() {
	decoder := json.NewDecoder(s.stream)
	for {
		var msg proto.Message
		err := decoder.Decode(&msg)
		if err == io.EOF {
			break
		}
		if err != nil {
			s.fatal(err)
			return
		}
		s.processMessage(msg)
	}
}

func (s *stream) write(msg proto.Message) error {
	d, err := json.Marshal(&msg)
	if err != nil {
		return err
	}
	_, err = s.stream.Write(d)
	if err != nil {
		return err
	}
	return nil
}

func (s *stream) processMessage(msg proto.Message) {
	s.logger.Debug(context.Background(), "processing message", slog.F("msg", msg))

	if msg.Error != "" {
		s.fatal(xerrors.New(msg.Error))
		return
	}

	if msg.Candidate != "" {
		if s.rtc == nil {
			s.fatal(xerrors.New("rtc connection must be started before candidates are sent"))
			return
		}

		s.logger.Debug(context.Background(), "accepted ice candidate", slog.F("candidate", msg.Candidate))
		err := proto.AcceptICECandidate(s.rtc, &msg)
		if err != nil {
			s.fatal(err)
			return
		}
	}

	if msg.Offer != nil {
		rtc, err := xwebrtc.NewPeerConnection()
		if err != nil {
			s.fatal(fmt.Errorf("create connection: %w", err))
			return
		}
		flushCandidates := proto.ProxyICECandidates(rtc, s.stream)

		err = rtc.SetRemoteDescription(*msg.Offer)
		if err != nil {
			s.fatal(fmt.Errorf("set remote desc: %w", err))
			return
		}
		answer, err := rtc.CreateAnswer(nil)
		if err != nil {
			s.fatal(fmt.Errorf("create answer: %w", err))
			return
		}
		err = rtc.SetLocalDescription(answer)
		if err != nil {
			s.fatal(fmt.Errorf("set local desc: %w", err))
			return
		}
		flushCandidates()

		err = s.write(proto.Message{
			Answer: rtc.LocalDescription(),
		})
		if err != nil {
			s.fatal(fmt.Errorf("send local desc: %w", err))
			return
		}

		rtc.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
			s.logger.Info(context.Background(), "state changed", slog.F("new", pcs))
		})
		rtc.OnDataChannel(s.processDataChannel)
		s.rtc = rtc
	}
}

func (s *stream) processDataChannel(channel *webrtc.DataChannel) {
	if channel.Protocol() == "ping" {
		channel.OnOpen(func() {
			rw, err := channel.Detach()
			if err != nil {
				return
			}
			d := make([]byte, 64)
			_, err = rw.Read(d)
			if err != nil {
				s.logger.Error(context.Background(), "read ping", slog.Error(err))
				return
			}
			_, err = rw.Write(d)
			if err != nil {
				s.logger.Error(context.Background(), "write ping", slog.Error(err))
				return
			}
		})
		return
	}

	prto, port, err := xwebrtc.ParseProxyDataChannel(channel)
	if err != nil {
		s.fatal(fmt.Errorf("failed to parse proxy data channel: %w", err))
		return
	}
	if prto != "tcp" {
		s.fatal(fmt.Errorf("client provided unsupported protocol: %s", prto))
		return
	}

	conn, err := net.Dial(prto, fmt.Sprintf("localhost:%d", port))
	if err != nil {
		s.fatal(fmt.Errorf("failed to dial client port: %d", port))
		return
	}

	channel.OnOpen(func() {
		s.logger.Debug(context.Background(), "proxying data channel to local port", slog.F("port", port))
		rw, err := channel.Detach()
		if err != nil {
			_ = channel.Close()
			s.logger.Error(context.Background(), "detach client data channel", slog.Error(err))
			return
		}
		go func() {
			_, err = io.Copy(rw, conn)
			if err != nil {
				s.logger.Error(context.Background(), "copy to conn", slog.Error(err))
			}
		}()
		go func() {
			_, _ = io.Copy(conn, rw)
			if err != nil {
				s.logger.Error(context.Background(), "copy from conn", slog.Error(err))
			}
		}()
	})
}
