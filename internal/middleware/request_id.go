package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
)


type contextKey string

const RequestIDKey contextKey = "request_id"

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func RequestID(next http.Handler) http.Handler{
	return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		requestid:=r.Header.Get("X-Request-ID")
		if requestid==""{
			requestid=generateID()

	}
	ctx:= context.WithValue(r.Context(),RequestIDKey,requestid)
	r=r.WithContext(ctx)

	r.Header.Set("X-Request-ID",requestid)
	w.Header().Set("X-Request-ID",requestid)
	next.ServeHTTP(w,r)
	})
		
}

func GetRequestID(ctx context.Context) string{
	if id,ok:=ctx.Value(RequestIDKey).(string);ok{
		return id
	}
	return ""	

}


