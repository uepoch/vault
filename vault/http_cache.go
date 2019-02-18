package vault

import (
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/helper/namespace"
	"strings"
	"github.com/armon/go-radix"
	"context"
	"github.com/hashicorp/vault/helper/cacheutils"
)

// CachePath checks if the given path support caching
func (r *Router) CachePath(ctx context.Context, path string) (bool, cacheutils.HTTPHashFunc) {
	ns, err := namespace.FromContext(ctx)
	if err != nil {
		return false, nil
	}

	adjustedPath := ns.Path + path

	r.l.RLock()
	mount, raw, ok := r.root.LongestPrefix(adjustedPath)
	r.l.RUnlock()
	if !ok {
		return false, nil
	}
	re := raw.(*routeEntry)

	// Trim to get remaining path
	remain := strings.TrimPrefix(adjustedPath, mount)

	// Check the rootPaths of this backend
	cachePaths := re.cachedPaths.Load().(*radix.Tree)
	match, raw, ok := cachePaths.LongestPrefix(remain)
	if !ok {
		return false, nil
	}

	prefixMatch := raw.(bool)

	var hashFunc = cacheutils.DefaultHashFunc
	if sp := re.backend.SpecialPaths(); sp != nil && sp.CacheablesPaths != nil {
		if f, ok := sp.CacheablesPaths[match]; ok {
			hashFunc = f
		}
	}

	// Handle the prefix match case
	if prefixMatch {
		return strings.HasPrefix(remain, match), hashFunc
	}

	// Handle the exact match case
	return match == remain, hashFunc
}

func (c *Core) handleCacheRequest(ctx context.Context, req *logical.Request) (*logical.Response, error, bool) {
	handleCache, hashFunc := c.router.CachePath(ctx, req.Path)

	if !handleCache {
		return nil, nil, false
	}

	switch req.Operation {
	case logical.DeleteOperation, logical.CreateOperation:
		return nil, nil, false
	case logical.ReadOperation, logical.ListOperation:

	case logical.UpdateOperation:
	}

}
