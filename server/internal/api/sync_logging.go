package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

const defaultSyncLogFilePath = "logs/sync.log"

func newSyncLogger(path string) (*log.Logger, io.Closer, error) {
	if path == "" {
		return log.New(io.Discard, "", log.LstdFlags), nil, nil
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, nil, fmt.Errorf("create sync log directory %s: %w", dir, err)
		}
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("open sync log file %s: %w", path, err)
	}

	return log.New(file, "", log.LstdFlags), file, nil
}

func compactSyncLogPayload(payload any) string {
	var raw []byte

	switch value := payload.(type) {
	case nil:
		return "null"
	case json.RawMessage:
		raw = bytes.TrimSpace(value)
	case []byte:
		raw = bytes.TrimSpace(value)
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return strconv.Quote(fmt.Sprintf("%+v", value))
		}
		raw = bytes.TrimSpace(encoded)
	}

	if len(raw) == 0 {
		return `""`
	}

	if json.Valid(raw) {
		var compact bytes.Buffer
		if err := json.Compact(&compact, raw); err == nil {
			raw = compact.Bytes()
		}
	}

	if len(raw) > maxTicketSyncBodyLogBytes {
		raw = append(raw[:maxTicketSyncBodyLogBytes], []byte("...(truncated)")...)
	}

	return string(raw)
}

func (s *Server) syncLogf(format string, args ...any) {
	if s == nil || s.syncLogger == nil {
		return
	}

	s.syncLogger.Printf(format, args...)
}

func (s *Server) logSyncEntityReceived(remoteAddr string, entityType string, payload any) {
	s.syncLogf(
		"sync received remote=%q entity_type=%q payload=%s",
		remoteAddr,
		entityType,
		compactSyncLogPayload(payload),
	)
}

func (s *Server) logSyncDecodeError(remoteAddr string, entityType string, payload any, contentType string, contentLength int64, err error) {
	s.syncLogf(
		"sync rejected: invalid request body remote=%q entity_type=%q content_type=%q content_length=%d decode_err=%q payload=%s",
		remoteAddr,
		entityType,
		contentType,
		contentLength,
		err.Error(),
		compactSyncLogPayload(payload),
	)
}

func (s *Server) logSyncProcessingError(remoteAddr string, entityType string, payload any, stage string, err error) {
	s.syncLogf(
		"sync failed remote=%q entity_type=%q stage=%q err=%q payload=%s",
		remoteAddr,
		entityType,
		stage,
		err.Error(),
		compactSyncLogPayload(payload),
	)
}

func (s *Server) logTicketSyncBadRequest(remoteAddr string, input syncTicketRequest, format string, args ...any) {
	s.syncLogf(
		"ticket sync rejected: "+format+" remote=%q entity_type=%q payload=%s",
		append(args, remoteAddr, "ticket", compactSyncLogPayload(input))...,
	)
}

func (s *Server) logDeviceSyncBadRequest(remoteAddr string, input syncDeviceRequest, format string, args ...any) {
	s.syncLogf(
		"device sync rejected: "+format+" remote=%q entity_type=%q payload=%s",
		append(args, remoteAddr, "device", compactSyncLogPayload(input))...,
	)
}

func (s *Server) logClassificatorSyncBadRequest(remoteAddr string, input syncClassificatorRequest, format string, args ...any) {
	s.syncLogf(
		"classificator sync rejected: "+format+" remote=%q entity_type=%q payload=%s",
		append(args, remoteAddr, "classificator", compactSyncLogPayload(input))...,
	)
}

func (s *Server) logContactSyncBadRequest(remoteAddr string, input syncContactRequest, format string, args ...any) {
	s.syncLogf(
		"contact sync rejected: "+format+" remote=%q entity_type=%q payload=%s",
		append(args, remoteAddr, "contact", compactSyncLogPayload(input))...,
	)
}
