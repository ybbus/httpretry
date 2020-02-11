package httpretry_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/ybbus/httpretry"
	"testing"
	"time"
)

func TestConstantBackoff(t *testing.T) {
	check := assert.New(t)

	t.Run("backoff should be constant over retries", func(t *testing.T) {
		waitDuration := 3 * time.Second

		backoff := httpretry.ConstantBackoff(waitDuration, 0)

		check.Equal(waitDuration, backoff(1))
		check.Equal(waitDuration, backoff(2))
		check.Equal(waitDuration, backoff(3))
	})

	t.Run("backoff should be 0 if negative", func(t *testing.T) {
		backoffNegativ := httpretry.ConstantBackoff(-1*time.Second, 0)

		check.Equal(0*time.Second, backoffNegativ(1))
		check.Equal(0*time.Second, backoffNegativ(2))
		check.Equal(0*time.Second, backoffNegativ(3))
	})

	t.Run("maxJitter should be in correct interval", func(t *testing.T) {
		constantTime := 10 * time.Second
		maxJitter := 10 * time.Second
		backoffJitter := httpretry.ConstantBackoff(constantTime, maxJitter)
		minWait := constantTime.Milliseconds()
		maxWait := minWait + maxJitter.Milliseconds()

		probe1 := backoffJitter(1).Milliseconds()
		probe2 := backoffJitter(2).Milliseconds()
		probe3 := backoffJitter(3).Milliseconds()

		check.GreaterOrEqual(probe1, minWait)
		check.NotEqual(minWait, probe1, "this may fail in very rare cases")
		check.Less(probe1, maxWait)

		check.GreaterOrEqual(probe2, minWait)
		check.NotEqual(minWait, probe1, "this may fail in very rare cases")
		check.Less(probe2, maxWait)

		check.GreaterOrEqual(probe3, minWait)
		check.NotEqual(minWait, probe1, "this may fail in very rare cases")
		check.Less(probe3, maxWait)
	})

	t.Run("maxJitter should be 0 if negative", func(t *testing.T) {
		backoffJitterNegativ := httpretry.ConstantBackoff(1*time.Second, -1*time.Second)

		check.Equal(1*time.Second, backoffJitterNegativ(1))
		check.Equal(1*time.Second, backoffJitterNegativ(2))
		check.Equal(1*time.Second, backoffJitterNegativ(3))
	})
}

func TestLinearBackoff(t *testing.T) {
	check := assert.New(t)

	t.Run("backoff should be linear over retries", func(t *testing.T) {
		minWait := 2 * time.Second

		backoff := httpretry.LinearBackoff(minWait, 0, 0)

		check.Equal(2*time.Second, backoff(1))
		check.Equal(4*time.Second, backoff(2))
		check.Equal(6*time.Second, backoff(3))
	})

	t.Run("backoff should stop at maxwait", func(t *testing.T) {
		minWait := 1 * time.Second
		maxWait := 3 * time.Second

		backoff := httpretry.LinearBackoff(minWait, maxWait, 0)

		check.Equal(1*time.Second, backoff(1))
		check.Equal(2*time.Second, backoff(2))
		check.Equal(3*time.Second, backoff(3))
		check.Equal(3*time.Second, backoff(4))
	})

	t.Run("maxWait should be 0 if < minWait", func(t *testing.T) {
		minWait := 2 * time.Second
		maxWait := 1 * time.Second

		backoff := httpretry.LinearBackoff(minWait, maxWait, 0)

		check.Equal(2*time.Second, backoff(1))
		check.Equal(4*time.Second, backoff(2))
		check.Equal(6*time.Second, backoff(3))
		check.Equal(8*time.Second, backoff(4))
	})

	t.Run("backoff should be 0 if negative", func(t *testing.T) {
		backoffNegativ := httpretry.LinearBackoff(-1*time.Second, 0, 0)

		check.Equal(0*time.Second, backoffNegativ(1))
		check.Equal(0*time.Second, backoffNegativ(2))
		check.Equal(0*time.Second, backoffNegativ(3))
	})

	t.Run("maxJitter should be in correct interval", func(t *testing.T) {
		initialWait := 10 * time.Second
		maxJitter := 10 * time.Second
		backoffNoJitter := httpretry.LinearBackoff(initialWait, 0, 0)
		backoffJitter := httpretry.LinearBackoff(initialWait, 0, maxJitter)

		probe1NoJitter := backoffNoJitter(1)
		probe1Jitter := backoffJitter(1)
		probe2NoJitter := backoffNoJitter(2)
		probe2Jitter := backoffJitter(2)
		probe3NoJitter := backoffNoJitter(3)
		probe3Jitter := backoffJitter(3)

		check.GreaterOrEqual(probe1Jitter.Milliseconds(), probe1NoJitter.Milliseconds())
		check.GreaterOrEqual(probe2Jitter.Milliseconds(), probe2NoJitter.Milliseconds())
		check.GreaterOrEqual(probe3Jitter.Milliseconds(), probe3NoJitter.Milliseconds())

		check.Less(probe1Jitter.Milliseconds(), (probe1NoJitter + maxJitter).Milliseconds())
		check.Less(probe1Jitter.Milliseconds(), (probe1NoJitter + maxJitter).Milliseconds())
		check.Less(probe1Jitter.Milliseconds(), (probe1NoJitter + maxJitter).Milliseconds())

		check.NotEqual(probe1Jitter.Milliseconds(), probe1NoJitter.Milliseconds(), "this may fail in very rare cases")
		check.NotEqual(probe2Jitter.Milliseconds(), probe2NoJitter.Milliseconds(), "this may fail in very rare cases")
		check.NotEqual(probe3Jitter.Milliseconds(), probe3NoJitter.Milliseconds(), "this may fail in very rare cases")
	})

	t.Run("maxJitter should be 0 if negative", func(t *testing.T) {
		backoffJitterNegativ := httpretry.LinearBackoff(2*time.Second, 0, -1*time.Second)

		check.Equal(2*time.Second, backoffJitterNegativ(1))
		check.Equal(4*time.Second, backoffJitterNegativ(2))
		check.Equal(6*time.Second, backoffJitterNegativ(3))
	})
}

func TestExponentialBackoff(t *testing.T) {
	check := assert.New(t)

	t.Run("backoff should be exponential over retries", func(t *testing.T) {
		minWait := 1 * time.Second

		backoff := httpretry.ExponentialBackoff(minWait, 0, 0)

		check.Equal(1*time.Second, backoff(1))
		check.Equal(2*time.Second, backoff(2))
		check.Equal(4*time.Second, backoff(3))
		check.Equal(8*time.Second, backoff(4))
	})

	t.Run("backoff should stop at maxWait", func(t *testing.T) {
		minWait := 1 * time.Second
		maxWait := 3 * time.Second

		backoff := httpretry.ExponentialBackoff(minWait, maxWait, 0)

		check.Equal(1*time.Second, backoff(1))
		check.Equal(2*time.Second, backoff(2))
		check.Equal(3*time.Second, backoff(3))
		check.Equal(3*time.Second, backoff(4))
	})

	t.Run("maxWait should be 0 if < minWait", func(t *testing.T) {
		minWait := 2 * time.Second
		maxWait := 1 * time.Second

		backoff := httpretry.ExponentialBackoff(minWait, maxWait, 0)

		check.Equal(2*time.Second, backoff(1))
		check.Equal(4*time.Second, backoff(2))
		check.Equal(8*time.Second, backoff(3))
		check.Equal(16*time.Second, backoff(4))
	})

	t.Run("backoff should be 0 if negative", func(t *testing.T) {
		backoffNegativ := httpretry.ExponentialBackoff(-1*time.Second, 0, 0)

		check.Equal(0*time.Second, backoffNegativ(0))
		check.Equal(0*time.Second, backoffNegativ(1))
		check.Equal(0*time.Second, backoffNegativ(2))
	})

	t.Run("maxJitter should be in correct interval", func(t *testing.T) {
		initialWait := 10 * time.Second
		maxJitter := 10 * time.Second
		backoffNoJitter := httpretry.ExponentialBackoff(initialWait, 0, 0)
		backoffJitter := httpretry.ExponentialBackoff(initialWait, 0, maxJitter)

		probe1NoJitter := backoffNoJitter(1)
		probe1Jitter := backoffJitter(1)
		probe2NoJitter := backoffNoJitter(2)
		probe2Jitter := backoffJitter(2)
		probe3NoJitter := backoffNoJitter(3)
		probe3Jitter := backoffJitter(3)

		check.GreaterOrEqual(probe1Jitter.Milliseconds(), probe1NoJitter.Milliseconds())
		check.GreaterOrEqual(probe2Jitter.Milliseconds(), probe2NoJitter.Milliseconds())
		check.GreaterOrEqual(probe3Jitter.Milliseconds(), probe3NoJitter.Milliseconds())

		check.Less(probe1Jitter.Milliseconds(), (probe1NoJitter + maxJitter).Milliseconds())
		check.Less(probe1Jitter.Milliseconds(), (probe1NoJitter + maxJitter).Milliseconds())
		check.Less(probe1Jitter.Milliseconds(), (probe1NoJitter + maxJitter).Milliseconds())

		check.NotEqual(probe1Jitter.Milliseconds(), probe1NoJitter.Milliseconds(), "this may fail in very rare cases")
		check.NotEqual(probe2Jitter.Milliseconds(), probe2NoJitter.Milliseconds(), "this may fail in very rare cases")
		check.NotEqual(probe3Jitter.Milliseconds(), probe3NoJitter.Milliseconds(), "this may fail in very rare cases")
	})

	t.Run("maxJitter should be 0 if negative", func(t *testing.T) {
		backoffJitterNegativ := httpretry.ExponentialBackoff(1*time.Second, 0, -1*time.Second)

		check.Equal(1*time.Second, backoffJitterNegativ(1))
		check.Equal(2*time.Second, backoffJitterNegativ(2))
		check.Equal(4*time.Second, backoffJitterNegativ(3))
		check.Equal(8*time.Second, backoffJitterNegativ(4))
	})
}
