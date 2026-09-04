package router

import (
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/karnesanthosh/Averon/internal/balancer"
	"github.com/karnesanthosh/Averon/internal/server"
	"github.com/karnesanthosh/Averon/internal/middleware"
)

// Route represents a path-to-backend mapping.
// Each route has its own load balancer instance that distributes
// traffic only among the backends assigned to this route.
type Route struct {
	PathPrefix  string             // e.g., "/api/auth"
	StripPrefix bool               // if true, remove PathPrefix before forwarding
	Backends    []*server.Server   // backend servers for this route
	LB          *balancer.LoadBalancer  // per-route load balancer
	RateLimitRate float64
	RateLimitBurst int
	limiter *middleware.IPRateLimiter
	
}

// Router matches incoming requests to routes by path prefix
// and delegates to the route's load balancer.
// Routes are sorted by prefix length (longest first) so the
// most specific route always wins.
type Router struct {
	routes []*Route
}

// NewRouter creates a Router from a list of routes.
// It sorts routes by path prefix length (longest first) for
// longest-prefix matching and initializes a load balancer for each route.
func NewRouter(routes []*Route) (*Router, error) {
	// Sort by prefix length descending — longest prefix matches first.
	// This ensures "/api/auth/admin" matches before "/api/auth".
	sort.Slice(routes, func(i, j int) bool {
		return len(routes[i].PathPrefix) > len(routes[j].PathPrefix)
	})

	// Initialize load balancer for each route
	for _, route := range routes {
		lb, err := balancer.NewLoadBalancer(route.Backends)
		if err != nil {
			return nil, err
		}
		route.LB = lb
		if route.RateLimitRate >0 && route.RateLimitBurst >0{
			route.limiter=middleware.NewIPRateLimiter(route.RateLimitRate,route.RateLimitBurst)
			log.Printf("[ROUTER] Registered route: %s → %d backend(s) [rate limit: %.0f req/s]",

				route.PathPrefix, len(route.Backends), route.RateLimitRate)
		}else {
			log.Printf("[ROUTER] Registered route: %s → %d backend(s) [no rate limit]",
				route.PathPrefix, len(route.Backends))

		}

		log.Printf("[ROUTER] Registered route: %s → %d backend(s)", route.PathPrefix, len(route.Backends))
	}

	

	return &Router{routes: routes}, nil
}

// match finds the first route whose PathPrefix matches the given path.
// Because routes are sorted longest-first, this implements longest-prefix matching.
func (r *Router) match(path string) *Route {
	for _, route := range r.routes {
		if strings.HasPrefix(path, route.PathPrefix) {
			return route
		}
	}
	return nil
}

// ServeHTTP implements the http.Handler interface.
// It matches the request path to a route, optionally strips the prefix,
// and forwards the request to the route's load balancer.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	route := r.match(req.URL.Path)
	if route == nil {
		http.Error(w, "Not Found: no route matches this path", http.StatusNotFound)
		return
	}
	if route.limiter != nil{
		if !route.limiter.CheckHTTP(w,req) {
			return
		}	
	}

	// URL rewriting: strip the route prefix before forwarding to the backend.
	// e.g., /api/products/42 → /42 (the backend doesn't know about /api/products)
	if route.StripPrefix {
		originalPath := req.URL.Path
		req.URL.Path = strings.TrimPrefix(req.URL.Path, route.PathPrefix)
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		// Also rewrite RawPath if it's set (for encoded URLs)
		if req.URL.RawPath != "" {
			req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, route.PathPrefix)
			if req.URL.RawPath == "" {
				req.URL.RawPath = "/"
			}
		}

		log.Printf("[ROUTER] %s → %s (prefix stripped)", originalPath, req.URL.Path)
	}

	// Delegate to the route's load balancer
	route.LB.ServeHTTP(w, req)
}

// AllBackends returns all backend servers across all routes.
// Useful for health checking and metrics (which need the full server list).
func (r *Router) AllBackends() []*server.Server {
	var all []*server.Server
	for _, route := range r.routes {
		all = append(all, route.Backends...)
	}
	return all
}
