package ledger

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

func recoverJournal(path string) (Recovery, error) {
	recovery := Recovery{
		Cases:       make(map[string]json.RawMessage),
		Idempotency: make(map[string]IdempotencyRecord),
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return recovery, nil
	}
	if err != nil {
		return Recovery{}, fmt.Errorf("打开事件账本: %w", err)
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	sequence := uint64(1)
	previous := ""
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			if line[len(line)-1] != '\n' {
				return Recovery{}, &IntegrityError{Sequence: sequence, Reason: "末尾事件未完整写入"}
			}
			var event Event
			if err := json.Unmarshal(line, &event); err != nil {
				return Recovery{}, &IntegrityError{Sequence: sequence, Reason: "事件 JSON 无效"}
			}
			if err := verifyEvent(event, sequence, previous); err != nil {
				return Recovery{}, err
			}
			recovery.Cases[event.CaseID] = append(json.RawMessage(nil), event.State...)
			if event.Idempotency != nil {
				recovery.Idempotency[idempotencyMapKey(event.Idempotency.Scope, event.Idempotency.Key)] = cloneIdempotency(*event.Idempotency)
			}
			recovery.LastSequence = event.Sequence
			recovery.LastDigest = event.Digest
			previous = event.Digest
			sequence++
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return Recovery{}, fmt.Errorf("读取事件账本: %w", readErr)
		}
	}
	return recovery, nil
}
