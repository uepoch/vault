package cacheutils

import (
	"github.com/hashicorp/vault/logical"
	"time"
)

const (
	IfMatchHeader = "If-Match"
	IfNoneMatchHeader = "If-None-Match"

	IfModifiedSinceHeader = "If-Modified-Since"
	IfUnmodifiedSinceHeader = "If-Unmodified-Since"

	EtagHeader = "Etag"
)


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

func checkTimeInHeader(headerMap map[string][]string, name string, value time.Time, before bool) (found, valid bool) {
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

// https://tools.ietf.org/html/rfc7232#section-6 Precedence
func CheckPreconditionalHeaders(req *logical.Request, entryHash string) (present, ok bool) {
	IfMatchfound, IfMatchValid := checkStringInHeader(req.Headers, IfMatchHeader, entryHash)
	//   1.  When recipient is the origin server and If-Match is present,
	//       evaluate the If-Match precondition:
	//
	//       *  if true, continue to step 3
	//
	//       *  if false, respond 412 (Precondition Failed)
	if IfMatchfound && !IfMatchValid {
		return true, false
	}

	IfUnmodifiedFound, IfUnmodifiedValid := checkStringInHeader(req.Headers, IfUnmodifiedSinceHeader, entryHash)

	//	2.  When recipient is the origin server, If-Match is not present, and
	//	If-Unmodified-Since is present, evaluate the If-Unmodified-Since
	//precondition:
	//
	//	*  if true, continue to step 3
	//
	//	*  if false, respond 412 (Precondition Failed)

	if !IfMatchfound && IfUnmodifiedFound && !IfUnmodifiedValid {
		return true, false
	}

	IfNoneMatchFound, IfNoneMatchValid := checkStringInHeader(req.Headers, IfNoneMatchHeader, entryHash)
	//    3.  When If-None-Match is present, evaluate the If-None-Match
	//       precondition:
	//
	//       *  if true, continue to step 5
	//
	//       *  if false for GET/HEAD, respond 304 (Not Modified)
	//
	//       *  if false for other methods, respond 412 (Precondition Failed)
	if IfNoneMatchFound && !IfNoneMatchValid {
		return true, false
	} else {
		IfModifiedSinceFound, IfModifiedSinceValid := checkStringInHeader(req.Headers, IfModifiedSinceHeader, entryHash)
		if !IfNoneMatchFound && IfModifiedSinceFound && (req.Operation == logical.ReadOperation || req.Operation == logical.ListOperation){

		}
	}



}


func HTTPEmptyResponse(code int) (*logical.Response) {
	return &logical.Response{
		Data: map[string]interface{}{
			logical.HTTPStatusCode: code,
		},
	}
}

