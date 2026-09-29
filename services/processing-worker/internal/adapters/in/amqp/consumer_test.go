package amqp

import (
	"testing"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

func TestGetRetryCount(t *testing.T) {
	cases := []struct {
		name    string
		headers amqp091.Table
		want    int
	}{
		{"nil", nil, 0},
		{"sem header", amqp091.Table{}, 0},
		{"int32", amqp091.Table{retryHeader: int32(2)}, 2},
		{"int64", amqp091.Table{retryHeader: int64(2)}, 2},
		{"float64", amqp091.Table{retryHeader: float64(2)}, 2},
		{"int", amqp091.Table{retryHeader: 2}, 2},
		{"tipo desconhecido", amqp091.Table{retryHeader: "2"}, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := getRetryCount(tc.headers); got != tc.want {
				t.Fatalf("getRetryCount = %d, want %d", got, tc.want)
			}
		})
	}
}
