package main

import (
	"bufio"
	"bytes"
	"fmt"
	// "strconv"
	"strings"

	"albertzhong.com/go-json/json"
)

func main() {
	o := `{"siren_song": "hello there!!!", "bye":    [1, 2,    4, 5]}`
	reader := bufio.NewReader(strings.NewReader(o))

	_, err := json.UnmarshalValue(reader)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	k := [4]string{"h", "a", "l", "o"}
	if err = json.MarshalValue(k, writer); err != nil {
		panic(err)
	}
	writer.Flush()
	s := buf.String()
	fmt.Println(len(s))
	fmt.Println(buf.String())
}
