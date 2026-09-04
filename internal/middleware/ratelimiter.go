package middleware

import (
	"log"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Tokenbucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

func NewTokenBucket(rate float64, burst int) *Tokenbucket {
	return &Tokenbucket{
		tokens:     float64(burst),
		maxTokens:  float64(burst),
		refillRate: rate,
		lastRefill: time.Now(),
	}

}

func (tb *Tokenbucket) Allow() (bool, int, time.Duration) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate

	if tb.tokens > tb.maxTokens{
		tb.tokens=tb.maxTokens
	}
	tb.lastRefill=now
	if tb.tokens >= 1.0 {
		tb.tokens--
		return true,int(tb.tokens),0
	}
	deficit := 1.0 - tb.tokens
	waitSeconds := deficit / tb.refillRate
	retryAfter := time.Duration(waitSeconds * float64(time.Second))
	return false, 0, retryAfter

}

type ipEntry struct{
	bucket *Tokenbucket
	lastseen time.Time

}

type IPRateLimiter struct{
	buckets map[string]*ipEntry
	rate float64
	burst int
	mu sync.RWMutex
}

func NewIPRateLimiter(rate float64,burst int) *IPRateLimiter{
	limiter:= &IPRateLimiter{
		buckets : make(map[string]*ipEntry),
		rate: rate,
		burst: burst,
	}

	go limiter.cleanup()
	return limiter
}

func (l *IPRateLimiter) getBucket(ip string) *Tokenbucket {
	l.mu.RLock()
	entry, exists := l.buckets[ip]
	l.mu.RUnlock()

	if exists {
		l.mu.Lock()
		entry.lastseen = time.Now()
		l.mu.Unlock()
		return entry.bucket
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if entry, exists := l.buckets[ip]; exists {
		entry.lastseen = time.Now()
		return entry.bucket
	}

	bucket := NewTokenBucket(l.rate,l.burst)
	l.buckets[ip]=&ipEntry{
		bucket: bucket,
		lastseen: time.Now(),
	}
	return bucket	
}

func (l *IPRateLimiter) cleanup() {
	ticker := time.NewTicker(5*time.Minute)
	defer ticker.Stop()

	for range ticker.C{
		l.mu.Lock()
		before:=len(l.buckets)
		for ip, entry := range l.buckets {
			if time.Since(entry.lastseen) > 10*time.Minute{
				delete(l.buckets,ip)
			}
		}
		after := len(l.buckets)
		l.mu.Unlock()

		if before != after{
			log.Printf("[RATELIMIT] Cleanup: removed %d stale IP(s), %d active", before-after, after)
		}		
	}
}


func (l *IPRateLimiter) CheckHTTP(w http.ResponseWriter, r *http.Request) bool {
	ip := ExtractClientIP(r)
	bucket := l.getBucket(ip)
	allowed, remaining, retryAfter := bucket.Allow()
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(l.burst))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	resetTime := time.Now().Add(time.Duration(float64(time.Second) / l.rate)).Unix()
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))
	if !allowed {
		retrySeconds := int(math.Ceil(retryAfter.Seconds()))
		if retrySeconds < 1 {
			retrySeconds = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
		log.Printf("[RATELIMIT] %s rate limited (retry after %ds)", ip, retrySeconds)
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return false
	}
	return true
}
func ExtractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

