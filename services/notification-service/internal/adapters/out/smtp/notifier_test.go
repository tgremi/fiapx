package smtp

import (
	"context"
	"strings"
	"testing"

	"github.com/fiapx/notification-service/internal/domain"
)

func TestBuildMessage(t *testing.T) {
	msg := buildMessage("from@x.com", "to@y.com", "Assunto", "Corpo")

	for _, want := range []string{
		"From: from@x.com",
		"To: to@y.com",
		"Subject: Assunto",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"Corpo",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("mensagem não contém %q:\n%s", want, msg)
		}
	}
}

func TestSendCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	n := NewSMTPNotifier("localhost", "1025", "from@x.com")
	err := n.Send(ctx, domain.Notification{Email: "to@y.com", Subject: "s", Body: "b"})
	if err == nil {
		t.Fatal("esperado erro com contexto cancelado")
	}
}
