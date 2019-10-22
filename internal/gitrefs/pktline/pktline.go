// Package pktline implements git pktline format
// https://github.com/git/git/blob/master/Documentation/technical/pack-protocol.txt
// Encoded as hexlen + string where len is 16 bit hex encoded len(string) + len(hexlen)
// Ex: "a" is "0005a"
// Ex: "" is "0000" (special case)
package pktline

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
)

// Read a pktline
func Read(r io.Reader) (string, error) {
	var err error

	var lenHexBuf [4]byte
	_, err = io.ReadFull(r, lenHexBuf[:])
	if err != nil {
		return "", err
	}
	var lenBuf [2]byte
	_, err = hex.Decode(lenBuf[:], lenHexBuf[:])
	if err != nil {
		return "", err
	}
	len := binary.BigEndian.Uint16(lenBuf[:])
	if len == 0 {
		return "", nil
	}
	if len < 4 {
		return "", fmt.Errorf("short len %d", len)
	}
	lineBuf := make([]byte, len-4)
	_, err = io.ReadFull(r, lineBuf[:])
	if err != nil {
		return "", err
	}

	return string(lineBuf), nil
}

// Write a pktline
func Write(w io.Writer, s string) (int, error) {
	return w.Write(Encode(s))
}

// Encode a pktline
func Encode(s string) []byte {
	if len(s) == 0 {
		return []byte("0000")
	}

	return []byte(fmt.Sprintf("%04x%s", uint16(len(s)+4), s))
}
