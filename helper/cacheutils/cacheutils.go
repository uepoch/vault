package cacheutils

import (
	"net/http"
	"time"
	"context"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/physical"
	"github.com/hashicorp/vault/helper/namespace"
	"fmt"
	"path"
)

const (
	IfMatchHeader     = "If-Match"
	IfNoneMatchHeader = "If-None-Match"

	IfModifiedSinceHeader   = "If-Modified-Since"
	IfUnmodifiedSinceHeader = "If-Unmodified-Since"

	EtagHeader = "Etag"
)

type CacheEntry struct {
	Key  string
	Date time.Time
}

type HTTPHashFunc func(req *logical.Request) string

func DefaultHashFunc(req *logical.Request) string {
	return req.ID
}

func checkStringInHeader(headerMap map[string][]string, name, value string) (found, valid bool) {
	vs, ok := headerMap[name]
	if ok {
		for _, v := range vs {
			if v == value {
				return true, true
			}
		}
		return true, false
	}
	return false, false
}

func checkTimeInHeader(headerMap map[string][]string, name string, value time.Time, before bool) (found, valid bool, err error) {
	vs, ok := headerMap[name]
	if ok && len(vs) > 0 {
		v := vs[0]
		t, err := http.ParseTime(v)
		if err != nil {
			return true, false, err
		}
		delta := t.Sub(value)
		if before {
			return true, delta >= 0, nil
		}
		return true, delta <= 0, nil
	}
	return ok, false, nil
}

func ReadCacheEntry(ctx context.Context, p *physical.PhysicalAccess, entryPath string) (*CacheEntry, error){
	ns, err := namespace.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("can't determine namespace from context: %s", err.Error())
	}

	adjustedPath := path.Join("cache/", ns.Path, entryPath)

	pe, err := p.Get(ctx, adjustedPath)
	if err != nil {
		return nil, err
	}
	return &CacheEntry{Key: string(pe.Value)}, nil
}

func DeleteCacheEntry(ctx context.Context, p *physical.PhysicalAccess, entryPath string) error{
	ns, err := namespace.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("can't determine namespace from context: %s", err.Error())
	}

	adjustedPath := path.Join("cache/", ns.Path, entryPath)

	return p.Delete(ctx, adjustedPath)
}

func WriteCacheEntry(ctx context.Context, p *physical.PhysicalAccess, entryPath string, ce *CacheEntry) error{
	ns, err := namespace.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("can't determine namespace from context: %s", err.Error())
	}
	adjustedPath := path.Join("cache/", ns.Path, entryPath)

	return p.Put(ctx, &physical.Entry{Key:adjustedPath, Value:[]byte(ce.Key)})
}

// https://tools.ietf.org/html/rfc7232#section-6 Precedence
func CheckPreconditionalHeaders(req *logical.Request, entry *CacheEntry, checkTimeHeaders bool) (shouldPass, ok bool, err error) {
	isReadReq := req.Operation == logical.ReadOperation || req.Operation == logical.ListOperation
	IfMatchfound, IfMatchValid := checkStringInHeader(req.Headers, IfMatchHeader, entry.Key)
	//   1.  When recipient is the origin server and If-Match is present,
	//       evaluate the If-Match precondition:
	//
	//       *  if true, continue to step 3
	//
	//       *  if false, respond 412 (Precondition Failed)
	if IfMatchfound && !IfMatchValid {
		return false, false, nil
	}

	if checkTimeHeaders {
		IfUnmodifiedSinceFound, IfUnmodifiedSinceValid, err := checkTimeInHeader(req.Headers, IfUnmodifiedSinceHeader, entry.Date, true)
		//	2.  When recipient is the origin server, If-Match is not present, and
		//	If-Unmodified-Since is present, evaluate the If-Unmodified-Since
		//precondition:
		//
		//	*  if true, continue to step 3
		//
		//	*  if false, respond 412 (Precondition Failed)

		if (err != nil) || (!IfMatchfound && IfUnmodifiedSinceFound && !IfUnmodifiedSinceValid) {
			return false, false, err
		}
	}

	IfNoneMatchFound, IfNoneMatchValid := checkStringInHeader(req.Headers, IfNoneMatchHeader, entry.Key)
	//    3.  When If-None-Match is present, evaluate the If-None-Match
	//       precondition:
	//
	//       *  if true, continue to step 5
	//
	//       *  if false for GET/HEAD, respond 304 (Not Modified)
	//
	//       *  if false for other methods, respond 412 (Precondition Failed)
	if IfNoneMatchFound && !IfNoneMatchValid {
		return false, isReadReq, nil
	}

	if checkTimeHeaders {
		IfModifiedSinceFound, IfModifiedSinceValid, err := checkTimeInHeader(req.Headers, IfModifiedSinceHeader, entry.Date, false)
		// 4.    When the method is GET or HEAD, If-None-Match is not present, and
		//       If-Modified-Since is present, evaluate the If-Modified-Since
		//       precondition:
		//
		//       *  if true, continue to step 5
		//
		//       *  if false, respond 304 (Not Modified)
		if (err != nil) || (!IfNoneMatchFound && IfModifiedSinceFound && !IfModifiedSinceValid && isReadReq) {
			return false, true, err
		}
	}
	return true, true, nil
}

func HTTPEmptyResponse(code int) *logical.Response {
	return &logical.Response{
		Data: map[string]interface{}{
			logical.HTTPStatusCode: code,
		},
	}
}
