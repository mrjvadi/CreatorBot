package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func prompt(label string) string {
	fmt.Print(label)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(line)
}
