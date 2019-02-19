package vault

import (
	"context"
	"net/http"
	"strings"
	"github.com/armon/go-radix"
	"github.com/hashicorp/vault/helper/cacheutils"
	"github.com/hashicorp/vault/helper/namespace"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	"fmt"
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

func (s *SystemBackend) handleCacheRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error){
	pathRaw := d.Get("path")
	entryPath, ok := pathRaw.(string)
	if !ok {
		return nil, fmt.Errorf("path can't be converted into string")
	}
	if entryPath == "" {
		return nil, fmt.Errorf("path can't be empty")
	}
	ce, err := cacheutils.ReadCacheEntry(ctx, s.Core.PhysicalAccess(), entryPath)
	if err != nil {
		return nil, err
	}
	return &logical.Response{
		Data: map[string]interface{}{
			"etag": ce.Key,
		},
	}, nil
}

func (c *Core) handleCacheRequest(ctx context.Context, req *logical.Request) (*logical.Response, bool, error) {
	handleCache, hashFunc := c.router.CachePath(ctx, req.Path)

	if !handleCache {
		return nil, true, nil
	}

	physAccess := c.PhysicalAccess()

	storedCacheEntry, err := cacheutils.ReadCacheEntry(ctx, physAccess, req.Path)

	if err != nil {
		return nil, true, err
	}

	if storedCacheEntry != nil || req.Operation == logical.CreateOperation {
		 cacheutils.WriteCacheEntry(ctx, physAccess, req.Path, &cacheutils.CacheEntry{Key: hashFunc(req)})
		 return nil, true, nil
	}

	if req.Operation == logical.DeleteOperation {
		cacheutils.DeleteCacheEntry(ctx, physAccess, req.Path)
		return nil, true, nil
	}

	shouldContinue, valid, err := cacheutils.CheckPreconditionalHeaders(req, storedCacheEntry, false)
	if err != nil || shouldContinue {
		return nil, shouldContinue, err
	}

	httpRespCode := http.StatusPreconditionFailed
	if valid {
		httpRespCode = http.StatusNotModified
	}

	return &logical.Response{
		Data: map[string]interface{}{
			logical.HTTPStatusCode: httpRespCode,
		}}, false, nil
}
