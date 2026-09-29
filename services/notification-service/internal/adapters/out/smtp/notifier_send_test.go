package smtp

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fiapx/notification-service/internal/domain"
)

func startFakeSMTP(t *testing.T) (port int, received chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	received = make(chan string, 1)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		r := bufio.NewReader(conn)
		_, _ = fmt.Fprint(conn, "220 fake ESMTP\r\n")

		var data strings.Builder
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				_, _ = fmt.Fprint(conn, "250 fake\r\n")
			case strings.HasPrefix(cmd, "MAIL FROM"), strings.HasPrefix(cmd, "RCPT TO"):
				_, _ = fmt.Fprint(conn, "250 ok\r\n")
			case strings.HasPrefix(cmd, "DATA"):
				_, _ = fmt.Fprint(conn, "354 end with .\r\n")
				for {
					l, err := r.ReadString('\n')
					if err != nil {
						return
					}
					if strings.TrimSpace(l) == "." {
						break
					}
					data.WriteString(l)
				}
				_, _ = fmt.Fprint(conn, "250 queued\r\n")
			case strings.HasPrefix(cmd, "QUIT"):
				_, _ = fmt.Fprint(conn, "221 bye\r\n")
				received <- data.String()
				return
			}
		}
	}()

	return ln.Addr().(*net.TCPAddr).Port, received
}

func TestSendSuccess(t *testing.T) {
	port, received := startFakeSMTP(t)

	n := NewSMTPNotifier("127.0.0.1", strconv.Itoa(port), "from@x.com")
	err := n.Send(context.Background(), domain.Notification{
		Email: "to@y.com", Subject: "Assunto", Body: "Corpo",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	select {
	case body := <-received:
		if !strings.Contains(body, "Subject: Assunto") || !strings.Contains(body, "Corpo") {
			t.Fatalf("corpo inesperado:\n%s", body)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout esperando entrega SMTP")
	}
}
