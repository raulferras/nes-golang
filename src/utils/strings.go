package utils

import (
	"encoding/hex"
	"regexp"
	"strings"
)

func StringSplitByRegex(content string) []string {
	re := regexp.MustCompile("[\\s]{2,}")

	return re.Split(content, -1)
}

func HexStringToByteArray(field string) []byte {
	result, err := hex.DecodeString(field)
	if err != nil {
		panic(err)
	}
	return result
}

func ByteToHex(value byte) string {
	s := strings.Repeat("0", 2)
	i := 2
	for i >= 0 {
		c := "0123456789ABCDEF"[value&0xF]
		s = s[:i] + string(c) + s[i+1:]
		i--
		value >>= 4
	}

	return s
}
