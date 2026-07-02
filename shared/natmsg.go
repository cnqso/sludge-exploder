package shared

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Chrome's Native Messaging stdio framing caps a single message at 1MB from
// the host's side (https://developer.chrome.com/docs/apps/nativeMessaging).
const maxNativeMessageBytes = 1024 * 1024

// ReadNativeMessage reads one length-prefixed message from a Native
// Messaging stdio pipe: a 4-byte little-endian length followed by that many
// bytes of JSON.
func ReadNativeMessage(r io.Reader) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	n := binary.LittleEndian.Uint32(lenBuf[:])
	if n > maxNativeMessageBytes {
		return nil, fmt.Errorf("native message too large: %d bytes", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// WriteNativeMessage writes one length-prefixed message to a Native
// Messaging stdio pipe.
func WriteNativeMessage(w io.Writer, payload []byte) error {
	if len(payload) > maxNativeMessageBytes {
		return fmt.Errorf("native message too large: %d bytes", len(payload))
	}
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(payload)))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}
