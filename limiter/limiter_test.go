package limiter

import "testing"

func TestNewLimiter(t *testing.T) {
    NewLimiterWait()
}

func TestNewLimiterAllow(t *testing.T) {
    NewLimiterAllow()
}