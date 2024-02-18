/**
 *
 */
package metrics

import "testing"

func TestUseGauge(t *testing.T) {
	UseGauge()
	t.Logf("Over")
}

func TestUseCounter(t *testing.T) {
	UseCounter()
	t.Logf("Over")
}

func TestUseMeter(t *testing.T) {
	UseMeter()
	t.Logf("Over")
}

func TestUseHistogram(t *testing.T) {
	UseHistogram()
	t.Logf("Over")
}

func TestUseTimer(t *testing.T) {
	UseTimer()
	t.Logf("Over")
}
