package telegram

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gotd/td/tg"
)

// StdinCodeSource reads the Telegram login code from the terminal. This is
// for the standalone cmd/login tool only — real workers are headless and
// use NATSCodeSource instead.
type StdinCodeSource struct{}

func (StdinCodeSource) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	fmt.Print("Enter the login code Telegram just sent you: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
